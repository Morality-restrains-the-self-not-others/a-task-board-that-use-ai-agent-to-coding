/**
 * Pure functions: container management for TaskDetail.
 */
import { apiFetch } from '../../utils/apiUtils.js'
import {
  isServerRuntimeNotServingStatus,
  bootstrapFailureMessageFromCloneLogPayload,
} from '../../utils/commentLayerZtreeUiState.js'
import {
  flushPendingContainerHeartbeat,
  promoteContainerHeartbeatFromIdle,
} from './applyContainerHeartbeatSse.js'
import { appendCommentIdPath } from '../../utils/containerForwardCommentId.js'
import { getContainerCompute } from './containerComputeRequest.js'
import { resolveContainerUiContextCommentId } from './resolveContainerUiContextCommentId.js'
import { createApplyLayerGraphFromPayload } from './taskDetailLayerGraphPayload.js'
import { missingLayerPrHtmlUrls } from './layerGraphPrBackfillRefetch.js'

export { resolveContainerUiContextCommentId }

/**
 * OPT-20260903-002：记录每个 task 已为重拉 comments 覆盖过的 PR URL（canonical）。
 * 层图 GET 会幂等补写 git_pr 子评论；若并行 comments 响应早于补写提交，前端在
 * 看到「Feed 缺失的 pr_html_url」时补一次 fetchTaskDetail。同一 PR URL 只补一次，
 * 避免随层图心跳无限重拉。
 */
const prBackfillRefetchedUrlsByTask = new Map()

export async function probeContainerReachability(deps) {
  const { containerEndpointRegistered, containerHttpUnreachable, effectiveTenantId, effectiveWorkspaceId, effectiveTaskId, ensureServerRuntimeAllowsContainerLayerGraph, refreshLayerGraphFromServer, applyLayerGraphFromPayload, resetLayerGraphFetchBackoff, markContainerTransportOk } = deps
  if (!containerEndpointRegistered.value || !containerHttpUnreachable.value) { deps.clearContainerUnreachableProbeTimer(); return }
  const taskId = effectiveTaskId.value
  const tenantId = effectiveTenantId.value
  const workspaceId = effectiveWorkspaceId.value
  if (!taskId || !tenantId || !workspaceId) return
  try {
    const runtimeOk = await ensureServerRuntimeAllowsContainerLayerGraph(true)
    if (!runtimeOk) { deps.scheduleContainerUnreachableProbe(deps.containerUnreachableProbeBackoffMs); return }
    const response = await getContainerCompute(deps, 'container-layer-graph')
    if (response.ok) { resetLayerGraphFetchBackoff(); markContainerTransportOk(); const data = await response.json(); if (data && typeof data === 'object' && Array.isArray(data.layers)) { applyLayerGraphFromPayload(data) }; return }
  } catch (e) { console.error('容器可达性探测失败', e) }
  if (!containerEndpointRegistered.value || !containerHttpUnreachable.value) return
  deps.scheduleContainerUnreachableProbe(deps.containerUnreachableProbeBackoffMs)
  deps.containerUnreachableProbeBackoffMs = Math.min(deps.CONTAINER_UNREACHABLE_PROBE_BACKOFF_MAX_MS, Math.max(deps.CONTAINER_UNREACHABLE_PROBE_BACKOFF_INITIAL_MS, deps.containerUnreachableProbeBackoffMs * 2))
}

export async function fetchContainerTaskUiContext(deps, commentIdOverride) {
  const { effectiveTenantId, effectiveWorkspaceId, effectiveTaskId, containerEndpointRegistered, containerPageUrl, containerVscodeUrl, containerPageLinkPendingReveal, markContainerTransportOk, refreshLayerGraphFromServer, ensureServerRuntimeAllowsContainerLayerGraph } = deps
  try {
    const taskId = effectiveTaskId.value; const tenantId = effectiveTenantId.value; const workspaceId = effectiveWorkspaceId.value
    if (!taskId || !tenantId || !workspaceId) return
    const commentId = String(commentIdOverride || '').trim() || resolveContainerUiContextCommentId(deps)
    if (!commentId) {
      containerEndpointRegistered.value = false
      containerPageUrl.value = ''
      containerVscodeUrl.value = ''
      return
    }
    const apiPath = appendCommentIdPath(
      `/api/cloud/compute/container-task-ui-context/tenant_id/${tenantId}/workspace_id/${workspaceId}?task_id=${encodeURIComponent(String(taskId))}`,
      commentId,
    )
    const response = await apiFetch(apiPath, { credentials: 'include', headers: { Accept: 'application/json' } })
    if (!response.ok) return
    const data = await response.json()
    if (data.status !== 'success') return
    if (!data.container_endpoint_registered) { markContainerTransportOk() }
    const endpoint = !!data.container_endpoint_registered
    const pageUrl = typeof data.container_page_url === 'string' ? data.container_page_url : ''
    const vscodeUrl = typeof data.container_vscode_url === 'string' ? data.container_vscode_url : ''
    if (deps.layerPanelStore && typeof deps.layerPanelStore.patch === 'function') {
      deps.layerPanelStore.patch(commentId, {
        containerEndpointRegistered: endpoint,
        containerPageUrl: pageUrl,
        containerVscodeUrl: vscodeUrl,
        uiContextFetched: true,
      })
    }
    const activeId = resolveContainerUiContextCommentId(deps)
    if (!commentIdOverride || !activeId || commentId === activeId) {
      containerEndpointRegistered.value = endpoint
      containerPageUrl.value = pageUrl
      containerVscodeUrl.value = vscodeUrl
    }
    const serverUp = deps.isServerRunning?.value === true || deps.isServerStarting?.value === true
    if (
      (endpoint || serverUp) &&
      typeof deps.syncContainerHeartbeatAfterServingHint === 'function'
    ) {
      deps.syncContainerHeartbeatAfterServingHint()
    } else if ((endpoint || serverUp) && (!commentIdOverride || commentId === activeId)) {
      promoteContainerHeartbeatFromIdle(deps)
      flushPendingContainerHeartbeat(deps.containerHeartbeatSseBuffer, deps)
    }
    if (!containerPageLinkPendingReveal.value) {
      await refreshLayerGraphFromServer(true, { bypassBackoff: true, commentId })
    }
    if (!endpoint && typeof ensureServerRuntimeAllowsContainerLayerGraph === 'function') {
      await ensureServerRuntimeAllowsContainerLayerGraph(true)
    }
  } catch (error) { console.error('获取容器任务 UI 上下文失败:', error) }
}

/**
 * 拉取容器引导批量克隆日志，供任务详情在 SSE 进度 Map 清空后恢复子仓状态。
 * @param {{
 *   effectiveTenantId: import('vue').Ref,
 *   effectiveWorkspaceId: import('vue').Ref,
 *   effectiveTaskId: import('vue').Ref,
 *   containerEndpointRegistered: import('vue').Ref,
 *   onBootstrapCloneLogUpdate: (payload: { text: string, segments?: unknown }) => void,
 *   onBootstrapCloneLogFailure?: (message: string) => void,
 * }} deps
 */
export async function fetchContainerBootstrapCloneLog(deps) {
  const {
    effectiveTenantId,
    effectiveWorkspaceId,
    effectiveTaskId,
    containerEndpointRegistered,
    onBootstrapCloneLogUpdate,
    onBootstrapCloneLogFailure,
  } = deps
  if (!containerEndpointRegistered.value) return
  const taskId = effectiveTaskId.value
  const tenantId = effectiveTenantId.value
  const workspaceId = effectiveWorkspaceId.value
  if (!taskId || !tenantId || !workspaceId) return
  if (typeof onBootstrapCloneLogUpdate !== 'function') return
  try {
    const response = await getContainerCompute(deps, 'container-bootstrap-clone-log')
    if (!response.ok) return
    const data = await response.json().catch(() => null)
    if (!data || typeof data !== 'object') return
    const text = typeof data.text === 'string' ? data.text : ''
    const segments = Array.isArray(data.segments) ? data.segments : null
    if (text.trim() || (segments && segments.length)) {
      onBootstrapCloneLogUpdate({ text, segments })
    }
    const failMsg = bootstrapFailureMessageFromCloneLogPayload(data)
    if (failMsg && typeof onBootstrapCloneLogFailure === 'function') {
      onBootstrapCloneLogFailure(failMsg)
    }
  } catch (error) {
    console.error('拉取引导克隆日志失败:', error)
  }
}

export async function ensureServerRuntimeAllowsContainerLayerGraph(bypassCache, deps) {
  const { containerEndpointRegistered, containerHeartbeatStatus, containerHttpUnreachable, effectiveTenantId, effectiveWorkspaceId, effectiveTaskId } = deps
  if (containerEndpointRegistered.value && containerHeartbeatStatus.value === 'connected' && !containerHttpUnreachable.value) { deps.serverRuntimeLayerGraphGateCheckedAt = Date.now(); deps.serverRuntimeLayerGraphGateAllowed = true; return true }
  const now = Date.now()
  if (!bypassCache && now - deps.serverRuntimeLayerGraphGateCheckedAt < deps.SERVER_RUNTIME_LAYER_GRAPH_GATE_MS) { return deps.serverRuntimeLayerGraphGateAllowed }
  const taskId = effectiveTaskId.value; const tenantId = effectiveTenantId.value; const workspaceId = effectiveWorkspaceId.value
  const commentId = resolveContainerUiContextCommentId(deps)
  if (!taskId || !tenantId || !workspaceId || !commentId) return false
  try {
    const response = await apiFetch(appendCommentIdPath(
      `/api/cloud/compute/server-runtime-status/tenant_id/${tenantId}/workspace_id/${workspaceId}?task_id=${encodeURIComponent(String(taskId))}`,
      commentId,
    ), { credentials: 'include', headers: { Accept: 'application/json' } })
    const data = await response.json().catch(() => null)
    deps.serverRuntimeLayerGraphGateCheckedAt = Date.now()
    if (!response.ok || !data || typeof data !== 'object') { deps.serverRuntimeLayerGraphGateAllowed = true; return true }
    const rs = String(data.runtime_status ?? '').trim().toLowerCase()
    if (!rs || rs === 'running' || data.mock === true) {
      deps.serverRuntimeLayerGraphGateAllowed = true
      if (rs === 'running' && typeof deps.onServerRuntimeServing === 'function') {
        deps.onServerRuntimeServing(String(data.runtime_status ?? 'Running'))
      }
      return true
    }
    const transitional = new Set(['initializing', 'starting', 'pending'])
    if (transitional.has(rs)) { deps.serverRuntimeLayerGraphGateAllowed = false; return false }
    if (isServerRuntimeNotServingStatus(rs)) {
      deps.serverRuntimeLayerGraphGateAllowed = false
      if (typeof deps.onServerRuntimeNotServing === 'function') {
        deps.onServerRuntimeNotServing(rs)
      }
      return false
    }
    deps.serverRuntimeLayerGraphGateAllowed = true; return true
  } catch (e) { console.error('查询服务器运行状态（层图门控）失败:', e); deps.serverRuntimeLayerGraphGateCheckedAt = Date.now(); deps.serverRuntimeLayerGraphGateAllowed = true; return true }
}

export async function refreshLayerGraphFromServer(force, opts, deps) {
  const { containerEndpointRegistered, containerLayerGraphAuthInvalid, effectiveTenantId, effectiveWorkspaceId, effectiveTaskId, ensureServerRuntimeAllowsContainerLayerGraph, applyLayerGraphFromPayload, registerLayerGraphFetchFailure, resetLayerGraphFetchBackoff, markContainerTransportOk, markContainerTransportUnreachableIfForwardingFailed, selectedLayerGraphNode, layerGraphZNodes, onLayerGraphNodeSelect, layerGraphSnapshot, layerGraphSnapshotHasContent, layerGraphSnapshotHasActiveJob, layerGraphSelectionDismissedByUser } = deps
  const bypassBackoff = Boolean(opts && opts.bypassBackoff)
  const commentId = String(opts?.commentId || '').trim() || resolveContainerUiContextCommentId(deps)
  const scopedDeps = { ...deps, commentId }
  const slotEndpoint = commentId && deps.layerPanelStore
    ? Boolean(deps.layerPanelStore.get(commentId).containerEndpointRegistered)
    : false
  const hasLiveEndpoint = Boolean(containerEndpointRegistered.value || slotEndpoint)
  if (!force && containerLayerGraphAuthInvalid.value) return
  if (!bypassBackoff && Date.now() < deps.layerGraphFetchBackoffUntil) return
  const slotSnap = commentId && deps.layerPanelStore ? deps.layerPanelStore.get(commentId).snapshot : null
  const skipIfIdle = (snap) => snap != null && layerGraphSnapshotHasContent(snap) && !layerGraphSnapshotHasActiveJob(snap)
  if (!force && skipIfIdle(slotSnap || deps.layerGraphSnapshot?.value)) return
  if (commentId && deps.layerPanelStore?.get(commentId).hydrateInFlight) return
  if (!commentId && deps.layerGraphHydrateInFlight) return
  const taskId = effectiveTaskId.value; const tenantId = effectiveTenantId.value; const workspaceId = effectiveWorkspaceId.value
  if (!taskId || !tenantId || !workspaceId) return
  if (commentId && deps.layerPanelStore) deps.layerPanelStore.patch(commentId, { hydrateInFlight: true, refreshing: true })
  else deps.layerGraphHydrateInFlight = true
  try {
    const bypassRuntimeCache = Boolean(force) || Boolean(opts && opts.bypassRuntimeGateCache)
    if (hasLiveEndpoint) {
      // runtime gate 只用于 live 转发；失败不得跳过 SaaS DB hydrate（关容器/gate 未就绪仍要出树）
      await ensureServerRuntimeAllowsContainerLayerGraph(bypassRuntimeCache)
    }
    const response = await getContainerCompute(scopedDeps, 'container-layer-graph')
    if (!response.ok) {
      registerLayerGraphFetchFailure()
      if (response.status === 502 || response.status === 503) {
        const t = (await response.text().catch(() => '')) || ''
        markContainerTransportUnreachableIfForwardingFailed(response.status, t)
      }
      return
    }
    const data = await response.json()
    if (data.detail && !Array.isArray(data.layers)) return
    const activeId = resolveContainerUiContextCommentId(deps)
    if (commentId && deps.layerPanelStore) {
      const applyToSlot = createApplyLayerGraphFromPayload({
        layerGraphSnapshot: deps.layerPanelStore.refsFor(commentId).layerGraphSnapshot,
        containerLayerGraphAuthInvalid: containerLayerGraphAuthInvalid,
      })
      applyToSlot(data)
    }
    if (!commentId || !activeId || commentId === activeId) {
      applyLayerGraphFromPayload(data)
    }
    resetLayerGraphFetchBackoff(); markContainerTransportOk()
    maybeRefetchCommentsForPrBackfill(deps, taskId, data)
    const userDismissed = Boolean(layerGraphSelectionDismissedByUser?.value)
    const snapAfter = commentId && deps.layerPanelStore
      ? deps.layerPanelStore.get(commentId).snapshot
      : deps.layerGraphSnapshot?.value
    if (selectedLayerGraphNode.value === null && !userDismissed && snapAfter) {
      await deps.nextTick()
      const layers = snapAfter.layers
      if (Array.isArray(layers) && layers.length > 0) {
        const sortedLayers = [...layers].sort((a, b) => { const tA = Date.parse(a?.created_at || '') || 0; const tB = Date.parse(b?.created_at || '') || 0; return tB - tA })
        const newestLayer = sortedLayers[0]; const newestLayerId = newestLayer?.layer_id ? String(newestLayer.layer_id).trim() : ''
        if (newestLayerId) {
          const zNode = (layerGraphZNodes.value || []).find((n) => n.nodeKind === 'layer' && String(n.layerId || '').trim() === newestLayerId)
          if (zNode) { onLayerGraphNodeSelect(zNode, commentId) }
        }
      }
    }
  } catch (error) { registerLayerGraphFetchFailure(); console.error('拉取容器层级快照失败:', error) }
  finally {
    if (commentId && deps.layerPanelStore) deps.layerPanelStore.patch(commentId, { hydrateInFlight: false, refreshing: false })
    if (deps.layerGraphHydrateInFlight && (!commentId || commentId === resolveContainerUiContextCommentId(deps))) {
      deps.layerGraphHydrateInFlight = false
    }
  }
}

// OPT-20260903-002：层图 GET 返回的快照出现评论 Feed 尚未带回的 pr_html_url 时，
// 补一次 comments 拉取（taskCloudService 会在同一次 GET 内幂等补写 git_pr 子评论，
// 前端并行 comments 请求可能早于该补写提交）。同一 task + PR URL 只触发一次。
function maybeRefetchCommentsForPrBackfill(deps, taskId, data) {
  if (typeof deps.onLayerPrBackfillDetected !== 'function') return
  if (!data || !Array.isArray(data.layers)) return
  const existingUrls = typeof deps.getExistingGitPrHtmlUrls === 'function'
    ? deps.getExistingGitPrHtmlUrls()
    : []
  const missing = missingLayerPrHtmlUrls(data.layers, existingUrls)
  if (!missing.length) return
  const taskKey = String(taskId || '')
  const already = prBackfillRefetchedUrlsByTask.get(taskKey) || new Set()
  const fresh = missing.filter((u) => !already.has(u))
  if (!fresh.length) return
  prBackfillRefetchedUrlsByTask.set(taskKey, new Set([...already, ...fresh]))
  deps.onLayerPrBackfillDetected()
}

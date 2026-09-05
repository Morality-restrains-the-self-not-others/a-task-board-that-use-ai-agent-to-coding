import { apiFetch } from '../../utils/apiUtils.js'
import { extractTraceId } from '../../utils/traceId.js'
import { resolveApiErrorMessage } from '../../utils/workPanelFormat.js'
import { getContainerComputeLegacyTask, containerComputeFuncFirstUrl } from './containerComputeRequest.js'
import {
  ztreeLogCreatedAtMs,
  normalizeLayerChangesPayload,
  mergeLayerChangesPage,
  layerChangesContentFingerprint,
} from './taskDetailZTreeExecLogDerived.js'

/**
 * 层变动「刷新 / 续拉 / 预取」控制面（OPT-20260815-027 拆分）。
 * 纯 ref 操作：接收外层工厂的 refs/依赖，返回可注入回状态对象的控制器。
 * @param {object} ctx
 * @param {object} ctx.deps 外层 useTaskDetail deps（用于 resolveContainerUiContextCommentId）
 * @param {import('vue').Ref} ctx.layerChangesByLayerId
 * @param {import('vue').Ref} ctx.layerChangesRefreshBusy
 * @param {import('vue').Ref} ctx.layerChangesRefreshError
 * @param {import('vue').Ref} ctx.layerChangesRefreshErrorTraceId
 * @param {import('vue').Ref} ctx.layerChangesLoadMoreBusy
 * @param {import('vue').Ref} ctx.zTreeLogTargets
 * @param {import('vue').Ref} ctx.selectedLayerGraphFileTreeLayerId
 * @param {import('vue').Ref} ctx.layerGraphSnapshot
 * @param {import('vue').Ref} ctx.effectiveTenantId
 * @param {import('vue').Ref} ctx.effectiveWorkspaceId
 * @param {import('vue').Ref} ctx.effectiveTaskId
 * @param {import('vue').Ref} ctx.containerEndpointRegistered
 * @param {(path: string) => void} [ctx.bumpProjectFileTreeRefresh]
 * @param {(deps: object) => string} ctx.resolveContainerUiContextCommentId
 */
export function createLayerChangesController(ctx) {
  const {
    deps,
    layerChangesByLayerId,
    layerChangesRefreshBusy,
    layerChangesRefreshError,
    layerChangesRefreshErrorTraceId,
    layerChangesLoadMoreBusy,
    zTreeLogTargets,
    selectedLayerGraphFileTreeLayerId,
    layerGraphSnapshot,
    effectiveTenantId,
    effectiveWorkspaceId,
    effectiveTaskId,
    containerEndpointRegistered,
    bumpProjectFileTreeRefresh,
    resolveContainerUiContextCommentId,
  } = ctx

  const layerChangesPrefetchInFlight = new Set()

  function ingestLayerChangesFromExecutionPayload(body, fetchTraceId = '') {
    if (!body || typeof body !== 'object') return
    const normalized = normalizeLayerChangesPayload(body.layer_changes)
    if (!normalized) return
    const tid = String(fetchTraceId || '').trim()
    if (tid) normalized.fetch_trace_id = tid
    const prev = layerChangesByLayerId.value?.[normalized.layer_id]
    // 执行日志首屏分页刷新时，保留用户已滚动加载的后续页，避免轮询冲掉续拉结果
    let next = normalized
    if (
      prev &&
      prev.layer_id === normalized.layer_id &&
      Array.isArray(prev.changes) &&
      prev.changes.length > normalized.changes.length
    ) {
      const firstPaths = new Set(normalized.changes.map((c) => c.path))
      const extras = prev.changes.filter((c) => c?.path && !firstPaths.has(c.path))
      if (extras.length) {
        const changes = [...normalized.changes, ...extras]
        next = {
          ...normalized,
          changes,
          change_count: Math.max(
            Number(normalized.change_count) || 0,
            Number(prev.change_count) || 0,
            changes.length,
          ),
          has_more: Boolean(normalized.has_more) || Boolean(prev.has_more),
          next_offset: Math.max(
            Number(normalized.next_offset) || 0,
            Number(prev.next_offset) || 0,
            changes.length,
          ),
          truncated: Boolean(normalized.truncated) || Boolean(prev.truncated),
          fetch_trace_id: tid || prev.fetch_trace_id || '',
        }
      }
    }
    const prevFp = layerChangesContentFingerprint(prev)
    const nextFp = layerChangesContentFingerprint(next)
    layerChangesByLayerId.value = {
      ...layerChangesByLayerId.value,
      [normalized.layer_id]: next,
    }
    // activeJobExecLogPoller 每 2s 拉执行日志并走此路径；相同变动集不得刷文件树
    if (prevFp !== nextFp) {
      bumpProjectFileTreeRefresh?.()
    }
  }

  async function refreshSelectedLayerChanges() {
    const tenantId = effectiveTenantId.value
    const workspaceId = effectiveWorkspaceId.value
    const taskId = effectiveTaskId.value
    const jobId = zTreeLogTargets.value.jobId
    layerChangesRefreshError.value = ''
    layerChangesRefreshErrorTraceId.value = ''
    if (!tenantId || !workspaceId || !taskId) {
      layerChangesRefreshError.value = '缺少任务上下文'
      return
    }
    if (!containerEndpointRegistered.value) {
      layerChangesRefreshError.value = '容器未就绪'
      return
    }
    if (!jobId) {
      layerChangesRefreshError.value = '当前节点没有可查询的执行任务'
      return
    }
    const commentId = resolveContainerUiContextCommentId(deps)
    if (!commentId) {
      layerChangesRefreshError.value = '缺少评论ID'
      return
    }
    layerChangesRefreshBusy.value = true
    try {
      const lid = zTreeLogTargets.value.layerId || ''
      const qs = new URLSearchParams({
        job_id: jobId,
        after_step: '0',
        limit: '1',
      })
      if (lid) qs.set('layer_id', lid)
      const response = await apiFetch(
        containerComputeFuncFirstUrl(
          tenantId,
          workspaceId,
          taskId,
          'container-job-execution-log',
          commentId,
          qs.toString(),
        ),
        {
          credentials: 'include',
          headers: { Accept: 'application/json' },
        },
      )
      const body = await response.json().catch(() => null)
      if (!response.ok) {
        layerChangesRefreshError.value = resolveApiErrorMessage(body, {
          fallback: '拉取执行日志失败',
          httpStatus: response.status,
        })
        layerChangesRefreshErrorTraceId.value =
          extractTraceId(response) || extractTraceId(body) || ''
        return
      }
      if (!body || typeof body !== 'object') {
        layerChangesRefreshError.value = '解析响应失败'
        layerChangesRefreshErrorTraceId.value = extractTraceId(response) || ''
        return
      }
      const normalized = normalizeLayerChangesPayload(body.layer_changes)
      if (!normalized) {
        layerChangesRefreshError.value = '响应中暂无有效的文件变动数据'
        layerChangesRefreshErrorTraceId.value =
          extractTraceId(response) || extractTraceId(body) || ''
        return
      }
      ingestLayerChangesFromExecutionPayload(
        body,
        extractTraceId(response) || extractTraceId(body) || '',
      )
    } catch (e) {
      layerChangesRefreshError.value = '网络错误，请重试'
      layerChangesRefreshErrorTraceId.value = extractTraceId(e) || ''
    } finally {
      layerChangesRefreshBusy.value = false
    }
  }

  async function loadMoreSelectedLayerChanges() {
    const tenantId = effectiveTenantId.value
    const workspaceId = effectiveWorkspaceId.value
    const taskId = effectiveTaskId.value
    const lid = selectedLayerGraphFileTreeLayerId.value
    if (!tenantId || !workspaceId || !taskId || !lid) return
    if (!containerEndpointRegistered.value) return
    if (layerChangesLoadMoreBusy.value || layerChangesRefreshBusy.value) return
    const prev = layerChangesByLayerId.value[lid]
    if (!prev || typeof prev !== 'object') return
    const loaded = Array.isArray(prev.changes) ? prev.changes.length : 0
    const total = Number(prev.change_count)
    const hasMore =
      Boolean(prev.has_more) || (Number.isFinite(total) && loaded < total)
    if (!hasMore) return
    const commentId = resolveContainerUiContextCommentId(deps)
    if (!commentId) {
      layerChangesRefreshError.value = '缺少评论ID'
      return
    }
    const offset =
      prev.next_offset != null && Number.isFinite(Number(prev.next_offset))
        ? Math.floor(Number(prev.next_offset))
        : loaded
    layerChangesLoadMoreBusy.value = true
    layerChangesRefreshError.value = ''
    layerChangesRefreshErrorTraceId.value = ''
    try {
      const qs = new URLSearchParams({
        layer_id: lid,
        offset: String(Math.max(0, offset)),
        limit: '100',
      })
      const response = await getContainerComputeLegacyTask(
        deps,
        'container-layer-diff-parent-files',
        qs.toString(),
      )
      const body = await response.json().catch(() => null)
      if (!response.ok) {
        layerChangesRefreshError.value = resolveApiErrorMessage(body, {
          fallback: '加载更多变动文件失败',
          httpStatus: response.status,
        })
        layerChangesRefreshErrorTraceId.value =
          extractTraceId(response) || extractTraceId(body) || ''
        return
      }
      const page = normalizeLayerChangesPayload(body)
      if (!page) {
        layerChangesRefreshError.value = '加载更多变动文件失败：响应无效'
        layerChangesRefreshErrorTraceId.value =
          extractTraceId(response) || extractTraceId(body) || ''
        return
      }
      page.fetch_trace_id =
        extractTraceId(response) || extractTraceId(body) || page.fetch_trace_id || ''
      const merged = mergeLayerChangesPage(prev, page)
      layerChangesByLayerId.value = {
        ...layerChangesByLayerId.value,
        [lid]: merged,
      }
    } catch (e) {
      layerChangesRefreshError.value = '网络错误，加载更多失败'
      layerChangesRefreshErrorTraceId.value = extractTraceId(e) || ''
    } finally {
      layerChangesLoadMoreBusy.value = false
    }
  }

  async function prefetchLayerChangeSummariesForDirtyLayers() {
    const tenantId = effectiveTenantId.value
    const workspaceId = effectiveWorkspaceId.value
    const taskId = effectiveTaskId.value
    const snap = layerGraphSnapshot.value
    if (!tenantId || !workspaceId || !taskId || !snap || !containerEndpointRegistered.value) {
      return
    }
    const commentId = resolveContainerUiContextCommentId(deps)
    if (!commentId) return
    const jobs = Array.isArray(snap.jobs) ? snap.jobs : []
    const layers = Array.isArray(snap.layers) ? snap.layers : []

    const latestNonCloneJobIdForLayer = (layerId) => {
      const vis = jobs.filter(
        (j) => j && String(j.layer_id || '') === layerId && j.command_kind !== 'clone',
      )
      vis.sort((a, b) => ztreeLogCreatedAtMs(a.created_at) - ztreeLogCreatedAtMs(b.created_at))
      return vis.length ? String(vis[vis.length - 1].id || '').trim() : ''
    }

    const tasks = []
    for (const layer of layers) {
      const lid = layer?.layer_id != null ? String(layer.layer_id).trim() : ''
      if (!lid || layerChangesByLayerId.value[lid]) continue
      const jobId = latestNonCloneJobIdForLayer(lid)
      if (!jobId || layerChangesPrefetchInFlight.has(jobId)) continue
      const jobRow = jobs.find((j) => j && String(j.id || '').trim() === jobId)
      const jobTerminal =
        jobRow &&
        (jobRow.status === 'completed' ||
          jobRow.status === 'interrupted' ||
          jobRow.status === 'error')
      const shouldPrefetch = jobTerminal && jobRow?.command_kind !== 'clone'
      if (!shouldPrefetch) continue
      layerChangesPrefetchInFlight.add(jobId)
      tasks.push(
        (async () => {
          try {
            const qs = new URLSearchParams({
              job_id: jobId,
              after_step: '0',
              limit: '1',
              layer_id: lid,
            })
            const response = await apiFetch(
              containerComputeFuncFirstUrl(
                tenantId,
                workspaceId,
                taskId,
                'container-job-execution-log',
                commentId,
                qs.toString(),
              ),
              {
                credentials: 'include',
                headers: { Accept: 'application/json' },
              },
            )
            if (!response.ok) return
            const body = await response.json().catch(() => null)
            ingestLayerChangesFromExecutionPayload(
              body,
              extractTraceId(response) || extractTraceId(body) || '',
            )
          } catch {
            /* 预取失败不阻断页面 */
          } finally {
            layerChangesPrefetchInFlight.delete(jobId)
          }
        })(),
      )
    }
    if (tasks.length) {
      await Promise.all(tasks)
    }
  }

  return {
    layerChangesPrefetchInFlight,
    ingestLayerChangesFromExecutionPayload,
    refreshSelectedLayerChanges,
    loadMoreSelectedLayerChanges,
    prefetchLayerChangeSummariesForDirtyLayers,
  }
}

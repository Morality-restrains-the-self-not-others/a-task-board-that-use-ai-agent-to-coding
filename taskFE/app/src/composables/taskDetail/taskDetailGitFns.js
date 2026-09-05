/**
 * Pure functions: Git identity management for TaskDetail.
 */
import { humanizeRequestErrorMessage } from '../../utils/requestErrorDisplay.js'
import { getContainerCompute, postContainerCompute } from './containerComputeRequest.js'

export async function fetchLayerRepoGitIdentities(deps) {
  const { effectiveTenantId, effectiveWorkspaceId, effectiveTaskId, containerEndpointRegistered, selectedLayerGraphFileTreeLayerId, layerRepoGitIdentityLoading, layerRepoGitIdentityFetchError, layerRepoGitIdentityRows } = deps
  const tenantId = effectiveTenantId.value; const workspaceId = effectiveWorkspaceId.value; const taskId = effectiveTaskId.value
  if (!tenantId || !workspaceId || !taskId) { layerRepoGitIdentityFetchError.value = '缺少任务上下文，无法拉取'; return }
  if (!containerEndpointRegistered.value) { layerRepoGitIdentityFetchError.value = '容器未注册，无法拉取'; return }
  const layerId = selectedLayerGraphFileTreeLayerId.value
  if (!layerId) { layerRepoGitIdentityFetchError.value = '请先在左侧层级图选中可写层或任务节点'; return }
  layerRepoGitIdentityLoading.value = true; layerRepoGitIdentityFetchError.value = ''
  try {
    const q = `layer_id=${encodeURIComponent(layerId)}`
    const response = await getContainerCompute(deps, 'container-layer-git-repo-identities', q)
    const data = await response.json().catch(() => ({}))
    if (!response.ok) { const parts = [data.detail || '拉取容器内 Git 身份失败']; if (data?.upstream?.detail) parts.push(String(data.upstream.detail)); throw new Error(parts.join(' | ')) }
    layerRepoGitIdentityRows.value = Array.isArray(data.repos) ? data.repos : []
  } catch (error) { console.error('拉取层级仓库 Git 身份失败:', error); layerRepoGitIdentityRows.value = []; layerRepoGitIdentityFetchError.value = humanizeRequestErrorMessage(error.message || '拉取失败') }
  finally { layerRepoGitIdentityLoading.value = false }
}

export async function syncPerRepoGitIdentitiesToContainer(deps) {
  const { effectiveTenantId, effectiveWorkspaceId, effectiveTaskId, containerEndpointRegistered, containerHeartbeatStatus, selectedLayerGraphFileTreeLayerId, containerPageUrl, taskRepoRows, repoCloneIdentityIdForUrl, perRepoGitIdentitySyncing, perRepoGitIdentitySyncError, containerLayerGraphAuthInvalid, fetchLayerRepoGitIdentities } = deps
  const tenantId = effectiveTenantId.value; const workspaceId = effectiveWorkspaceId.value; const taskId = effectiveTaskId.value
  perRepoGitIdentitySyncError.value = ''
  if (!tenantId || !workspaceId || !taskId) { perRepoGitIdentitySyncError.value = '缺少任务上下文，无法同步'; return }
  if (!containerEndpointRegistered.value) { perRepoGitIdentitySyncError.value = '容器未注册，无法同步'; return }
  const layerId = selectedLayerGraphFileTreeLayerId.value
  if (!layerId) { perRepoGitIdentitySyncError.value = '请先在左侧层级图选中可写层或任务节点'; return }
  if (containerHeartbeatStatus.value === 'disconnected') { perRepoGitIdentitySyncError.value = '容器已断开连接'; return }
  const repos = []
  for (const row of taskRepoRows.value) { const u = String(row?.url || '').trim(); if (!u) continue; const identityId = String(repoCloneIdentityIdForUrl(u) || '').trim(); if (!identityId) continue; repos.push({ repo_url: u, identity_id: identityId }) }
  if (repos.length === 0) { perRepoGitIdentitySyncError.value = '请先在各仓库的「Git 提交身份」下拉框中至少选择一个身份'; return }
  perRepoGitIdentitySyncing.value = true
  try {
    const response = await postContainerCompute(deps, 'container-layer-git-repo-identities-sync', {
      layer_id: layerId, repos, container_page_url: containerPageUrl.value || '',
    })
    const data = await response.json().catch(() => ({}))
    if (!response.ok) { const parts = [data.detail || '同步各仓库 Git 身份失败']; const us = data?.upstream?.status ?? data?.http_status; const uu = data?.upstream?.url; const ud = data?.upstream?.detail; if (us) parts.push(`下游HTTP: ${us}`); if (uu) parts.push(`下游URL: ${uu}`); if (ud && ud !== data.detail) parts.push(`下游详情: ${ud}`); throw new Error(parts.join(' | ')) }
    if (data.ok === false && Array.isArray(data.results)) { const bad = data.results.filter((r) => r && !r.ok); if (bad.length) { perRepoGitIdentitySyncError.value = bad.map((r) => `${r.repo_match_key || '?'}: ${r.detail || '失败'}`).join('；') } }
    containerLayerGraphAuthInvalid.value = false; await fetchLayerRepoGitIdentities()
  } catch (error) { console.error('同步各仓库 Git 身份失败:', error); perRepoGitIdentitySyncError.value = humanizeRequestErrorMessage(error.message || '同步失败') }
  finally { perRepoGitIdentitySyncing.value = false }
}

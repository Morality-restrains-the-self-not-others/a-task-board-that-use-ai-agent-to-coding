/**
 * Repo-clone operations for TaskDetail.
 * Split from taskDetailFetchFns.js (OPT-20260816-005): reclone / clone-identity
 * 变更 / 默认克隆身份自动应用。
 */
import { apiFetch } from '../../utils/apiUtils.js'
import { extractTraceId } from '../../utils/traceId.js'
import { humanizeRequestErrorMessage } from '../../utils/requestErrorDisplay.js'
import { resolveDefaultCompanyGitIdentityId } from '../../utils/taskDetailBranchAndRepoUtils.js'
import { appendCommentIdPath, withCommentIdBody } from '../../utils/containerForwardCommentId.js'

function commentIdFromRecloneDeps(repoUrlOrPayload, deps) {
  if (repoUrlOrPayload && typeof repoUrlOrPayload === 'object') {
    const fromPayload = String(repoUrlOrPayload.commentId || repoUrlOrPayload.comment_id || '').trim()
    if (fromPayload) return fromPayload
  }
  const dep = deps?.containerForwardCommentId
  if (dep && typeof dep === 'object' && 'value' in dep) {
    return String(dep.value || '').trim()
  }
  return String(dep || '').trim()
}

export async function onRepoReclone(repoUrlOrPayload, deps) {
  const { effectiveTenantId, effectiveWorkspaceId, effectiveTaskId, recloneLoadingByUrl, recloneStatusByUrl, recloneErrorTraceIdByUrl, repoRecloneGlobalLoading, repoCloneIdentityIdForUrl } = deps
  const tid = effectiveTenantId.value; const wid = effectiveWorkspaceId.value; const taskId = effectiveTaskId.value
  let u = ''
  let parentRepoUrl = ''
  let cloneAlias = ''
  if (repoUrlOrPayload && typeof repoUrlOrPayload === 'object') {
    u = String(repoUrlOrPayload.repoUrl || repoUrlOrPayload.repo_url || '').trim()
    parentRepoUrl = String(repoUrlOrPayload.parentRepoUrl || repoUrlOrPayload.parent_repo_url || '').trim()
    cloneAlias = String(repoUrlOrPayload.cloneAlias || repoUrlOrPayload.clone_alias || '').trim()
  } else {
    u = String(repoUrlOrPayload || '').trim()
  }
  if (!tid || !wid || !taskId || !u) return
  const setRecloneTrace = (tidVal) => {
    if (!recloneErrorTraceIdByUrl) return
    recloneErrorTraceIdByUrl.value = { ...recloneErrorTraceIdByUrl.value, [u]: String(tidVal || '').trim() }
  }
  const commentId = commentIdFromRecloneDeps(repoUrlOrPayload, deps)
  if (!commentId) {
    recloneStatusByUrl.value = { ...recloneStatusByUrl.value, [u]: '缺少评论ID' }
    setRecloneTrace('')
    return
  }
  // 有 comment_id 即走评论级 CSC 转发，禁止用任务页单例 containerEndpointRegistered 短路。
  // 引导克隆失败时常尚未登记业务端点，但容器已在跑；Cloud handleRepoReclone 按 comment_id 路由。
  recloneLoadingByUrl.value = { ...recloneLoadingByUrl.value, [u]: true }; recloneStatusByUrl.value = { ...recloneStatusByUrl.value, [u]: '' }; setRecloneTrace(''); repoRecloneGlobalLoading.value = true
  try {
    const identityId =
      repoCloneIdentityIdForUrl(u) ||
      (parentRepoUrl ? repoCloneIdentityIdForUrl(parentRepoUrl) : '') ||
      ''
    const body = withCommentIdBody({
      repo_url: u,
      ...(identityId ? { repo_clone_git_identity_id: identityId } : {}),
      ...(parentRepoUrl ? { parent_repo_url: parentRepoUrl } : {}),
      ...(cloneAlias ? { clone_alias: cloneAlias } : {}),
    }, commentId)
    const apiPath = appendCommentIdPath(
      `/api/cloud/repo-reclone/tenant_id/${tid}/workspace_id/${wid}/task_id/${taskId}`,
      commentId,
    )
    const response = await apiFetch(apiPath, { method: 'POST', credentials: 'include', headers: { 'Content-Type': 'application/json', Accept: 'application/json' }, body: JSON.stringify(body) })
    const data = await response.json().catch(() => ({}))
    const tidFromReq = extractTraceId(response) || extractTraceId(data)
    if (!response.ok) {
      const detail = typeof data.detail === 'string' ? data.detail : (data.message || '重新克隆失败')
      recloneStatusByUrl.value = { ...recloneStatusByUrl.value, [u]: detail }
      setRecloneTrace(tidFromReq)
    }
    else if (data.async) { recloneStatusByUrl.value = { ...recloneStatusByUrl.value, [u]: '' }; setRecloneTrace('') }
    else { recloneStatusByUrl.value = { ...recloneStatusByUrl.value, [u]: 'ok' }; setRecloneTrace('') }
  } catch (error) {
    console.error('重新克隆请求失败:', error)
    recloneStatusByUrl.value = { ...recloneStatusByUrl.value, [u]: humanizeRequestErrorMessage(error.message || '请求失败') }
    setRecloneTrace(extractTraceId(error))
  }
  finally { recloneLoadingByUrl.value = { ...recloneLoadingByUrl.value, [u]: false }; repoRecloneGlobalLoading.value = false }
}

export async function onRepoCloneIdentityChange(url, nextId, deps) {
  const { effectiveTenantId, effectiveWorkspaceId, effectiveTaskId, repoCloneIdentityByUrl, repoCloneIdentitySaving, repoCloneIdentitySaveError, syncRepoCloneIdentityMapFromTask, repoCloneIdentityUserTouchedByUrl, localTask } = deps
  const tid = effectiveTenantId.value; const wid = effectiveWorkspaceId.value; const taskId = effectiveTaskId.value
  const u = String(url || '').trim()
  if (!tid || !wid || !taskId || !u) return
  if (repoCloneIdentityUserTouchedByUrl) {
    repoCloneIdentityUserTouchedByUrl.value = { ...repoCloneIdentityUserTouchedByUrl.value, [u]: true }
  }
  repoCloneIdentitySaving.value = true; repoCloneIdentitySaveError.value = ''
  const payload = { repo_clone_git_identities: { [u]: nextId ? String(nextId) : null } }
  try {
    const response = await apiFetch(`/api/tasks/todos/tenant_id/${tid}/workspace_id/${wid}/${taskId}/repo-clone-git-identities/`, { method: 'PATCH', credentials: 'include', headers: { 'Content-Type': 'application/json', Accept: 'application/json' }, body: JSON.stringify(payload) })
    const data = await response.json().catch(() => ({}))
    if (!response.ok) { throw new Error(typeof data.detail === 'string' ? data.detail : '保存失败') }
    repoCloneIdentityByUrl.value = { ...repoCloneIdentityByUrl.value, [u]: nextId ? String(nextId) : '' }
    if (localTask?.value && data?.repo_clone_git_identities && typeof data.repo_clone_git_identities === 'object') {
      localTask.value = {
        ...localTask.value,
        parameters: {
          ...(localTask.value.parameters || {}),
          repo_clone_git_identities: { ...data.repo_clone_git_identities },
        },
      }
    }
  } catch (error) { repoCloneIdentitySaveError.value = humanizeRequestErrorMessage(error.message || '保存失败'); syncRepoCloneIdentityMapFromTask() }
  finally { repoCloneIdentitySaving.value = false }
}

/** 任务关联仓库未保存克隆身份时，自动选用公司默认 Git 身份并持久化 */
export async function maybeAutoApplyDefaultRepoCloneIdentities(deps) {
  const {
    effectiveTenantId, effectiveWorkspaceId, effectiveTaskId,
    layerGitIdentityOptions, layerGitIdentityLoading, workspaceProjectsLoading,
    taskRepoRows, repoCloneIdentityIdForUrl,
    repoCloneIdentityByUrl, repoCloneIdentitySaving, repoCloneIdentitySaveError,
    localTask, repoCloneIdentityUserTouchedByUrl, repoCloneIdentityAutoApplyInFlight,
  } = deps

  if (repoCloneIdentityAutoApplyInFlight?.value) return
  if (repoCloneIdentitySaving.value) return
  if (layerGitIdentityLoading.value || workspaceProjectsLoading.value) return

  const defaultId = resolveDefaultCompanyGitIdentityId(layerGitIdentityOptions.value)
  if (!defaultId) return

  const tid = effectiveTenantId.value
  const wid = effectiveWorkspaceId.value
  const taskId = effectiveTaskId.value
  if (!tid || !wid || !taskId) return

  const touched = repoCloneIdentityUserTouchedByUrl?.value || {}
  const missing = []
  for (const row of taskRepoRows.value || []) {
    const u = String(row?.url || '').trim()
    if (!u || touched[u]) continue
    if (!String(repoCloneIdentityIdForUrl(u) || '').trim()) missing.push(u)
  }
  if (!missing.length) return

  const payload = { repo_clone_git_identities: {} }
  for (const u of missing) payload.repo_clone_git_identities[u] = defaultId

  if (repoCloneIdentityAutoApplyInFlight) repoCloneIdentityAutoApplyInFlight.value = true
  repoCloneIdentitySaving.value = true
  repoCloneIdentitySaveError.value = ''
  try {
    const response = await apiFetch(
      `/api/tasks/todos/tenant_id/${tid}/workspace_id/${wid}/${taskId}/repo-clone-git-identities/`,
      {
        method: 'PATCH',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
        body: JSON.stringify(payload),
      },
    )
    const data = await response.json().catch(() => ({}))
    if (!response.ok) {
      throw new Error(typeof data.detail === 'string' ? data.detail : '自动保存默认 Git 身份失败')
    }
    const saved = data?.repo_clone_git_identities && typeof data.repo_clone_git_identities === 'object'
      ? data.repo_clone_git_identities
      : payload.repo_clone_git_identities
    const nextByUrl = { ...repoCloneIdentityByUrl.value }
    for (const u of missing) {
      nextByUrl[u] = String(saved[u] || defaultId)
    }
    repoCloneIdentityByUrl.value = nextByUrl
    if (localTask?.value) {
      localTask.value = {
        ...localTask.value,
        parameters: {
          ...(localTask.value.parameters || {}),
          repo_clone_git_identities: { ...saved },
        },
      }
    }
  } catch (error) {
    console.error('自动应用默认 Git 身份失败:', error)
    repoCloneIdentitySaveError.value = humanizeRequestErrorMessage(error.message || '自动保存默认 Git 身份失败')
  } finally {
    repoCloneIdentitySaving.value = false
    if (repoCloneIdentityAutoApplyInFlight) repoCloneIdentityAutoApplyInFlight.value = false
  }
}

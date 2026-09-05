/**
 * 任务详情 → 容器 compute 转发请求：统一带 path `/comment_id/{id}/`（POST body 仍写入 comment_id）。
 */
import { apiFetch } from '../../utils/apiUtils.js'
import {
  appendCommentIdPath,
  jsonPostWithCommentId,
} from '../../utils/containerForwardCommentId.js'
import { resolveContainerUiContextCommentId } from './resolveContainerUiContextCommentId.js'

export function containerComputeFuncFirstUrl(tenantId, workspaceId, taskId, action, commentId, extraQuery = '') {
  const a = String(action || '').trim()
  const path =
    `/api/cloud/compute/${a}/tenant_id/${encodeURIComponent(tenantId)}/workspace_id/${encodeURIComponent(workspaceId)}/task_id/${encodeURIComponent(taskId)}/`
  const q = String(extraQuery || '').replace(/^\?/, '')
  return appendCommentIdPath(q ? `${path}?${q}` : path, commentId)
}

export function containerComputeKvLastUrl(tenantId, workspaceId, taskId, action, commentId, extraQuery = '') {
  const a = String(action || '').trim().replace(/(?:^\/+|\/+$)/g, '')
  const path =
    `/api/cloud/compute/tenant_id/${encodeURIComponent(tenantId)}/workspace_id/${encodeURIComponent(workspaceId)}/task_id/${encodeURIComponent(taskId)}/${a}/`
  const q = String(extraQuery || '').replace(/^\?/, '')
  return appendCommentIdPath(q ? `${path}?${q}` : path, commentId)
}

export function containerComputeLegacyTaskUrl(tenantId, workspaceId, taskId, action, commentId, extraQuery = '') {
  const a = String(action || '').trim().replace(/(?:^\/+|\/+$)/g, '')
  const path =
    `/api/cloud/compute/${a}/tenant_id/${encodeURIComponent(tenantId)}/workspace_id/${encodeURIComponent(workspaceId)}` +
    `/task_id/${encodeURIComponent(taskId)}/`
  const q = String(extraQuery || '').replace(/^\?/, '')
  return appendCommentIdPath(q ? `${path}?${q}` : path, commentId)
}

function scopeIds(deps) {
  return {
    tenantId: deps.effectiveTenantId.value,
    workspaceId: deps.effectiveWorkspaceId.value,
    taskId: deps.effectiveTaskId.value,
    commentId: resolveContainerUiContextCommentId(deps),
  }
}

export function postContainerCompute(deps, action, body) {
  const { tenantId, workspaceId, taskId, commentId } = scopeIds(deps)
  return apiFetch(
    containerComputeFuncFirstUrl(tenantId, workspaceId, taskId, action, commentId),
    jsonPostWithCommentId(body, commentId),
  )
}

export function getContainerCompute(deps, action, extraQuery = '') {
  const { tenantId, workspaceId, taskId, commentId } = scopeIds(deps)
  return apiFetch(containerComputeFuncFirstUrl(tenantId, workspaceId, taskId, action, commentId, extraQuery), {
    credentials: 'include',
    headers: { Accept: 'application/json' },
  })
}

export function getContainerComputeKvLast(deps, action, extraQuery = '', fetchOpts = {}) {
  const { tenantId, workspaceId, taskId, commentId } = scopeIds(deps)
  return apiFetch(containerComputeKvLastUrl(tenantId, workspaceId, taskId, action, commentId, extraQuery), {
    credentials: 'include',
    headers: { Accept: 'application/json' },
    ...fetchOpts,
  })
}

export function postContainerComputeLegacyTask(deps, action, body) {
  const { tenantId, workspaceId, taskId, commentId } = scopeIds(deps)
  return apiFetch(
    containerComputeLegacyTaskUrl(tenantId, workspaceId, taskId, action, commentId),
    jsonPostWithCommentId(body, commentId),
  )
}

export function getContainerComputeLegacyTask(deps, action, extraQuery = '') {
  const { tenantId, workspaceId, taskId, commentId } = scopeIds(deps)
  return apiFetch(containerComputeLegacyTaskUrl(tenantId, workspaceId, taskId, action, commentId, extraQuery), {
    credentials: 'include',
    headers: { Accept: 'application/json' },
  })
}

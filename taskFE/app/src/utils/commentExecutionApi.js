/**
 * 评论依赖模式 PATCH + 评论容器绑定 API。
 */
import { apiFetch } from './apiUtils.js'

export function normalizeExecutionMode(raw) {
  const s = String(raw || '').trim()
  if (s === 'independent') return 'independent'
  return 'wait_previous'
}

export async function patchHumanCommentExecutionMode({ tenantId, taskId, commentId, executionMode }) {
  const tid = String(tenantId || '').trim()
  const mode = normalizeExecutionMode(executionMode)
  // taskTaskService handleCommentRoutes：/api/tenant/{tid}/tasks/{taskId}/comments/{commentId}
  //（网关前缀 /api/tasks/ 下等价约定：位置段在前、kv 键值对在后）
  const response = await apiFetch(
    `/api/tasks/${encodeURIComponent(String(taskId))}/comments/${encodeURIComponent(String(commentId))}/tenant_id/${encodeURIComponent(tid)}/`,
    {
      method: 'PATCH',
      credentials: 'include',
      headers: { Accept: 'application/json', 'Content-Type': 'application/json' },
      body: JSON.stringify({ execution_mode: mode }),
    },
  )
  if (!response.ok) {
    throw new Error(`更新评论依赖模式失败: ${response.status}`)
  }
  return response.json().catch(() => ({ execution_mode: mode }))
}

export async function patchAICommentExecutionMode({
  tenantId,
  workspaceId,
  taskId,
  commentId,
  executionMode,
}) {
  const tid = String(tenantId || '').trim()
  const wid = String(workspaceId || '').trim()
  const mode = normalizeExecutionMode(executionMode)
  // taskAIComment handleAICommentRoutes：/api/tenant/{tid}/workspace/{wid}/task-detail/{taskId}/ai-comments/{commentId}
  //（网关前缀 /api/ai-comment/ 下等价约定）
  const response = await apiFetch(
    `/api/ai-comment/task-detail/tenant_id/${encodeURIComponent(tid)}/workspace_id/${encodeURIComponent(wid)}/task_id/${encodeURIComponent(String(taskId))}/ai-comments/${encodeURIComponent(String(commentId))}/`,
    {
      method: 'PATCH',
      credentials: 'include',
      headers: { Accept: 'application/json', 'Content-Type': 'application/json' },
      body: JSON.stringify({ execution_mode: mode }),
    },
  )
  if (!response.ok) {
    throw new Error(`更新 AI 评论依赖模式失败: ${response.status}`)
  }
  return response.json().catch(() => ({ execution_mode: mode }))
}

export async function fetchCommentContainerBindings({ tenantId, workspaceId, taskId }) {
  const tid = String(tenantId || '').trim()
  const wid = String(workspaceId || '').trim()
  const response = await apiFetch(
    `/api/cloud/compute/comment-container-bindings/tenant_id/${encodeURIComponent(tid)}/workspace_id/${encodeURIComponent(wid)}?task_id=${encodeURIComponent(String(taskId))}`,
    { credentials: 'include', headers: { Accept: 'application/json' } },
  )
  if (!response.ok) {
    throw new Error(`拉取评论容器绑定失败: ${response.status}`)
  }
  const data = await response.json()
  return Array.isArray(data?.bindings) ? data.bindings : []
}

export async function ensureCommentContainerBinding({
  tenantId,
  workspaceId,
  taskId,
  commentId,
  executionMode,
  dependsOnCommentId = '',
  dependsOnCommentIds = null,
}) {
  const tid = String(tenantId || '').trim()
  const wid = String(workspaceId || '').trim()
  const body = {
    comment_id: String(commentId),
    execution_mode: normalizeExecutionMode(executionMode),
  }
  const ids = Array.isArray(dependsOnCommentIds)
    ? dependsOnCommentIds.map((id) => String(id || '').trim()).filter(Boolean)
    : String(dependsOnCommentId || '')
      .split(',')
      .map((id) => id.trim())
      .filter(Boolean)
  if (ids.length) {
    body.depends_on_comment_id = ids.join(',')
    body.depends_on_comment_ids = ids
  }
  const response = await apiFetch(
    `/api/cloud/compute/comment-container-bindings/tenant_id/${encodeURIComponent(tid)}/workspace_id/${encodeURIComponent(wid)}?task_id=${encodeURIComponent(String(taskId))}`,
    {
      method: 'POST',
      credentials: 'include',
      headers: { Accept: 'application/json', 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    },
  )
  // 200 = 已存在并已同步 mode/depends；201 = 新建；兼容旧网关仍返回 409
  if (response.status === 409) {
    return { status: 'exists' }
  }
  if (!response.ok) {
    throw new Error(`创建评论容器绑定失败: ${response.status}`)
  }
  return response.json()
}

export async function advanceCommentContainerBindings({ tenantId, workspaceId, taskId }) {
  const tid = String(tenantId || '').trim()
  const wid = String(workspaceId || '').trim()
  const response = await apiFetch(
    `/api/cloud/compute/comment-container-bindings/tenant_id/${encodeURIComponent(tid)}/workspace_id/${encodeURIComponent(wid)}/advance/?task_id=${encodeURIComponent(String(taskId))}`,
    {
      method: 'POST',
      credentials: 'include',
      headers: { Accept: 'application/json', 'Content-Type': 'application/json' },
      body: '{}',
    },
  )
  if (!response.ok) {
    throw new Error(`推进评论容器调度失败: ${response.status}`)
  }
  return response.json()
}

/** 终止等待前序的评论容器绑定（waiting_previous → cancelled） */
export async function cancelCommentContainerBinding({ tenantId, workspaceId, taskId, commentId }) {
  const tid = String(tenantId || '').trim()
  const wid = String(workspaceId || '').trim()
  const cid = String(commentId || '').trim()
  const response = await apiFetch(
    `/api/cloud/compute/comment-container-bindings/tenant_id/${encodeURIComponent(tid)}/workspace_id/${encodeURIComponent(wid)}/${encodeURIComponent(cid)}/cancel/?task_id=${encodeURIComponent(String(taskId))}`,
    {
      method: 'POST',
      credentials: 'include',
      headers: { Accept: 'application/json', 'Content-Type': 'application/json' },
      body: '{}',
    },
  )
  if (!response.ok) {
    let detail = ''
    try {
      const data = await response.json()
      detail = String(data?.message || data?.error || '')
    } catch {
      /* ignore */
    }
    const err = new Error(detail || `终止等待前序失败: ${response.status}`)
    err.status = response.status
    throw err
  }
  return response.json()
}

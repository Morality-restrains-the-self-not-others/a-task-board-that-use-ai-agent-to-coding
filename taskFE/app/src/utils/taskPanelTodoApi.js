/**
 * TaskPanel 任务 PATCH / 评论 API（从视图抽出以控制组件行数）。
 */
import { apiFetch } from './apiUtils.js'
import { isCommentContentTaskIdEcho } from '../composables/taskDetail/buildDisplayComments.js'

export function buildTodoPatchUrl({ tenantId, workspaceId, currentWorkspace, taskId }) {
  const tid = tenantId != null ? String(tenantId).trim() : ''
  const wid =
    workspaceId != null
      ? String(workspaceId).trim()
      : currentWorkspace?.id != null
        ? String(currentWorkspace.id).trim()
        : ''
  if (!tid || !wid || wid === 'default') {
    return null
  }
  return `/api/tasks/todos/tenant_id/${tid}/workspace_id/${wid}/${taskId}/`
}

export async function patchTodo(deps, taskId, updateData) {
  const url = buildTodoPatchUrl({ ...deps, taskId })
  if (!url) {
    throw new Error('缺少租户或工作空间，无法更新任务进度列')
  }
  const response = await apiFetch(url, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(updateData),
  })
  if (!response.ok) {
    throw new Error(`HTTP错误！状态：${response.status}`)
  }
  const data = await response.json()
  console.log(`任务 ${taskId} 更新成功`, data)
  return data
}

export async function createTaskComment(tenantId, taskId, contentOrPayload) {
  const tid = tenantId != null ? String(tenantId).trim() : ''
  if (!tid) {
    throw new Error('创建评论失败: 缺少租户 ID')
  }
  let content = contentOrPayload
  let mentions
  if (contentOrPayload && typeof contentOrPayload === 'object') {
    content = contentOrPayload.content
    mentions = contentOrPayload.mentions
  }
  if (isCommentContentTaskIdEcho(content, taskId)) {
    throw new Error('评论内容不能仅为任务编号')
  }
  const body = { task: taskId, content }
  if (Array.isArray(mentions) && mentions.length > 0) {
    body.mentions = mentions
  }
  const response = await apiFetch(
    `/api/tasks/${encodeURIComponent(String(taskId))}/comments/tenant_id/${encodeURIComponent(tid)}`,
    {
      method: 'POST',
      credentials: 'include',
      headers: {
        Accept: 'application/json',
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(body),
    },
  )
  if (!response.ok) {
    throw new Error(`创建评论失败: ${response.status}`)
  }
  return response
}

export async function updateTaskComment(tenantId, taskId, commentId, content) {
  const tid = tenantId != null ? String(tenantId).trim() : ''
  if (!tid) {
    throw new Error('更新评论失败: 缺少租户 ID')
  }
  if (taskId == null || taskId === '') {
    throw new Error('更新评论失败: 无法解析任务 ID')
  }
  const response = await apiFetch(
    `/api/tasks/${encodeURIComponent(String(taskId))}/comments/${encodeURIComponent(String(commentId))}/tenant_id/${encodeURIComponent(tid)}/`,
    {
      method: 'PUT',
      credentials: 'include',
      headers: {
        Accept: 'application/json',
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ content }),
    },
  )
  if (!response.ok) {
    throw new Error(`更新评论失败: ${response.status}`)
  }
  return response
}

export function resolveCommentUpdateArgs(todos, args) {
  if (args.length >= 3) {
    return { taskId: args[0], commentId: args[1], content: args[2] }
  }
  const commentId = args[0]
  const content = args[1]
  const todo = (Array.isArray(todos) ? todos : []).find((t) =>
    (Array.isArray(t.comments) ? t.comments : []).some(
      (c) => String(c.id) === String(commentId),
    ),
  )
  return { taskId: todo?.id, commentId, content }
}

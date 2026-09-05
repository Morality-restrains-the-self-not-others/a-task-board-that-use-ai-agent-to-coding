/**
 * 评论卡片上的云主机 API 须带 comment_id，后端按该评论 CSC 解析。
 * 无 comment_id 时不得发任务级请求。comment_id 放 path kv，query 仅兼容旧客户端。
 */
import { appendCommentIdPath } from './containerForwardCommentId.js'

export function trimCommentId(commentId) {
  return String(commentId || '').trim()
}

/** 在 compute URL 的 path 追加 `/comment_id/{id}/`（空则原样返回，调用方应先拒绝请求）。 */
export function withCommentIdQuery(url, commentId) {
  return appendCommentIdPath(url, commentId)
}

/** stop-vm JSON 体：有评论 id 时带上 comment_id。 */
export function stopVmBodyWithCommentId(taskId, commentId) {
  const body = { task_id: taskId, stop_reason: 'user_stop' }
  const cid = trimCommentId(commentId)
  if (cid) body.comment_id = cid
  return body
}

/** 仅接受显式字符串/数字评论 id；点击事件等非字符串不是评论 id。 */
export function commentIdFromButtonArg(value) {
  if (typeof value === 'string' || typeof value === 'number') {
    return String(value).trim()
  }
  return ''
}

/**
 * 评论卡刷新会记住 comment_id，供无参轮询沿用。
 * 点击事件不是评论 id，沿用上次值，禁止清空后走任务级请求。
 */
export function createScopedCommentIdMemory() {
  let last = ''
  return {
    fromAction(value) {
      if (typeof value === 'string' || typeof value === 'number') {
        last = String(value).trim()
        return last
      }
      return last
    },
    forPoll() {
      return last
    },
    reset() {
      last = ''
    },
  }
}

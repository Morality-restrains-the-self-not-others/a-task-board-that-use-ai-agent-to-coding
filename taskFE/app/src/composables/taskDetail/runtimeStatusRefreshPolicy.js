/**
 * 启动成功/停止成功推送如何路由到评论快照。
 * 禁止无 comment_id 时回落到 scopedComment.forPoll()；禁止在此路径发 HTTP。
 */
import { isCloudServerStopSuccessMessage } from '../../utils/stopVmSseMessage.js'

export const SERVER_START_SUCCESS_MESSAGE = 'aliyun服务器启动成功！'

export function shouldRefreshRuntimeOnStatusMessage(message) {
  const msg = String(message || '')
  if (!msg) return false
  return msg.includes(SERVER_START_SUCCESS_MESSAGE) || isCloudServerStopSuccessMessage(msg)
}

/**
 * @param {{ message?: string, eventCommentId?: string|null }} input
 * @returns {string} 可 Describe 的 comment_id；空字符串表示跳过（含成功文案但无事件 id）
 */
export function commentIdForRuntimeRefresh({ message, eventCommentId } = {}) {
  if (!shouldRefreshRuntimeOnStatusMessage(message)) return ''
  return String(eventCommentId || '').trim()
}

/** SSE status 载荷上的评论 id（禁止用任务级最近操作槽代替） */
export function commentIdFromStatusEvent(statusData) {
  if (!statusData || typeof statusData !== 'object') return ''
  return String(statusData.comment_id || statusData.commentId || '').trim()
}

export function runtimeStatusFromStatusEvent(statusData) {
  if (!statusData || typeof statusData !== 'object') return ''
  return String(statusData.runtime_status || statusData.runtimeStatus || '').trim()
}

/**
 * 推送到达时写入评论快照；返回是否写入。禁止在此函数内发 HTTP。
 */
export function applyPushedRuntimeSnapshot(store, { commentId, runtimeStatus, message, traceId }) {
  const cid = String(commentId || '').trim()
  const rs = String(runtimeStatus || '').trim()
  if (!store || typeof store.patch !== 'function' || !cid || !rs) return false
  store.patch(cid, {
    status: rs,
    message: String(message || ''),
    traceId: String(traceId || ''),
    loading: false,
  })
  return true
}

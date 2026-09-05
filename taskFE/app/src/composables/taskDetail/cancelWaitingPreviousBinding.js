import { ref } from 'vue'
import { cancelCommentContainerBinding } from '../../utils/commentExecutionApi.js'

/**
 * 终止 waiting_previous 评论容器绑定（用户主动取消串行等待）。
 */
export function createCancelWaitingPreviousBinding({ ids, refreshBindings, appendBindingStatusLog }) {
  const cancelWaitingBusyByCommentId = ref({})

  function isCancelWaitingBusy(commentId) {
    return Boolean(cancelWaitingBusyByCommentId.value[String(commentId || '')])
  }

  async function cancelWaitingPreviousBinding(commentId) {
    const cid = String(commentId || '').trim()
    const { tid, wid, tk } = typeof ids === 'function' ? ids() : ids
    if (!cid || !tid || !wid || !tk) return null
    const busy = { ...(cancelWaitingBusyByCommentId.value || {}) }
    busy[cid] = true
    cancelWaitingBusyByCommentId.value = busy
    try {
      const data = await cancelCommentContainerBinding({
        tenantId: tid,
        workspaceId: wid,
        taskId: tk,
        commentId: cid,
      })
      if (typeof appendBindingStatusLog === 'function') {
        appendBindingStatusLog(cid, 'cancelled')
      }
      if (typeof refreshBindings === 'function') {
        await refreshBindings()
      }
      return data
    } finally {
      const next = { ...(cancelWaitingBusyByCommentId.value || {}) }
      delete next[cid]
      cancelWaitingBusyByCommentId.value = next
    }
  }

  return {
    cancelWaitingBusyByCommentId,
    isCancelWaitingBusy,
    cancelWaitingPreviousBinding,
  }
}

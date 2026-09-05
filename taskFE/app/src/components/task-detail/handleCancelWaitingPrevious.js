import { showRequestError } from '../../utils/requestErrorDisplay.js'

/** 执行细节「终止等待前序」：调用 cancel API，失败经统一错误展示。 */
export async function handleCancelWaitingPrevious(cancelFn, payload) {
  const cid = String(payload?.commentId || '').trim()
  if (!cid || typeof cancelFn !== 'function') return
  try {
    await cancelFn(cid)
  } catch (e) {
    showRequestError(e?.message || '终止等待前序失败', e)
  }
}

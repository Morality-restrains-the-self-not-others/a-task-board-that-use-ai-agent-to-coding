import { trimCommentId } from '../../utils/cloudComputeCommentQuery.js'

/**
 * 从运行态面板快照取出该评论 raw/展示状态，供执行细节 summary 徽章覆盖。
 * @param {object|null|undefined} panel
 * @param {unknown} commentId
 * @returns {string}
 */
export function runtimeStatusFromCommentPanel(panel, commentId) {
  if (!panel || typeof panel.snapshotForComment !== 'function') return ''
  const snap = panel.snapshotForComment(commentId) || {}
  return String(snap.serverRuntimeStatus || snap.serverRuntimeStatusDisplayText || '').trim()
}

/**
 * 评论执行细节里的运行态面板：把 Workbench / 刷新 / 停止 绑到该评论 id，
 * 并用 snapshotForComment 覆盖展示字段，避免多评论共用任务级单例文案。
 * 无 commentId 时原样返回，调用方不得把它当任务级运行实例面板。
 */
export function bindCommentRuntimePanel(panel, commentId, bindingStatus = '', commentCreatedAt = '') {
  if (!panel || typeof panel !== 'object') return panel
  const cid = trimCommentId(commentId)
  if (!cid) return panel
  const snap = typeof panel.snapshotForComment === 'function'
    ? (panel.snapshotForComment(cid, { commentCreatedAt }) || {})
    : {}
  const bound = {
    ...panel,
    ...snap,
    commentId: cid,
    openWorkbenchLink: () => panel.openWorkbenchLink?.(cid),
    fetchServerRuntimeStatus: () => panel.fetchServerRuntimeStatus?.(cid),
    stopServer: () => panel.stopServer?.(cid),
  }
  if (String(bindingStatus || '').trim() === 'waiting_previous') {
    return {
      ...bound,
      serverRuntimeStatusDisplayText: '等待前序',
      showRuntimeActionButtons: false,
      serverRuntimeStatusMessage: '串行评论需等待前序完成后再启动本评论服务器。',
    }
  }
  return bound
}

import { trimCommentId } from '../../utils/cloudComputeCommentQuery.js'

/**
 * 评论执行细节里的「服务器内容」面板：把刷新绑到该评论 id，
 * 并用 snapshotForComment 覆盖展示字段，避免多评论共用任务级单例文案。
 */
export function bindCommentServerContentPanel(panel, commentId) {
  if (!panel || typeof panel !== 'object') return panel
  const cid = trimCommentId(commentId)
  if (!cid) return panel
  const snap = typeof panel.snapshotForComment === 'function'
    ? (panel.snapshotForComment(cid) || {})
    : {}
  return {
    ...panel,
    ...snap,
    commentId: cid,
    fetchServerContent: () => panel.fetchServerContent?.(cid),
  }
}

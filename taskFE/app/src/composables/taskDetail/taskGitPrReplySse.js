/**
 * task_git_pr_reply_created SSE：任务详情在未整页刷新时插入 git_pr 子评论。
 */

/** @param {unknown} statusData */
export function isTaskGitPrReplyCreatedSse(statusData) {
  return Boolean(statusData && typeof statusData === 'object' && statusData.event_name === 'task_git_pr_reply_created')
}

/**
 * @param {unknown} statusData
 * @param {Set<string>} seenCommentIds
 * @returns {{ shouldRefetch: boolean, commentId: string }}
 */
export function planGitPrReplyCreatedSseRefetch(statusData, seenCommentIds = new Set()) {
  if (!isTaskGitPrReplyCreatedSse(statusData)) {
    return { shouldRefetch: false, commentId: '' }
  }
  const commentId = String(statusData.comment_id || '').trim()
  if (!commentId) {
    return { shouldRefetch: true, commentId: '' }
  }
  if (seenCommentIds.has(commentId)) {
    return { shouldRefetch: false, commentId }
  }
  seenCommentIds.add(commentId)
  return { shouldRefetch: true, commentId }
}

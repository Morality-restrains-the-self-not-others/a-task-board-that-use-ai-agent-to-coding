/**
 * 评论级容器名：保证恰好一个 task_ 前缀。
 * taskId 已是 task_* 时不再二次拼接（避免 task_task_1566…_cmt_…）。
 */
export function buildCommentContainerName(taskId, commentId) {
  const tid = String(taskId || '').trim()
  const cid = String(commentId || '').trim()
  if (!tid || !cid) return ''
  if (tid.startsWith('task_')) return `${tid}_${cid}`
  return `task_${tid}_${cid}`
}

/**
 * 优先用规范名覆盖存量双前缀 task_ + task_*；其它自定义名原样返回。
 */
export function normalizeCommentContainerName(taskId, commentId, stored) {
  const want = buildCommentContainerName(taskId, commentId)
  const s = String(stored || '').trim()
  if (!s) return want
  if (!want) return s
  const tid = String(taskId || '').trim()
  const cid = String(commentId || '').trim()
  const legacyDouble = tid && cid ? `task_${tid}_${cid}` : ''
  if (legacyDouble && s === legacyDouble && s !== want) return want
  return s
}

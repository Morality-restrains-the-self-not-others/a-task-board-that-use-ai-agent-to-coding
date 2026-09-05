/**
 * SaaS→容器转发附带 comment_id，命中评论级 CSC（两评论绑不同实例时禁止串台）。
 * 约定：comment_id 放 path kv（`/comment_id/{id}/`），query 仅兼容旧客户端。
 */

export function trimCommentId(commentId) {
  return String(commentId || '').trim()
}

/** 在 path 追加 `/comment_id/{id}/`（已有 path 段则不重复；保留既有 query）。 */
export function appendCommentIdPath(apiPath, commentId) {
  const cid = trimCommentId(commentId)
  const raw = String(apiPath || '')
  if (!cid || !raw) return raw
  const qIndex = raw.indexOf('?')
  const path = qIndex >= 0 ? raw.slice(0, qIndex) : raw
  const qs = qIndex >= 0 ? raw.slice(qIndex + 1) : ''
  if (/\/comment_id\//.test(path)) return raw
  const withSlash = path.endsWith('/') ? path : `${path}/`
  const next = `${withSlash}comment_id/${encodeURIComponent(cid)}/`
  return qs ? `${next}?${qs}` : next
}

/** @deprecated 旧 query 形态；compute 转发请用 appendCommentIdPath。 */
export function appendCommentIdQuery(apiPath, commentId) {
  return appendCommentIdPath(apiPath, commentId)
}

export function setCommentIdSearchParam(qs, commentId) {
  const cid = trimCommentId(commentId)
  if (cid && qs && typeof qs.set === 'function') qs.set('comment_id', cid)
  return qs
}

export function withCommentIdBody(body, commentId) {
  const cid = trimCommentId(commentId)
  const base = body && typeof body === 'object' && !Array.isArray(body) ? { ...body } : {}
  if (!cid) return base
  return { ...base, comment_id: cid }
}

export function jsonPostWithCommentId(body, commentId) {
  return {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
    body: JSON.stringify(withCommentIdBody(body, commentId)),
  }
}

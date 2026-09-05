/**
 * Split auto-run skip / nested-git auth errors out of comment body
 * so they can render in the author-row chips (div.flex.flex-wrap).
 */

export const COMMENT_START_SKIP_MARKER = '未启动服务器：'

export function splitCommentStartSkipNotice(content) {
  const raw = String(content ?? '')
  if (!raw) {
    return { body: '', skipNotice: '' }
  }
  const idx = raw.lastIndexOf(COMMENT_START_SKIP_MARKER)
  if (idx < 0) {
    return { body: raw, skipNotice: '' }
  }
  const notice = raw.slice(idx).trim()
  if (!notice.startsWith(COMMENT_START_SKIP_MARKER)) {
    return { body: raw, skipNotice: '' }
  }
  const body = raw.slice(0, idx).replace(/[ \t]+$/g, '').replace(/\n+$/g, '')
  return { body, skipNotice: notice }
}

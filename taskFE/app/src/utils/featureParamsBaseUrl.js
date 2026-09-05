/**
 * Feature-params provider base_url：只修复「两个绝对 URL 粘在一起」的脏数据，
 * 不按运营商改写用户填写的端点。
 */

const ABSOLUTE_URL_SCHEME_RE = /https?:\/\//gi

/**
 * 若字符串里连续出现两个绝对 URL（无 query），保留最后一个。
 * @param {unknown} raw
 * @returns {string}
 */
export function repairConcatenatedAbsoluteUrl(raw) {
  const s = String(raw || '').trim()
  if (!s) return ''
  const starts = []
  ABSOLUTE_URL_SCHEME_RE.lastIndex = 0
  let match = ABSOLUTE_URL_SCHEME_RE.exec(s)
  while (match) {
    starts.push(match.index)
    match = ABSOLUTE_URL_SCHEME_RE.exec(s)
  }
  if (starts.length <= 1) {
    return s.replace(/\/+$/, '')
  }
  if (s.slice(0, starts[1]).includes('?')) {
    return s.replace(/\/+$/, '')
  }
  return s.slice(starts[starts.length - 1]).replace(/\/+$/, '')
}

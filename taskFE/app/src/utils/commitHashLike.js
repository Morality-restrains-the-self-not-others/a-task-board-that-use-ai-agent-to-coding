/**
 * 判断字符串是否像 git commit hash（短/完整 SHA，7–40 位十六进制）。
 * @param {unknown} value
 * @returns {boolean}
 */
export function isCommitHashLike(value) {
  const s = String(value ?? '').trim()
  return /^[0-9a-f]{7,40}$/i.test(s)
}

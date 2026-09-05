/** GitHub App 授权回跳：跳转 GitHub 前写入本地；成功时后端优先用 session 的 next，此处为兜底（如换票失败回跳）。 */

const STORAGE_PREFIX = 'github_app_return:'
/** GitHub 授权页停留常超过 30s；过短会导致回跳页读不到 return_url。 */
const TTL_MS = 900_000
const RETURN_URL_MAX_LEN = 2048

function isSafeRelativeReturnUrl(s) {
  if (s == null || typeof s !== 'string') return false
  const t = s.trim()
  if (!t || t.length > RETURN_URL_MAX_LEN) return false
  if (!t.startsWith('/')) return false
  if (t.startsWith('//')) return false
  if (t.includes('://')) return false
  if (/[\n\r\0]/.test(t)) return false
  return true
}

/** 生成供 state 使用的 return_key（仅 [0-9a-f]）。 */
export function createGithubAppReturnKey() {
  const a = new Uint8Array(16)
  crypto.getRandomValues(a)
  return Array.from(a, (x) => x.toString(16).padStart(2, '0')).join('')
}

export function setGithubAppReturnTarget(returnKey, returnUrl) {
  if (!returnKey || typeof returnKey !== 'string') return
  if (!isSafeRelativeReturnUrl(returnUrl)) return
  try {
    localStorage.setItem(
      STORAGE_PREFIX + returnKey,
      JSON.stringify({
        return_url: returnUrl,
        expires_at: Date.now() + TTL_MS,
      }),
    )
  } catch {
    /* quota / private mode */
  }
}

export function consumeGithubAppReturnTarget(returnKey) {
  if (!returnKey || typeof returnKey !== 'string') return null
  const key = STORAGE_PREFIX + returnKey
  let raw
  try {
    raw = localStorage.getItem(key)
  } catch {
    return null
  }
  if (!raw) return null
  try {
    localStorage.removeItem(key)
  } catch {
    /* ignore */
  }
  let o
  try {
    o = JSON.parse(raw)
  } catch {
    return null
  }
  if (!o || typeof o.return_url !== 'string' || typeof o.expires_at !== 'number') return null
  if (o.expires_at < Date.now()) return null
  if (!isSafeRelativeReturnUrl(o.return_url)) return null
  return o.return_url
}

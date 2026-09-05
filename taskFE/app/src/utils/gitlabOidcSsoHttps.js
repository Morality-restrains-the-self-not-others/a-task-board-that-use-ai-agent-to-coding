/**
 * Tenant GitLab OIDC SSO URL checks.
 * HTTP and HTTPS Path A base URLs are both issuable; HTTP is a warning, not a block.
 */

const LOOPBACK_HOSTS = new Set(['127.0.0.1', 'localhost', '::1'])

/**
 * @param {string} hostname
 * @returns {boolean}
 */
export function isLoopbackGitLabHost(hostname) {
  return LOOPBACK_HOSTS.has(String(hostname || '').toLowerCase())
}

/**
 * @param {string} raw
 * @returns {URL | null}
 */
export function parseGitLabBaseUrl(raw) {
  const s = String(raw || '').trim()
  if (!s) return null
  try {
    const u = new URL(s)
    if (!u.hostname) return null
    return u
  } catch {
    return null
  }
}

/**
 * Non-blocking reminder when OmniAuth callback would be plaintext HTTP.
 * Empty for HTTPS, loopback HTTP, empty URL, and invalid URL.
 * @param {string} baseUrl
 * @returns {string}
 */
export function gitlabOidcSsoHttpWarning(baseUrl) {
  const u = parseGitLabBaseUrl(baseUrl)
  if (!u || u.protocol !== 'http:') return ''
  if (isLoopbackGitLabHost(u.hostname)) return ''
  return `当前 GitLab 使用 HTTP（${u.host}）。授权回调为明文，签发后即可登录。GitLab 服务器仍须能 HTTPS 访问平台 issuer 的 token 与 JWKS。建议日后为 GitLab 配置 TLS。`
}

/**
 * Empty string means SSO enable is allowed from the URL scheme perspective.
 * @param {string} baseUrl
 * @returns {string}
 */
export function gitlabOidcSsoBlockReason(baseUrl) {
  const s = String(baseUrl || '').trim()
  if (!s) {
    return ''
  }
  const u = parseGitLabBaseUrl(s)
  if (!u || (u.protocol !== 'http:' && u.protocol !== 'https:')) {
    return 'GitLab Base URL 无效，请填写以 http:// 或 https:// 开头的地址。'
  }
  return ''
}

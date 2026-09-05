/**
 * Cross-subdomain SSO cookie Domain helpers.
 * Host-only cookies on www.* are NOT sent to api.* during top-level OIDC navigation.
 */

/**
 * @param {string | null | undefined} hostname
 * @returns {boolean}
 */
export function isIpOrLocalHostname(hostname) {
  const host = String(hostname || '')
    .trim()
    .toLowerCase()
  if (!host) return true
  if (host === 'localhost' || host.endsWith('.localhost')) return true
  if (host === '::1') return true
  // IPv4
  if (/^\d{1,3}(\.\d{1,3}){3}$/.test(host)) return true
  // Bracketed / raw IPv6 (coarse)
  if (host.includes(':')) return true
  return false
}

/**
 * @param {string | null | undefined} hostname
 * @param {string | null | undefined} configuredDomain e.g. ".example.com" or "example.com"
 * @returns {string} cookie Domain attribute without leading strategy noise; empty = host-only
 */
export function resolveSharedCookieDomain(hostname, configuredDomain) {
  const host = String(hostname || '')
    .trim()
    .toLowerCase()
  // localhost / IP 只能写 host-only cookie；带 Domain=.example.com 会被浏览器（及 jsdom）丢弃
  if (!host || isIpOrLocalHostname(host)) {
    return ''
  }
  const configured = String(configuredDomain || '').trim()
  if (configured) {
    const normalized = configured.startsWith('.') ? configured : `.${configured}`
    const bare = normalized.slice(1).toLowerCase()
    // 仅当当前主机属于该共享域时才写入 Domain，避免跨站/本地误用
    if (host === bare || host.endsWith(`.${bare}`)) {
      return normalized
    }
    return ''
  }
  const labels = host.split('.').filter(Boolean)
  if (labels.length < 2) {
    return ''
  }
  // eTLD+1 style for our product domains (*.example.com)
  return `.${labels.slice(-2).join('.')}`
}

/**
 * Derive public base origin from a frontend hostname（使用 base 域名，避免跳转到 api.* 网关域）。
 * www.daydaymoney.com → https://example.com
 *
 * @param {{ protocol?: string, hostname?: string } | null | undefined} loc
 * @returns {string}
 */
export function deriveGatewayPublicBaseFromLocation(loc) {
  const hostname = String(loc?.hostname || '')
    .trim()
    .toLowerCase()
  const protocol = String(loc?.protocol || 'https:').trim() || 'https:'
  if (!hostname || isIpOrLocalHostname(hostname)) {
    return ''
  }
  const labels = hostname.split('.').filter(Boolean)
  if (labels.length < 2) {
    return ''
  }
  const baseDomain = labels.slice(-2).join('.')
  return `${protocol}//${baseDomain}`.replace(/\/$/, '')
}

import {
  deriveGatewayPublicBaseFromLocation,
  isIpOrLocalHostname,
} from './ssoCookieDomain.js'

const OIDC_AUTHORIZE_PATH = '/api/oidc/authorize'
const OIDC_TENANT_AUTHORIZE = /^\/api\/oidc\/([^/]+)\/authorize$/
const MAX_LEN = 2048

/**
 * Apex / www / api of the same registrable domain (e.g. daydaymoney.com).
 * Rejects sibling hosts such as evil.daydaymoney.com.
 * @param {string} hostname
 * @param {string} expectedHostname
 * @returns {boolean}
 */
export function isOidcResumeSameSiteHost(hostname, expectedHostname) {
  const a = String(hostname || '').toLowerCase()
  const b = String(expectedHostname || '').toLowerCase()
  if (!a || !b) return false
  if (a === b) return true
  const apexOf = (host) => host.replace(/^www\./, '').replace(/^api\./, '')
  const apex = apexOf(a)
  if (!apex || apex !== apexOf(b)) return false
  const allowed = new Set([apex, `www.${apex}`, `api.${apex}`])
  return allowed.has(a) && allowed.has(b)
}

/**
 * Global `/api/oidc/authorize` or tenant-scoped `/api/oidc/{tenantId}/authorize`.
 * @param {string} pathname
 * @returns {boolean}
 */
export function isOidcAuthorizePathname(pathname) {
  if (pathname === OIDC_AUTHORIZE_PATH) return true
  const m = typeof pathname === 'string' ? pathname.match(OIDC_TENANT_AUTHORIZE) : null
  if (!m) return false
  const tenantId = m[1]
  return Boolean(tenantId && tenantId !== '.' && tenantId !== '..')
}

function gatewayPublicBase() {
  const fromGateway = import.meta.env.VITE_TASK_GATEWAY_PUBLIC_BASE
  if (typeof fromGateway === 'string' && fromGateway.trim()) {
    return fromGateway.trim().replace(/\/$/, '')
  }
  // Do NOT fall back to VITE_API_BASE_URL when it is intentionally empty
  // (same-origin SPA). Empty apiBaseUrl must not poison OIDC resume host checks.
  const fromConfig =
    typeof window !== 'undefined' ? window.config?.TASK_GATEWAY_PUBLIC_BASE : ''
  if (typeof fromConfig === 'string' && fromConfig.trim()) {
    return fromConfig.trim().replace(/\/$/, '')
  }
  if (typeof window !== 'undefined' && window.location?.hostname) {
    const derived = deriveGatewayPublicBaseFromLocation(window.location)
    if (derived) {
      return derived
    }
    // Legacy same-host multi-port (IP/localhost): gateway on :18081
    if (isIpOrLocalHostname(window.location.hostname)) {
      return `${window.location.protocol}//${window.location.hostname}:18081`
    }
  }
  return ''
}

/**
 * @param {string | null | undefined} raw
 * @returns {string | null}
 */
export function sanitizeOidcResumeNext(raw) {
  if (raw == null || typeof raw !== 'string') return null
  const candidate = raw.trim()
  if (!candidate || candidate.length > MAX_LEN) return null
  if (/[\n\r\0]/.test(candidate)) return null

  const gatewayBase = gatewayPublicBase()

  const relPath = candidate.split('?')[0]
  if (!candidate.startsWith('//') && !candidate.includes('://') && isOidcAuthorizePathname(relPath)) {
    return candidate
  }

  let parsed
  try {
    parsed = new URL(candidate)
  } catch {
    return null
  }
  if (parsed.protocol !== 'http:' && parsed.protocol !== 'https:') return null
  if (!isOidcAuthorizePathname(parsed.pathname)) return null
  if (gatewayBase) {
    const expected = new URL(gatewayBase)
    if (parsed.protocol !== expected.protocol || !isOidcResumeSameSiteHost(parsed.hostname, expected.hostname)) {
      return null
    }
  }
  return candidate
}

/**
 * @param {string} resumeNext
 * @returns {string}
 */
export function resolveOidcResumeTarget(resumeNext) {
  if (resumeNext.startsWith('http://') || resumeNext.startsWith('https://')) {
    return resumeNext
  }
  const gatewayBase = gatewayPublicBase()
  if (!gatewayBase) {
    throw new Error('TASK_GATEWAY_PUBLIC_BASE is required for relative OIDC resume URLs')
  }
  return `${gatewayBase}${resumeNext}`
}

/** @internal exported for unit tests */
export const _test = { gatewayPublicBase }

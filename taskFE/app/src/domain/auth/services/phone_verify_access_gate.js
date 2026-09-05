import { userHasVerifiedPhone } from '../value_objects/phone_verified_status.js'

const EXEMPT_EXACT = new Set([
  '/',
  '/pricing/',
  '/faq/',
  '/acknowledgments/',
  '/profile/',
])

const EXEMPT_PREFIXES = [
  '/auth/login/',
  '/login/',
  '/auth/register/',
  '/register/',
  '/auth/admin-login/',
  '/auth/reset-password',
  '/reset-password',
  '/auth/activate/',
  '/activate/',
  '/oauth/github-app/callback/',
  '/redirect/gitsite/',
]

function normalizePathname(path) {
  const raw = String(path || '').split('#')[0].split('?')[0]
  if (!raw) return '/'
  return raw.endsWith('/') ? raw : `${raw}/`
}

export function isPhoneVerifyStaffBypass(source) {
  if (!source || typeof source !== 'object') return false
  if (source.is_superuser === true || source.is_superuser === 'True') return true
  const roles = source.platform_roles
  if (!Array.isArray(roles)) return false
  return roles.some((r) => r === 'super_admin' || r === 'employee')
}

export function isPhoneVerifyGateExemptPath(path, { query, isImpersonating } = {}) {
  if (isImpersonating) return true
  const p = normalizePathname(path)
  if (EXEMPT_EXACT.has(p)) return true
  if (EXEMPT_PREFIXES.some((pre) => p.startsWith(pre))) return true
  if (p.includes('/people/join/') || p.includes('/people/invite/')) return true
  const accessCode = query && (query.accessCode || query.access_code)
  if (accessCode && p.includes('/task-detail/')) return true
  return false
}

/**
 * sessionKnown=false（/me/ 失败）fail-open；已验证/豁免/员工不阻断。
 */
export function shouldBlockForUnverifiedPhone({
  sessionKnown = false,
  source,
  path,
  query,
  isImpersonating = false,
} = {}) {
  if (!sessionKnown) return false
  if (isPhoneVerifyStaffBypass(source)) return false
  if (userHasVerifiedPhone(source)) return false
  if (isPhoneVerifyGateExemptPath(path, { query, isImpersonating })) return false
  return true
}

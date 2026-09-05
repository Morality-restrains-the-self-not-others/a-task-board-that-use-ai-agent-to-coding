import { sanitizeOidcResumeNext, resolveOidcResumeTarget } from './oidcResumeUrl.js'
import { PostLoginReturnUrl } from '../domain/auth/value_objects/post_login_return_url_value_object.js'
import { POST_LOGIN_REDIRECT_STORAGE_KEY } from './authReturnUrl.js'
import { resolveSwitchHref } from '../domain/auth/services/activate_session_service.js'

const DEFAULT_REDIRECT = '/system-admin/'

function firstQueryValue(raw) {
  if (Array.isArray(raw)) {
    return typeof raw[0] === 'string' ? raw[0] : ''
  }
  return typeof raw === 'string' ? raw : ''
}

/**
 * Resolve post-login redirect.
 * Priority: OIDC resume next → same-origin business next → localStorage → API redirect_url → default.
 * When addAccount + newUserId and result still has /tenant/, apply resolveSwitchHref (AC7).
 *
 * @param {string | string[] | null | undefined} routeNextRaw
 * @param {{ redirect_url?: string }} [data]
 * @param {Pick<Storage, 'getItem' | 'removeItem'>} [storage]
 * @param {{ addAccount?: boolean, newUserId?: string }} [opts]
 * @returns {string}
 */
export function resolvePostLoginRedirectUrl(routeNextRaw, data = {}, storage = globalThis.localStorage, opts = {}) {
  const raw = firstQueryValue(routeNextRaw)
  const resumeNext = sanitizeOidcResumeNext(raw)
  let result = null
  if (resumeNext) {
    try {
      result = resolveOidcResumeTarget(resumeNext)
    } catch (error) {
      console.warn('[Login] invalid OIDC resume next, falling back:', error)
    }
  }

  if (!result) {
    const businessNext = PostLoginReturnUrl.normalize(raw)
    if (businessNext) {
      result = businessNext
    }
  }

  if (!result && storage && typeof storage.getItem === 'function') {
    const postLoginRedirect = storage.getItem(POST_LOGIN_REDIRECT_STORAGE_KEY)
    if (postLoginRedirect) {
      if (typeof storage.removeItem === 'function') {
        storage.removeItem(POST_LOGIN_REDIRECT_STORAGE_KEY)
      }
      result = postLoginRedirect
    }
  }

  if (!result) {
    // 禁止 API 返回 redirect_url="/" 导致登录后跳转到公开首页（用户无公司时后端曾返回 '/'）
    if (typeof data?.redirect_url === 'string' && data.redirect_url.trim() && data.redirect_url !== '/') {
      result = data.redirect_url
    } else {
      result = DEFAULT_REDIRECT
    }
  }

  const uid = String(opts?.newUserId ?? '').trim()
  const addRaw = opts?.addAccountQuery
  const addAccount =
    opts?.addAccount === true ||
    addRaw === '1' ||
    addRaw === 1 ||
    (Array.isArray(addRaw) && String(addRaw[0]) === '1')
  if (addAccount && uid && /\/tenant\//.test(String(result))) {
    return resolveSwitchHref(result, uid)
  }
  return result
}

import { PostLoginReturnUrl } from '../domain/auth/value_objects/post_login_return_url_value_object.js'

/**
 * 个人资料「手机号绑定」深链契约。
 * 与 emailBindingDeepLink 同构：#rg= 对应 UserProfile data-rg-key。
 */
export const PHONE_BINDING_RG_KEY = 'profile.phone_binding'

/** 登录引导「去验证」的相对路径（当前用户资料页） */
export function buildPhoneBindingRedirectUrl() {
  return `/profile/#rg=${PHONE_BINDING_RG_KEY}`
}

export function isPhoneBindingHash(hash) {
  const raw = String(hash || '')
  return raw.includes(`rg=${PHONE_BINDING_RG_KEY}`)
}

/**
 * OPT-20260825-002：登录后未绑手机引导「去验证」时，把原 next / 工作面板落点暂存到
 * sessionStorage；绑定成功后回到该落点。与 POST_LOGIN_REDIRECT_STORAGE_KEY 区分开，
 * 避免互相覆盖。sessionStorage 使同标签页内的资料页跳转仍保留，关闭标签页即清除。
 */
export const PHONE_VERIFY_REDIRECT_STORAGE_KEY = 'postLoginPhoneVerifyRedirect'

/** 「去验证」前写入原登录落点；非法路径不写。 */
export function savePhoneVerifyRedirect(rawPath) {
  const normalized = PostLoginReturnUrl.normalize(rawPath)
  if (!normalized) return false
  try {
    sessionStorage.setItem(PHONE_VERIFY_REDIRECT_STORAGE_KEY, normalized)
    return true
  } catch {
    return false
  }
}

/** 读取并清除暂存的原登录落点；返回 '' 表示无。 */
export function consumePhoneVerifyRedirect() {
  try {
    const raw = sessionStorage.getItem(PHONE_VERIFY_REDIRECT_STORAGE_KEY)
    if (!raw) return ''
    sessionStorage.removeItem(PHONE_VERIFY_REDIRECT_STORAGE_KEY)
    return raw
  } catch {
    return ''
  }
}

/** 若存在暂存的原登录落点则跳回并返回 true；否则返回 false。 */
export function maybeNavigateToPhoneVerifyRedirect(navigate = (href) => { window.location.href = href }) {
  const pending = consumePhoneVerifyRedirect()
  if (pending) {
    navigate(pending)
    return true
  }
  return false
}

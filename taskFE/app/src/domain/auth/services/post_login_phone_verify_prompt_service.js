import { userHasVerifiedPhone } from '../value_objects/phone_verified_status.js'
import { buildPhoneBindingRedirectUrl, savePhoneVerifyRedirect } from '../../../utils/phoneBindingDeepLink.js'

export const LOGIN_PHONE_VERIFY_PROMPT_TITLE = '请验证手机号'
export const LOGIN_PHONE_VERIFY_PROMPT_MESSAGE =
  '须先完成手机号验证后才能进入工作台等业务页面。请前往个人资料绑定手机号并完成短信验证。'
export const LOGIN_PHONE_VERIFY_PROMPT_CONFIRM = '去验证'

export function shouldSkipPhoneVerifyPrompt({
  adminLogin = false,
  loginMethod = '',
  isImpersonating = false,
} = {}) {
  return Boolean(adminLogin) || loginMethod === 'phonePassword' || Boolean(isImpersonating)
}

function gateToBinding(redirectUrl, log, reason) {
  log(`[login-phone-verify] action=verify reason=${reason}`)
  savePhoneVerifyRedirect(redirectUrl)
  return { action: 'verify', href: buildPhoneBindingRedirectUrl() }
}

/**
 * 登录成功后：未验证手机则硬门禁到资料绑定页（不可跳过）。
 * acknowledge 语义：resolve=去验证；reject/缺失仍去验证。
 */
export async function promptPostLoginPhoneVerify({
  skipPrompt = false,
  source,
  redirectUrl,
  acknowledge,
  confirm,
  log = (...args) => console.log(...args),
} = {}) {
  const href = typeof redirectUrl === 'string' && redirectUrl.trim() ? redirectUrl : '/system-admin/'
  if (skipPrompt) {
    log('[login-phone-verify] action=skip reason=skip_prompt')
    return { action: 'redirect', href }
  }
  let verified = false
  try {
    verified = userHasVerifiedPhone(source)
  } catch {
    return gateToBinding(href, log, 'predicate_error')
  }
  if (verified) {
    log('[login-phone-verify] action=skip reason=already_verified')
    return { action: 'redirect', href }
  }
  const ask = typeof acknowledge === 'function' ? acknowledge : confirm
  if (typeof ask === 'function') {
    try {
      await ask(
        LOGIN_PHONE_VERIFY_PROMPT_MESSAGE,
        LOGIN_PHONE_VERIFY_PROMPT_TITLE,
        LOGIN_PHONE_VERIFY_PROMPT_CONFIRM,
      )
    } catch {
      // 关闭/拒绝不可跳过门禁
    }
  } else {
    log('[login-phone-verify] action=verify reason=no_acknowledge')
  }
  return gateToBinding(href, log, 'unverified_gate')
}

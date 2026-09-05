import { persistLoginSuccessCredentials } from '../domain/auth/services/activate_session_service.js'
import { promptPostLoginPhoneVerify } from '../domain/auth/services/post_login_phone_verify_prompt_service.js'
import { resolveWechatSessionIdentity } from './sessionUserIdUtils.js'
import { resolveWechatCallbackRedirect } from './wechatLoginFlow.js'
import modalService from './modalService.js'

/**
 * 微信回调 token 分支：落盘凭据 → 解析 profile → 未绑手机则引导验证 → 跳转。
 * 身份解析失败 fail-open：不弹窗，仍跳原落点（与修复前「不阻断登录」一致）。
 */
export async function finishWechatCallbackLogin({
  wechatToken,
  next,
  persist = persistLoginSuccessCredentials,
  resolveIdentity = resolveWechatSessionIdentity,
  resolveRedirect = resolveWechatCallbackRedirect,
  promptPhoneVerify = promptPostLoginPhoneVerify,
  confirm = (message, title, confirmText) =>
    modalService.alert(message, title, { confirmText, showCloseButton: false }),
  navigate = (href) => {
    window.location.href = href
  },
    replaceHistory = (cleanUrl) => {
      if (typeof window === 'undefined' || !window.history?.replaceState) return
      window.history.replaceState({}, document.title, cleanUrl)
    },
  log = (...args) => console.log(...args),
} = {}) {
  await persist({ token: wechatToken })
  const identity = await resolveIdentity(wechatToken)
  const identityOk = Boolean(identity?.ok)
  log(`[wechat-callback] identity_resolved=${identityOk}`)
  const redirectUrl = resolveRedirect(next)
  const pathname = typeof window !== 'undefined' ? window.location.pathname : '/'
  const search = typeof window !== 'undefined' ? window.location.search : ''
  const cleanUrl =
    pathname +
    search
      .replace(/[?&]wechat_token=[^&]*/g, '')
      .replace(/[?&]next=[^&]*/g, '')
      .replace(/\?$/, '')
  replaceHistory(cleanUrl)
  const decision = await promptPhoneVerify({
    skipPrompt: !identityOk,
    source: identity?.profile || {},
    redirectUrl,
    confirm,
  })
  log(`[wechat-callback] navigate href=${decision.href} phoneVerify=${decision.action}`)
  navigate(decision.href)
  return decision
}

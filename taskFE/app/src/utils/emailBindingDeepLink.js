/**
 * 邮箱绑定深链契约（OPT-20260812-016）：
 * SSO bridge（taskAuth sso_bridge.go）、ImageMarket、UserProfile 三处共用同一
 * 深链 key 与引导 URL，避免字面量漂移导致「toast 仍在、区块对不上」。
 *
 * - `sso_error=email_required`：taskAuth 对无邮箱用户（典型微信扫码）拒绝签发
 *   vendor_bridge 时重定向携带的查询参数。
 * - `#rg=profile.email_binding`：UserProfile 邮箱绑定面板 data-rg-key，页面加载后
 *   rgDeepLink 据此滚动定位并短暂高亮。
 */
export const EMAIL_BINDING_RG_KEY = 'profile.email_binding'

export const EMAIL_REQUIRED_SSO_ERROR = 'email_required'

/** 构造邮箱绑定引导 URL（相对路径，与 taskAuth sso_bridge.go 拼接 GatewayPublicBase 一致） */
export function buildEmailBindingRedirectUrl() {
  return `/profile/?sso_error=${EMAIL_REQUIRED_SSO_ERROR}#rg=${EMAIL_BINDING_RG_KEY}`
}

/** 判断当前查询参数是否来自邮箱绑定 SSO 引导 */
export function isEmailRequiredSsoError(query) {
  return Boolean(query && query.sso_error === EMAIL_REQUIRED_SSO_ERROR)
}

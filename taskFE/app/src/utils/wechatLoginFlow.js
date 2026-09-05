import { PostLoginReturnUrl } from '../domain/auth/value_objects/post_login_return_url_value_object.js'
import { resolvePostLoginRedirectUrl } from './resolvePostLoginRedirect.js'
import { extractTraceId, looksLikeTraceId } from './traceId.js'
import { readStoredReferralAccessCode } from './referralAccessCodeUtils.js'

const WECHAT_LOGIN_PATH = '/api/auth/wechat/login/'

/**
 * 构建微信 OAuth 入口 URL，携带净化后的 next 回跳地址。
 * next 仅接受同源相对路径（PostLoginReturnUrl.normalize），非法值直接丢弃 —
 * 与后端 taskAuth sanitizeNextPath 双向防护，防开放重定向。
 *
 * @param {'web' | 'inapp' | string} app 微信应用 key（web=扫码 / inapp=公众号授权）
 * @param {string | null | undefined} [next] 当前登录页的 next 参数
 * @returns {string} 例如 /api/auth/wechat/login/?app=web&next=%2Fprojects%2F
 */
export function buildWechatOAuthUrl(app, next) {
  const query = new URLSearchParams({ app: String(app || 'web') })
  const normalized = PostLoginReturnUrl.normalize(String(next ?? ''))
  // OPT-20260806-043: next 指向 /onboarding 引导页时置空，让后端按角色计算落点
  // （后端回调守卫已纠正该回显；前端过滤消除误导性请求参数，双保险）。
  if (normalized && normalized !== '/onboarding' && !normalized.startsWith('/onboarding/')) {
    query.set('next', normalized)
  }
  // OPT-20260820-036: 分享链接带 ?accessCode= 时经登录页写入 sessionStorage，
  // 这里透传到微信 OAuth 入口；taskAuth state 携带该码，首次扫码自动注册后回填推荐关系。
  const accessCode = readStoredReferralAccessCode()
  if (accessCode) {
    query.set('accessCode', accessCode)
  }
  return `${WECHAT_LOGIN_PATH}?${query.toString()}`
}

/**
 * 解析微信回调成功后的登录落点。
 * next 为后端回调携带（用户原始回跳或按角色计算的落点，taskAuth 保证必有）；
 * 此处仅做规范化/防开放重定向校验，不再读 localStorage 兜底。
 * 永不落到公开首页 "/"。
 *
 * @param {string | null | undefined} [next] 回调 URL 的 next 参数
 * @returns {string}
 */
export function resolveWechatCallbackRedirect(next) {
  return resolvePostLoginRedirectUrl(next, {})
}

/**
 * 预检微信 OAuth 入口：上游（taskAuth）不可达时 APISIX 返回 502，
 * 禁止整页跳进裸 HTML 502。健康时再 window.location 跳转。
 *
 * @param {string} url buildWechatOAuthUrl 结果（同源相对或绝对路径）
 * @param {{ fetchImpl?: typeof fetch, assign?: (href: string) => void }} [deps]
 * @returns {Promise<{status:'ok',url:string}|{status:'unavailable',message:string,traceId:string,httpStatus?:number}>}
 */
export async function navigateWechatOAuth(url, deps = {}) {
  const fetchImpl = deps.fetchImpl || globalThis.fetch.bind(globalThis)
  const assign =
    deps.assign ||
    ((href) => {
      window.location.assign(href)
    })

  let response
  try {
    response = await fetchImpl(url, {
      method: 'GET',
      redirect: 'manual',
      credentials: 'same-origin',
      headers: { Accept: 'text/html,application/xhtml+xml,*/*' },
    })
  } catch (e) {
    const traceId = extractTraceId(e) || `fe-wechat-${Date.now().toString(36)}`
    console.warn('[wechat-login] preflight network error', { traceId, err: String(e?.message || e) })
    return {
      status: 'unavailable',
      message: '登录服务暂时不可用，请稍后重试',
      traceId,
    }
  }

  const headerTrace = extractTraceId(response)
  const traceId =
    (looksLikeTraceId(headerTrace) && headerTrace) || `fe-wechat-${Date.now().toString(36)}`

  const redirected =
    response.type === 'opaqueredirect' ||
    (response.status >= 300 && response.status < 400)

  if (redirected) {
    console.log(`[wechat-login] preflight ok status=${response.status} type=${response.type} traceId=${traceId}`)
    assign(url)
    return { status: 'ok', url }
  }

  if (response.status >= 500 || response.status === 0) {
    console.warn(
      `[wechat-login] preflight upstream unavailable status=${response.status} traceId=${traceId}`,
    )
    return {
      status: 'unavailable',
      message: '登录服务暂时不可用，请稍后重试',
      traceId,
      httpStatus: response.status,
    }
  }

  if (response.status === 403 || response.status === 404 || response.status === 429) {
    console.warn(`[wechat-login] preflight rejected status=${response.status} traceId=${traceId}`)
    return {
      status: 'unavailable',
      message: '微信登录暂不可用，请使用其他方式登录或联系管理员',
      traceId,
      httpStatus: response.status,
    }
  }

  console.log(`[wechat-login] preflight pass-through status=${response.status} traceId=${traceId}`)
  assign(url)
  return { status: 'ok', url }
}

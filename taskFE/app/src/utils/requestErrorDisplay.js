/**
 * 请求失败错误展示统一出口：经 modal/toast 挂载 data-traceId。
 * 禁止对请求失败使用无法挂 DOM 属性的 window.alert。
 */
import modalService from './modalService.js'
import toastService from './toastService.js'
import { extractTraceId } from './traceId.js'
import { CHUNK_STALE_RELOAD_MESSAGE, isChunkLoadError } from './chunkLoadGuard.js'
import { humanizePaymentProviderError } from './humanizePaymentProviderError.js'

/**
 * @param {unknown} message
 * @param {unknown} [source] Error / Response / 含 traceId 的对象；禁止把展示文案当 trace 源
 * @param {{ title?: string, autoClose?: number, traceId?: string }} [options]
 */
export function showRequestError(message, source, options = {}) {
  // 单参 Error/Response：从对象取 trace；展示文案永不作为 fallback（避免 data-traceId=错误中文）
  const traceSource =
    source !== undefined && source !== null
      ? source
      : message != null && typeof message === 'object'
        ? message
        : undefined
  const traceId =
    extractTraceId(traceSource) ||
    extractTraceId(options.traceId) ||
    extractTraceId(options.source)
  let text = message
  if (message instanceof Error) {
    text = message.message
  } else if (text != null && typeof text === 'object' && 'message' in /** @type {object} */ (text)) {
    text = /** @type {{ message: string }} */ (text).message
  }
  return modalService.alert(humanizeRequestErrorMessage(text ?? '请求失败'), options.title || '错误', {
    ...options,
    traceId,
  })
}

// 原始浏览器网络错误的常见英文文案（fetch/AbortController 直抛，未经 apiFetch 翻译）。
const RAW_NETWORK_ERROR_MARKERS = [
  'failed to fetch',
  'networkerror when attempting to fetch resource',
  'load failed',
  'network request failed',
  'the internet connection appears to be offline',
]

/**
 * 统一 humanize 请求失败文案：原始英文网络错误 → 中文；其余原样返回。
 * 用于非 modal/toast 的内联 error.message 展示点（UserProfile、AccessTokenManagementPanel、
 * PendingInvitations、UserGitIdentities 等），避免继续暴露裸英文（OPT-20260811-070/073）。
 * @param {unknown} message
 * @returns {string}
 */
export function humanizeRequestErrorMessage(message) {
  const text = String(message ?? '').trim()
  if (!text) return '请求失败，请稍后重试'
  if (isChunkLoadError(text)) return CHUNK_STALE_RELOAD_MESSAGE
  const lowered = text.toLowerCase()
  if (RAW_NETWORK_ERROR_MARKERS.some((m) => lowered.includes(m))) {
    return '网络错误，请稍后重试'
  }
  if (lowered === 'git oauth not connected') {
    return '尚未绑定 Git 网站 OAuth，请先在账号中心完成授权后再试'
  }
  if (
    lowered.includes('refresh http 400') ||
    lowered.includes('refresh http 401') ||
    lowered.includes('decrypt_failed')
  ) {
    return 'Git OAuth 授权已失效，请重新绑定后再试'
  }
  return humanizePaymentProviderError(text, text)
}

/**
 * @param {unknown} message
 * @param {unknown} [source]
 * @param {number} [duration]
 */
/**
 * 统一从 API 响应提取错误文案，消除全站重复的 `message → error → detail → fallback` 链。
 * @param {unknown} result - API 返回的 JSON 对象或错误体
 * @param {string} [fallback='未知错误'] - 所有字段均不可用时的兜底文案
 * @returns {string}
 */
export function formatApiError(result, fallback = '未知错误') {
  let raw = fallback
  if (result && typeof result === 'object') {
    if (typeof result.message === 'string' && result.message) raw = result.message
    else if (typeof result.error === 'string' && result.error) raw = result.error
    else if (typeof result.detail === 'string' && result.detail) raw = result.detail
  }
  return humanizePaymentProviderError(raw, fallback)
}

// 网关 forward-auth 会话失效的标准文案（taskAuth gateway_forward_auth.go）。
// 触发时浏览器持有的凭据（裸 userId cookie/失效 token）已无法解析 → 唯一出路是重新登录。
const FORWARD_AUTH_SESSION_EXPIRED = '无法解析登录凭据，请重新登录'

/**
 * 判断 API 错误是否为网关 forward-auth 会话失效（401 + 特定 detail）。
 * @param {unknown} errorData - 已解析的响应体（可能含 detail）
 * @param {Response} [response] - fetch Response（可选，辅助取 _errorData）
 * @returns {boolean}
 */
export function isForwardAuthSessionExpired(errorData, response) {
  const body = errorData && typeof errorData === 'object' ? errorData : (response?._errorData ?? {})
  return body?.detail === FORWARD_AUTH_SESSION_EXPIRED
}

/**
 * 会话失效恢复：跳转登录页并携带回跳地址。
 * 仅当无法解析登录凭据（无凭据/凭据全失效）时使用——其他 401（账号禁用、
 * 归档等）语义不同，不应误跳登录。
 * @param {string} [next] 回跳路径，默认当前路径
 */
export function redirectToLoginOnSessionExpired(next) {
  const target =
    next ||
    (typeof window !== 'undefined' ? window.location.pathname + window.location.search : '/')
  const nextParam = encodeURIComponent(target)
  window.location.href = `/auth/login/?next=${nextParam}`
}

/** 当前文档生命周期内已触发过统一跳转：并发多个 401 只跳一次（页面导航后模块状态自然重置）。 */
let _sessionExpiredRedirectFired = false

/**
 * 任务详情 accessCode 分享页：访客不持平台会话，探测接口 401 是预期态。
 * @param {string} pathname
 * @param {string} search
 */
export function isTaskDetailAccessCodeSharePage(pathname, search) {
  const path = String(pathname || '')
  if (!path.includes('/task-detail/')) return false
  const raw = String(search || '')
  const params = new URLSearchParams(raw.startsWith('?') ? raw.slice(1) : raw)
  return params.has('accessCode') && Boolean(String(params.get('accessCode') || '').trim())
}

const ANONYMOUS_PUBLIC_PATHS = new Set([
  '/',
  '/faq/',
  '/pricing/',
  '/acknowledgments/',
  '/onboarding/',
])

/**
 * 未登录可打开的营销/文档页：Navbar /me 401 是预期，不得整页踢登录。
 * @param {string} pathname
 */
export function isAnonymousPublicPage(pathname) {
  const raw = String(pathname || '').split('#')[0].split('?')[0]
  if (!raw || raw === '/') return true
  const p = raw.endsWith('/') ? raw : `${raw}/`
  return ANONYMOUS_PUBLIC_PATHS.has(p)
}

/**
 * 网关 forward-auth 会话失效的全局统一恢复入口（apiFetch 层调用，全站单点收口）。
 * 任意页面/组件的请求命中「无法解析登录凭据」即触发，页面无需各自接线。
 * 守卫：
 *  - 非浏览器环境（单测/SSR）不处理；
 *  - 已处于 /auth/* 公开页（登录/找回密码等）不处理，避免循环跳转；
 *  - Git OAuth 回调落地页（/redirect/gitsite/*）不处理：该流程会话无关
 *    （state 签名自足，换票不依赖登录态）。回调落在裸域时 Navbar 等
 *    鉴权请求会因跨子域凭据缺失而 401，若被本收口劫持 → 跳登录 →
 *    OAuth 授权中途中断（OPT-20260807-071 线上回归根因）。
 *  - 任务详情 accessCode 分享页不处理：访客凭分享码阅读，绑定探测 401
 *    不得整页踢登录（否则「去绑定」回流后看不到芯片刷新）。
 *  - 公网 FAQ/定价/首页等匿名页不处理：Navbar 会话探测 401 不得挡住文档。
 *  - 同页生命周期内已触发过不重复触发。
 * @returns {boolean} 是否触发了跳转
 */
export function handleForwardAuthSessionExpired() {
  if (_sessionExpiredRedirectFired) return false
  if (typeof window === 'undefined') return false
  const pathname = window.location.pathname || ''
  if (pathname.startsWith('/auth/')) return false
  // Git OAuth v2 回调契约路径：/redirect/gitsite/<gitsite>/oauth/callback/
  if (pathname.startsWith('/redirect/gitsite/')) return false
  if (isTaskDetailAccessCodeSharePage(pathname, window.location.search || '')) return false
  if (isAnonymousPublicPage(pathname)) return false
  _sessionExpiredRedirectFired = true
  console.warn('[session] 网关 forward-auth 会话失效，引导重新登录:', pathname)
  redirectToLoginOnSessionExpired()
  return true
}

export function toastRequestError(message, source, duration = 4000) {
  const traceSource =
    source !== undefined && source !== null
      ? source
      : message != null && typeof message === 'object'
        ? message
        : undefined
  const traceId = extractTraceId(traceSource)
  let text = message instanceof Error ? message.message : message
  toastService.error(humanizeRequestErrorMessage(text ?? '请求失败'), duration, { traceId })
}

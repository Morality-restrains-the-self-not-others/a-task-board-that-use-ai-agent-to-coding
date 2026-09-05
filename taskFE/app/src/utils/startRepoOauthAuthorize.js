import { extractTraceId } from './traceId.js'

/**
 * 请求 OAuth start URL，成功则跳转 authorize_url；失败 Error 带 traceId。
 * @param {(url: string, init?: RequestInit) => Promise<Response>} apiFetch
 * @param {string} targetUrl
 */
export async function fetchAndFollowOauthAuthorizeUrl(apiFetch, targetUrl) {
  const response = await apiFetch(targetUrl, {
    credentials: 'include',
    headers: { Accept: 'application/json' },
  })
  const data = await response.json().catch(() => ({}))
  const requestTraceId = extractTraceId(response) || extractTraceId(data) || ''
  if (!response.ok) {
    const err = new Error(typeof data.detail === 'string' ? data.detail : '无法启动 OAuth 授权')
    if (requestTraceId) err.traceId = requestTraceId
    throw err
  }
  if (data.authorize_url) {
    window.location.href = data.authorize_url
    return
  }
  const missingUrlErr = new Error('无法启动 OAuth 授权')
  if (requestTraceId) missingUrlErr.traceId = requestTraceId
  throw missingUrlErr
}

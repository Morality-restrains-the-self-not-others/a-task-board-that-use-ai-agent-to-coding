/**
 * HTTP 错误响应解析通用函数。
 * 从 fetch Response（含 apiFetch 注入的 _errorData）提取错误文案与 traceId，生成带 traceId 的 Error。
 */
import { extractTraceId } from './traceId.js'

function firstNonEmptyString(value) {
  if (typeof value === 'string' && value.trim()) return value.trim()
  if (Array.isArray(value) && value.length) {
    const first = value[0]
    if (typeof first === 'string' && first.trim()) return first.trim()
    if (first && typeof first === 'object' && typeof first.msg === 'string' && first.msg.trim()) {
      return first.msg.trim()
    }
  }
  return ''
}

function extractMessageFromErrorBody(body) {
  if (!body || typeof body !== 'object') return ''
  for (const key of ['message', 'error', 'error_msg', 'detail']) {
    const found = firstNonEmptyString(body[key])
    if (found) return found
  }
  return ''
}

function stripHtmlErrorText(raw) {
  if (typeof raw !== 'string' || !raw.trim()) return ''
  const stripped = raw.replace(/<[^>]*>/g, ' ').replace(/\s+/g, ' ').trim()
  return stripped ? stripped.slice(0, 200) : ''
}

/**
 * 网关/上游短不可用（APISIX connection refused → 502 HTML 等）。
 * @param {unknown} status
 * @returns {boolean}
 */
export function isTransientHttpStatus(status) {
  const n = Number(status)
  return n === 502 || n === 503 || n === 504
}

/**
 * 网关/上游短不可用（APISIX connection refused → 502 HTML 等）的用户可读文案。
 * JSON 业务错误体优先，本函数只在无 message/error/detail 时使用。
 * @param {unknown} status
 * @returns {string}
 */
export function gatewayUnavailableMessage(status) {
  if (isTransientHttpStatus(status)) {
    return '服务暂时不可用，请稍后重试'
  }
  return ''
}

/**
 * 从失败 Response 提取用户可读文案。
 * 优先 _errorData.message / error / detail；502/503/504 无业务字段时给重试文案；
 * 其余 JSON 解析失败时用 apiFetch 注入的 _rawErrorText。
 * @param {Response|{status?: number, _errorData?: object}} response
 * @param {string} [fallback]
 * @returns {string}
 */
export function messageFromFailedResponse(response, fallback) {
  const data = response?._errorData
  const fromJson = extractMessageFromErrorBody(data)
  if (fromJson) return fromJson
  const fromGateway = gatewayUnavailableMessage(response?.status)
  if (fromGateway) return fromGateway
  const fromHtml = stripHtmlErrorText(data?._rawErrorText)
  if (fromHtml) return fromHtml
  if (typeof fallback === 'string' && fallback) return fallback
  return response?.status ? `HTTP ${response.status}` : '请求失败'
}

/**
 * @param {Response} response - fetch Response（可选带 _errorData）
 * @param {string} fallback - 无法从响应体提取文案时的兜底
 * @returns {Error} 带 traceId 属性的 Error
 */
export function errorFromFailedResponse(response, fallback) {
  const msg = messageFromFailedResponse(response, fallback)
  const err = new Error(msg)
  const tid = response?.traceId || extractTraceId(response) || extractTraceId(response?._errorData)
  if (tid) err.traceId = tid
  return err
}

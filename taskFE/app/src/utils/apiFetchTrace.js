import { resolveRequestTraceId } from './traceId.js'

export const TRACE_ID_HEADER = 'X-Trace-Id'
export const PARENT_SPAN_ID_HEADER = 'X-Parent-Span-Id'
export const TRACEPARENT_HEADER = 'traceparent'

/**
 * @param {Response} response
 * @param {Record<string, string>} requestHeaders
 * @param {unknown} [bodyHint]
 */
export function attachTraceIdToResponse(response, requestHeaders, bodyHint) {
  const tid = resolveRequestTraceId({
    responseHeaders: response.headers,
    requestHeaders,
    body: bodyHint ?? response._errorData,
  })
  if (tid) {
    response.traceId = tid
  }
  return tid
}

/**
 * @param {Error} error
 * @param {string} [traceId]
 */
export function attachTraceIdToError(error, traceId) {
  if (error && traceId) {
    error.traceId = traceId
  }
  return error
}

function newRequestTraceId() {
  try {
    if (typeof crypto !== 'undefined' && crypto.randomUUID) {
      return crypto.randomUUID()
    }
  } catch {
    /* ignore */
  }
  return `web-${Date.now()}-${Math.random().toString(36).slice(2, 14)}`
}

function newClientSpanId() {
  try {
    if (typeof crypto !== 'undefined' && crypto.getRandomValues) {
      const bytes = new Uint8Array(8)
      crypto.getRandomValues(bytes)
      return Array.from(bytes, (b) => b.toString(16).padStart(2, '0')).join('')
    }
  } catch {
    /* ignore */
  }
  return Math.random().toString(16).slice(2, 10) + Math.random().toString(16).slice(2, 10)
}

async function otelTraceIdHex(externalId) {
  const raw = String(externalId || '').trim()
  const compact = raw.replace(/-/g, '')
  if (/^[0-9a-fA-F]{32}$/.test(compact)) return compact.toLowerCase()
  try {
    if (typeof crypto !== 'undefined' && crypto.subtle?.digest) {
      const data = new TextEncoder().encode(raw)
      const buf = await crypto.subtle.digest('SHA-256', data)
      return Array.from(new Uint8Array(buf))
        .map((b) => b.toString(16).padStart(2, '0'))
        .join('')
        .slice(0, 32)
    }
  } catch {
    /* ignore */
  }
  return compact.padEnd(32, '0').slice(0, 32)
}

async function formatTraceparent(traceId, spanId) {
  const tid = String(traceId || '').trim()
  const sid = String(spanId || '').trim().toLowerCase()
  if (!tid || !sid) return ''
  return `00-${await otelTraceIdHex(tid)}-${sid}-01`
}

/**
 * @param {Record<string, string>} headers
 */
export async function attachTracePropagationHeaders(headers) {
  if (!headers[TRACE_ID_HEADER] && !headers['x-trace-id']) {
    headers[TRACE_ID_HEADER] = newRequestTraceId()
  }
  if (!headers[PARENT_SPAN_ID_HEADER] && !headers['x-parent-span-id']) {
    const clientSpan = newClientSpanId()
    headers[PARENT_SPAN_ID_HEADER] = clientSpan
    const tp = await formatTraceparent(headers[TRACE_ID_HEADER], clientSpan)
    if (tp && !headers[TRACEPARENT_HEADER]) {
      headers[TRACEPARENT_HEADER] = tp
    }
  }
}

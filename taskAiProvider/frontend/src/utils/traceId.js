/** 与项目元规则 data-traceId 对齐的 traceId 解析 */

export function newRequestTraceId() {
  try {
    if (typeof crypto !== 'undefined' && crypto.randomUUID) {
      return crypto.randomUUID()
    }
  } catch {
    /* ignore */
  }
  return `aip-${Date.now()}-${Math.random().toString(36).slice(2, 12)}`
}

/** 16-hex span id；与 shareLib/tracelog X-Parent-Span-Id 校验一致 */
export function newRequestSpanId() {
  try {
    if (typeof crypto !== 'undefined' && typeof crypto.getRandomValues === 'function') {
      const bytes = new Uint8Array(8)
      crypto.getRandomValues(bytes)
      return Array.from(bytes, (b) => b.toString(16).padStart(2, '0')).join('')
    }
  } catch {
    /* ignore */
  }
  let hex = ''
  for (let i = 0; i < 16; i++) hex += ((Math.random() * 16) | 0).toString(16)
  return hex
}

export function traceIdFromHeaders(headers) {
  if (!headers) return ''
  if (typeof headers.get === 'function') {
    return String(headers.get('X-Trace-Id') || headers.get('x-trace-id') || '').trim()
  }
  for (const [k, v] of Object.entries(headers)) {
    if (String(k).toLowerCase() === 'x-trace-id' && v) return String(v).trim()
  }
  return ''
}

export function parentSpanIdFromHeaders(headers) {
  if (!headers) return ''
  if (typeof headers.get === 'function') {
    return String(headers.get('X-Parent-Span-Id') || headers.get('x-parent-span-id') || '').trim()
  }
  for (const [k, v] of Object.entries(headers)) {
    if (String(k).toLowerCase() === 'x-parent-span-id' && v) return String(v).trim()
  }
  return ''
}

/**
 * 浏览器出站须完整传播：有 X-Trace-Id 时必须同时有 X-Parent-Span-Id（或合法 traceparent）。
 * 见 shareLib/tracelog RejectTraceIdOnlyHTTP。
 */
export function buildOutboundTraceHeaders(existingHeaders) {
  const requestTraceId = traceIdFromHeaders(existingHeaders) || newRequestTraceId()
  const parentSpanId = parentSpanIdFromHeaders(existingHeaders) || newRequestSpanId()
  return {
    requestTraceId,
    parentSpanId,
    headers: {
      'X-Trace-Id': requestTraceId,
      'X-Parent-Span-Id': parentSpanId,
    },
  }
}

export function extractTraceId(source) {
  if (source == null) return ''
  if (typeof source === 'string') return source.trim()
  if (typeof source !== 'object') return ''
  if (typeof source.traceId === 'string' && source.traceId.trim()) return source.traceId.trim()
  if (typeof source.trace_id === 'string' && source.trace_id.trim()) return source.trace_id.trim()
  if (source.data && typeof source.data === 'object') {
    const d = source.data
    if (typeof d.trace_id === 'string' && d.trace_id.trim()) return d.trace_id.trim()
    if (typeof d.traceId === 'string' && d.traceId.trim()) return d.traceId.trim()
  }
  return ''
}

/** 网络层失败无响应头时，把本端已发出的 X-Trace-Id 挂到 Error，供 data-traceId */
export function attachClientTraceId(err, requestTraceId) {
  const id = typeof requestTraceId === 'string' ? requestTraceId.trim() : ''
  if (!err || typeof err !== 'object' || !id) return err
  if (!err.traceId) err.traceId = id
  return err
}

/** @param {Element|null|undefined} el @param {unknown} source */
export function setDataTraceId(el, source) {
  if (!el?.setAttribute) return
  const id = extractTraceId(source)
  if (id) el.setAttribute('data-traceId', id)
  else el.removeAttribute('data-traceId')
}

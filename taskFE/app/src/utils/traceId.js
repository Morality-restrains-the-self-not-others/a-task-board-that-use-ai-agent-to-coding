/**
 * 请求失败展示用的 traceId 解析（与元规则 data-traceId 对齐）。
 */

const TRACE_HEADER_CANDIDATES = ['x-trace-id', 'X-Trace-Id']

/**
 * 可观测 trace 形态：UUID / web-* / 网关短 token 等。
 * 禁止空白与 CJK（避免「创建分组失败」误入 data-traceId）；最短 2 字符。
 */
const TRACE_ID_SHAPE = /^[A-Za-z0-9][A-Za-z0-9._:-]{1,127}$/

/**
 * 判断字符串是否像有效 traceId（禁止把「创建分组失败」等用户文案写入 data-traceId）。
 * @param {unknown} value
 * @returns {boolean}
 */
export function looksLikeTraceId(value) {
  if (typeof value !== 'string') return false
  const s = value.trim()
  if (!s || s.length > 128) return false
  // 含空白、CJK、全角标点的几乎都是错误文案而非 traceId
  if (/[\s\u3400-\u9FFF\uF900-\uFAFF\uFF00-\uFFEF]/.test(s)) return false
  return TRACE_ID_SHAPE.test(s)
}

/**
 * 真实网关格式：APISIX 网关与上游服务各自注入一行 X-Trace-Id，浏览器把多行同名字头
 * 合并为 "id1, id2"（逗号+空格）。合法 traceId 不含逗号（TRACE_ID_SHAPE），
 * 拆分安全；取首段即可全链路检索（网关与上游共享同一 trace）。
 */
function pickTraceIdHeaderValue(v) {
  if (!v) return ''
  const s = String(v).trim()
  if (!s) return ''
  const first = s.split(',')[0].trim()
  return first || ''
}

/**
 * @param {Headers|Record<string, string>|undefined|null} headers
 * @returns {string}
 */
export function traceIdFromHeaders(headers) {
  if (!headers) return ''
  if (typeof headers.get === 'function') {
    for (const name of TRACE_HEADER_CANDIDATES) {
      const picked = pickTraceIdHeaderValue(headers.get(name))
      if (picked) return picked
    }
    return ''
  }
  for (const [k, v] of Object.entries(headers)) {
    if (String(k).toLowerCase() === 'x-trace-id') {
      const picked = pickTraceIdHeaderValue(v)
      if (picked) return picked
    }
  }
  return ''
}

/**
 * @param {unknown} body
 * @returns {string}
 */
export function traceIdFromBody(body) {
  if (!body || typeof body !== 'object') return ''
  const o = /** @type {Record<string, unknown>} */ (body)
  for (const key of ['trace_id', 'traceId', '_traceId']) {
    const v = o[key]
    if (typeof v === 'string' && v.trim()) return v.trim()
  }
  return ''
}

/**
 * 按元规则优先级合并：响应头 → 请求头 → 响应体 → 已有 error.traceId。
 * @param {{ responseHeaders?: Headers|Record<string,string>|null, requestHeaders?: Headers|Record<string,string>|null, body?: unknown, fallback?: unknown }} parts
 * @returns {string}
 */
export function resolveRequestTraceId(parts = {}) {
  const fromResponse = traceIdFromHeaders(parts.responseHeaders)
  if (fromResponse) return fromResponse
  const fromRequest = traceIdFromHeaders(parts.requestHeaders)
  if (fromRequest) return fromRequest
  const fromBody = traceIdFromBody(parts.body)
  if (fromBody) return fromBody
  return extractTraceId(parts.fallback)
}

/**
 * 从 Error / Response / 字符串 / 任意对象上提取 traceId。
 * @param {unknown} source
 * @returns {string}
 */
export function extractTraceId(source) {
  if (source == null || source === '') return ''
  if (typeof source === 'string') {
    const s = source.trim()
    if (!s) return ''
    // 响应体 JSON 字符串：只抽取 body 内 trace 字段，禁止把整段 JSON 当成 traceId
    if (s.startsWith('{') || s.startsWith('[')) {
      try {
        const fromBody = traceIdFromBody(JSON.parse(s))
        return looksLikeTraceId(fromBody) ? fromBody : ''
      } catch {
        return ''
      }
    }
    return looksLikeTraceId(s) ? s : ''
  }
  if (typeof source !== 'object') return ''
  const o = /** @type {Record<string, unknown>} */ (source)
  if (typeof o.traceId === 'string' && looksLikeTraceId(o.traceId)) return o.traceId.trim()
  if (typeof o.trace_id === 'string' && looksLikeTraceId(o.trace_id)) return o.trace_id.trim()
  if (typeof o._traceId === 'string' && looksLikeTraceId(o._traceId)) return o._traceId.trim()
  if (o.headers) {
    const fromH = traceIdFromHeaders(/** @type {any} */ (o.headers))
    if (fromH && looksLikeTraceId(fromH)) return fromH
  }
  if (o._errorData) {
    const fromBody = traceIdFromBody(o._errorData)
    if (fromBody && looksLikeTraceId(fromBody)) return fromBody
  }
  return ''
}

/**
 * 仅在有非空 traceId 时设置 data-traceId；否则移除属性。
 * @param {Element|null|undefined} el
 * @param {unknown} traceIdOrSource
 */
export function setDataTraceId(el, traceIdOrSource) {
  if (!el || typeof el.setAttribute !== 'function') return
  const id = extractTraceId(traceIdOrSource)
  if (id) {
    el.setAttribute('data-traceId', id)
  } else {
    el.removeAttribute('data-traceId')
  }
}

/** OAuth 回调落地后拼回跳 href，并拆成 Vue Router location（保留 accessCode 等 query）。 */

/**
 * 授权跳转前的 next/return_url：当前页 path + search；location.search 空时回退 route.query。
 * @param {{ routePath?: string, pathname?: string, search?: string, query?: Record<string, unknown> }} [opts]
 * @returns {string}
 */
export function buildOauthReturnPath(opts = {}) {
  const routePath = String(opts.routePath || '').trim()
  const pathname = String(opts.pathname || '').trim()
  const locSearch = String(opts.search || '')
  const basePath = routePath || pathname || '/'
  if (locSearch) return `${basePath}${locSearch}`
  const q = opts.query && typeof opts.query === 'object' ? opts.query : {}
  const params = new URLSearchParams()
  Object.entries(q).forEach(([k, v]) => {
    if (v == null || v === '') return
    const val = Array.isArray(v) ? v[0] : v
    if (val == null || val === '') return
    params.set(k, String(val))
  })
  const qs = params.toString()
  return qs ? `${basePath}?${qs}` : basePath
}

export function mergeQueryParam(pathWithSearch, key, value) {
  if (!key || value == null || value === '') {
    return pathWithSearch
  }
  const k = String(key)
  const s = String(value)
  if (new RegExp(`(?:^|[?&])${k}=`).test(pathWithSearch)) {
    return pathWithSearch
  }
  const joiner = pathWithSearch.includes('?') ? '&' : '?'
  return `${pathWithSearch}${joiner}${encodeURIComponent(k)}=${encodeURIComponent(s)}`
}

/**
 * @param {{ returnUrl: string, provider: string, providerParam: string, traceId?: string }} opts
 * @returns {string}
 */
export function resolveGithubAppContinueHref(opts) {
  const returnUrl = String(opts?.returnUrl || '')
  const provider = String(opts?.provider || 'github')
  const providerParam = opts?.providerParam != null ? String(opts.providerParam) : 'ok'
  const traceId = String(opts?.traceId || '').trim()
  let finalPath = mergeQueryParam(returnUrl, provider, providerParam)
  if (traceId && providerParam !== 'ok') {
    finalPath = mergeQueryParam(finalPath, 'trace_id', traceId)
  }
  return finalPath
}

/**
 * @param {string} pathWithSearch
 * @returns {{ path: string, query: Record<string, string> }}
 */
export function hrefToRouterLocation(pathWithSearch) {
  const raw = String(pathWithSearch || '')
  const q = raw.indexOf('?')
  if (q < 0) {
    return { path: raw, query: {} }
  }
  return {
    path: raw.slice(0, q),
    query: Object.fromEntries(new URLSearchParams(raw.slice(q + 1))),
  }
}

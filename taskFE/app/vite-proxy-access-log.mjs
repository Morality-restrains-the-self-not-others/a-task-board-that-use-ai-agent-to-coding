/**
 * Dev-only structured log for Vite → taskContainerGateway proxy hops (Loki/Promtail).
 */

const proxyStarts = new WeakMap()

function traceIdFromReq(req) {
  const raw = req?.headers?.['x-trace-id']
  if (Array.isArray(raw)) return String(raw[0] || '').trim()
  return String(raw || '').trim()
}

export function emitProxyForwardLog(fields) {
  const line = {
    ts: new Date().toISOString(),
    level: 'info',
    service: 'taskFE',
    msg: 'proxy_forward',
    proxy_target: 'task-container-gateway',
    ...fields,
  }
  process.stdout.write(`${JSON.stringify(line)}\n`)
}

/**
 * @param {import('http-proxy').Server} proxy
 * @param {{ target?: string }} [_opts]
 */
export function attachContainerGitCommitProxyAccessLog(proxy, _opts = {}) {
  proxy.on('proxyReq', (proxyReq, req) => {
    const traceId = traceIdFromReq(req)
    if (traceId) proxyReq.setHeader('X-Trace-Id', traceId)
    proxyStarts.set(req, { start: Date.now(), traceId, path: req.url || '' })
  })
  proxy.on('proxyRes', (proxyRes, req) => {
    const meta = proxyStarts.get(req)
    if (!meta) return
    proxyStarts.delete(req)
    emitProxyForwardLog({
      trace_id: meta.traceId,
      path: meta.path,
      status: proxyRes.statusCode || 0,
      duration_ms: Date.now() - meta.start,
    })
  })
  proxy.on('error', (err, req) => {
    const meta = proxyStarts.get(req)
    if (!meta) return
    proxyStarts.delete(req)
    emitProxyForwardLog({
      trace_id: meta.traceId,
      path: meta.path,
      status: 502,
      duration_ms: Date.now() - meta.start,
      detail: String(err?.message || err).slice(0, 200),
    })
  })
}

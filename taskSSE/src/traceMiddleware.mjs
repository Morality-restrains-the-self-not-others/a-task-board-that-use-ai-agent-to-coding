import crypto from 'node:crypto';

export const TRACE_HEADER = 'x-trace-id';
export const SPAN_HEADER = 'x-span-id';
export const PARENT_SPAN_HEADER = 'x-parent-span-id';
export const TRACEPARENT_HEADER = 'traceparent';

const SAFE_TRACE = /^[A-Za-z0-9._:-]{8,256}$/;
const HEX16 = /^[0-9a-fA-F]{16}$/;
const TRACEPARENT_RE = /^00-([0-9a-fA-F]{32})-([0-9a-fA-F]{16})-([0-9a-fA-F]{2})$/i;

function normalizeTraceId(raw) {
  const s = String(raw || '').trim();
  if (!s || s.length > 256 || !SAFE_TRACE.test(s)) return null;
  return s;
}

function normalizeSpanId(raw) {
  const s = String(raw || '').trim();
  if (!s || s.length > 32 || !HEX16.test(s)) return null;
  return s.toLowerCase();
}

export function otelTraceIdHex(external) {
  const raw = String(external || '').trim();
  const compact = raw.replace(/-/g, '');
  if (/^[0-9a-fA-F]{32}$/.test(compact)) return compact.toLowerCase();
  return crypto.createHash('sha256').update(raw).digest('hex').slice(0, 32);
}

function newTraceId() {
  return `tsse-${Date.now()}-${crypto.randomBytes(6).toString('hex')}`;
}

function newSpanId() {
  return crypto.randomBytes(8).toString('hex');
}

function formatTraceparent(traceId, spanId) {
  if (!traceId) return '';
  const traceHex = otelTraceIdHex(traceId);
  const parentHex = spanId || '0000000000000000';
  return `00-${traceHex}-${parentHex}-01`;
}

function parseTraceparent(raw) {
  const m = TRACEPARENT_RE.exec(String(raw || '').trim());
  if (!m) return null;
  let parent = m[2].toLowerCase();
  if (parent === '0000000000000000') parent = '';
  return { traceId: `tp-${m[1].toLowerCase()}`, parentSpanId: parent };
}

/** Resolve inbound trace/span correlation; always generates a new span id. */
export function resolveInboundCorrelation(req) {
  let traceId =
    normalizeTraceId(req?.headers?.[TRACE_HEADER]) ||
    normalizeTraceId(req?.headers?.[TRACE_HEADER.toLowerCase()]);
  let parentSpanId =
    normalizeSpanId(req?.headers?.[PARENT_SPAN_HEADER]) ||
    normalizeSpanId(req?.headers?.[PARENT_SPAN_HEADER.toLowerCase()]) ||
    '';
  if (!traceId) {
    const tp = parseTraceparent(req?.headers?.[TRACEPARENT_HEADER] ?? req?.headers?.traceparent);
    if (tp) {
      traceId = tp.traceId;
      if (!parentSpanId) parentSpanId = tp.parentSpanId;
    }
  }
  if (!traceId) traceId = newTraceId();
  return { traceId, spanId: newSpanId(), parentSpanId };
}

/** Headers for downstream HTTP calls: current span becomes parent span. */
export function outboundTraceHeaders(correlation) {
  const headers = {};
  if (correlation?.traceId) headers['X-Trace-Id'] = correlation.traceId;
  if (correlation?.spanId) headers['X-Parent-Span-Id'] = correlation.spanId;
  const tp = formatTraceparent(correlation?.traceId, correlation?.spanId);
  if (tp) headers.traceparent = tp;
  return headers;
}

function isTraceIdOnlyRequest(req) {
  const tid =
    normalizeTraceId(req?.headers?.[TRACE_HEADER]) ||
    normalizeTraceId(req?.headers?.[TRACE_HEADER.toLowerCase()]);
  if (!tid) return false;
  const parent =
    normalizeSpanId(req?.headers?.[PARENT_SPAN_HEADER]) ||
    normalizeSpanId(req?.headers?.[PARENT_SPAN_HEADER.toLowerCase()]);
  if (parent) return false;
  const tpRaw = req?.headers?.[TRACEPARENT_HEADER] ?? req?.headers?.traceparent;
  if (tpRaw && parseTraceparent(tpRaw)) return false;
  return true;
}

/** Wrap an HTTP handler with trace/span propagation and JSON access logs for Loki. */
export function withTraceMiddleware(serviceName, handler) {
  return (req, res) => {
    if (isTraceIdOnlyRequest(req)) {
      res.writeHead(400, { 'Content-Type': 'application/json' });
      res.end(
        JSON.stringify({
          detail:
            'trace propagation incomplete: require X-Parent-Span-Id or traceparent with X-Trace-Id',
        }),
      );
      return;
    }
    const corr = resolveInboundCorrelation(req);
    req.traceId = corr.traceId;
    req.spanId = corr.spanId;
    req.parentSpanId = corr.parentSpanId;
    res.setHeader('X-Trace-Id', corr.traceId);
    res.setHeader('X-Span-Id', corr.spanId);
    const tp = formatTraceparent(corr.traceId, corr.spanId);
    if (tp) res.setHeader('traceparent', tp);
    const start = Date.now();
    let status = 200;
    const origWriteHead = res.writeHead.bind(res);
    res.writeHead = (code, ...args) => {
      status = code;
      return origWriteHead(code, ...args);
    };
    let logged = false;
    const logRequest = () => {
      if (logged) return;
      logged = true;
      const path = String(req.url || '/').split('?')[0];
      const payload = {
        ts: new Date().toISOString(),
        level: 'info',
        service: serviceName,
        msg: 'http_request',
        trace_id: corr.traceId,
        otel_trace_id: otelTraceIdHex(corr.traceId),
        method: req.method,
        path,
        status,
        duration_ms: Date.now() - start,
      };
      if (corr.spanId) payload.span_id = corr.spanId;
      if (corr.parentSpanId) payload.parent_span_id = corr.parentSpanId;
      process.stdout.write(`${JSON.stringify(payload)}\n`);
    };
    res.on('finish', logRequest);
    res.on('close', logRequest);
    return handler(req, res);
  };
}

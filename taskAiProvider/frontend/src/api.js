import {
  attachClientTraceId,
  buildOutboundTraceHeaders,
  extractTraceId,
  parentSpanIdFromHeaders,
  traceIdFromHeaders,
} from './utils/traceId.js'
import { enrichResolveArchitectureError } from './utils/privateRegistryHint.js'

function authHeaders(path) {
  if (path.includes("/api/auth/sso/exchange")) {
    return {};
  }
  if (path.includes("/api/admin/") && !path.includes("/api/admin/auth/login")) {
    const t = localStorage.getItem("staff_token");
    return t ? { Authorization: `Bearer ${t}` } : {};
  }
  if (path.includes("/api/vendor/") && !path.includes("/api/vendor/auth/login") && !path.includes("/api/vendor/auth/register")) {
    const t = localStorage.getItem("vendor_token");
    return t ? { Authorization: `Bearer ${t}` } : {};
  }
  return {};
}

function hasTraceparent(headers) {
  if (!headers) return false;
  if (typeof headers.get === "function") {
    return Boolean(headers.get("traceparent") || headers.get("Traceparent"));
  }
  for (const [k, v] of Object.entries(headers)) {
    if (String(k).toLowerCase() === "traceparent" && v) return true;
  }
  return false;
}

export async function api(path, options = {}) {
  const { requestTraceId, parentSpanId, headers: traceHeaders } =
    buildOutboundTraceHeaders(options.headers);
  const headers = {
    "Content-Type": "application/json",
    ...traceHeaders,
    ...authHeaders(path),
    ...options.headers,
  };
  if (!traceIdFromHeaders(headers)) {
    headers["X-Trace-Id"] = requestTraceId;
  }
  // tracelog 拒绝「仅 X-Trace-Id」；合并后若仍缺 parent/traceparent 则补齐。
  if (!parentSpanIdFromHeaders(headers) && !hasTraceparent(headers)) {
    headers["X-Parent-Span-Id"] = parentSpanId;
  }
  let res;
  try {
    res = await fetch(path, { credentials: "include", ...options, headers });
  } catch (networkErr) {
    const err =
      networkErr instanceof Error ? networkErr : new Error(String(networkErr));
    attachClientTraceId(err, requestTraceId);
    throw err;
  }
  const text = await res.text();
  let data = null;
  try {
    data = text ? JSON.parse(text) : null;
  } catch {
    data = { detail: text };
  }
  const responseTraceId =
    traceIdFromHeaders(res.headers) ||
    extractTraceId(data) ||
    requestTraceId;
  if (!res.ok) {
    let detail = data?.detail || data?.message || res.statusText;
    if (String(path).includes('resolve-target-architectures')) {
      detail = enrichResolveArchitectureError(detail, options.body);
    }
    const err = new Error(detail);
    err.status = res.status;
    err.data = data;
    err.traceId = responseTraceId;
    throw err;
  }
  if (data && typeof data === 'object' && !Array.isArray(data) && responseTraceId) {
    data.traceId = data.traceId || responseTraceId;
  }
  return data;
}

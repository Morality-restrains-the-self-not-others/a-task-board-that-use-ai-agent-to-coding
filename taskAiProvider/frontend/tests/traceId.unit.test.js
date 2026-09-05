import test from "node:test";
import assert from "node:assert/strict";
import {
  attachClientTraceId,
  buildOutboundTraceHeaders,
  extractTraceId,
  newRequestSpanId,
  parentSpanIdFromHeaders,
  setDataTraceId,
  traceIdFromHeaders,
} from "../src/utils/traceId.js";

test("extractTraceId reads traceId and trace_id from objects", () => {
  assert.equal(extractTraceId({ traceId: " tid-1 " }), "tid-1");
  assert.equal(extractTraceId({ trace_id: "tid-2" }), "tid-2");
  assert.equal(extractTraceId({ data: { trace_id: "tid-3" } }), "tid-3");
  assert.equal(extractTraceId(null), "");
});

test("traceIdFromHeaders reads X-Trace-Id from fetch Headers", () => {
  const headers = new Headers({ "X-Trace-Id": "hdr-1" });
  assert.equal(traceIdFromHeaders(headers), "hdr-1");
});

test("parentSpanIdFromHeaders reads X-Parent-Span-Id", () => {
  const headers = new Headers({ "X-Parent-Span-Id": "a1b2c3d4e5f67890" });
  assert.equal(parentSpanIdFromHeaders(headers), "a1b2c3d4e5f67890");
});

test("newRequestSpanId returns 16-char hex", () => {
  const id = newRequestSpanId();
  assert.match(id, /^[0-9a-f]{16}$/);
});

test("buildOutboundTraceHeaders always pairs X-Trace-Id with X-Parent-Span-Id", () => {
  const { requestTraceId, parentSpanId, headers } = buildOutboundTraceHeaders();
  assert.ok(requestTraceId);
  assert.match(parentSpanId, /^[0-9a-f]{16}$/);
  assert.equal(headers["X-Trace-Id"], requestTraceId);
  assert.equal(headers["X-Parent-Span-Id"], parentSpanId);
});

test("buildOutboundTraceHeaders preserves caller-provided ids", () => {
  const { requestTraceId, parentSpanId, headers } = buildOutboundTraceHeaders({
    "X-Trace-Id": "fixed-trace",
    "X-Parent-Span-Id": "aabbccddeeff0011",
  });
  assert.equal(requestTraceId, "fixed-trace");
  assert.equal(parentSpanId, "aabbccddeeff0011");
  assert.equal(headers["X-Trace-Id"], "fixed-trace");
  assert.equal(headers["X-Parent-Span-Id"], "aabbccddeeff0011");
});

test("setDataTraceId sets or removes data-traceId", () => {
  const attrs = {};
  const el = {
    setAttribute(k, v) {
      attrs[k] = v;
    },
    removeAttribute(k) {
      delete attrs[k];
    },
  };
  setDataTraceId(el, { traceId: "dom-1" });
  assert.equal(attrs["data-traceId"], "dom-1");
  setDataTraceId(el, "");
  assert.equal(attrs["data-traceId"], undefined);
});

test("attachClientTraceId stamps requestTraceId without overwriting", () => {
  const err = new TypeError("Failed to fetch");
  attachClientTraceId(err, " client-tid ");
  assert.equal(err.traceId, "client-tid");
  attachClientTraceId(err, "other");
  assert.equal(err.traceId, "client-tid");
  attachClientTraceId(err, "");
  assert.equal(err.traceId, "client-tid");
});

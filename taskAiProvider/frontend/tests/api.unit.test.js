import test from "node:test";
import assert from "node:assert/strict";
import { api } from "../src/api.js";

test("api attaches client X-Trace-Id when fetch throws Failed to fetch", async () => {
  const orig = globalThis.fetch;
  let sentTraceId = "";
  globalThis.fetch = async (_url, opts) => {
    sentTraceId = opts.headers["X-Trace-Id"];
    throw new TypeError("Failed to fetch");
  };
  try {
    await assert.rejects(
      () => api("/api/public/catalog/"),
      (err) => {
        assert.equal(err.message, "Failed to fetch");
        assert.ok(sentTraceId, "client must send X-Trace-Id before fetch");
        assert.equal(err.traceId, sentTraceId);
        return true;
      },
    );
  } finally {
    globalThis.fetch = orig;
  }
});

test("api HTTP error still prefers response X-Trace-Id", async () => {
  const orig = globalThis.fetch;
  globalThis.fetch = async () =>
    new Response(JSON.stringify({ detail: "boom" }), {
      status: 500,
      headers: { "X-Trace-Id": "resp-tid-1", "Content-Type": "application/json" },
    });
  try {
    await assert.rejects(
      () => api("/api/public/catalog/"),
      (err) => {
        assert.equal(err.message, "boom");
        assert.equal(err.traceId, "resp-tid-1");
        assert.equal(err.status, 500);
        return true;
      },
    );
  } finally {
    globalThis.fetch = orig;
  }
});

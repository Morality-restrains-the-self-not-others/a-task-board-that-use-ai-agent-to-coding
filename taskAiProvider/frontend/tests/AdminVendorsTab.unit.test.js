import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import path from "node:path";
import { describe, it } from "node:test";
import { fileURLToPath } from "node:url";

const here = path.dirname(fileURLToPath(import.meta.url));
const tabVue = readFileSync(
  path.join(here, "../src/components/AdminVendorsTab.vue"),
  "utf8",
);

describe("AdminVendorsTab 审核写路径 (OPT-20260828-007)", () => {
  it("不再使用原生 confirm/prompt/alert", () => {
    assert.doesNotMatch(tabVue, /\bconfirm\s*\(/);
    assert.doesNotMatch(tabVue, /\bprompt\s*\(/);
    assert.doesNotMatch(tabVue, /\balert\s*\(/);
  });

  it("通过/驳回复用原因弹窗并走 createClickGuard + Idempotency-Key", () => {
    assert.match(tabVue, /AdminImageReviewNoteModal/);
    assert.match(tabVue, /createClickGuard/);
    assert.match(tabVue, /Idempotency-Key|headers/);
    assert.match(tabVue, /aria-busy/);
  });

  it("刷新为只读、证照打开为 GET", () => {
    assert.match(tabVue, /Anti-Replay-OK:.*refresh|Anti-Replay-OK: read-refresh/);
    assert.match(tabVue, /Anti-Replay-OK:.*(?:GET|证照|blob)/);
  });

  it("openDoc 走 buildOutboundTraceHeaders 且失败时挂 traceId (OPT-20260828-019)", () => {
    assert.match(tabVue, /buildOutboundTraceHeaders\(\)/);
    assert.match(tabVue, /\.\.\.trace\.headers/);
    assert.match(tabVue, /attachClientTraceId\(e,\s*requestTraceId\)/);
  });
});

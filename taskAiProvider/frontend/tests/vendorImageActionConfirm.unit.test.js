import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import path from "node:path";
import { describe, it } from "node:test";
import { fileURLToPath } from "node:url";
import { createVendorImageActionConfirm } from "../src/composables/vendorImageActionConfirm.js";
import { IDEMPOTENCY_HEADER } from "../src/utils/clickGuard.js";

const here = path.dirname(fileURLToPath(import.meta.url));
const confirmJs = readFileSync(
  path.join(here, "../src/composables/vendorImageActionConfirm.js"),
  "utf8",
);
const groupsJs = readFileSync(
  path.join(here, "../src/composables/useVendorPortalImageGroups.js"),
  "utf8",
);
const tabVue = readFileSync(
  path.join(here, "../src/components/VendorImageGroupsTab.vue"),
  "utf8",
);

describe("提交审核防重放与 Idempotency-Key（OPT-20260829-022）", () => {
  it("双击只发一次 POST 且请求头携带 Idempotency-Key", async () => {
    const calls = [];
    const mockApi = async (apiPath, opts = {}) => {
      calls.push({ apiPath, opts });
      return {};
    };
    const c = createVendorImageActionConfirm({
      api: mockApi,
      setMsgError: () => {},
      refreshGroups: async () => {},
      refreshImages: async () => {},
    });

    const p1 = c.submitVersionReview({ id: "img1" });
    const p2 = c.submitVersionReview({ id: "img1" });
    await Promise.all([p1, p2]);

    assert.equal(calls.length, 1, "in-flight duplicate submit must be skipped");
    assert.equal(calls[0].apiPath, "/api/vendor/container-images/img1/submit/");
    assert.equal(calls[0].opts.method, "POST");
    assert.ok(
      calls[0].opts.headers[IDEMPOTENCY_HEADER],
      "submit POST must carry Idempotency-Key",
    );
    assert.equal(c.versionActionBusy.value, false, "busy must reset after submit");
  });

  it("composable 不再裸 POST submit，按钮 disabled + aria-busy", () => {
    // submit 逻辑不应再在 composable 里裸 POST（无 headers）
    assert.doesNotMatch(
      groupsJs,
      /await api\(`\/api\/vendor\/container-images\/\$\{im\.id\}\/submit\//,
    );
    assert.match(confirmJs, /submitVersionReview/);
    assert.match(confirmJs, /versionWriteGuard\.run/);
    assert.match(tabVue, /:disabled="vp\.versionActionBusy"/);
    assert.match(tabVue, /aria-busy="vp\.versionActionBusy/);
  });
});

import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import path from "node:path";
import { describe, it } from "node:test";
import { fileURLToPath } from "node:url";

const here = path.dirname(fileURLToPath(import.meta.url));
const tabVue = readFileSync(
  path.join(here, "../src/components/VendorImageGroupsTab.vue"),
  "utf8",
);

describe("VendorImageGroupsTab — 提交审核按钮防连点（OPT-20260829-022）", () => {
  it("提交审核按钮在 versionActionBusy 时 disabled 且 aria-busy", () => {
    const submitBlock = tabVue.slice(tabVue.indexOf("提交审核"));
    assert.match(submitBlock, /:disabled="vp\.versionActionBusy"/);
    assert.match(submitBlock, /aria-busy="vp\.versionActionBusy/);
  });
});

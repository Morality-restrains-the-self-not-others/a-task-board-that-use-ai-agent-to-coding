import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import path from "node:path";
import { describe, it } from "node:test";
import { fileURLToPath } from "node:url";

const here = path.dirname(fileURLToPath(import.meta.url));
const portalVue = readFileSync(
  path.join(here, "../src/views/VendorPortal.vue"),
  "utf8",
);

describe("vendorApplyCta", () => {
  it("T7 VendorPortal 不再挂载申请面板", () => {
    assert.doesNotMatch(portalVue, /VendorApplyPanel/);
    assert.match(portalVue, /镜像市场/);
  });
});

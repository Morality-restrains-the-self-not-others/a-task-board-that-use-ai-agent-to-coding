import { readFileSync } from "node:fs";
import path from "node:path";
import { describe, it } from "node:test";
import assert from "node:assert/strict";
import { fileURLToPath } from "node:url";

const here = path.dirname(fileURLToPath(import.meta.url));
const frontendSrc = path.join(here, "../src");

const SPLIT_FILES = [
  "views/VendorPortal.vue",
  "composables/useVendorPortal.js",
  "composables/useVendorPortalSession.js",
  "composables/useVendorPortalImageGroups.js",
  "composables/useVendorPortalCloudServers.js",
  "components/VendorImageGroupsTab.vue",
  "components/VendorCloudServerImagesTab.vue",
  "components/VendorGroupFormModal.vue",
];

function lineCount(abs) {
  return readFileSync(abs, "utf8").split("\n").length - 1;
}

describe("VendorPortal 行数门禁 (OPT-20260817-016)", () => {
  it("门户壳与拆出的 composable/tab 均 ≤500 行", () => {
    for (const rel of SPLIT_FILES) {
      const abs = path.join(frontendSrc, rel);
      const n = lineCount(abs);
      assert.ok(n <= 500, `${rel} 有 ${n} 行，超过 500`);
    }
  });

  it("VendorPortal.vue 不再保留 500 行例外注释", () => {
    const portal = readFileSync(path.join(frontendSrc, "views/VendorPortal.vue"), "utf8");
    assert.doesNotMatch(portal, /500-line rule exception/);
    assert.doesNotMatch(portal, /VendorApplyPanel/);
    assert.match(portal, /VendorImageGroupsTab/);
    assert.match(portal, /VendorCloudServerImagesTab/);
    assert.match(portal, /VendorContainerImageVersionModals/);
  });
});

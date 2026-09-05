import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import path from "node:path";
import { describe, it } from "node:test";
import { fileURLToPath } from "node:url";

const here = path.dirname(fileURLToPath(import.meta.url));
const groupsJs = readFileSync(
  path.join(here, "../src/composables/useVendorPortalImageGroups.js"),
  "utf8",
);

describe("useVendorPortalImageGroups — submitReview 委托受 guard 版本（OPT-20260829-022）", () => {
  it("composable 不再裸 POST submit，改为委托 submitVersionReview", () => {
    assert.doesNotMatch(
      groupsJs,
      /await api\(`\/api\/vendor\/container-images\/\$\{im\.id\}\/submit\//,
    );
    assert.match(groupsJs, /async function submitReview\(im\) \{/);
    assert.match(groupsJs, /await submitVersionReview\(im\)/);
  });
});

import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import path from "node:path";
import { describe, it } from "node:test";
import { fileURLToPath } from "node:url";
import { createClickGuard } from "../src/utils/clickGuard.js";
import {
  formatSkillVersionBadge,
  requireWritableSkillVersion,
  SKILL_VERSION_REQUIRED_MESSAGE,
  skillVersionOptionLabel,
  writableSkillVersions,
} from "../src/lib/saasInboundSkillVersion.js";
import { saasMachineContainerSkillMdHref } from "../src/lib/saasMachineContainerSkill.js";

const here = path.dirname(fileURLToPath(import.meta.url));
const skillView = readFileSync(
  path.join(here, "../src/views/SaasMachineContainerSkill.vue"),
  "utf8",
);
const portalVue = readFileSync(
  path.join(here, "../src/views/VendorPortal.vue"),
  "utf8",
);
const modalVue = readFileSync(
  path.join(here, "../src/components/VendorContainerImageVersionModals.vue"),
  "utf8",
);
const selectVue = readFileSync(
  path.join(here, "../src/components/SaasInboundSkillVersionSelect.vue"),
  "utf8",
);
const adminVue = readFileSync(
  path.join(here, "../src/components/AdminImageReviewTab.vue"),
  "utf8",
);
const adminTableVue = readFileSync(
  path.join(here, "../src/components/AdminImageReviewTable.vue"),
  "utf8",
);
const catalogVue = readFileSync(
  path.join(here, "../src/views/PublicCatalog.vue"),
  "utf8",
);

describe("saas inbound skill version helpers", () => {
  it("writableSkillVersions drops sunset entries", () => {
    const out = writableSkillVersions({
      current: "1",
      versions: [
        { version: "1", status: "current", summary: "now" },
        { version: "0", status: "sunset", summary: "old" },
        { version: "2", status: "deprecated", summary: "still" },
      ],
    });
    assert.deepEqual(
      out.map((v) => v.version),
      ["1", "2"],
    );
  });

  it("requireWritableSkillVersion rejects empty and strips v prefix", () => {
    assert.equal(requireWritableSkillVersion("v1"), "1");
    assert.throws(
      () => requireWritableSkillVersion(""),
      (err) => err.message === SKILL_VERSION_REQUIRED_MESSAGE,
    );
  });

  it("formatSkillVersionBadge and option label", () => {
    assert.equal(formatSkillVersionBadge("1"), "v1");
    assert.match(
      skillVersionOptionLabel({ version: "1", status: "current", summary: "cid in path" }),
      /v1（当前）/,
    );
  });

  it("md href appends version query", () => {
    assert.equal(saasMachineContainerSkillMdHref(""), "/saas-machine-container.md");
    assert.equal(
      saasMachineContainerSkillMdHref("1"),
      "/saas-machine-container.md?version=1",
    );
  });
});

describe("saas inbound skill version UI wiring", () => {
  it("skill page shows badge and loads md by selected version", () => {
    assert.match(skillView, /data-testid="saas-inbound-skill-version-badge"/);
    assert.match(skillView, /saasMachineContainerSkillMdHref\(/);
    assert.match(skillView, /\/api\/ai-provider\/saas-inbound-skill-versions\//);
  });

  it("vendor add/edit modal requires select and posts saas_inbound_skill_version", () => {
    assert.match(selectVue, /data-testid="saas-inbound-skill-version-select"/);
    assert.match(selectVue, /data-traceId/);
    assert.match(modalVue, /requireWritableSkillVersion/);
    assert.match(modalVue, /saas_inbound_skill_version: skill/);
    assert.match(portalVue, /VendorContainerImageVersionModals/);
    assert.doesNotMatch(portalVue, /uploadForm/);
    assert.doesNotMatch(selectVue, /skill version catalog unavailable/);
  });

  it("admin review table shows 接口版本", () => {
    assert.match(adminTableVue, /接口版本/);
    assert.match(adminTableVue, /saas_inbound_skill_version/);
    assert.match(adminVue, /AdminImageReviewTable/);
  });

  it("public catalog table shows 接口版本", () => {
    assert.match(catalogVue, /data-testid="public-catalog-saas-inbound-skill-version"/);
    assert.match(catalogVue, /saas_inbound_skill_version/);
  });
});

describe("clickGuard on image save", () => {
  it("skips in-flight second click (FE-5)", async () => {
    const guard = createClickGuard({ debounceMs: 0, newKey: () => "k1" });
    let started = 0;
    let resolveFirst;
    const first = guard.run(
      () =>
        new Promise((resolve) => {
          started += 1;
          resolveFirst = resolve;
        }),
    );
    const second = await guard.run(() => {
      started += 1;
    });
    assert.equal(second.skipped, true);
    assert.equal(second.reason, "in-flight");
    resolveFirst();
    const done = await first;
    assert.equal(done.skipped, false);
    assert.equal(started, 1);
    assert.match(modalVue, /createClickGuard/);
    assert.match(modalVue, /Idempotency-Key|headers/);
  });
});

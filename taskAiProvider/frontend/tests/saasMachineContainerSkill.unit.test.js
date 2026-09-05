import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import path from "node:path";
import { describe, it } from "node:test";
import { fileURLToPath } from "node:url";
import {
  SAAS_MACHINE_CONTAINER_SKILL_HREF,
  SAAS_MACHINE_CONTAINER_SKILL_LABEL,
  SAAS_MACHINE_CONTAINER_SKILL_MD_HREF,
} from "../src/lib/saasMachineContainerSkill.js";

const here = path.dirname(fileURLToPath(import.meta.url));
const appVue = readFileSync(path.join(here, "../src/App.vue"), "utf8");
const routerJs = readFileSync(path.join(here, "../src/router/index.js"), "utf8");
const viewVue = readFileSync(
  path.join(here, "../src/views/SaasMachineContainerSkill.vue"),
  "utf8",
);

describe("saasMachineContainerSkill nav & in-app page", () => {
  it("exports the in-app route href plus the raw md href", () => {
    assert.equal(SAAS_MACHINE_CONTAINER_SKILL_HREF, "/saas-machine-container");
    assert.equal(SAAS_MACHINE_CONTAINER_SKILL_LABEL, "容器→SaaS 接口");
    assert.equal(SAAS_MACHINE_CONTAINER_SKILL_MD_HREF, "/saas-machine-container.md");
  });

  it("App.vue nav uses router-link to the in-app page keeping testid", () => {
    assert.match(appVue, /name: 'saas-machine-container-skill'/);
    assert.match(appVue, /data-testid="nav-saas-machine-container-skill"/);
    assert.doesNotMatch(appVue, /@click\.prevent/);
  });

  it("router registers the static page before the root catch-all", () => {
    const adminIdx = routerJs.indexOf('path: "/admin"');
    const skillIdx = routerJs.indexOf('path: "/saas-machine-container"');
    const rootIdx = routerJs.indexOf('path: "/", name: "vendor"');
    assert.ok(skillIdx >= 0 && skillIdx < rootIdx, "skill route must precede root");
    assert.ok(adminIdx > skillIdx, "static sub-paths should stay ordered before root");
  });

  it("view fetches the selected version of the SSOT md and renders markdown body", () => {
    assert.match(viewVue, /saasMachineContainerSkillMdHref\(/);
    assert.match(viewVue, /renderMarkdown\(/);
    assert.match(viewVue, /data-testid="saas-machine-container-skill-body"/);
    assert.match(viewVue, /data-testid="saas-inbound-skill-version-badge"/);
  });
});

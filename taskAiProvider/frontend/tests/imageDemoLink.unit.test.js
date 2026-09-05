import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import path from "node:path";
import { describe, it } from "node:test";
import { fileURLToPath } from "node:url";
import { IMAGE_DEMO_HREF, IMAGE_DEMO_LABEL } from "../src/lib/imageDemoLink.js";

const here = path.dirname(fileURLToPath(import.meta.url));
const appVue = readFileSync(path.join(here, "../src/App.vue"), "utf8");

describe("imageDemo GitHub nav link", () => {
  it("exports the public GitHub demo href and label", () => {
    assert.equal(IMAGE_DEMO_HREF, "https://github.com/task2money/trae-agent");
    assert.equal(IMAGE_DEMO_LABEL, "镜像Demo");
  });

  it("App.vue nav uses a real external anchor with testid", () => {
    assert.match(appVue, /:href="IMAGE_DEMO_HREF"/);
    assert.match(appVue, /data-testid="nav-image-demo"/);
    assert.match(appVue, /target="_blank"/);
    assert.match(appVue, /rel="noopener noreferrer"/);
    assert.doesNotMatch(appVue, /@click\.prevent/);
  });
});

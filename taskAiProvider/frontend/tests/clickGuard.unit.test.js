import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import path from "node:path";
import { describe, it } from "node:test";
import { fileURLToPath } from "node:url";

const here = path.dirname(fileURLToPath(import.meta.url));
const src = readFileSync(path.join(here, "../src/utils/clickGuard.js"), "utf8");

describe("createClickGuard busy is a Vue ref", () => {
  it("uses vue ref so template :disabled=isBusy() re-renders after await", () => {
    assert.match(src, /import \{ ref \} from ["']vue["']/);
    assert.match(src, /const busy = ref\(false\)/);
    assert.match(src, /isBusy:\s*\(\)\s*=>\s*busy\.value/);
    assert.match(src, /busy\.value = true/);
    assert.match(src, /busy\.value = false/);
  });
});

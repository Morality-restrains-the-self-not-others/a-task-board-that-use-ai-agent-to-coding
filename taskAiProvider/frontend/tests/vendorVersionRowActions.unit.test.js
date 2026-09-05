import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import path from "node:path";
import { describe, it } from "node:test";
import { fileURLToPath } from "node:url";
import {
  versionHasAction,
  versionRowActions,
  withdrawButtonLabel,
} from "../src/lib/vendorVersionRowActions.js";

const here = path.dirname(fileURLToPath(import.meta.url));
const tabVue = readFileSync(
  path.join(here, "../src/components/VendorImageGroupsTab.vue"),
  "utf8",
);
const css = readFileSync(
  path.join(here, "../src/views/VendorPortal.css"),
  "utf8",
);
const groupsJs = readFileSync(
  path.join(here, "../src/composables/useVendorPortalImageGroups.js"),
  "utf8",
);
const confirmJs = readFileSync(
  path.join(here, "../src/composables/vendorImageActionConfirm.js"),
  "utf8",
);
const portalVue = readFileSync(
  path.join(here, "../src/views/VendorPortal.vue"),
  "utf8",
);

describe("versionRowActions — 已上架行须有管理手段", () => {
  it("已上架非激活：区域明细 + 设为激活 + 下架 + 删除", () => {
    const im = { status: "approved", is_active: false };
    assert.deepEqual(versionRowActions(im), [
      "runtimeEnv",
      "activate",
      "delete",
      "withdraw",
    ]);
    assert.equal(versionHasAction(im, "delete"), true);
    assert.equal(withdrawButtonLabel("approved"), "下架");
  });

  it("已上架激活：可下架，不可删除", () => {
    const im = { status: "approved", is_active: true };
    assert.equal(versionHasAction(im, "withdraw"), true);
    assert.equal(versionHasAction(im, "delete"), false);
    assert.equal(versionHasAction(im, "activate"), false);
  });

  it("草稿：编辑/删除/提交审核", () => {
    const im = { status: "draft" };
    assert.equal(versionHasAction(im, "edit"), true);
    assert.equal(versionHasAction(im, "delete"), true);
    assert.equal(versionHasAction(im, "submit"), true);
    assert.equal(versionHasAction(im, "withdraw"), false);
  });

  it("待审核：撤回审核，不可删除", () => {
    const im = { status: "pending_review" };
    assert.equal(versionHasAction(im, "withdraw"), true);
    assert.equal(versionHasAction(im, "delete"), false);
    assert.equal(withdrawButtonLabel("pending_review"), "撤回审核");
  });
});

describe("version-row 布局 — URL 不得挤掉操作列", () => {
  it("version-row 五列且 version-url 独占下一行", () => {
    assert.match(
      css,
      /\.version-row\s*\{[^}]*grid-template-columns:\s*9\.5rem\s+minmax\(/s,
    );
    assert.match(css, /\.version-url\s*\{[^}]*grid-column:\s*1\s*\/\s*-1/s);
  });

  it("版本行模板用 versionHasAction 且操作列在 URL 之前", () => {
    const actionsIdx = tabVue.indexOf('data-testid="version-row-actions"');
    const urlIdx = tabVue.indexOf("version-url");
    assert.ok(actionsIdx > 0, "missing version-row-actions testid");
    assert.ok(urlIdx > actionsIdx, "version-url must come after actions in DOM");
    assert.match(tabVue, /versionHasAction\(im,\s*['"]delete['"]\)/);
    assert.match(tabVue, /versionHasAction\(im,\s*['"]withdraw['"]\)/);
  });
});

describe("写操作防重放与确认弹窗", () => {
  it("composable 对删除/下架走 createClickGuard 且不使用 window.confirm", () => {
    assert.match(confirmJs, /createClickGuard/);
    assert.match(confirmJs, /headers/);
    assert.doesNotMatch(groupsJs, /confirm\(/);
    assert.doesNotMatch(confirmJs, /confirm\(/);
  });

  it("门户挂载版本操作确认弹窗", () => {
    assert.match(portalVue, /VendorVersionActionConfirmModal/);
  });
});

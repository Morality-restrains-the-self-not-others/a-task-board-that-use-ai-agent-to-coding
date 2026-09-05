import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import path from "node:path";
import { describe, it } from "node:test";
import { fileURLToPath } from "node:url";

const here = path.dirname(fileURLToPath(import.meta.url));
const modalVue = readFileSync(
  path.join(here, "../src/components/VendorContainerImageVersionModals.vue"),
  "utf8",
);
const portalCss = readFileSync(
  path.join(here, "../src/views/VendorPortal.css"),
  "utf8",
);

describe("VendorContainerImageVersionModals 底部按钮行宽度", () => {
  it("两个弹窗（添加/编辑镜像版本）均保留 取消 + 主按钮", () => {
    assert.match(modalVue, />取消<\/button>/);
    assert.match(modalVue, /保存为草稿/);
    assert.match(modalVue, /"保存"/);
  });

  it("按钮等宽分摊整行的 scoped 样式仅在本组件声明", () => {
    assert.match(
      modalVue,
      /\.row \.btn\s*\{\s*flex:\s*1 1 0;\s*\}/,
      "modal 内应有 .row .btn { flex: 1 1 0; }",
    );
    assert.match(modalVue, /style scoped/, "样式为 scoped，不泄漏到其他组件");
  });

  it("全局 VendorPortal.css 的 .row 未被修改（其他弹窗不受影响）", () => {
    const rowBlock = portalCss.match(/^\.row \{[\s\S]*?^\}/m);
    assert.ok(rowBlock, ".row 全局定义应存在");
    assert.doesNotMatch(rowBlock[0], /flex: 1/, "全局 .row 不应含按钮拉伸规则");
  });
});

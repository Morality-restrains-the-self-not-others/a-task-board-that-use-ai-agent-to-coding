import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import path from "node:path";
import { describe, it } from "node:test";
import { fileURLToPath } from "node:url";
import {
  formatArchList,
  hasAction,
  rejectHistories,
  resolvePanelProps,
  reviewHistoryActionLabel,
  reviewRowActions,
} from "../src/lib/adminImageReviewActions.js";

const here = path.dirname(fileURLToPath(import.meta.url));
const tabVue = readFileSync(
  path.join(here, "../src/components/AdminImageReviewTab.vue"),
  "utf8",
);
const tableVue = readFileSync(
  path.join(here, "../src/components/AdminImageReviewTable.vue"),
  "utf8",
);
const detailVue = readFileSync(
  path.join(here, "../src/components/AdminImageReviewDetailModal.vue"),
  "utf8",
);

describe("reviewRowActions — 操作列按状态提供完整选项", () => {
  it("待审核：查看详情 + 通过 + 驳回", () => {
    assert.deepEqual(reviewRowActions("pending_review"), [
      "detail",
      "approve",
      "reject",
    ]);
    assert.equal(hasAction("pending_review", "detail"), true);
    assert.equal(hasAction("pending_review", "unpublish"), false);
  });

  it("已上架：查看详情 + 撤销上架", () => {
    assert.deepEqual(reviewRowActions("approved"), ["detail", "unpublish"]);
    assert.equal(hasAction("approved", "approve"), false);
  });

  it("已驳回/草稿/未知：至少查看详情，操作列不为空", () => {
    assert.deepEqual(reviewRowActions("rejected"), ["detail"]);
    assert.deepEqual(reviewRowActions("draft"), ["detail"]);
    assert.deepEqual(reviewRowActions(""), ["detail"]);
  });
});

describe("rejectHistories — 兼容 reject / rejected", () => {
  it("filters backend action=reject (not only rejected)", () => {
    const rows = rejectHistories([
      { id: "1", action: "approve", note: "ok" },
      { id: "2", action: "reject", note: "描述不完整" },
      { id: "3", action: "rejected", note: "legacy" },
    ]);
    assert.equal(rows.length, 2);
    assert.equal(rows[0].note, "描述不完整");
    assert.equal(rows[1].note, "legacy");
  });

  it("empty or missing histories → []", () => {
    assert.deepEqual(rejectHistories(null), []);
    assert.deepEqual(rejectHistories(undefined), []);
  });

  it("labels review actions in zh", () => {
    assert.equal(reviewHistoryActionLabel("reject"), "驳回");
    assert.equal(reviewHistoryActionLabel("approve"), "通过");
    assert.equal(reviewHistoryActionLabel("unpublish"), "撤销上架");
  });
});

describe("resolvePanelProps / formatArchList", () => {
  it("maps persisted skills and auto-run onto the resolve panel", () => {
    const p = resolvePanelProps({
      image_skills: [{ name: "code" }],
      image_skills_extract_status: "ok",
      auto_run_steps_md: "# boot",
      auto_run_steps_extract_status: "ok",
    });
    assert.equal(p.skills[0].name, "code");
    assert.equal(p.skillsStatus, "ok");
    assert.equal(p.autoRunStepsMd, "# boot");
  });

  it("treats populated skills/md without extract status as ok", () => {
    const p = resolvePanelProps({
      image_skills: [{ name: "x" }],
      auto_run_steps_md: "step",
    });
    assert.equal(p.skillsStatus, "ok");
    assert.equal(p.autoRunStepsStatus, "ok");
  });

  it("joins architecture list", () => {
    assert.equal(formatArchList(["amd64", "arm64"]), "amd64, arm64");
    assert.equal(formatArchList([]), "—");
  });
});

describe("admin review UI wiring", () => {
  it("操作列始终提供查看详情，并按状态显示通过/驳回/撤销上架", () => {
    assert.match(tableVue, /查看详情/);
    assert.match(tableVue, /data-testid="admin-review-detail"/);
    assert.match(tableVue, /reviewRowActions|hasAction/);
    assert.match(tableVue, />通过</);
    assert.match(tableVue, />驳回</);
    assert.match(tableVue, /撤销上架/);
  });

  it("历史驳回原因使用 rejectHistories 而非仅 action===rejected", () => {
    assert.match(tableVue, /rejectHistories/);
    assert.doesNotMatch(tableVue, /action === ['"]rejected['"]/);
  });

  it("写操作走 createClickGuard，详情为 ui-only", () => {
    assert.match(tabVue, /createClickGuard/);
    assert.match(tableVue, /Anti-Replay-OK: ui-only/);
  });

  it("详情弹窗展示镜像地址、技能/自动运行与运行环境", () => {
    assert.match(detailVue, /image_url/);
    assert.match(detailVue, /ImageResolveInfoPanel/);
    assert.match(detailVue, /runtime_environments/);
    assert.match(detailVue, /data-testid="admin-review-detail-modal"/);
  });

  it("tab 挂载 table + detail modal，去掉重复双表", () => {
    assert.match(tabVue, /AdminImageReviewTable/);
    assert.match(tabVue, /AdminImageReviewDetailModal/);
  });

  it("openDetail prefers GET /api/admin/container-images/{id}/ then falls back to list row", () => {
    assert.match(tabVue, /async function openDetail/);
    assert.match(tabVue, /\/api\/admin\/container-images\/\$\{im\.id\}\//);
    assert.match(tabVue, /detailImage\.value = fresh/);
  });

  it("组内激活 hint 可见文案含版本且说明审批不切换激活", () => {
    assert.match(tableVue, /group-active-hint/);
    assert.match(
      tableVue,
      /组内激活：\{\{ im\.group_active_version\.version \}\}（审批不切换激活）/,
    );
    assert.match(tableVue, /激活由厂商在版本列表切换/);
    assert.doesNotMatch(tableVue, /审批通过不影响激活，激活状态由厂商/);
  });
});

describe("admin review 镜像组图标（OPT-20260828-014）", () => {
  it("表格通过 imageGroupIconSrc 渲染 36px 图标，缺省走占位块", () => {
    assert.match(tableVue, /imageGroupIconSrc/);
    assert.match(tableVue, /import\s*\{[\s\S]*imageGroupIconSrc[\s\S]*\}\s*from\s*"\.\.\/utils\/imageGroupIcon\.js"/);
    assert.match(tableVue, /<img[\s\S]*v-if="imageGroupIconSrc\(im\)"/);
    assert.match(tableVue, /:src="imageGroupIconSrc\(im\)"/);
    assert.match(tableVue, /width="36"\s*height="36"/);
    assert.match(tableVue, /group-icon-sm group-icon-placeholder/);
    assert.match(tableVue, /\{\{ im\.name \}\}/);
  });
});

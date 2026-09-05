import test from "node:test";
import assert from "node:assert/strict";
import {
  containerImageGroupId,
  enrichImageGroups,
  formatUnavailableBadge,
  getGroupVersions,
  isMarketplaceUnavailable,
  groupActiveVersion,
  isActiveVersion,
  statusDisplay,
} from "../src/utils/containerImageGroup.js";

test("containerImageGroupId 读取列表 API 的 image_group 字符串 FK", () => {
  assert.equal(
    containerImageGroupId({ id: "1", image_group: "859670587958513664" }),
    "859670587958513664",
  );
});

test("containerImageGroupId 读取嵌套 image_group.id", () => {
  assert.equal(
    containerImageGroupId({ id: "1", image_group: { id: "42", name: "g" } }),
    "42",
  );
});

test("getGroupVersions 按 image_group 过滤（回归：误用 image_group_id 会导致列表恒空）", () => {
  const groupId = "859670587958513664";
  const images = [
    {
      id: "7308131032227841",
      image_group: groupId,
      version: "",
      status: "draft",
    },
    {
      id: "859670982529273856",
      image_group: groupId,
      version: "1344",
      status: "draft",
    },
    {
      id: "other",
      image_group: "999",
      version: "1.0.0",
      status: "draft",
    },
  ];
  // 模拟旧 bug：只认 image_group_id
  const broken = images.filter((im) => String(im.image_group_id) === String(groupId));
  assert.equal(broken.length, 0);

  const versions = getGroupVersions(images, groupId);
  assert.equal(versions.length, 2);
  assert.deepEqual(
    versions.map((x) => x.id),
    ["7308131032227841", "859670982529273856"],
  );
});

test("enrichImageGroups 补全 versions_count 与 latest_version", () => {
  const groups = [{ id: "g1", name: "trae0630", description: "" }];
  const images = [
    { id: "a", image_group: "g1", version: "2.0", status: "draft" },
    { id: "b", image_group: "g1", version: "1.0", status: "pending_review" },
  ];
  const enriched = enrichImageGroups(groups, images);
  assert.equal(enriched[0].versions_count, 2);
  assert.equal(enriched[0].latest_version.version, "2.0");
  assert.equal(enriched[0].latest_version.status_display, "草稿");
});

test("statusDisplay 覆盖领域状态文案", () => {
  assert.equal(statusDisplay("approved"), "已上架");
  assert.equal(statusDisplay("pending_review"), "待审核");
});

test("formatUnavailableBadge 有原因时内联展示", () => {
  assert.equal(
    formatUnavailableBadge("未设置任何区域运行环境，无法在镜像市场展示"),
    "不可用：未设置任何区域运行环境，无法在镜像市场展示",
  );
});

test("formatUnavailableBadge 无原因时仅显示不可用", () => {
  assert.equal(formatUnavailableBadge(""), "不可用");
  assert.equal(formatUnavailableBadge(null), "不可用");
});

test("isMarketplaceUnavailable 仅对明确 false 生效", () => {
  assert.equal(isMarketplaceUnavailable({ is_ai_provider_available: false }), true);
  assert.equal(isMarketplaceUnavailable({ is_ai_provider_available: true }), false);
  assert.equal(isMarketplaceUnavailable({}), false);
  assert.equal(isMarketplaceUnavailable(undefined), false);
});

test("groupActiveVersion 返回组内激活版本（非自身）", () => {
  assert.deepEqual(
    groupActiveVersion({
      id: "100",
      image_group: "20",
      status: "pending_review",
      group_active_version: { id: "99", version: "x86_64-latest" },
    }),
    { id: "99", version: "x86_64-latest" },
  );
});

test("groupActiveVersion 无激活版本或自身即激活时返回 null", () => {
  assert.equal(groupActiveVersion({ id: "1", group_active_version: undefined }), null);
  assert.equal(groupActiveVersion({ id: "1" }), null);
  assert.equal(groupActiveVersion(null), null);
  assert.equal(groupActiveVersion({ id: "99", group_active_version: {} }), null);
  assert.equal(groupActiveVersion({ id: "99", group_active_version: { version: "x" } }), null);
});

test("isActiveVersion 识别后端 is_active=true 的激活版本", () => {
  assert.equal(isActiveVersion({ id: "1", is_active: true }), true);
  assert.equal(isActiveVersion({ id: "2", is_active: false }), false);
  assert.equal(isActiveVersion({ id: "3" }), false);
  assert.equal(isActiveVersion(null), false);
  assert.equal(isActiveVersion(undefined), false);
  assert.equal(isActiveVersion({ is_active: 1 }), false);
});

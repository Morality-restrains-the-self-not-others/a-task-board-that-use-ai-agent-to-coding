/** 列表 API 返回 FK 为 `image_group`（字符串 ID）；公开目录嵌套为 `{ id, name }`。 */
export function containerImageGroupId(image) {
  if (!image || typeof image !== "object") {
    throw new Error("container image is required");
  }
  const g = image.image_group;
  if (g != null && typeof g === "object") {
    if (g.id == null || g.id === "") {
      throw new Error(`container image ${image.id} has image_group without id`);
    }
    return String(g.id);
  }
  if (g != null && g !== "") {
    return String(g);
  }
  if (image.image_group_id != null && image.image_group_id !== "") {
    return String(image.image_group_id);
  }
  throw new Error(`container image ${image.id} missing image_group`);
}

export function getGroupVersions(images, groupId) {
  const gid = String(groupId);
  return (Array.isArray(images) ? images : []).filter(
    (im) => containerImageGroupId(im) === gid,
  );
}

const STATUS_DISPLAY = {
  draft: "草稿",
  pending_review: "待审核",
  approved: "已上架",
  rejected: "已驳回",
};

export function statusDisplay(status) {
  return STATUS_DISPLAY[status] || status || "";
}

/** 仅当 API 明确返回 false 时视为市场不可用（字段缺失不得误判）。
 *  后端返回 is_ai_provider_available，表示该区域运行环境是否配置了 UserData 模板。 */
export function isMarketplaceUnavailable(image) {
  return Boolean(image && image.is_ai_provider_available === false);
}

/** 同一镜像组允许多个 approved 但仅一个激活版本：返回组内当前激活版本（非自身）
 *  {id, version}，无则 null。后端列表 API 通过 group_active_version 标注。 */
export function groupActiveVersion(image) {
  const a = image && image.group_active_version;
  return a && typeof a === "object" && a.id ? a : null;
}

/** 是否为组内当前「激活」版本（公开目录生效）。后端列表 API 通过 is_active 标注，
 *  仅 approved 版本可激活。 */
export function isActiveVersion(image) {
  return Boolean(image && image.is_active === true);
}

/** 不可用徽章可见文案：有原因时内联展示，避免仅靠 title。 */
export function formatUnavailableBadge(reason) {
  const text = typeof reason === "string" ? reason.trim() : "";
  return text ? `不可用：${text}` : "不可用";
}

/** 用容器镜像列表补全镜像组的 versions_count / latest_version（组列表 API 本身不带）。 */
export function enrichImageGroups(groups, images) {
  return (Array.isArray(groups) ? groups : []).map((g) => {
    const versions = getGroupVersions(images, g.id);
    const latest = versions[0] || null;
    return {
      ...g,
      versions_count: versions.length,
      latest_version: latest
        ? {
            version: latest.version,
            status: latest.status,
            status_display: latest.status_display || statusDisplay(latest.status),
          }
        : null,
    };
  });
}

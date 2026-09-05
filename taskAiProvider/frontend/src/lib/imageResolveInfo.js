/**
 * 「添加镜像版本」弹窗解析结果归一化：resolve-target-architectures 响应中
 * 技能列表（imageSkills.yaml）与自动运行说明（autoRunStep.md）字段的读取/展示。
 */

/** 解析去重键：同一 image_url@version 已在弹窗内成功解析时不再重复向仓库请求。 */
export function resolveRefKey(imageUrl, version) {
  return `${String(imageUrl ?? "").trim()}@${String(version ?? "").trim()}`;
}

/** 归一化 resolve-target-architectures 响应中技能与自动运行说明字段。 */
export function parseResolvePayload(payload) {
  const p = payload ?? {};
  return {
    skills: Array.isArray(p.skills?.skills) ? p.skills.skills : [],
    skillsStatus: String(p.skills_status ?? ""),
    skillsDetail: String(p.skills_detail ?? ""),
    autoRunStepsMd: String(p.auto_run_steps_md ?? ""),
    autoRunStepsStatus: String(p.auto_run_steps_status ?? ""),
    autoRunStepsDetail: String(p.auto_run_steps_detail ?? ""),
  };
}

/**
 * 已保存镜像记录是否已含可复用的持久化技能/自动运行说明（提取状态为 ok）。
 * 列表接口已返回 image_skills / auto_run_steps_md 与各自 extract_status；
 * 编辑弹窗打开时若均已提取成功，直接展示即可，无需再按地址+版本向仓库全层扫描。
 */
export function hasPersistedResolveInfo(im) {
  return (
    im?.image_skills_extract_status === "ok" &&
    im?.auto_run_steps_extract_status === "ok"
  );
}

/** 从已保存镜像记录构建弹窗解析结果；未完整提取时返回 null（调用方退回解析端点）。 */
export function persistedResolveInfo(im) {
  if (!hasPersistedResolveInfo(im)) return null;
  return parseResolvePayload({
    skills: { skills: Array.isArray(im.image_skills) ? im.image_skills : [] },
    skills_status: "ok",
    auto_run_steps_md: im.auto_run_steps_md || "",
    auto_run_steps_status: "ok",
  });
}

/** 技能列表非 ok 状态的提示文案（ok 返回空串，不展示）。 */
export function skillsStatusHint(status) {
  switch (status) {
    case "not_found":
      return "镜像中未发现 imageSkills.yaml";
    case "auth_failed":
      return "镜像仓库认证失败，未能提取技能列表";
    case "failed":
      return "技能列表提取失败";
    default:
      return "";
  }
}

/** 自动运行说明非 ok 状态的提示文案（ok 返回空串，不展示）。 */
export function autoRunStatusHint(status) {
  switch (status) {
    case "not_found":
      return "镜像中未发现 autoRunStep.md";
    case "auth_failed":
      return "镜像仓库认证失败，未能提取自动运行说明";
    case "failed":
      return "自动运行说明提取失败";
    default:
      return "";
  }
}

/** 取 markdown 第一行有意义的文本作为摘要（跳过标题/空行，去列表/引用标记）。 */
function markdownFirstMeaningfulLine(md) {
  const stripMarkers = (l) => l.replace(/^[>*\-]+\s*|\d+[.)]\s*/, "").trim();
  const line =
    String(md ?? "")
      .split(/\r?\n/)
      .map((l) => l.trim())
      .find((l) => l && !/^#/.test(l) && stripMarkers(l).length > 0) ?? "";
  return stripMarkers(line);
}

/**
 * OPT-20260824-063：镜像列表行「技能/运行说明」摘要。
 * 厂商无需重开弹窗即可核对已保存版本的技能与自动运行说明。
 * 返回：skillNames（技能名列表）、skillsHint（未 ok 时提示）、
 * autoRunPreview（自动运行说明首行摘要）、autoRunHint（未 ok 时提示）。
 */
export function summarizeImageResolveInfo(im) {
  const record = im ?? {};
  const skillsStatus = String(record.image_skills_extract_status ?? "");
  const skills = Array.isArray(record.image_skills) ? record.image_skills : [];
  const autoRunStepsStatus = String(record.auto_run_steps_extract_status ?? "");
  const autoRunStepsMd = String(record.auto_run_steps_md ?? "");

  const skillNames =
    skillsStatus === "ok" ? skills.map((s) => s.name).filter(Boolean) : [];
  const skillsHint =
    skillNames.length > 0
      ? ""
      : skillsStatus === "ok"
        ? "镜像未声明技能"
        : skillsStatusHint(skillsStatus) || "技能未提取";

  const autoRunPreview =
    autoRunStepsStatus === "ok" ? markdownFirstMeaningfulLine(autoRunStepsMd) : "";
  const autoRunHint =
    autoRunPreview
      ? ""
      : autoRunStepsStatus === "ok"
        ? "镜像未声明自动运行说明"
        : autoRunStatusHint(autoRunStepsStatus) || "运行说明未提取";

  return { skillNames, skillsHint, autoRunPreview, autoRunHint };
}

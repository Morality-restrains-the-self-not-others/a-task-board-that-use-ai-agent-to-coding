/**
 * 平台审核「容器镜像」操作列与审核历史的纯函数。
 * 待审核不能只给通过/驳回：审核员必须先能查看镜像地址、技能、自动运行与运行环境。
 */

export const ACTION_DETAIL = "detail";
export const ACTION_APPROVE = "approve";
export const ACTION_REJECT = "reject";
export const ACTION_UNPUBLISH = "unpublish";

export function reviewRowActions(status) {
  const actions = [ACTION_DETAIL];
  if (status === "pending_review") {
    actions.push(ACTION_APPROVE, ACTION_REJECT);
  } else if (status === "approved") {
    actions.push(ACTION_UNPUBLISH);
  }
  return actions;
}

export function hasAction(status, action) {
  return reviewRowActions(status).includes(action);
}

export function isRejectHistoryAction(action) {
  return action === "reject" || action === "rejected";
}

export function rejectHistories(histories) {
  if (!Array.isArray(histories)) {
    return [];
  }
  return histories.filter((h) => isRejectHistoryAction(h?.action));
}

export function reviewHistoryActionLabel(action) {
  switch (action) {
    case "reject":
    case "rejected":
      return "驳回";
    case "approve":
    case "approved":
      return "通过";
    case "unpublish":
      return "撤销上架";
    default:
      return action || "—";
  }
}

export function formatArchList(arch) {
  if (Array.isArray(arch) && arch.length > 0) {
    return arch.join(", ");
  }
  if (typeof arch === "string" && arch.trim()) {
    return arch.trim();
  }
  return "—";
}

export function resolvePanelProps(im) {
  const skills = Array.isArray(im?.image_skills) ? im.image_skills : [];
  const autoRunStepsMd = String(im?.auto_run_steps_md || "");
  let skillsStatus = String(im?.image_skills_extract_status || "");
  let autoRunStepsStatus = String(im?.auto_run_steps_extract_status || "");
  if (!skillsStatus && skills.length > 0) {
    skillsStatus = "ok";
  }
  if (!autoRunStepsStatus && autoRunStepsMd) {
    autoRunStepsStatus = "ok";
  }
  return {
    skills,
    skillsStatus,
    skillsDetail: String(im?.image_skills_digest || ""),
    autoRunStepsMd,
    autoRunStepsStatus,
    autoRunStepsDetail: String(im?.auto_run_steps_digest || ""),
  };
}

export function formatReviewDate(dateStr) {
  if (!dateStr) {
    return "";
  }
  const d = new Date(dateStr);
  if (Number.isNaN(d.getTime())) {
    return "";
  }
  return d.toLocaleString("zh-CN", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  });
}

export function userdataTemplateLabel(rt) {
  const tpl = rt?.userdata_template;
  if (!tpl || typeof tpl !== "object") {
    return "未配置模板";
  }
  const name = String(tpl.name || "").trim();
  if (!name) {
    return "未配置模板";
  }
  const ver = tpl.version != null && String(tpl.version).trim() !== "" ? ` v${tpl.version}` : "";
  return `${name}${ver}`;
}

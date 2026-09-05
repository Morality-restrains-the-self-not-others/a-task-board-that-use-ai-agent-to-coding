/**
 * 厂商门户镜像版本行操作矩阵。
 * 已上架行必须能下架；非激活已上架可删除；激活版本须先下架。
 */

export const ACTION_RUNTIME_ENV = "runtimeEnv";
export const ACTION_EDIT = "edit";
export const ACTION_DELETE = "delete";
export const ACTION_SUBMIT = "submit";
export const ACTION_REJECT_NOTE = "rejectNote";
export const ACTION_ACTIVATE = "activate";
export const ACTION_WITHDRAW = "withdraw";

export function versionRowActions(im) {
  const status = im && im.status;
  const isActive = Boolean(im && im.is_active);
  const actions = [ACTION_RUNTIME_ENV];
  if (status === "draft" || status === "rejected") {
    actions.push(ACTION_EDIT, ACTION_DELETE, ACTION_SUBMIT);
    if (im && im.review_note) {
      actions.push(ACTION_REJECT_NOTE);
    }
  }
  if (status === "approved" && !isActive) {
    actions.push(ACTION_ACTIVATE, ACTION_DELETE);
  }
  if (status === "approved" || status === "pending_review") {
    actions.push(ACTION_WITHDRAW);
  }
  return actions;
}

export function versionHasAction(im, action) {
  return versionRowActions(im).includes(action);
}

export function withdrawButtonLabel(status) {
  return status === "approved" ? "下架" : "撤回审核";
}

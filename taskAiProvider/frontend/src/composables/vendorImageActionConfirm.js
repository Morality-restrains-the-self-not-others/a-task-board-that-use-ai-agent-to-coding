import { ref } from "vue";
import { createClickGuard } from "../utils/clickGuard.js";
import { api as defaultApi } from "../api.js";
import { groupActiveVersion } from "../utils/containerImageGroup.js";

export function createVendorImageActionConfirm({
  setMsgError,
  refreshGroups,
  refreshImages,
  api = defaultApi,
}) {
  const versionWriteGuard = createClickGuard();
  const versionActionBusy = ref(false);
  const actionConfirm = ref(null);

  function openVersionDelete(im) {
    actionConfirm.value = { kind: "delete", image: im };
  }
  function openVersionWithdraw(im) {
    actionConfirm.value = { kind: "withdraw", image: im };
  }
  function openVersionActivate(im) {
    actionConfirm.value = { kind: "activate", image: im };
  }
  function openGroupDelete(g) {
    actionConfirm.value = { kind: "deleteGroup", group: g };
  }

  function actionConfirmTitle() {
    const spec = actionConfirm.value;
    if (!spec) return "";
    if (spec.kind === "deleteGroup") return "删除镜像组";
    if (spec.kind === "delete") return "删除版本";
    if (spec.kind === "withdraw") {
      return spec.image?.status === "approved" ? "下架版本" : "撤回审核";
    }
    if (spec.kind === "activate") return "设为激活版本";
    return "确认";
  }

  function actionConfirmMessage() {
    const spec = actionConfirm.value;
    if (!spec) return "";
    if (spec.kind === "deleteGroup") {
      return `确定删除镜像组「${spec.group?.name}」？`;
    }
    const ver = spec.image?.version || spec.image?.id;
    if (spec.kind === "delete") {
      return `确定删除版本「${ver}」？此操作不可恢复。`;
    }
    if (spec.kind === "withdraw") {
      return spec.image?.status === "approved"
        ? `确定下架版本「${ver}」？下架后公开目录不再展示该版本，状态变为草稿。`
        : `确定撤回版本「${ver}」的审核？`;
    }
    if (spec.kind === "activate") {
      const cur = groupActiveVersion(spec.image);
      return cur
        ? `将「${ver}」设为激活版本，并自动取消「${cur.version}」的激活？两者均保持已上架，仅生效版本切换。`
        : `将「${ver}」设为激活版本？激活版本在公开镜像市场对外生效。`;
    }
    return "";
  }

  function actionConfirmLabel() {
    const spec = actionConfirm.value;
    if (!spec) return "确定";
    if (spec.kind === "delete" || spec.kind === "deleteGroup") return "删除";
    if (spec.kind === "withdraw") {
      return spec.image?.status === "approved" ? "下架" : "撤回审核";
    }
    if (spec.kind === "activate") return "设为激活";
    return "确定";
  }

  function actionConfirmDanger() {
    const kind = actionConfirm.value?.kind;
    return kind === "delete" || kind === "deleteGroup" || kind === "withdraw";
  }

  async function submitVersionReview(im) {
    // OPT-20260829-022: submit 与删除/下架/激活共享 versionWriteGuard，
    // 双击/超时重试只发一次 POST，并携带 Idempotency-Key 供服务端去重。
    await versionWriteGuard.run(async ({ headers }) => {
      versionActionBusy.value = true;
      try {
        await api(`/api/vendor/container-images/${im.id}/submit/`, {
          method: "POST",
          body: "{}",
          headers,
        });
        await refreshImages();
        await refreshGroups();
      } catch (e) {
        setMsgError(e);
      } finally {
        versionActionBusy.value = false;
      }
    });
  }

  async function confirmPortalAction() {
    const spec = actionConfirm.value;
    if (!spec) return;
    await versionWriteGuard.run(async ({ headers }) => {
      versionActionBusy.value = true;
      try {
        if (spec.kind === "deleteGroup") {
          await api(`/api/vendor/image-groups/${spec.group.id}/`, { method: "DELETE", headers });
          actionConfirm.value = null;
          await refreshGroups();
          return;
        }
        const id = spec.image.id;
        if (spec.kind === "delete") {
          await api(`/api/vendor/container-images/${id}/`, { method: "DELETE", headers });
        } else if (spec.kind === "withdraw") {
          await api(`/api/vendor/container-images/${id}/withdraw/`, {
            method: "POST",
            body: "{}",
            headers,
          });
        } else if (spec.kind === "activate") {
          await api(`/api/vendor/container-images/${id}/activate/`, {
            method: "POST",
            body: "{}",
            headers,
          });
        }
        actionConfirm.value = null;
        await refreshImages();
        await refreshGroups();
      } catch (e) {
        setMsgError(e);
      } finally {
        versionActionBusy.value = false;
      }
    });
  }

  return {
    actionConfirm,
    versionActionBusy,
    openVersionDelete,
    openVersionWithdraw,
    openVersionActivate,
    openGroupDelete,
    actionConfirmTitle,
    actionConfirmMessage,
    actionConfirmLabel,
    actionConfirmDanger,
    confirmPortalAction,
    submitVersionReview,
  };
}

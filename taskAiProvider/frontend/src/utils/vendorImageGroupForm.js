import { uploadImageGroupIcon, validateImageGroupFields } from "./imageGroupIcon.js";

export function createVendorImageGroupFormApi({
  api,
  groupEditingId,
  groupForm,
  groupIconFile,
  groupIconPreview,
  groupFormError,
  groupFormErrorTraceId,
  groupOpen,
  groupSaving,
  groupSaveGuard,
  clearMsg,
  setMsgError,
  msg,
  refreshGroups,
}) {
  function resetIconDraft() {
    groupIconFile.value = null;
    if (groupIconPreview.value) {
      URL.revokeObjectURL(groupIconPreview.value);
    }
    groupIconPreview.value = "";
    groupFormError.value = "";
    groupFormErrorTraceId.value = "";
  }

  function openGroupCreate() {
    groupEditingId.value = null;
    groupForm.value = { name: "", description: "", icon_file_key: "", icon_url: "" };
    resetIconDraft();
    groupOpen.value = true;
  }

  function openGroupEdit(g) {
    groupEditingId.value = g.id;
    groupForm.value = {
      name: g.name,
      description: g.description || "",
      icon_file_key: g.icon_file_key || "",
      icon_url: g.icon_url || "",
    };
    resetIconDraft();
    groupOpen.value = true;
  }

  function onGroupIconFile(ev) {
    const file = ev?.target?.files?.[0] || null;
    groupIconFile.value = file;
    if (groupIconPreview.value) {
      URL.revokeObjectURL(groupIconPreview.value);
      groupIconPreview.value = "";
    }
    if (file) {
      groupIconPreview.value = URL.createObjectURL(file);
    }
  }

  async function saveGroup() {
    return groupSaveGuard.run(async ({ headers }) => {
      const name = (groupForm.value.name || "").trim();
      const description = (groupForm.value.description || "").trim();
      groupFormError.value = "";
      groupFormErrorTraceId.value = "";
      clearMsg();
      const localErr = validateImageGroupFields({
        name,
        description,
        iconFileKey: groupForm.value.icon_file_key,
        iconFile: groupIconFile.value,
      });
      if (localErr) {
        groupFormError.value = localErr;
        msg.value = localErr;
        return;
      }
      groupSaving.value = true;
      try {
        let iconKey = (groupForm.value.icon_file_key || "").trim();
        if (groupIconFile.value) {
          iconKey = await uploadImageGroupIcon(groupIconFile.value, { api });
        }
        const payload = { name, description, icon_file_key: iconKey };
        if (groupEditingId.value) {
          await api(`/api/vendor/image-groups/${groupEditingId.value}/`, {
            method: "PUT",
            headers,
            body: JSON.stringify(payload),
          });
        } else {
          await api("/api/vendor/image-groups/", {
            method: "POST",
            headers,
            body: JSON.stringify(payload),
          });
        }
        groupOpen.value = false;
        await refreshGroups();
      } catch (e) {
        groupFormError.value = e?.message || String(e);
        groupFormErrorTraceId.value = e?.traceId || "";
        setMsgError(e);
      } finally {
        groupSaving.value = false;
      }
    });
  }

  return { openGroupCreate, openGroupEdit, onGroupIconFile, saveGroup };
}

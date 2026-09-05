<script setup>
import { ref } from "vue";
import { api } from "../api.js";
import { createClickGuard } from "../utils/clickGuard.js";
import { formatFileSize, parseSizeToBytes } from "../utils/imageSize.js";
import { requireWritableSkillVersion } from "../lib/saasInboundSkillVersion.js";
import { parseResolvePayload, persistedResolveInfo, resolveRefKey } from "../lib/imageResolveInfo.js";
import ImageResolveInfoPanel from "./ImageResolveInfoPanel.vue";
import SaasInboundSkillVersionSelect from "./SaasInboundSkillVersionSelect.vue";

const emit = defineEmits(["saved", "error"]);

const uploadGuard = createClickGuard();
const editGuard = createClickGuard();

function emptyResolveInfo() {
  return {
    skills: [],
    skillsStatus: "",
    skillsDetail: "",
    autoRunStepsMd: "",
    autoRunStepsStatus: "",
    autoRunStepsDetail: "",
  };
}

const uploadOpen = ref(false);
const uploadForm = ref({
  image_group_id: "",
  version: "",
  image_url: "",
  target_architectures: [],
  size_display: "",
  saas_inbound_skill_version: "",
});
const resolving = ref(false);
const resolveErr = ref("");
const resolveErrTraceId = ref("");
const resolveInfo = ref(emptyResolveInfo());
const lastResolveKey = ref("");
const uploading = ref(false);

const editOpen = ref(false);
const editing = ref(false);
const editId = ref(null);
const editResolving = ref(false);
const editResolveErr = ref("");
const editResolveErrTraceId = ref("");
const editResolveInfo = ref(emptyResolveInfo());
const editLastResolveKey = ref("");
const editForm = ref({
  version: "",
  image_url: "",
  size_display: "",
  target_architectures: [],
  saas_inbound_skill_version: "",
});

function openCreate(group) {
  uploadForm.value = {
    image_group_id: String(group.id),
    version: "",
    image_url: "",
    target_architectures: [],
    size_display: "",
    saas_inbound_skill_version: "",
  };
  resolveErr.value = "";
  resolveErrTraceId.value = "";
  resolveInfo.value = emptyResolveInfo();
  lastResolveKey.value = "";
  uploadOpen.value = true;
}

function openEdit(im) {
  editId.value = im.id;
  editForm.value = {
    version: im.version,
    image_url: im.image_url,
    size_display: im.size ? formatFileSize(Number(im.size)) : "",
    target_architectures: Array.isArray(im.target_architectures)
      ? [...im.target_architectures]
      : [],
    saas_inbound_skill_version: String(im.saas_inbound_skill_version || ""),
  };
  editResolveErr.value = "";
  editResolveErrTraceId.value = "";
  editResolveInfo.value = emptyResolveInfo();
  editLastResolveKey.value = "";
  editOpen.value = true;
  // 已保存版本已含持久化技能/自动运行说明（提取状态 ok）时直接展示，
  // 免去按地址+版本向仓库全层扫描下载（编辑场景多为改大小/接口版本，镜像内容未变）。
  const persisted = persistedResolveInfo(im);
  if (persisted) {
    editResolveInfo.value = persisted;
    editLastResolveKey.value = resolveRefKey(editForm.value.image_url, editForm.value.version);
    return;
  }
  void resolveEditArch();
}

async function resolveArch() {
  const url = uploadForm.value.image_url?.trim();
  if (!url) {
    uploadForm.value.target_architectures = [];
    resolveInfo.value = emptyResolveInfo();
    lastResolveKey.value = "";
    return;
  }
  const key = resolveRefKey(url, uploadForm.value.version);
  if (key === lastResolveKey.value) {
    return; // 同一地址+版本已成功解析，避免重复触发镜像层下载
  }
  resolving.value = true;
  resolveErr.value = "";
  resolveErrTraceId.value = "";
  try {
    const payload = await api("/api/vendor/container-images/resolve-target-architectures/", {
      method: "POST",
      body: JSON.stringify({
        image_url: url,
        version: uploadForm.value.version?.trim() || "",
      }),
    });
    uploadForm.value.target_architectures = payload.target_architectures || [];
    uploadForm.value.size_display = payload.size ? formatFileSize(Number(payload.size)) : "";
    resolveInfo.value = parseResolvePayload(payload);
    lastResolveKey.value = key;
  } catch (e) {
    uploadForm.value.target_architectures = [];
    resolveInfo.value = emptyResolveInfo();
    resolveErr.value = e.message;
    resolveErrTraceId.value = e.traceId || "";
  } finally {
    resolving.value = false;
  }
}

async function resolveEditArch() {
  const url = editForm.value.image_url?.trim();
  if (!url) {
    editForm.value.target_architectures = [];
    editResolveInfo.value = emptyResolveInfo();
    editLastResolveKey.value = "";
    return;
  }
  const key = resolveRefKey(url, editForm.value.version);
  if (key === editLastResolveKey.value) {
    return; // 同一地址+版本已成功解析，避免重复触发镜像层下载
  }
  editResolving.value = true;
  editResolveErr.value = "";
  editResolveErrTraceId.value = "";
  try {
    const payload = await api("/api/vendor/container-images/resolve-target-architectures/", {
      method: "POST",
      body: JSON.stringify({
        image_url: url,
        version: editForm.value.version?.trim() || "",
      }),
    });
    editForm.value.target_architectures = payload.target_architectures || [];
    editForm.value.size_display = payload.size ? formatFileSize(Number(payload.size)) : "";
    editResolveInfo.value = parseResolvePayload(payload);
    editLastResolveKey.value = key;
  } catch (e) {
    editResolveErr.value = e.message;
    editResolveErrTraceId.value = e.traceId || "";
  } finally {
    editResolving.value = false;
  }
}

async function doUpload() {
  const out = await uploadGuard.run(async ({ headers }) => {
    uploading.value = true;
    try {
      const skill = requireWritableSkillVersion(uploadForm.value.saas_inbound_skill_version);
      const parsedSize = parseSizeToBytes(uploadForm.value.size_display);
      const body = {
        image_group_id: uploadForm.value.image_group_id,
        version: uploadForm.value.version,
        image_url: uploadForm.value.image_url,
        target_architectures: uploadForm.value.target_architectures,
        saas_inbound_skill_version: skill,
      };
      if (parsedSize && parsedSize !== "") {
        body.size = parsedSize;
      }
      await api("/api/vendor/container-images/", {
        method: "POST",
        headers,
        body: JSON.stringify(body),
      });
      uploadOpen.value = false;
      emit("saved");
    } catch (e) {
      emit("error", e);
    } finally {
      uploading.value = false;
    }
  });
  return out;
}

async function doEdit() {
  const out = await editGuard.run(async ({ headers }) => {
    editing.value = true;
    try {
      const skill = requireWritableSkillVersion(editForm.value.saas_inbound_skill_version);
      const body = {
        version: editForm.value.version,
        image_url: editForm.value.image_url,
        target_architectures: editForm.value.target_architectures,
        size: parseSizeToBytes(editForm.value.size_display),
        saas_inbound_skill_version: skill,
      };
      await api(`/api/vendor/container-images/${editId.value}/`, {
        method: "PUT",
        headers,
        body: JSON.stringify(body),
      });
      editOpen.value = false;
      emit("saved");
    } catch (e) {
      emit("error", e);
    } finally {
      editing.value = false;
    }
  });
  return out;
}

defineExpose({ openCreate, openEdit });
</script>

<template>
  <div>
    <div v-if="uploadOpen" class="modal-bg" @click.self="uploadOpen = false">
      <div class="card modal">
        <h3>添加镜像版本</h3>
        <label class="lbl">版本号</label>
        <input v-model="uploadForm.version" class="inp" placeholder="例如: 1.0.0" @blur="resolveArch" />
        <label class="lbl">镜像地址</label>
        <input v-model="uploadForm.image_url" class="inp" @blur="resolveArch" />
        <p class="hint">镜像地址或版本号失焦后，按地址与版本向仓库解析架构、大小、技能列表与自动运行说明（地址已含标签或摘要时以地址为准；需网络可达）</p>
        <div v-if="resolving" class="hint">解析中…</div>
        <div v-else-if="uploadForm.target_architectures?.length" class="arch">
          <span v-for="a in uploadForm.target_architectures" :key="a" class="pill">{{ a }}</span>
        </div>
        <p v-if="resolveErr" class="err" v-bind="resolveErrTraceId ? { 'data-traceId': resolveErrTraceId } : {}">{{ resolveErr }}</p>
        <ImageResolveInfoPanel
          v-if="!resolving"
          :skills="resolveInfo.skills"
          :skills-status="resolveInfo.skillsStatus"
          :skills-detail="resolveInfo.skillsDetail"
          :auto-run-steps-md="resolveInfo.autoRunStepsMd"
          :auto-run-steps-status="resolveInfo.autoRunStepsStatus"
          :auto-run-steps-detail="resolveInfo.autoRunStepsDetail"
        />
        <label class="lbl">大小（可选，如 850 MB）</label>
        <input v-model="uploadForm.size_display" class="inp" />
        <SaasInboundSkillVersionSelect v-model="uploadForm.saas_inbound_skill_version" />
        <div class="row">
          <button class="btn btn-ghost" type="button" @click="uploadOpen = false">取消</button>
          <button
            class="btn btn-primary"
            type="button"
            :disabled="uploading"
            :aria-busy="uploading ? 'true' : 'false'"
            @click="doUpload"
          >
            {{ uploading ? "提交中…" : "保存为草稿" }}
          </button>
        </div>
      </div>
    </div>

    <div v-if="editOpen" class="modal-bg" @click.self="editOpen = false">
      <div class="card modal">
        <h3>编辑镜像版本</h3>
        <label class="lbl">版本号</label>
        <input v-model="editForm.version" class="inp" @blur="resolveEditArch" />
        <label class="lbl">镜像地址</label>
        <input v-model="editForm.image_url" class="inp" @blur="resolveEditArch" />
        <p class="hint">镜像地址或版本号失焦后，按地址与版本向仓库解析架构、大小、技能列表与自动运行说明（地址已含标签或摘要时以地址为准；需网络可达）</p>
        <div v-if="editResolving" class="hint">解析中…</div>
        <div v-else-if="editForm.target_architectures?.length" class="arch">
          <span v-for="a in editForm.target_architectures" :key="a" class="pill">{{ a }}</span>
        </div>
        <p v-if="editResolveErr" class="err" v-bind="editResolveErrTraceId ? { 'data-traceId': editResolveErrTraceId } : {}">{{ editResolveErr }}</p>
        <ImageResolveInfoPanel
          v-if="!editResolving"
          :skills="editResolveInfo.skills"
          :skills-status="editResolveInfo.skillsStatus"
          :skills-detail="editResolveInfo.skillsDetail"
          :auto-run-steps-md="editResolveInfo.autoRunStepsMd"
          :auto-run-steps-status="editResolveInfo.autoRunStepsStatus"
          :auto-run-steps-detail="editResolveInfo.autoRunStepsDetail"
        />
        <label class="lbl">大小</label>
        <input v-model="editForm.size_display" class="inp" />
        <SaasInboundSkillVersionSelect v-model="editForm.saas_inbound_skill_version" />
        <div class="row">
          <button class="btn btn-ghost" type="button" @click="editOpen = false">取消</button>
          <button
            class="btn btn-primary"
            type="button"
            :disabled="editing"
            :aria-busy="editing ? 'true' : 'false'"
            @click="doEdit"
          >
            {{ editing ? "保存中…" : "保存" }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped src="../views/VendorPortal.css"></style>

<style scoped>
/* 弹窗底部按钮行：取消/保存按钮等宽分摊整行，视觉更饱满（仅本组件生效，
   .row 为 VendorPortal.css 全局类，其他弹窗不受影响） */
.row .btn {
  flex: 1 1 0;
}
</style>

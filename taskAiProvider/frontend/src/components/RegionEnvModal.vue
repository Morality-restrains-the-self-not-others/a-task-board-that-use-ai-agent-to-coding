<template>
  <div v-if="visible" class="modal-bg" @click.self="$emit('close')">
    <div class="card modal">
      <h3>设置区域运行环境</h3>
      <p class="hint">
        为 <strong>{{ platformLabel }}</strong> - <strong>{{ region?.name }}</strong> 选择服务器镜像
      </p>
      <p class="hint">列出当前云平台已登记的全部服务器镜像；与本区域、容器架构匹配的项排在前面。</p>
      <p v-if="targetArchitectures.length" class="hint">
        当前容器镜像架构：<span v-for="a in targetArchitectures" :key="a" class="pill">{{ a }}</span>
      </p>
      <label class="lbl">服务器镜像</label>
      <select
        v-model="form.cloud_server_image_id"
        class="inp region-env-server-select"
        @change="onServerImageChange"
      >
        <option value="">— 不选（清除本区域关联）—</option>
        <option
          v-for="opt in serverOptions"
          :key="opt.id"
          :value="String(opt.id)"
          :title="opt.image_name + ' (' + opt.region + ') ' + (opt.architecture || '') + (hardwareSummary(opt) !== '—' ? ' · ' + hardwareSummary(opt) : '')"
        >
          {{ opt.image_name }} ({{ opt.region }}) {{ opt.architecture || "" }}
          {{ hardwareSummary(opt) !== "—" ? " · " + hardwareSummary(opt) : "" }}
        </option>
      </select>
      <label class="lbl">UserData 模板</label>
      <select
        v-model="form.userdata_template_id"
        class="inp region-env-userdata-select"
        :disabled="!form.cloud_server_image_id"
      >
        <option value="">不选用</option>
        <option v-for="t in userdataTemplateOptions" :key="t.id" :value="String(t.id)">
          {{ t.name }} v{{ t.version }}（{{ t.os_type }}）
        </option>
      </select>
      <p class="hint">
        须先选择服务器镜像；模板保存在该条云平台服务器镜像上，与「云平台服务器镜像」页登记时一致。选择「不选」并保存将清除本区域关联。
      </p>
      <p
        v-if="error"
        class="err"
        role="alert"
        v-bind="errorTraceId ? { 'data-traceId': errorTraceId } : {}"
      >{{ error }}</p>
      <div class="row">
        <button class="btn btn-ghost" type="button" @click="openCsCreate">登记新镜像</button>
        <button class="btn btn-ghost" type="button" @click="$emit('close')">取消</button>
        <button class="btn btn-primary" type="button" :disabled="saving" @click="doSave">
          {{ saving ? "保存中…" : "保存" }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, ref, watch } from "vue";
import { buildRegionEnvServerOptions } from "../utils/regionEnvServerOptions.js";
import { normalizeUserdataTemplate } from "../utils/userdataTemplateDisplay.js";

const props = defineProps({
  visible: { type: Boolean, default: false },
  serverImages: { type: Array, default: () => [] },
  userdataTemplateOptions: { type: Array, default: () => [] },
  cloudPlatforms: { type: Array, default: () => [] },
  containerImageId: { type: [String, Number], default: null },
  platformType: { type: String, default: "" },
  /** { id, name } */
  region: { type: Object, default: null },
  targetArchitectures: { type: Array, default: () => [] },
  initialCloudServerImageId: { type: String, default: "" },
  initialUserdataTemplateId: { type: String, default: "" },
  saving: { type: Boolean, default: false },
  error: { type: String, default: "" },
  errorTraceId: { type: String, default: "" },
});

const emit = defineEmits(["close", "save", "open-cs-create"]);

const form = ref({ cloud_server_image_id: "", userdata_template_id: "" });

watch(
  () => props.visible,
  (visible) => {
    if (visible) {
      form.value = {
        cloud_server_image_id: props.initialCloudServerImageId,
        userdata_template_id: props.initialUserdataTemplateId,
      };
    }
  },
);

const platformLabel = computed(() => {
  if (!props.platformType) return "";
  return props.cloudPlatforms.find((p) => p.value === props.platformType)?.label || props.platformType;
});

const serverOptions = computed(() =>
  buildRegionEnvServerOptions(props.serverImages, {
    platform: props.platformType,
    region: props.region,
    targetArchs: props.targetArchitectures,
    selectedId: form.value.cloud_server_image_id,
  }),
);

function hardwareSummary(s) {
  if (!s) return "—";
  if (s.default_instance_type_label) return s.default_instance_type_label;
  if (s.base_cpu_cores != null || s.base_memory_gib != null) {
    const c = s.base_cpu_cores != null ? `${s.base_cpu_cores}` : "?";
    const m = s.base_memory_gib != null ? `${s.base_memory_gib}` : "?";
    return `${c}vCPU / ${m}GiB`;
  }
  return "—";
}

function onServerImageChange() {
  const id = form.value.cloud_server_image_id;
  if (!id) {
    form.value.userdata_template_id = "";
    return;
  }
  const s = props.serverImages.find((x) => String(x.id) === String(id));
  const ut = normalizeUserdataTemplate(s?.userdata_template);
  form.value.userdata_template_id = ut?.id != null ? String(ut.id) : "";
}

function doSave() {
  emit("save", {
    containerImageId: props.containerImageId,
    platformType: props.platformType,
    regionId: props.region?.id,
    cloudServerImageId: form.value.cloud_server_image_id,
    userdataTemplateId: form.value.userdata_template_id,
  });
}

function openCsCreate() {
  emit("open-cs-create");
}
</script>

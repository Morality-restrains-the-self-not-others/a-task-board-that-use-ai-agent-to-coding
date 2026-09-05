<template>
  <div v-if="open" class="modal-bg" @click.self="$emit('close')">
    <div class="card modal">
      <h3>{{ editingId ? "编辑测试密钥" : "添加测试密钥" }}</h3>
      <label class="lbl">云平台</label>
      <select v-model="form.platform_type" class="inp" :disabled="!!editingId">
        <option v-for="p in cloudPlatforms" :key="p.value" :value="p.value">{{ p.label }}</option>
      </select>
      <label class="lbl">AccessKey ID</label>
      <input v-model="form.secret_id" class="inp" autocomplete="off" />
      <label class="lbl">AccessKey Secret</label>
      <input
        v-model="form.secret_key"
        class="inp"
        type="password"
        autocomplete="new-password"
        :placeholder="editingId ? '留空则不修改' : ''"
      />
      <label class="lbl">备注</label>
      <input v-model="form.remark" class="inp" placeholder="可选" />
      <p v-if="err" class="err" v-bind="errTraceId ? { 'data-traceId': errTraceId } : {}">{{ err }}</p>
      <div class="row">
        <button class="btn btn-ghost" type="button" @click="$emit('close')">取消</button>
        <button class="btn btn-primary" type="button" :disabled="saving" @click="save">
          {{ saving ? "保存中…" : "保存" }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, watch } from "vue";
import { api } from "../api";

const props = defineProps({
  open: { type: Boolean, default: false },
  editingId: { type: [String, Number], default: null },
  initial: { type: Object, default: null },
  cloudPlatforms: { type: Array, default: () => [] },
});

const emit = defineEmits(["close", "saved", "created"]);

const form = ref({
  platform_type: "aliyun",
  secret_id: "",
  secret_key: "",
  remark: "",
});
const saving = ref(false);
const err = ref("");
const errTraceId = ref("");

watch(
  () => props.open,
  (v) => {
    if (!v) return;
    err.value = "";
    errTraceId.value = "";
    if (props.initial) {
      form.value = {
        platform_type: props.initial.platform_type || "aliyun",
        secret_id: "",
        secret_key: "",
        remark: props.initial.remark || "",
      };
    } else {
      form.value = { platform_type: "aliyun", secret_id: "", secret_key: "", remark: "" };
    }
  }
);

async function save() {
  err.value = "";
  errTraceId.value = "";
  if (!form.value.secret_id && !props.editingId) {
    err.value = "请填写 AccessKey ID";
    return;
  }
  if (!form.value.secret_key && !props.editingId) {
    err.value = "请填写 AccessKey Secret";
    return;
  }
  saving.value = true;
  try {
    const body = {
      platform_type: form.value.platform_type,
      remark: form.value.remark,
    };
    if (form.value.secret_id) body.secret_id = form.value.secret_id;
    if (form.value.secret_key) body.secret_key = form.value.secret_key;
    if (props.editingId) {
      await api(`/api/vendor/cloud-platform-credentials/${props.editingId}/`, {
        method: "PATCH",
        body: JSON.stringify(body),
      });
    } else {
      const created = await api("/api/vendor/cloud-platform-credentials/", {
        method: "POST",
        body: JSON.stringify(body),
      });
      emit("created", created);
    }
    emit("saved");
    emit("close");
  } catch (e) {
    err.value = e.data?.detail || e.message || "保存失败";
    errTraceId.value = e.traceId || "";
  } finally {
    saving.value = false;
  }
}
</script>

<style scoped src="../views/VendorPortal.css"></style>

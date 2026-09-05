<template>
  <div class="card" style="max-width: 640px">
    <h2 class="h2">证照对象路径</h2>
    <p class="sub">写回 conf/ai/ai-provider/vendor-docs-path.yaml，立即热加载。不展示密钥。</p>
    <p v-if="msg" class="msg" v-bind="msgTraceId ? { 'data-traceId': msgTraceId } : {}">{{ msg }}</p>
    <div class="row gap" style="margin-top: 0.75rem">
      <label>backend</label>
      <span class="muted">{{ form.backend || "—" }}</span>
    </div>
    <div class="row gap">
      <label>bucket / region</label>
      <span class="muted">{{ form.bucket || "—" }} / {{ form.region || "—" }}</span>
    </div>
    <label class="muted">keyPrefix</label>
    <input v-model.trim="form.keyPrefix" class="inp" maxlength="64">
    <label class="muted">pathRule</label>
    <input v-model.trim="form.pathRule" class="inp" maxlength="255">
    <p class="muted">占位符：{keyPrefix} {userId} {kind} {id} {ext} {yyyy} {mm} {dd}</p>
    <div class="row" style="margin-top: 0.75rem">
      <button class="btn btn-primary" type="button" :disabled="saving" @click="save">保存</button>
      <button class="btn btn-ghost" type="button" @click="load">刷新</button>
    </div>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from "vue";
import { api } from "../api";
import { showRequestError } from "../utils/showRequestError.js";

const form = reactive({
  backend: "",
  bucket: "",
  region: "",
  keyPrefix: "",
  pathRule: "",
});
const saving = ref(false);
const msg = ref("");
const msgTraceId = ref("");

async function load() {
  msg.value = "";
  msgTraceId.value = "";
  try {
    const d = await api("/api/admin/vendor-docs-storage/");
    form.backend = d.backend || "";
    form.bucket = d.bucket || "";
    form.region = d.region || "";
    form.keyPrefix = d.keyPrefix || "";
    form.pathRule = d.pathRule || "";
  } catch (e) {
    showRequestError("读取路径规则失败: " + (e.message || "网络错误"), e);
  }
}

async function save() {
  saving.value = true;
  msg.value = "";
  try {
    const d = await api("/api/admin/vendor-docs-storage/", {
      method: "PATCH",
      body: JSON.stringify({ keyPrefix: form.keyPrefix, pathRule: form.pathRule }),
    });
    form.keyPrefix = d.keyPrefix || form.keyPrefix;
    form.pathRule = d.pathRule || form.pathRule;
    msg.value = "已写回 conf 并热加载";
  } catch (e) {
    msg.value = e?.message || "保存失败";
    msgTraceId.value = e?.traceId || "";
    showRequestError("保存路径规则失败: " + (e.message || "网络错误"), e);
  } finally {
    saving.value = false;
  }
}

onMounted(load);
</script>

<style scoped src="../views/AdminPortal.css"></style>

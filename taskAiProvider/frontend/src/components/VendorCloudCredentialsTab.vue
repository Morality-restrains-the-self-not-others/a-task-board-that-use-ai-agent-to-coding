<template>
  <div>
    <div class="row-between">
      <h2 class="h2">云平台测试密钥</h2>
      <button class="btn btn-primary" type="button" @click="openCreate">添加测试密钥</button>
    </div>
    <p class="sub">
      登记「云平台服务器镜像」前，须在此配置各云厂商测试用 AccessKey；配置后请点击「测试密钥」校验。
    </p>
    <div v-if="credentialHint" class="credential-alert">
      {{ credentialHint }}
    </div>
    <div class="table-wrap">
      <table>
        <thead>
          <tr>
            <th>云平台</th>
            <th>AccessKey ID</th>
            <th>备注</th>
            <th>最近校验</th>
            <th>状态</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="c in credentials" :key="c.id">
            <td>{{ platformLabel(c.platform_type) }}</td>
            <td class="mono">{{ c.secret_id || "—" }}</td>
            <td>{{ c.remark || "—" }}</td>
            <td>{{ formatVerified(c.last_verified_at) }}</td>
            <td>
              <span :class="c.is_active ? 'status-badge set' : 'status-badge unset'">
                {{ c.is_active ? "已启用" : "已禁用" }}
              </span>
            </td>
            <td class="cred-actions">
              <button
                class="btn-mini primary"
                type="button"
                :disabled="verifyingId === c.id"
                @click="verify(c)"
              >
                {{ verifyingId === c.id ? "校验中…" : "测试密钥" }}
              </button>
              <button class="btn-mini" type="button" @click="openEdit(c)">编辑</button>
              <button class="btn-mini" type="button" @click="toggleActive(c)">
                {{ c.is_active ? "禁用" : "启用" }}
              </button>
              <button class="btn-mini danger" type="button" @click="remove(c)">删除</button>
            </td>
          </tr>
        </tbody>
      </table>
      <p v-if="credentials.length === 0" class="hint empty-cred">暂无测试密钥，点击「添加测试密钥」开始配置。</p>
    </div>
    <p v-if="msg" class="msg" v-bind="msgTraceId ? { 'data-traceId': msgTraceId } : {}">{{ msg }}</p>

    <VendorCloudCredentialFormModal
      :open="formOpen"
      :editing-id="editingId"
      :initial="editingRow"
      :cloud-platforms="cloudPlatforms"
      @close="formOpen = false"
      @saved="onSaved"
      @created="onCreated"
    />
  </div>
</template>

<script setup>
import { onMounted, ref } from "vue";
import { api } from "../api";
import VendorCloudCredentialFormModal from "./VendorCloudCredentialFormModal.vue";

const props = defineProps({
  cloudPlatforms: { type: Array, default: () => [] },
  credentialHint: { type: String, default: "" },
});

const emit = defineEmits(["credentials-changed"]);

const credentials = ref([]);
const msg = ref("");
const msgTraceId = ref("");
const formOpen = ref(false);

function clearMsg() {
  msg.value = "";
  msgTraceId.value = "";
}

function setMsgError(e, fallback = "操作失败") {
  msg.value = e?.data?.detail || e?.message || fallback;
  msgTraceId.value = e?.traceId || "";
}

function setMsgText(text) {
  msg.value = text;
  msgTraceId.value = "";
}
const editingId = ref(null);
const editingRow = ref(null);
const verifyingId = ref(null);

function platformLabel(pt) {
  return props.cloudPlatforms.find((p) => p.value === pt)?.label || pt;
}

function formatVerified(v) {
  if (!v) return "—";
  try {
    return new Date(v).toLocaleString();
  } catch {
    return String(v);
  }
}

async function load() {
  try {
    credentials.value = await api("/api/vendor/cloud-platform-credentials/");
  } catch {
    credentials.value = [];
  }
}

function openCreate() {
  editingId.value = null;
  editingRow.value = null;
  formOpen.value = true;
}

function openEdit(c) {
  editingId.value = c.id;
  editingRow.value = c;
  formOpen.value = true;
}

async function onSaved() {
  await load();
  emit("credentials-changed");
  setMsgText("已保存");
}

function onCreated(cred) {
  // Immediately add the new credential to the list so it renders without
  // waiting for a re-fetch (which could fail or return cached/stale data).
  if (cred && cred.id) {
    const exists = credentials.value.some((c) => c.id === cred.id);
    if (!exists) {
      credentials.value = [...credentials.value, cred];
    }
  }
  emit("credentials-changed");
  setMsgText("已保存");
}

async function verify(c) {
  verifyingId.value = c.id;
  clearMsg();
  try {
    const data = await api(`/api/vendor/cloud-platform-credentials/${c.id}/verify-credentials/`, {
      method: "POST",
    });
    if (data?.success) {
      setMsgText("密钥校验通过");
      await load();
      emit("credentials-changed");
    } else {
      setMsgText(data?.detail || "校验失败");
    }
  } catch (e) {
    setMsgError(e, "校验失败");
  } finally {
    verifyingId.value = null;
  }
}

async function toggleActive(c) {
  try {
    await api(`/api/vendor/cloud-platform-credentials/${c.id}/`, {
      method: "PATCH",
      body: JSON.stringify({ is_active: !c.is_active }),
    });
    await load();
    emit("credentials-changed");
  } catch (e) {
    setMsgError(e, "操作失败");
  }
}

async function remove(c) {
  if (!window.confirm(`确定删除 ${platformLabel(c.platform_type)} 的测试密钥？`)) return;
  try {
    await api(`/api/vendor/cloud-platform-credentials/${c.id}/`, { method: "DELETE" });
    await load();
    emit("credentials-changed");
  } catch (e) {
    setMsgError(e, "删除失败");
  }
}

onMounted(load);

defineExpose({ reload: load });
</script>

<style scoped>
.credential-alert {
  background: #fef3c7;
  border: 1px solid #fcd34d;
  color: #92400e;
  padding: 0.65rem 0.85rem;
  border-radius: 8px;
  margin-bottom: 1rem;
  font-size: 0.9rem;
}
.cred-actions {
  white-space: nowrap;
}
.cred-actions .btn-mini {
  margin-left: 0.25rem;
}
.empty-cred {
  padding: 1rem 0;
}
</style>

<style scoped src="../views/VendorPortal.css"></style>

<template>
  <div>
    <div class="row gap">
      <span class="muted">厂商申请审核（提交后自动建档为 is_active=0 待审核）</span>
      <!-- Anti-Replay-OK: read-refresh -->
      <button class="btn btn-ghost" type="button" :disabled="loadBusy" @click="loadVendors">刷新</button>
    </div>
    <div class="table-wrap" style="margin-top: 1rem">
      <table>
        <thead>
          <tr>
            <th>ID</th>
            <th>邮箱</th>
            <th>公司名称</th>
            <th>联系人</th>
            <th>状态</th>
            <th>证照</th>
            <th>驳回原因</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="v in vendors" :key="v.id">
            <td>{{ v.id }}</td>
            <td>{{ v.email }}</td>
            <td>{{ v.company_name || "—" }}</td>
            <td>{{ v.contact_name || "—" }}</td>
            <td>
              <span v-if="v.is_active" class="badge badge-ok">已获准</span>
              <span v-else-if="v.review_note" class="badge badge-warn">已驳回</span>
              <span v-else class="badge badge-pending">待审核</span>
            </td>
            <td>
              <!-- Anti-Replay-OK: GET blob 新标签打开证照 -->
              <button v-if="v.has_id_card" class="btn btn-ghost sm" type="button" @click="openDoc(v, 'id_card')">身份证</button>
              <button v-if="v.has_business_license" class="btn btn-ghost sm" type="button" @click="openDoc(v, 'business_license')">执照</button>
              <span v-if="!v.has_id_card && !v.has_business_license" class="muted">—</span>
            </td>
            <td>{{ v.review_note || "—" }}</td>
            <td>
              <template v-if="!v.is_active">
                <button
                  class="btn btn-primary sm"
                  type="button"
                  :disabled="writeBusy"
                  :aria-busy="writeBusy ? 'true' : 'false'"
                  @click="openApprove(v)"
                >通过</button>
                <button
                  class="btn btn-warning sm"
                  type="button"
                  :disabled="writeBusy"
                  :aria-busy="writeBusy ? 'true' : 'false'"
                  @click="openReject(v)"
                >驳回</button>
              </template>
              <span v-else class="muted">—</span>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
    <AdminImageReviewNoteModal
      v-if="actionModal"
      :title="actionModalMeta.title"
      :confirm-label="actionModalMeta.confirmLabel"
      :confirm-class="actionModalMeta.confirmClass"
      :require-note="actionModalMeta.requireNote"
      :placeholder="actionModalMeta.placeholder"
      :busy="writeBusy"
      :note="actionNote"
      :error="actionError"
      :error-trace-id="actionErrorTraceId"
      @update:note="actionNote = $event"
      @cancel="closeActionModal"
      @confirm="confirmAction"
    />
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from "vue";
import { api } from "../api";
import { showRequestError } from "../utils/showRequestError.js";
import { createClickGuard } from "../utils/clickGuard.js";
import { attachClientTraceId, buildOutboundTraceHeaders } from "../utils/traceId.js";
import AdminImageReviewNoteModal from "./AdminImageReviewNoteModal.vue";

const vendors = ref([]);
const actionModal = ref(null);
const actionNote = ref("");
const actionError = ref("");
const actionErrorTraceId = ref("");
const writeBusy = ref(false);
const loadBusy = ref(false);
const writeGuard = createClickGuard();
const loadGuard = createClickGuard();

function vendorLabel(v) {
  return v.company_name || v.email || String(v.id);
}

const actionModalMeta = computed(() => {
  const kind = actionModal.value?.kind;
  const label = actionModal.value ? vendorLabel(actionModal.value.vendor) : "";
  if (kind === "approve") {
    return {
      title: `确认通过厂商「${label}」的申请？`,
      confirmLabel: "确认通过",
      confirmClass: "btn-primary",
      requireNote: false,
      placeholder: "",
    };
  }
  return {
    title: `驳回厂商「${label}」的申请`,
    confirmLabel: "确认驳回",
    confirmClass: "btn-warning",
    requireNote: true,
    placeholder: "必填驳回原因",
  };
});

function openApprove(v) {
  actionModal.value = { kind: "approve", vendor: v };
  actionNote.value = "";
  actionError.value = "";
  actionErrorTraceId.value = "";
}

function openReject(v) {
  actionModal.value = { kind: "reject", vendor: v };
  actionNote.value = "";
  actionError.value = "";
  actionErrorTraceId.value = "";
}

function closeActionModal() {
  if (writeBusy.value) return;
  actionModal.value = null;
}

async function loadVendors() {
  await loadGuard.run(async () => {
    loadBusy.value = true;
    try {
      vendors.value = await api("/api/admin/vendors/");
    } catch (e) {
      showRequestError("加载厂商列表失败: " + (e.message || "网络错误"), e);
    } finally {
      loadBusy.value = false;
    }
  });
}

async function confirmAction() {
  const modal = actionModal.value;
  if (!modal) return;
  if (modal.kind !== "approve" && !actionNote.value.trim()) {
    actionError.value = "驳回原因必填";
    actionErrorTraceId.value = "";
    return;
  }
  await writeGuard.run(async ({ headers }) => {
    writeBusy.value = true;
    actionError.value = "";
    actionErrorTraceId.value = "";
    try {
      const body =
        modal.kind === "approve"
          ? { action: "approve" }
          : { action: "reject", note: actionNote.value.trim() };
      await api(`/api/admin/vendors/${modal.vendor.id}/`, {
        method: "PATCH",
        headers,
        body: JSON.stringify(body),
      });
      actionModal.value = null;
      await loadVendors();
    } catch (e) {
      const label = modal.kind === "approve" ? "审核通过失败" : "驳回失败";
      actionError.value = `${label}: ${e.message || "网络错误"}`;
      actionErrorTraceId.value = e.traceId || "";
      showRequestError(actionError.value, e);
    } finally {
      writeBusy.value = false;
    }
  });
}

async function openDoc(v, kind) {
  let requestTraceId = "";
  try {
    // 证照是二进制 blob，不能走 api() 的 JSON 解析；手动补出站 trace 头
    // 让失败同样可对 Loki（OPT-20260828-019）。
    const trace = buildOutboundTraceHeaders();
    requestTraceId = trace.requestTraceId;
    const t = localStorage.getItem("staff_token");
    const headers = {
      ...trace.headers,
      ...(t ? { Authorization: `Bearer ${t}` } : {}),
    };
    const r = await fetch(`/api/admin/vendors/${v.id}/documents/${kind}/`, {
      headers,
    });
    if (!r.ok) throw new Error("证照不可用");
    const blob = await r.blob();
    window.open(URL.createObjectURL(blob), "_blank", "noopener");
  } catch (e) {
    attachClientTraceId(e, requestTraceId);
    showRequestError("打开证照失败: " + (e.message || "网络错误"), e);
  }
}

onMounted(loadVendors);
</script>

<style scoped src="../views/AdminPortal.css"></style>

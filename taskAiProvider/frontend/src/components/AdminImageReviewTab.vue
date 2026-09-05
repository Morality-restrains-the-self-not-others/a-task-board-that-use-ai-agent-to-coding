<template>
  <div>
    <div class="row gap">
      <label>状态筛选</label>
      <select v-model="statusFilter" class="inp sm" @change="load">
        <option value="">全部</option>
        <option value="pending_review">待审核</option>
        <option value="approved">已上架</option>
        <option value="rejected">已驳回</option>
        <option value="draft">草稿</option>
      </select>
      <!-- Anti-Replay-OK: read-refresh -->
      <button class="btn btn-ghost" type="button" :disabled="loadBusy" @click="load">刷新</button>
    </div>
    <div class="table-wrap" style="margin-top: 1rem">
      <AdminImageReviewTable
        v-for="(group, platformType) in groupedImages"
        :key="platformType"
        :title="getPlatformDisplayName(platformType)"
        :images="group"
        :write-busy="writeBusy"
        @detail="openDetail"
        @approve="openApprove"
        @reject="openReject"
        @unpublish="openUnpublish"
      />
      <AdminImageReviewTable
        v-if="unassociatedImages.length > 0"
        title="未关联云平台"
        :images="unassociatedImages"
        :write-busy="writeBusy"
        @detail="openDetail"
        @approve="openApprove"
        @reject="openReject"
        @unpublish="openUnpublish"
      />
      <div v-if="images.length === 0" class="empty-msg">
        <p class="muted">暂无数据</p>
      </div>
    </div>

    <AdminImageReviewDetailModal
      v-if="detailImage"
      :image="detailImage"
      @close="detailImage = null"
    />
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
import { getPlatformDisplayName } from "../utils/platformLabel.js";
import { createClickGuard } from "../utils/clickGuard.js";
import AdminImageReviewTable from "./AdminImageReviewTable.vue";
import AdminImageReviewDetailModal from "./AdminImageReviewDetailModal.vue";
import AdminImageReviewNoteModal from "./AdminImageReviewNoteModal.vue";

const images = ref([]);
const statusFilter = ref("pending_review");
const detailImage = ref(null);
const actionModal = ref(null);
const actionNote = ref("");
const actionError = ref("");
const actionErrorTraceId = ref("");
const writeBusy = ref(false);
const loadBusy = ref(false);
const writeGuard = createClickGuard();
const loadGuard = createClickGuard();

const groupedImages = computed(() => {
  const groups = {};
  for (const im of images.value) {
    if (im.platform_types && im.platform_types.length > 0) {
      for (const pt of im.platform_types) {
        if (!groups[pt]) groups[pt] = [];
        if (!groups[pt].find((item) => item.id === im.id)) groups[pt].push(im);
      }
    }
  }
  const sortedGroups = {};
  Object.keys(groups).sort().forEach((key) => {
    sortedGroups[key] = groups[key];
  });
  return sortedGroups;
});

const unassociatedImages = computed(() =>
  images.value.filter((im) => !im.platform_types || im.platform_types.length === 0),
);

const actionModalMeta = computed(() => {
  const kind = actionModal.value?.kind;
  if (kind === "approve") {
    return {
      title: `确认通过「${actionModal.value.image.name} ${actionModal.value.image.version}」？`,
      confirmLabel: "确认通过",
      confirmClass: "btn-primary",
      requireNote: false,
      placeholder: "",
    };
  }
  if (kind === "reject") {
    return {
      title: "驳回原因",
      confirmLabel: "确认驳回",
      confirmClass: "btn-danger",
      requireNote: true,
      placeholder: "必填",
    };
  }
  return {
    title: "撤销上架原因",
    confirmLabel: "确认撤销上架",
    confirmClass: "btn-warning",
    requireNote: true,
    placeholder: "必填",
  };
});

async function openDetail(im) {
  detailImage.value = im;
  try {
    // Anti-Replay-OK: read-refresh
    const fresh = await api(`/api/admin/container-images/${im.id}/`);
    detailImage.value = fresh;
  } catch {
    // keep list row when GET-by-id fails
  }
}

function openApprove(im) {
  actionModal.value = { kind: "approve", image: im };
  actionNote.value = "";
  actionError.value = "";
  actionErrorTraceId.value = "";
}

function openReject(im) {
  actionModal.value = { kind: "reject", image: im };
  actionNote.value = "";
  actionError.value = "";
  actionErrorTraceId.value = "";
}

function openUnpublish(im) {
  actionModal.value = { kind: "unpublish", image: im };
  actionNote.value = "";
  actionError.value = "";
  actionErrorTraceId.value = "";
}

function closeActionModal() {
  if (writeBusy.value) return;
  actionModal.value = null;
}

async function load() {
  await loadGuard.run(async () => {
    loadBusy.value = true;
    try {
      const q = statusFilter.value ? `?status=${encodeURIComponent(statusFilter.value)}` : "";
      images.value = await api(`/api/admin/container-images/${q}`);
    } catch (e) {
      showRequestError("加载镜像列表失败: " + (e.message || "网络错误"), e);
    } finally {
      loadBusy.value = false;
    }
  });
}

async function confirmAction() {
  const modal = actionModal.value;
  if (!modal) return;
  if (modal.kind !== "approve" && !actionNote.value.trim()) {
    actionError.value = "请填写原因";
    actionErrorTraceId.value = "";
    return;
  }
  const out = await writeGuard.run(async ({ headers }) => {
    writeBusy.value = true;
    actionError.value = "";
    actionErrorTraceId.value = "";
    try {
      const id = modal.image.id;
      if (modal.kind === "approve") {
        await api(`/api/admin/container-images/${id}/approve/`, {
          method: "POST",
          headers,
          body: "{}",
        });
      } else if (modal.kind === "reject") {
        await api(`/api/admin/container-images/${id}/reject/`, {
          method: "POST",
          headers,
          body: JSON.stringify({ note: actionNote.value }),
        });
      } else {
        await api(`/api/admin/container-images/${id}/unpublish/`, {
          method: "POST",
          headers,
          body: JSON.stringify({ note: actionNote.value }),
        });
      }
      actionModal.value = null;
      detailImage.value = null;
      await load();
    } catch (e) {
      const label =
        modal.kind === "approve" ? "审核通过失败" : modal.kind === "reject" ? "驳回失败" : "撤销上架失败";
      actionError.value = `${label}: ${e.message || "网络错误"}`;
      actionErrorTraceId.value = e.traceId || "";
      showRequestError(actionError.value, e);
    } finally {
      writeBusy.value = false;
    }
  });
  return out;
}

onMounted(load);
defineExpose({ load });
</script>

<style scoped src="../views/AdminPortal.css"></style>

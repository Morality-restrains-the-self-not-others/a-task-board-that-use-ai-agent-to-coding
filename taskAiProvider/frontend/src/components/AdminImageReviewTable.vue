<script setup>
import { getPlatformDisplayName } from "../utils/platformLabel.js";
import { imageGroupIconSrc } from "../utils/imageGroupIcon.js";
import {
  formatReviewDate,
  hasAction,
  rejectHistories,
} from "../lib/adminImageReviewActions.js";

defineProps({
  title: { type: String, default: "" },
  images: { type: Array, required: true },
  writeBusy: { type: Boolean, default: false },
});

const emit = defineEmits(["detail", "approve", "reject", "unpublish"]);

function badgeClass(st) {
  return {
    badge: true,
    "badge-draft": st === "draft",
    "badge-pending": st === "pending_review",
    "badge-ok": st === "approved",
    "badge-reject": st === "rejected",
  };
}
</script>

<template>
  <div class="platform-group">
    <h3 v-if="title" class="platform-title">{{ title }}</h3>
    <table>
      <thead>
        <tr>
          <th>ID</th>
          <th>镜像组</th>
          <th>描述</th>
          <th>版本</th>
          <th>接口版本</th>
          <th>厂商</th>
          <th>运行环境</th>
          <th>状态</th>
          <th>历史驳回原因</th>
          <th>操作</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="im in images" :key="im.id">
          <td>{{ im.id }}</td>
          <td class="name-cell">
            <img
              v-if="imageGroupIconSrc(im)"
              class="group-icon-sm"
              :src="imageGroupIconSrc(im)"
              :alt="im.name + ' 图标'"
              width="36"
              height="36"
            />
            <span v-else class="group-icon-sm group-icon-placeholder" aria-hidden="true"></span>
            {{ im.name }}
          </td>
          <td class="desc-cell">{{ im.description || "-" }}</td>
          <td class="version-cell">{{ im.version }}</td>
          <td class="version-cell" data-testid="admin-saas-inbound-skill-version">{{ im.saas_inbound_skill_version || "—" }}</td>
          <td>{{ im.vendor_company || "—" }}</td>
          <td class="runtime-cell">
            <span v-if="im.runtime_environments && im.runtime_environments.length > 0" class="runtime-list">
              <span v-for="(rt, idx) in im.runtime_environments" :key="idx" class="runtime-item">
                {{ getPlatformDisplayName(rt.platform_type) }}/{{ rt.region }}
              </span>
            </span>
            <span v-else class="muted">未设置</span>
          </td>
          <td><span :class="badgeClass(im.status)">{{ im.status_display }}</span></td>
          <td class="history-cell">
            <div v-if="rejectHistories(im.review_histories).length > 0" class="history-list">
              <div v-for="h in rejectHistories(im.review_histories)" :key="h.id" class="history-item">
                <span class="history-date">{{ formatReviewDate(h.reviewed_at) }}</span>
                <span class="history-note">{{ h.note }}</span>
                <span v-if="h.reviewer_name" class="history-reviewer">（{{ h.reviewer_name }}）</span>
              </div>
            </div>
            <span v-else class="muted">无</span>
          </td>
          <td class="actions" data-testid="admin-image-review-actions">
            <!-- Anti-Replay-OK: ui-only -->
            <button
              v-if="hasAction(im.status, 'detail')"
              class="btn btn-ghost sm"
              type="button"
              data-testid="admin-review-detail"
              @click="emit('detail', im)"
            >查看详情</button>
            <button
              v-if="hasAction(im.status, 'approve')"
              class="btn btn-primary sm"
              type="button"
              :disabled="writeBusy"
              :aria-busy="writeBusy ? 'true' : 'false'"
              @click="emit('approve', im)"
            >通过</button>
            <button
              v-if="hasAction(im.status, 'reject')"
              class="btn btn-danger sm"
              type="button"
              :disabled="writeBusy"
              :aria-busy="writeBusy ? 'true' : 'false'"
              @click="emit('reject', im)"
            >驳回</button>
            <button
              v-if="hasAction(im.status, 'unpublish')"
              class="btn btn-warning sm"
              type="button"
              :disabled="writeBusy"
              :aria-busy="writeBusy ? 'true' : 'false'"
              @click="emit('unpublish', im)"
            >撤销上架</button>
            <span
              v-if="im.group_active_version"
              class="group-active-hint"
              :title="'激活由厂商在版本列表切换'"
            >组内激活：{{ im.group_active_version.version }}（审批不切换激活）</span>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<style scoped src="../views/AdminPortal.css"></style>

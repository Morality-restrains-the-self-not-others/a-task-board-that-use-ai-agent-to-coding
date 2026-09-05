<script setup>
import { computed } from "vue";
import ImageResolveInfoPanel from "./ImageResolveInfoPanel.vue";
import { getPlatformDisplayName } from "../utils/platformLabel.js";
import {
  formatArchList,
  formatReviewDate,
  rejectHistories,
  resolvePanelProps,
  reviewHistoryActionLabel,
  userdataTemplateLabel,
} from "../lib/adminImageReviewActions.js";

const props = defineProps({
  image: { type: Object, required: true },
});

const emit = defineEmits(["close"]);

const resolveProps = computed(() => resolvePanelProps(props.image));
const rejectRows = computed(() => rejectHistories(props.image.review_histories));
const allHistory = computed(() =>
  Array.isArray(props.image.review_histories) ? props.image.review_histories : [],
);
const runtimes = computed(() =>
  Array.isArray(props.image.runtime_environments) ? props.image.runtime_environments : [],
);
</script>

<template>
  <div class="modal-bg" @click.self="emit('close')">
    <div class="card modal modal-wide" role="dialog" aria-modal="true" data-testid="admin-review-detail-modal">
      <h3>镜像审核详情</h3>
      <dl class="detail-grid">
        <dt>镜像组</dt>
        <dd>{{ image.name || "—" }}</dd>
        <dt>版本</dt>
        <dd>{{ image.version || "—" }}</dd>
        <dt>厂商</dt>
        <dd>{{ image.vendor_company || "—" }}</dd>
        <dt>状态</dt>
        <dd>{{ image.status_display || image.status || "—" }}</dd>
        <dt>接口版本</dt>
        <dd>{{ image.saas_inbound_skill_version || "—" }}</dd>
        <dt>架构</dt>
        <dd>{{ formatArchList(image.target_architectures) }}</dd>
        <dt>镜像地址</dt>
        <dd class="mono" data-testid="admin-review-image-url">{{ image.image_url || "—" }}</dd>
        <dt>描述</dt>
        <dd>{{ image.description || "—" }}</dd>
      </dl>

      <ImageResolveInfoPanel
        :skills="resolveProps.skills"
        :skills-status="resolveProps.skillsStatus"
        :skills-detail="resolveProps.skillsDetail"
        :auto-run-steps-md="resolveProps.autoRunStepsMd"
        :auto-run-steps-status="resolveProps.autoRunStepsStatus"
        :auto-run-steps-detail="resolveProps.autoRunStepsDetail"
      />

      <h4 class="h4">运行环境</h4>
      <div v-if="runtimes.length === 0" class="muted">未设置</div>
      <table v-else class="runtime-table">
        <thead>
          <tr>
            <th>云平台</th>
            <th>地域</th>
            <th>OS</th>
            <th>服务器镜像</th>
            <th>UserData</th>
            <th>规格</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(rt, idx) in runtimes" :key="idx">
            <td>{{ rt.platform_type_display || getPlatformDisplayName(rt.platform_type) }}</td>
            <td>{{ rt.region || "—" }}</td>
            <td>{{ [rt.os_type, rt.os_version].filter(Boolean).join(" ") || "—" }}</td>
            <td>{{ rt.image_name || rt.image_id || "—" }}</td>
            <td>{{ userdataTemplateLabel(rt) }}</td>
            <td>{{ rt.hardware_summary || "—" }}</td>
          </tr>
        </tbody>
      </table>

      <h4 class="h4">审核历史</h4>
      <div v-if="allHistory.length === 0" class="muted">无</div>
      <ul v-else class="history-list">
        <li v-for="h in allHistory" :key="h.id" class="history-item">
          <span class="history-date">{{ formatReviewDate(h.reviewed_at) }}</span>
          <span class="history-action">{{ reviewHistoryActionLabel(h.action) }}</span>
          <span class="history-note">{{ h.note || "—" }}</span>
          <span v-if="h.reviewer_name" class="history-reviewer">（{{ h.reviewer_name }}）</span>
        </li>
      </ul>
      <p v-if="rejectRows.length" class="muted reject-count">驳回记录 {{ rejectRows.length }} 条</p>

      <div class="row">
        <button class="btn btn-ghost" type="button" @click="emit('close')">关闭</button>
      </div>
    </div>
  </div>
</template>

<style scoped src="../views/AdminPortal.css"></style>

<style scoped>
.modal-wide {
  max-width: 760px;
  max-height: 90vh;
  overflow: auto;
}
.detail-grid {
  display: grid;
  grid-template-columns: 7rem 1fr;
  gap: 0.35rem 0.75rem;
  margin: 0 0 1rem;
}
.detail-grid dt {
  color: var(--muted);
  font-size: 0.8rem;
}
.detail-grid dd {
  margin: 0;
  word-break: break-all;
}
.mono {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 0.8rem;
}
.h4 {
  font-size: 0.95rem;
  margin: 1rem 0 0.4rem;
}
.runtime-table {
  font-size: 0.8rem;
}
.history-action {
  font-weight: 600;
  margin: 0 0.35rem;
}
.reject-count {
  margin-top: 0.35rem;
}
</style>

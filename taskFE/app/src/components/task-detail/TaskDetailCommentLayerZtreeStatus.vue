<template>
  <div
    v-if="showLoading"
    class="mb-4 p-3 rounded-lg border border-dashed border-gray-200 bg-gray-50/80"
    data-testid="comment-layer-ztree-loading"
  >
    <div class="flex flex-wrap items-center gap-2">
      <p class="text-xs font-medium text-gray-600">任务关联（可写层串行 · 容器推送）</p>
      <OpenContainerActionLinks
        :container-page-url="containerPageUrl"
        :display-container-vscode-url="displayContainerVscodeUrl"
        :tenant-id="tenantId"
        :workspace-id="workspaceId"
        :task-id="taskId"
        :pending-reveal="containerPageLinkPendingReveal"
        :http-unreachable="containerHttpUnreachable"
      />
    </div>
    <TaskDetailCommentLayerZtreeErrorBanner
      v-if="loadingIsError"
      :hint="loadingHint"
      :trace-id="loadingErrorTraceId"
    />
    <p
      v-else
      class="mt-2 text-xs flex items-start gap-2 text-gray-500"
    >
      <span
        class="inline-block h-3 w-3 shrink-0 animate-spin rounded-full border-2 border-gray-300 border-t-primary mt-0.5"
      ></span>
      <span>{{ loadingHint }}</span>
    </p>
  </div>
  <div
    v-else-if="showReleased"
    class="mb-4 p-3 rounded-lg border border-dashed border-slate-200 bg-slate-50/80"
    data-testid="comment-layer-ztree-released"
  >
    <p class="text-xs font-medium text-gray-600">任务关联（可写层串行 · 容器推送）</p>
    <p class="mt-2 text-xs font-medium text-slate-700">{{ releasedTitle }}</p>
    <p class="mt-1 text-xs text-slate-500">{{ releasedBody }}</p>
  </div>
</template>

<script setup>
import OpenContainerActionLinks from './OpenContainerActionLinks.vue'
import TaskDetailCommentLayerZtreeErrorBanner from './TaskDetailCommentLayerZtreeErrorBanner.vue'

defineProps({
  showLoading: { type: Boolean, default: false },
  showReleased: { type: Boolean, default: false },
  loadingHint: { type: String, default: '' },
  loadingIsError: { type: Boolean, default: false },
  loadingErrorTraceId: { type: String, default: '' },
  releasedTitle: { type: String, default: '' },
  releasedBody: { type: String, default: '' },
  tenantId: { type: String, default: '' },
  workspaceId: { type: String, default: '' },
  taskId: { type: String, default: '' },
  containerPageUrl: { type: String, default: '' },
  containerPageLinkPendingReveal: { type: Boolean, default: false },
  containerHttpUnreachable: { type: Boolean, default: false },
  displayContainerVscodeUrl: { type: String, default: '' },
})
</script>

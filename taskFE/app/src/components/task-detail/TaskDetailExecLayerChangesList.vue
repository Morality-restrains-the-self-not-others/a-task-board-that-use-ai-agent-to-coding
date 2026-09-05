<template>
  <div
    v-if="changes.length"
    ref="scrollRoot"
    class="max-h-72 overflow-auto space-y-3 rounded border border-gray-200 bg-white px-2 py-2"
    data-testid="task-detail-layer-changes-list"
    @scroll.passive="onScroll"
  >
    <div v-for="section in sections" v-show="section.items.length" :key="section.key">
      <p class="text-[11px] font-semibold text-gray-600 mb-1">{{ section.label }}</p>
      <ul class="space-y-1">
        <TaskDetailExecLayerChangesListItem
          v-for="(change, idx) in section.items"
          :key="`${section.key}-${change.path}-${idx}`"
          :change="change"
          :is-selected="selectedPath === change.path"
          :layer-id="layerId"
          :tenant-id="tenantId"
          :workspace-id="workspaceId"
          :task-id="taskId"
          :git-actions-disabled="gitActionsDisabled"
          :container-page-url="containerPageUrl"
          :comment-id="commentId"
          @select="onSelectPath"
          @preview-loading="onPreviewLoading"
          @preview-loaded="onPreviewLoaded"
          @preview-error="onPreviewError"
          @staged-success="onStagedSuccess"
        />
      </ul>
    </div>
    <div
      v-if="hasMore"
      ref="sentinel"
      class="h-1 w-full"
      data-testid="task-detail-layer-changes-load-more-sentinel"
      aria-hidden="true"
    />
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'
import TaskDetailExecLayerChangesListItem from './TaskDetailExecLayerChangesListItem.vue'

const props = defineProps({
  changes: {
    type: Array,
    default: () => [],
  },
  selectedPath: { type: String, default: '' },
  layerId: { type: String, default: '' },
  tenantId: { type: String, default: '' },
  workspaceId: { type: String, default: '' },
  taskId: { type: String, default: '' },
  gitActionsDisabled: { type: Boolean, default: false },
  containerPageUrl: { type: String, default: '' },
  commentId: { type: String, default: '' },
  hasMore: { type: Boolean, default: false },
  loadMoreBusy: { type: Boolean, default: false },
})

const emit = defineEmits([
  'select',
  'preview-loading',
  'preview-loaded',
  'preview-error',
  'staged-success',
  'load-more',
])

const scrollRoot = ref(null)

function maybeLoadMore() {
  if (!props.hasMore || props.loadMoreBusy) return
  const el = scrollRoot.value
  if (!el) return
  const remain = el.scrollHeight - el.scrollTop - el.clientHeight
  if (remain <= 48) {
    emit('load-more')
  }
}

function onScroll() {
  maybeLoadMore()
}

function onStagedSuccess(payload) {
  emit('staged-success', payload)
}

const sections = computed(() => [
  {
    key: 'staged',
    label: '暂存区（已加入索引）',
    items: props.changes.filter((c) => c.git_staged),
  },
  {
    key: 'unstaged',
    label: '未暂存（工作区）',
    items: props.changes.filter((c) => c.git_unstaged),
  },
  {
    key: 'layer_only',
    label: '相对父层（未出现在当前 Git 暂存/工作区）',
    items: props.changes.filter((c) => c.git_layer_diff_only),
  },
])

function onSelectPath(path) {
  if (!path || !props.changes.length) return
  emit('select', path)
}

function onPreviewLoading(payload) {
  emit('preview-loading', payload)
}

function onPreviewLoaded(payload) {
  emit('preview-loaded', payload)
}

function onPreviewError(payload) {
  emit('preview-error', payload)
}
</script>

<template>
  <div
    v-if="summary && summary.total > 0"
    class="p-3 bg-white border border-gray-200 rounded-lg space-y-2"
    data-testid="task-subtree-status"
  >
    <div class="flex items-center justify-between gap-2">
      <h3 class="text-sm font-medium text-gray-500">下级交付物</h3>
      <p class="text-sm text-gray-700" data-testid="task-subtree-summary">
        已关闭 {{ summary.settled }}/{{ summary.total }}
      </p>
    </div>
    <p v-if="loading" class="text-sm text-gray-500">加载中…</p>
    <p
      v-else-if="error"
      class="text-sm text-red-600"
      data-testid="task-subtree-error"
      v-bind="errorTraceId ? { 'data-traceId': errorTraceId } : {}"
    >{{ error }}</p>
    <ul v-else class="space-y-1.5" data-testid="task-subtree-list">
      <li
        v-for="node in nodes"
        :key="node.id"
        class="flex items-center justify-between gap-2 text-sm"
        :class="node.depth >= 2 ? 'pl-4' : ''"
        :data-testid="`task-subtree-node-${node.id}`"
      >
        <router-link
          v-if="nodeRoute(node)"
          :to="nodeRoute(node)"
          class="text-primary hover:underline truncate"
        >
          {{ node.title || node.id }}
        </router-link>
        <span v-else class="truncate text-gray-800">{{ node.title || node.id }}</span>
        <span
          class="shrink-0 px-2 py-0.5 rounded text-xs"
          :class="node.settled ? 'bg-green-50 text-green-700' : 'bg-amber-50 text-amber-800'"
        >
          {{ node.progress_column_name || (node.completed ? '已完成' : '未设置') }}
        </span>
      </li>
    </ul>
  </div>
</template>

<script setup>
const props = defineProps({
  summary: { type: Object, default: null },
  nodes: { type: Array, default: () => [] },
  loading: { type: Boolean, default: false },
  error: { type: String, default: '' },
  errorTraceId: { type: String, default: '' },
  tenantId: { type: String, default: '' },
  workspaceId: { type: String, default: '' },
})

function nodeRoute(node) {
  const tenantId = String(props.tenantId || '').trim()
  const workspaceId = String(props.workspaceId || '').trim()
  const taskId = String(node?.id || '').trim()
  if (!tenantId || !workspaceId || !taskId) return null
  return {
    name: 'task_detail',
    params: { tenant: tenantId, workspaceId, taskId },
  }
}
</script>

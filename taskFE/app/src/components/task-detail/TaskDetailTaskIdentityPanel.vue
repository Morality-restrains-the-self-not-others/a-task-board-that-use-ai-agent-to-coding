<template>
  <div class="space-y-4">
    <TaskDetailTaskAuxInfoPanel
      v-bind="$props"
      @progress-status-change="emit('progress-status-change', $event)"
      @task-updated="emit('task-updated', $event)"
    />

    <div class="p-3 bg-white border border-gray-200 rounded-lg">
      <h3 class="text-sm font-medium text-gray-500 mb-1">任务标题</h3>
      <div v-if="!isEditing">
        <p class="text-lg font-bold text-gray-900">{{ task.title }}</p>
      </div>
      <div v-else-if="editingTask">
        <input
          v-model="editingTask.title"
          type="text"
          class="w-full text-lg font-bold text-gray-900 border border-gray-300 rounded-md px-3 py-1.5 focus:outline-none focus:ring-primary focus:border-primary"
          placeholder="请输入任务标题"
        />
      </div>
    </div>

    <div class="p-3 bg-white border border-gray-200 rounded-lg">
      <h3 class="text-sm font-medium text-gray-500 mb-2">任务描述</h3>
      <div v-if="!isEditing">
        <MarkdownContent :content="task.description" />
      </div>
      <div v-else-if="editingTask">
        <textarea
          v-model="editingTask.description"
          rows="4"
          class="w-full text-gray-700 border border-gray-300 rounded-md px-3 py-2 resize-y focus:outline-none focus:ring-primary focus:border-primary"
          placeholder="请输入任务描述"
        />
      </div>
    </div>

    <TaskDetailLlmBudgetPanel
      v-if="budgetRouteIds.tenantId && budgetRouteIds.workspaceId && budgetRouteIds.taskId"
      :tenant-id="budgetRouteIds.tenantId"
      :workspace-id="budgetRouteIds.workspaceId"
      :task-id="budgetRouteIds.taskId"
      :can-edit="true"
      :can-raise="true"
    />
    <TaskDetailSubtreeStatusPanel
      :summary="subtreeSummary"
      :nodes="subtreeNodes"
      :loading="subtreeLoading"
      :error="subtreeError"
      :error-trace-id="subtreeErrorTraceId"
      :tenant-id="budgetRouteIds.tenantId"
      :workspace-id="budgetRouteIds.workspaceId"
    />
  </div>
</template>

<script setup>
import { computed, inject, unref } from 'vue'
import { useRoute } from 'vue-router'
import MarkdownContent from '../MarkdownContent.ui.vue'
import TaskDetailLlmBudgetPanel from './TaskDetailLlmBudgetPanel.vue'
import TaskDetailSubtreeStatusPanel from './TaskDetailSubtreeStatusPanel.vue'
import TaskDetailTaskAuxInfoPanel from './TaskDetailTaskAuxInfoPanel.vue'
import { resolveTaskRouteIds } from '../../utils/resolveTaskRouteIds.js'

const props = defineProps({
  task: { type: Object, required: true },
  isEditing: { type: Boolean, default: false },
  editingTask: { type: Object, default: null },
  collaboratorNameById: { type: Object, default: () => ({}) },
  collaboratorOptions: { type: Array, default: () => [] },
  forkSourceTaskRoute: { type: [Object, String], required: true },
  forkSourceTitle: { type: String, default: '' },
  forkSourceSeq: { type: Number, default: 0 },
  isForkSourceTitleLoading: { type: Boolean, default: false },
  parentDeliverableRoute: { type: [Object, String], default: undefined },
  parentTaskTitle: { type: String, default: undefined },
  parentTaskSeq: { type: Number, default: undefined },
  isParentTaskTitleLoading: { type: Boolean, default: undefined },
  progressStatusOptions: { type: Array, default: () => [] },
  isProgressStatusesLoading: { type: Boolean, default: false },
  isUpdatingProgressStatus: { type: Boolean, default: false },
  progressStatusError: { type: String, default: '' },
  progressStatusErrorTraceId: { type: String, default: '' },
  deliverableCategoryOptions: { type: Array, default: undefined },
  isDeliverableCategoriesLoading: { type: Boolean, default: undefined },
  deliverableCategoryError: { type: String, default: undefined },
  workspaceTodos: { type: Array, default: undefined },
  resolvedDeliverableCategoryErrorTraceId: { type: String, default: '' },
  containerEndpointRegistered: { type: Boolean, default: false },
  /** idle|connecting|connected|disconnected；用于推迟 live auto-run-steps，避免 exchange 后空 token 409 */
  containerHeartbeatStatus: { type: String, default: '' },
  tenantId: { type: String, default: '' },
  workspaceId: { type: String, default: '' },
  taskId: { type: String, default: '' },
  commentId: { type: String, default: '' },
})

const emit = defineEmits(['progress-status-change', 'task-updated'])

const injectedSubtree = inject('taskDetailSubtree', null)
const subtreeSummary = computed(() => unref(injectedSubtree?.summary) || null)
const subtreeNodes = computed(() => {
  const nodes = unref(injectedSubtree?.nodes)
  return Array.isArray(nodes) ? nodes : []
})
const subtreeLoading = computed(() => Boolean(unref(injectedSubtree?.loading)))
const subtreeError = computed(() => unref(injectedSubtree?.error) || '')
const subtreeErrorTraceId = computed(() => unref(injectedSubtree?.errorTraceId) || '')

const route = useRoute()
const budgetRouteIds = computed(() => resolveTaskRouteIds({
  tenantId: props.tenantId,
  workspaceId: props.workspaceId,
  taskId: props.taskId,
  task: props.task,
  routeParams: route.params,
}))
</script>

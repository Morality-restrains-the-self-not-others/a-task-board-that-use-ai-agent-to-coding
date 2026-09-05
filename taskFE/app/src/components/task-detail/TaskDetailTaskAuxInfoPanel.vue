<template>
  <div class="p-4 bg-gray-50 rounded-lg" data-testid="task-aux-info-panel">
    <div class="flex items-center justify-between gap-2">
      <h3 class="text-sm font-semibold text-gray-700">任务辅助信息</h3>
      <button
        type="button"
        class="inline-flex items-center gap-1 px-2 py-1 border border-gray-300 rounded-md text-xs text-gray-600 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-1 focus:ring-primary"
        data-testid="task-aux-info-toggle"
        :aria-expanded="auxInfoExpanded ? 'true' : 'false'"
        :aria-label="auxInfoExpanded ? '收起任务辅助信息' : '展开任务辅助信息'"
        @click="auxInfoExpanded = !auxInfoExpanded"
      >
        <span>{{ auxInfoExpanded ? '收起' : '展开' }}</span>
        <svg
          class="w-3.5 h-3.5 transition-transform duration-200"
          :class="{ 'rotate-180': auxInfoExpanded }"
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
          stroke-width="2"
          aria-hidden="true"
        >
          <path stroke-linecap="round" stroke-linejoin="round" d="M19 9l-7 7-7-7" />
        </svg>
      </button>
    </div>

    <div v-show="auxInfoExpanded" class="mt-4 space-y-4" data-testid="task-aux-info-body">
      <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
      <div>
        <h3 class="text-sm font-medium text-gray-500 mb-1">任务编号</h3>
        <div data-testid="task-aux-info-display-no" class="min-h-[1.5rem]">
          <TaskCardIdBadge
            v-if="taskDisplayNo"
            :task-id="task.id"
            :workspace-seq="task.workspace_seq ?? task.workspaceSeq"
            size-class="text-lg font-semibold text-gray-900"
          />
          <span v-else class="text-lg font-semibold text-gray-900 font-mono tabular-nums">—</span>
        </div>
        <div class="flex items-center gap-2 min-w-0 mt-1">
          <p
            class="text-xs font-mono text-gray-600 truncate"
            data-testid="task-aux-info-task-id"
            :title="String(task.id || '')"
          >{{ task.id }}</p>
          <button
            type="button"
            class="shrink-0 px-1.5 py-0.5 border border-gray-300 rounded text-[11px] text-gray-600 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-1 focus:ring-primary"
            data-testid="task-aux-info-task-id-copy"
            :aria-label="taskIdCopied ? '已复制任务 ID' : '复制任务 ID'"
            @click="copyTaskId"
          >{{ taskIdCopied ? '已复制' : '复制' }}</button>
        </div>
      </div>
      <div>
        <h3 class="text-sm font-medium text-gray-500 mb-1">创建时间</h3>
        <p class="text-lg font-semibold text-gray-900">{{ formatDateTime(task.created_at) }}</p>
      </div>
      <!-- v15 存续期：创建帖 12 个月有效期 + 续存按钮 -->
      <div data-testid="task-post-validity">
        <h3 class="text-sm font-medium text-gray-500 mb-1">存续期</h3>
        <div v-if="task.post_expires_at" class="flex items-center gap-2 flex-wrap">
          <span
            class="inline-block px-3 py-1 rounded-full text-sm font-medium"
            :class="task.post_expired ? 'bg-red-100 text-red-700' : 'bg-green-100 text-green-700'"
            data-testid="task-post-expiry-badge"
          >{{ task.post_expired ? '已到期' : formatPostExpiry(task.post_expires_at) }}</span>
          <button
            v-if="!isEditing"
            type="button"
            class="px-3 py-1 rounded-md text-sm font-medium bg-primary text-white hover:bg-primary/90 disabled:opacity-50"
            :disabled="renewing"
            data-testid="task-post-renew-button"
            @click="renewPost"
          >{{ renewing ? '续存中…' : '续存（+12个月）' }}</button>
        </div>
        <p v-else class="text-lg font-semibold text-gray-900">—</p>
        <p
          v-if="renewError"
          class="text-xs text-red-600 mt-1"
          v-bind="renewErrorTraceId ? { 'data-traceId': renewErrorTraceId } : {}"
        >{{ renewError }}</p>
      </div>
      <div>
        <h3 class="text-sm font-medium text-gray-500 mb-1">优先级</h3>
        <div v-if="!isEditing">
          <span
            class="inline-block px-3 py-1 rounded-full text-sm font-medium"
            :class="priorityClass"
          >
            {{ priorityText }}
          </span>
        </div>
        <div v-else-if="editingTask">
          <select
            v-model="editingTask.priority"
            class="px-3 py-1 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-primary focus:border-primary"
          >
            <option :value="0">高优先级</option>
            <option :value="1">中优先级</option>
            <option :value="2">低优先级</option>
          </select>
        </div>
      </div>
      <div data-testid="task-deliverable-category">
        <h3 class="text-sm font-medium text-gray-500 mb-1">交付物类别</h3>
        <p v-if="!isEditing" class="text-lg font-semibold text-gray-900">{{ deliverableCategoryDisplayText }}</p>
        <div v-else-if="editingTask">
          <select
            v-model="editingTask.deliverable_obj_id"
            class="w-full px-3 py-1 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-primary focus:border-primary"
            :disabled="resolvedIsDeliverableCategoriesLoading"
          >
            <option value="">未分类</option>
            <option
              v-for="cat in resolvedDeliverableCategoryOptions"
              :key="cat.id"
              :value="String(cat.id)"
            >
              {{ cat.name }}
            </option>
          </select>
          <p
            v-if="resolvedDeliverableCategoryError"
            class="text-xs text-red-600 mt-1"
            v-bind="resolvedDeliverableCategoryErrorTraceId ? { 'data-traceId': resolvedDeliverableCategoryErrorTraceId } : {}"
          >{{ resolvedDeliverableCategoryError }}</p>
        </div>
      </div>
      <div v-if="isEditing && editingTask" class="md:col-span-1">
        <CreateTaskParentDeliverableField
          v-model="editingTask.parent_task"
          :task-types="resolvedDeliverableCategoryOptions"
          :todos="resolvedWorkspaceTodos"
          :selected-category-id="editingTask.deliverable_obj_id"
          :exclude-task-id="task?.id"
        />
      </div>
      <div v-else-if="parentDeliverableId" data-testid="task-parent-deliverable">
        <h3 class="text-sm font-medium text-gray-500 mb-1">上层交付物</h3>
        <p v-if="resolvedIsParentTaskTitleLoading" class="text-sm text-gray-500">加载中…</p>
        <router-link
          v-else
          :to="resolvedParentDeliverableRoute"
          class="text-sm font-medium text-primary hover:underline"
          data-testid="task-parent-deliverable-link"
        >
          {{ parentDeliverableDisplayText }}
        </router-link>
      </div>
      <div>
        <h3 class="text-sm font-medium text-gray-500 mb-1">操作员</h3>
        <p v-if="!isEditing" class="text-gray-700">{{ operatorDisplayText }}</p>
        <div v-else-if="editingTask">
          <select
            v-model="editingTask.operator"
            class="w-full px-2 py-1 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-primary focus:border-primary"
          >
            <option value="">未指派</option>
            <option
              v-for="option in collaboratorOptions"
              :key="`op-${option.id}`"
              :value="option.id"
            >
              {{ option.name }}
            </option>
          </select>
        </div>
      </div>
      <div>
        <h3 class="text-sm font-medium text-gray-500 mb-1">负责人</h3>
        <p v-if="!isEditing" class="text-gray-700">{{ ownerDisplayText }}</p>
        <div v-else-if="editingTask">
          <select
            v-model="editingTask.owner"
            class="w-full px-2 py-1 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-primary focus:border-primary"
          >
            <option
              v-for="option in collaboratorOptions"
              :key="option.id"
              :value="option.id"
            >
              {{ option.name }}
            </option>
          </select>
        </div>
      </div>
      <div>
        <h3 class="text-sm font-medium text-gray-500 mb-1">协作者</h3>
        <p
          v-if="!isEditing"
          class="text-gray-700"
          :title="assigneesDisplayTitle"
        >{{ assigneesDisplayText }}</p>
        <div v-else-if="editingTask" class="w-full">
          <TagMemberInput
            :model-value="editingTask.assignees"
            :options="collaboratorOptions"
            placeholder="搜索添加协作者..."
            @update:model-value="onAssigneesChange"
          />
        </div>
      </div>
      <div>
        <h3 class="text-sm font-medium text-gray-500 mb-1">进度状态</h3>
        <div class="flex items-center gap-2">
          <select
            id="task-progress-status"
            class="w-full max-w-xs px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-primary focus:border-primary disabled:bg-gray-100 disabled:text-gray-500"
            :value="selectedProgressColumnId"
            :disabled="isProgressStatusesLoading || isUpdatingProgressStatus || progressStatusOptions.length === 0"
            @change="onProgressSelectChange"
          >
            <option value="" disabled>
              {{ isProgressStatusesLoading ? '加载中...' : '请选择进度状态' }}
            </option>
            <option
              v-for="status in progressStatusOptions"
              :key="status.id"
              :value="status.id"
            >
              {{ status.name }}
            </option>
          </select>
          <span v-if="isUpdatingProgressStatus" class="text-xs text-gray-500 whitespace-nowrap">更新中...</span>
        </div>
        <p
          v-if="progressStatusError"
          class="text-xs text-red-600 mt-1"
          v-bind="progressStatusErrorTraceId ? { 'data-traceId': progressStatusErrorTraceId } : {}"
        >{{ progressStatusError }}</p>
      </div>
      <div data-testid="task-auto-run-readonly">
        <h3 class="text-sm font-medium text-gray-500 mb-1">是否自动运行</h3>
        <p class="text-lg font-semibold text-gray-900">{{ autoRunDisplayText }}</p>
        <div
          v-if="task?.auto_run === true"
          class="mt-2"
          data-testid="task-detail-auto-run-steps"
        >
          <AutoRunStepsPreview
            :markdown="cachedAutoRunStepsMd"
            :extract-status="cachedAutoRunStepsStatus"
            :live-markdown="liveAutoRunStepsMd"
            :live-error="liveAutoRunStepsError"
            :default-expanded="false"
          />
        </div>
      </div>
      <div v-if="task.fork_from" class="md:col-span-3" data-testid="task-fork-from">
        <h3 class="text-sm font-medium text-gray-500 mb-1">派生自（fork_from）</h3>
        <p v-if="resolvedIsForkSourceTitleLoading" class="text-sm text-gray-500">加载中…</p>
        <router-link
          v-else
          :to="forkSourceTaskRoute"
          class="text-sm font-medium text-primary hover:underline"
          data-testid="task-fork-from-link"
        >
          {{ forkSourceDisplayText }}
        </router-link>
      </div>
    </div>
    </div>
  </div>
</template>

<script setup>
import { computed, inject, toRef, unref } from 'vue'
import TagMemberInput from '../TagMemberInput.vue'
import AutoRunStepsPreview from '../AutoRunStepsPreview.vue'
import CreateTaskParentDeliverableField from '../CreateTaskParentDeliverableField.vue'
import TaskCardIdBadge from '../TaskCardIdBadge.vue'
import {
  resolveDeliverableCategoryDisplayName,
  resolveTodoParentTaskId,
} from '../../utils/workPanelDeliverableAggregation.js'
import { useTaskIdentityAutoRunSteps } from '../../composables/taskDetail/useTaskIdentityAutoRunSteps.js'
import { useTaskIdentityPanelDisplay } from '../../composables/taskDetail/useTaskIdentityPanelDisplay.js'
import { useTaskPostRenew } from '../../composables/taskDetail/useTaskPostRenew.js'
import { useTaskAuxInfoExpanded } from '../../composables/useTaskAuxInfoExpanded.js'

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
const injectedParent = inject('taskDetailParentDeliverable', null)
const injectedCategory = inject('taskDetailDeliverableCategory', null)
const injectedWorkspaceTodos = inject('taskDetailWorkspaceTodos', null)

const resolvedParentDeliverableRoute = computed(() => {
  if (props.parentDeliverableRoute !== undefined) return props.parentDeliverableRoute
  return unref(injectedParent?.parentDeliverableRoute) || { path: '/' }
})
const resolvedParentTaskTitle = computed(() => {
  if (props.parentTaskTitle !== undefined) return props.parentTaskTitle
  return unref(injectedParent?.parentTaskTitle) || ''
})
const resolvedParentTaskSeq = computed(() => {
  if (props.parentTaskSeq !== undefined) return props.parentTaskSeq
  return Number(unref(injectedParent?.parentTaskSeq) || 0)
})
const resolvedIsParentTaskTitleLoading = computed(() => {
  if (props.isParentTaskTitleLoading !== undefined) return props.isParentTaskTitleLoading
  return Boolean(unref(injectedParent?.isParentTaskTitleLoading))
})
const resolvedDeliverableCategoryOptions = computed(() => {
  if (props.deliverableCategoryOptions !== undefined) return props.deliverableCategoryOptions
  return unref(injectedCategory?.deliverableCategoryOptions) || []
})
const resolvedIsDeliverableCategoriesLoading = computed(() => {
  if (props.isDeliverableCategoriesLoading !== undefined) return props.isDeliverableCategoriesLoading
  return Boolean(unref(injectedCategory?.isDeliverableCategoriesLoading))
})
const resolvedDeliverableCategoryError = computed(() => {
  if (props.deliverableCategoryError !== undefined) return props.deliverableCategoryError
  return unref(injectedCategory?.deliverableCategoryError) || ''
})
const resolvedWorkspaceTodos = computed(() => {
  if (props.workspaceTodos !== undefined) return props.workspaceTodos
  const fromInject = unref(injectedWorkspaceTodos)
  return Array.isArray(fromInject) ? fromInject : []
})

const {
  renewing,
  renewError,
  renewErrorTraceId,
  renewPost,
  formatPostExpiry,
} = useTaskPostRenew({
  task: toRef(props, 'task'),
  tenantId: () => props.tenantId,
  workspaceId: () => props.workspaceId,
  taskId: () => props.taskId,
  onUpdated: (payload) => emit('task-updated', payload),
})

const { auxInfoExpanded } = useTaskAuxInfoExpanded()

const formatDateTime = (datetime) => {
  if (!datetime) return ''
  const date = new Date(datetime)
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  })
}

const priorityClass = computed(() => {
  switch (props.task.priority) {
    case 0:
      return 'bg-red-100 text-red-800'
    case 1:
      return 'bg-yellow-100 text-yellow-800'
    default:
      return 'bg-green-100 text-green-800'
  }
})

const priorityText = computed(() => {
  switch (props.task.priority) {
    case 0:
      return '高优先级'
    case 1:
      return '中优先级'
    default:
      return '低优先级'
  }
})

const selectedProgressColumnId = computed(() => {
  const progressColumnId = props.task?.progress_column_id
  if (progressColumnId == null) return ''
  return String(progressColumnId)
})

const autoRunDisplayText = computed(() => (props.task?.auto_run === true ? '是' : '否'))

const {
  cachedAutoRunStepsMd,
  cachedAutoRunStepsStatus,
  liveAutoRunStepsMd,
  liveAutoRunStepsError,
} = useTaskIdentityAutoRunSteps({
  task: toRef(props, 'task'),
  tenantId: () => props.tenantId,
  workspaceId: () => props.workspaceId,
  taskId: () => props.taskId,
  commentId: () => props.commentId,
  containerEndpointRegistered: () => props.containerEndpointRegistered,
  containerHeartbeatStatus: () => props.containerHeartbeatStatus,
})

const deliverableCategoryDisplayText = computed(() =>
  resolveDeliverableCategoryDisplayName(props.task, resolvedDeliverableCategoryOptions.value),
)

const parentDeliverableId = computed(() => resolveTodoParentTaskId(props.task))

const {
  taskIdCopied,
  copyTaskId,
  taskDisplayNo,
  assigneesDisplayText,
  assigneesDisplayTitle,
  ownerDisplayText,
  operatorDisplayText,
  parentDeliverableDisplayText,
  resolvedIsForkSourceTitleLoading,
  forkSourceDisplayText,
} = useTaskIdentityPanelDisplay(props, {
  resolvedParentTaskTitle,
  resolvedParentTaskSeq,
  parentDeliverableId,
  workspaceTodos: resolvedWorkspaceTodos,
})

const onProgressSelectChange = (event) => {
  emit('progress-status-change', event)
}

const onAssigneesChange = (nextIds) => {
  if (props.editingTask) {
    props.editingTask.assignees = nextIds
  }
}
</script>

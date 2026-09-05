<template>
  <div data-alias="cmp-task-detail-card" class="task-card" :class="`priority-${task.priority}`" :data-task-id="task.id" @click="emit('task-clicked', task)">
    <!-- 元信息行：编号 + 进度下拉 + 优先级 -->
    <div class="flex items-center gap-2 mb-2 min-w-0" data-testid="task-card-meta-row">
      <TaskCardIdBadge :task-id="task.id" :workspace-seq="task.workspace_seq" />
      <div
        v-if="showProgressStatusSelect"
        class="flex-1 min-w-0 task-card-no-drag"
        @click.stop
      >
        <select
          :id="'task-progress-' + task.id"
          class="task-progress-select w-full min-w-0 text-xs border border-gray-200 rounded-md px-2 py-1 bg-white text-text focus:outline-none focus:ring-1 focus:ring-primary cursor-pointer"
          aria-label="进度状态"
          :value="progressSelectValue"
          @click.stop
          @mousedown.stop
          @change="onProgressColumnChange"
        >
          <option
            v-for="col in taskStatuses"
            :key="col.id"
            :value="String(col.id)"
          >
            {{ col.name }}
          </option>
        </select>
      </div>
      <span
        class="text-xs px-2 py-1 rounded shrink-0"
        :class="priorityClass"
        data-testid="task-card-priority"
      >
        {{ priorityText }}
      </span>
      <!-- v15 存续期：帖子已到期标记 -->
      <span
        v-if="task.post_expired"
        class="text-xs px-2 py-1 rounded shrink-0 bg-red-100 text-red-700"
        data-testid="task-card-post-expired-badge"
        title="任务帖已到期，续存后恢复使用"
      >已到期</span>
    </div>

    <!-- 标题行（含运行态指示） -->
    <div class="flex items-start gap-2 mb-2 min-w-0">
      <h4 class="font-medium text-text min-w-0 flex-1" data-testid="task-card-title">{{ task.title }}</h4>
      <div
        v-if="runtimeFlags.showRuntimeIndicators"
        class="task-card-no-drag relative inline-flex items-center justify-center shrink-0 mt-0.5 min-w-[1.25rem] min-h-[1.25rem] px-0.5"
        data-testid="task-card-runtime-indicators"
        @click.stop
        @mouseleave="activeRuntimeTip = null"
      >
        <span
          v-if="runtimeFlags.showMachineRing"
          class="relative inline-flex items-center justify-center w-3.5 h-3.5 rounded-full border-2 border-emerald-500 shrink-0 cursor-help"
          data-testid="task-card-machine-ring"
          :title="RUNTIME_MACHINE_TIP"
          :aria-label="RUNTIME_MACHINE_TIP"
          @mouseenter="activeRuntimeTip = 'machine'"
        >
          <span
            v-if="runtimeFlags.showContainerDot"
            class="absolute w-2 h-2 rounded-full bg-emerald-500 cursor-help"
            data-testid="task-card-container-dot"
            :title="RUNTIME_CONTAINER_TIP"
            :aria-label="RUNTIME_CONTAINER_TIP"
            @mouseenter.stop="activeRuntimeTip = 'container'"
            @mouseleave.stop="activeRuntimeTip = 'machine'"
          />
        </span>
        <span
          v-else-if="runtimeFlags.showContainerDot"
          class="inline-block w-2 h-2 rounded-full bg-emerald-500 shrink-0 cursor-help"
          data-testid="task-card-container-dot"
          :title="RUNTIME_CONTAINER_TIP"
          :aria-label="RUNTIME_CONTAINER_TIP"
          @mouseenter="activeRuntimeTip = 'container'"
        />
        <span
          v-show="activeRuntimeTip"
          role="tooltip"
          data-testid="task-card-runtime-tooltip"
          class="pointer-events-none absolute left-1/2 top-full z-20 mt-1.5 -translate-x-1/2 whitespace-nowrap rounded bg-gray-900 px-2 py-1 text-[11px] leading-tight text-white shadow-md"
        >
          {{ activeRuntimeTipLabel }}
        </span>
      </div>
    </div>

    <p class="text-sm text-text-light mb-3">{{ truncatedDescription }}</p>

    <div class="flex justify-between items-start mb-3 gap-2">
      <div class="flex flex-wrap items-start gap-x-3 gap-y-1 min-w-0 flex-1">
        <div
          class="min-w-0 leading-tight max-w-[40%]"
          :title="operatorLabel"
          data-testid="task-card-operator"
        >
          <span class="block text-[10px] text-gray-400">操作员</span>
          <span class="block text-xs text-gray-700 truncate" data-testid="task-card-operator-name">{{ operatorDisplayName }}</span>
        </div>
        <div
          class="min-w-0 leading-tight max-w-[40%]"
          :title="ownerLabel"
          data-testid="task-card-owner"
        >
          <span class="block text-[10px] text-gray-400">负责人</span>
          <span class="block text-xs text-gray-700 truncate" data-testid="task-card-owner-name">{{ ownerDisplayName }}</span>
        </div>
        <div
          class="min-w-0 leading-tight max-w-full"
          :title="collaboratorsLabel"
          data-testid="task-card-collaborators"
        >
          <span class="block text-[10px] text-gray-400">协作者</span>
          <span class="block text-xs text-gray-700 truncate" data-testid="task-card-collaborators-name">{{ collaboratorsDisplayName }}</span>
        </div>
      </div>
      <div
        v-if="formattedDate"
        class="text-right shrink-0 leading-tight"
        :title="createdAtTitle"
        data-testid="task-card-created-at"
      >
        <span class="block text-[10px] text-gray-400">创建于</span>
        <span class="block text-xs text-gray-500">{{ formattedDate }}</span>
      </div>
    </div>
    
    <TaskCardCommentsSection
      :task="task"
      :tenant-id="tenantId"
      :workspace-id="workspaceId"
      @create-comment="(taskId, payload) => emit('create-comment', taskId, payload)"
    />
  </div>
</template>

<script setup>
/* @alias:cmp-task-detail-card */
import { ref, computed } from 'vue'
import TaskCardIdBadge from './TaskCardIdBadge.vue'
import TaskCardCommentsSection from './TaskCardCommentsSection.vue'
import {
  resolveTaskRuntimeIndicatorFlags,
  RUNTIME_MACHINE_TIP,
  RUNTIME_CONTAINER_TIP,
} from '../utils/workPanelRuntimeIndicators.js'
import {
  formatCollaboratorsDisplay,
  formatSingleMemberDisplay,
} from '../utils/taskCardPeopleDisplay.js'

/** @type {import('vue').Ref<null | 'machine' | 'container'>} */
const activeRuntimeTip = ref(null)

const activeRuntimeTipLabel = computed(() => {
  if (activeRuntimeTip.value === 'machine') return RUNTIME_MACHINE_TIP
  if (activeRuntimeTip.value === 'container') return RUNTIME_CONTAINER_TIP
  return ''
})

const props = defineProps({
  task: {
    type: Object,
    required: true,
    default: () => ({
      id: '',
      title: '',
      priority: 2,
      description: '',
      created_by: {
        username: ''
      },
      created_at: '',
      comments: [],
      comments_count: 0,
      owner: '',
      operator: '',
      assignees: [],
    })
  },
  tenantId: {
    type: [String, Number],
    default: null
  },
  workspaceId: {
    type: [String, Number],
    default: null
  },
  taskStatuses: {
    type: Array,
    default: () => []
  },
  /** @type {{ machineRunning?: boolean, containerRunning?: boolean } | null} */
  runtimeIndicator: {
    type: Object,
    default: null
  },
  /** memberId → 展示名 */
  collaboratorNameById: {
    type: Object,
    default: () => ({})
  }
})

const operatorDisplayName = computed(() =>
  formatSingleMemberDisplay(props.task?.operator, props.collaboratorNameById),
)

const ownerDisplayName = computed(() =>
  formatSingleMemberDisplay(props.task?.owner, props.collaboratorNameById),
)

const collaboratorsDisplayName = computed(() =>
  formatCollaboratorsDisplay(props.task?.assignees, props.collaboratorNameById),
)

const operatorLabel = computed(() => `操作员：${operatorDisplayName.value}`)
const ownerLabel = computed(() => `负责人：${ownerDisplayName.value}`)
const collaboratorsLabel = computed(() => `协作者：${collaboratorsDisplayName.value}`)

const priorityClass = computed(() => {
  const priority = parseInt(props.task.priority) || 2
  switch (priority) {
    case 0:
      return 'bg-red-100 text-red-800'
    case 1:
      return 'bg-yellow-100 text-yellow-800'
    default:
      return 'bg-green-100 text-green-800'
  }
})

const priorityText = computed(() => {
  const priority = parseInt(props.task.priority) || 2
  switch (priority) {
    case 0:
      return '高优先级'
    case 1:
      return '中优先级'
    default:
      return '低优先级'
  }
})

const runtimeFlags = computed(() => resolveTaskRuntimeIndicatorFlags(props.runtimeIndicator))

const truncatedDescription = computed(() => {
  if (!props.task.description) return ''
  if (props.task.description.length <= 100) {
    return props.task.description
  }
  return props.task.description.substring(0, 100) + '...'
})

const formattedDate = computed(() => {
  if (!props.task.created_at) return ''
  const date = new Date(props.task.created_at)
  if (Number.isNaN(date.getTime())) return ''
  return `${date.getMonth() + 1}月${date.getDate()}日`
})

const createdAtTitle = computed(() => {
  if (!props.task.created_at) return ''
  const date = new Date(props.task.created_at)
  if (Number.isNaN(date.getTime())) return ''
  const y = date.getFullYear()
  const m = String(date.getMonth() + 1).padStart(2, '0')
  const d = String(date.getDate()).padStart(2, '0')
  const hh = String(date.getHours()).padStart(2, '0')
  const mm = String(date.getMinutes()).padStart(2, '0')
  return `创建于 ${y}-${m}-${d} ${hh}:${mm}`
})

const taskProgressColumnId = computed(() => {
  const t = props.task
  if (!t) return null
  const raw =
    t.progressColumn?.id ??
    t.progress_column_id ??
    t.progress_column?.id ??
    t.progress_column ??
    t.status?.id ??
    t.status
  if (raw == null || raw === '') return null
  return raw
})

const showProgressStatusSelect = computed(() => {
  return (
    props.taskStatuses.length > 0 &&
    props.tenantId != null &&
    String(props.tenantId).trim() !== '' &&
    props.workspaceId != null &&
    String(props.workspaceId).trim() !== '' &&
    String(props.workspaceId).trim() !== 'default'
  )
})

const progressSelectValue = computed(() => {
  const id = taskProgressColumnId.value
  return id != null ? String(id) : ''
})

const emit = defineEmits([
  'create-comment',
  'update-comment',
  'task-clicked',
  'progress-column-change'
])

const onProgressColumnChange = (e) => {
  const raw = e?.target?.value
  if (raw == null || raw === '') return
  if (String(raw) === String(taskProgressColumnId.value ?? '')) return
  emit('progress-column-change', {
    taskId: props.task.id,
    progressColumnId: raw
  })
}

</script>

<style scoped>
.task-card {
  background-color: white;
  border-radius: 8px;
  padding: 16px;
  margin-bottom: 12px;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
  transition: all 0.3s ease;
  cursor: pointer;
}

.task-card:hover {
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

.task-card.priority-0 {
  border-left: 4px solid #ff6b6b;
}

.task-card.priority-1 {
  border-left: 4px solid #4dabf7;
}

.task-card.priority-2 {
  border-left: 4px solid #51cf66;
}

.priority-0 {
  background-color: #fee2e2;
  color: #ef4444;
}

.priority-1 {
  background-color: #dbeafe;
  color: #3b82f6;
}

.priority-2 {
  background-color: #d1fae5;
  color: #10b981;
}

</style>

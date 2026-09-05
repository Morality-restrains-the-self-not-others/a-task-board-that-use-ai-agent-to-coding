<template>
  <div>
    <div v-if="isFieldEnabled('feature_params')">
      <!-- 已安装镜像字段已下线：镜像/技能选择收敛到任务描述 @ 弹层（TaskDescriptionSkillField），
           提交绑定仍走 container_image（由描述 mention 同步） -->
      <ServerConfigFeatureParamsBlock
        v-if="editingTask && isFieldEnabled('feature_params')"
        variant="form"
        :show-required-hint="true"
        :source-required-hint="createSubmitBlockedReason || '请先选择智能体资源配置'"
        :tenant-id="tenantId"
        :show-preview="Boolean(editingTask.id)"
        :feature-params-source="featureParamsSourceModel"
        :personal-configs="personalConfigs"
        :selected-personal-config-id="selectedPersonalConfigIdModel"
        :resolved-env-preview="resolvedEnvPreview"
        :is-env-preview-loading="isEnvPreviewLoading"
        :env-preview-expanded="envPreviewExpanded"
        :sources-available="featureParamsSourcesAvailable"
        @update:feature-params-source="onFeatureParamsSourceUpdate"
        @update:selected-personal-config-id="onPersonalConfigIdUpdate"
        @update:env-preview-expanded="envPreviewExpanded = $event"
        @source-change="onFeatureParamsSourceChange"
        @preview-env="fetchCreateTaskEnvPreview"
      />
    </div>
    <div v-if="isFieldEnabled('priority')">
      <label for="task-priority" class="block text-sm font-medium text-gray-700">优先级</label>
      <select id="task-priority" v-model="editingTask.priority" class="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-primary focus:border-primary">
        <option value="0">高优先级</option>
        <option value="1">中优先级</option>
        <option value="2">低优先级</option>
      </select>
    </div>
    <div v-if="isFieldEnabled('due_date')">
      <div class="flex items-center justify-between">
        <label for="task-deadline" class="block text-sm font-medium text-gray-700">截止日期</label>
        <span v-if="dueDateWeekHint" class="text-sm text-gray-500 whitespace-nowrap">（{{ dueDateWeekHint }}）</span>
      </div>
      <input type="datetime-local" id="task-deadline" v-model="editingTask.due_date" class="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-primary focus:border-primary">
    </div>
  </div>
</template>

<script setup>
import { computed, watch } from 'vue'
import ServerConfigFeatureParamsBlock from './ServerConfigFeatureParamsBlock.vue'
import { useCreateTaskFeatureParams } from '../composables/useCreateTaskFeatureParams.js'
import { defaultCreateTaskFieldSettings, isCreateTaskFieldEnabled } from '../utils/createTaskFieldSettings.js'

const props = defineProps({
  editingTask: {
    type: Object,
    required: true,
  },
  fieldSettings: {
    type: Object,
    default: () => defaultCreateTaskFieldSettings(),
  },
  currentWorkspace: {
    type: Object,
    default: null,
  },
  tenantId: {
    type: [String, Number],
    default: null,
  },
  show: {
    type: Boolean,
    default: false,
  },
  createSubmitBlockedReason: {
    type: String,
    default: null,
  },
})

const isFieldEnabled = (key) => isCreateTaskFieldEnabled(props.fieldSettings, key)

const {
  personalConfigs,
  resolvedEnvPreview,
  isEnvPreviewLoading,
  envPreviewExpanded,
  featureParamsSourcesAvailable,
  featureParamsSourceModel,
  selectedPersonalConfigIdModel,
  onFeatureParamsSourceUpdate,
  onPersonalConfigIdUpdate,
  onFeatureParamsSourceChange,
  fetchCreateTaskEnvPreview,
  syncFeatureParamsFromEditingTask,
} = useCreateTaskFeatureParams({
  editingTask: () => props.editingTask,
  currentWorkspace: () => props.currentWorkspace,
  tenantId: () => props.tenantId,
  show: () => props.show,
})

watch(
  () => [props.show, props.editingTask],
  () => {
    if (!props.show) return
    syncFeatureParamsFromEditingTask()
  },
  { immediate: true },
)

const dueDateWeekHint = computed(() => {
  const dueDate = props.editingTask?.due_date
  if (!dueDate) return ''

  const selectedDate = new Date(dueDate)
  if (Number.isNaN(selectedDate.getTime())) return ''

  const now = new Date()
  const todayStart = new Date(now.getFullYear(), now.getMonth(), now.getDate())
  const selectedStart = new Date(selectedDate.getFullYear(), selectedDate.getMonth(), selectedDate.getDate())
  const diffDays = Math.floor((selectedStart - todayStart) / (1000 * 60 * 60 * 24))
  const weekdayMap = ['星期日', '星期一', '星期二', '星期三', '星期四', '星期五', '星期六']
  const weekday = weekdayMap[selectedDate.getDay()]

  if (diffDays < 0) {
    const weeksBefore = Math.floor(Math.abs(diffDays) / 7)
    return `约${weeksBefore}周前的${weekday}`
  }

  if (diffDays < 7) {
    return `本周的${weekday}`
  }

  const weeksAfter = Math.floor(diffDays / 7)
  return `约${weeksAfter}周后的${weekday}`
})
</script>

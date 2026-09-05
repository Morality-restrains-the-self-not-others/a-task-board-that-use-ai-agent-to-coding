<template>
  <div>
    <div>
      <label for="task-title" class="block text-sm font-medium text-gray-700">任务标题</label>
      <input type="text" id="task-title" v-model="editingTask.title" class="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-primary focus:border-primary" placeholder="输入任务标题" required>
    </div>
    <div v-if="isFieldEnabled('description')">
      <label for="task-description" class="block text-sm font-medium text-gray-700">任务描述</label>
      <TaskDescriptionSkillField
        v-model="editingTask.description"
        :image="selectedImage"
        :installed-images="installedImages"
        @mention-change="onDescriptionMentionChange"
      />
    </div>

    <div v-if="isFieldEnabled('task_kind')">
      <label for="task-kind" class="block text-sm font-medium text-gray-700">任务类型</label>
      <select
        id="task-kind"
        v-model="editingTask.task_kind"
        class="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-primary focus:border-primary"
        data-testid="create-task-field-taskKind"
      >
        <option value="">未指定</option>
        <option v-for="opt in resolvedTaskKindOptions" :key="opt" :value="opt">{{ opt }}</option>
      </select>
    </div>

    <div v-if="isFieldEnabled('code_lang')">
      <label for="task-code-lang" class="block text-sm font-medium text-gray-700">主要编程语言</label>
      <select
        id="task-code-lang"
        v-model="editingTask.code_lang"
        class="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-primary focus:border-primary"
        data-testid="create-task-field-codeLang"
      >
        <option value="">未指定</option>
        <option v-for="opt in resolvedCodeLangOptions" :key="opt" :value="opt">{{ opt }}</option>
      </select>
    </div>

    <div
      v-if="isFieldEnabled('structured_fields')"
      class="rounded-md border border-gray-200 bg-gray-50/60"
      data-testid="create-task-structured-fields"
    >
      <button
        type="button"
        class="flex w-full items-center justify-between px-3 py-2 text-left text-sm font-medium text-gray-700 hover:bg-gray-100/80"
        data-testid="create-task-structured-toggle"
        :aria-expanded="structuredOpen ? 'true' : 'false'"
        @click="structuredOpen = !structuredOpen"
      >
        <span>结构化任务说明（可选）</span>
        <svg
          class="h-4 w-4 text-gray-500 transition-transform"
          :class="structuredOpen ? 'rotate-180' : ''"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
          aria-hidden="true"
        >
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
        </svg>
      </button>
      <div v-show="structuredOpen" class="space-y-3 border-t border-gray-200 px-3 py-3">
        <div v-for="field in textFields" :key="field.key">
          <label :for="`task-structured-${field.key}`" class="block text-sm font-medium text-gray-700">
            {{ field.heading }}
          </label>
          <textarea
            :id="`task-structured-${field.key}`"
            v-model="editingTask[field.key]"
            rows="2"
            class="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-primary focus:border-primary"
            :placeholder="`可选：${field.heading}`"
            :data-testid="`create-task-field-${field.key}`"
          />
        </div>
        <div>
          <label for="task-structured-forbidNewDeps" class="block text-sm font-medium text-gray-700">
            是否禁止新增依赖
          </label>
          <select
            id="task-structured-forbidNewDeps"
            v-model="editingTask.forbidNewDeps"
            class="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-primary focus:border-primary"
            data-testid="create-task-field-forbidNewDeps"
          >
            <option value="">未指定</option>
            <option value="yes">是</option>
            <option value="no">否</option>
          </select>
        </div>
      </div>
    </div>

    <div>
      <label for="task-status" class="block text-sm font-medium text-gray-700">进度状态</label>
      <select id="task-status" v-model="editingTask.progressColumn.id" class="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-primary focus:border-primary">
        <option v-for="status in taskStatuses" :key="status.id" :value="status.id">{{ status.name }}</option>
      </select>
    </div>
    <div>
      <label for="task-type" class="block text-sm font-medium text-gray-700">交付物类别</label>
      <div v-if="isTaskTypesLoading" class="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm bg-gray-50 flex items-center">
        <svg class="animate-spin -ml-1 mr-2 h-4 w-4 text-gray-500" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
        </svg>
        加载中...
      </div>
      <div
        v-else-if="taskTypesError"
        class="mt-1 block w-full px-3 py-2 border border-red-300 rounded-md shadow-sm bg-red-50 text-red-700"
        :data-traceId="taskTypesErrorTraceId || undefined"
      >
        {{ taskTypesError }}
      </div>
      <select v-else id="task-type" v-model="editingTask.task_type.id" class="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-primary focus:border-primary">
        <option v-if="taskTypes.length === 0" value="">无可用任务类别</option>
        <option v-for="type in taskTypes" :key="type.id" :value="type.id">{{ type.name }}</option>
      </select>
    </div>
  </div>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import TaskDescriptionSkillField from './TaskDescriptionSkillField.vue'
import { syncTaskContainerImageFromMention } from '../utils/createTaskImageMention.js'
import {
  CREATE_TASK_STRUCTURED_FIELDS,
  emptyCreateTaskStructuredFields,
} from '../utils/createTaskDescriptionCompose.js'
import { normalizeTaskKindOptions } from '../utils/taskKindOptions.js'
import { normalizeCodeLangOptions } from '../utils/codeLangOptions.js'
import { defaultCreateTaskFieldSettings, isCreateTaskFieldEnabled } from '../utils/createTaskFieldSettings.js'

const props = defineProps({
  editingTask: {
    type: Object,
    required: true,
  },
  taskStatuses: {
    type: Array,
    default: () => [],
  },
  taskTypes: {
    type: Array,
    default: () => [],
  },
  taskKindOptions: {
    type: Array,
    default: () => [],
  },
  codeLangOptions: {
    type: Array,
    default: () => [],
  },
  fieldSettings: {
    type: Object,
    default: () => defaultCreateTaskFieldSettings(),
  },
  isTaskTypesLoading: {
    type: Boolean,
    default: false,
  },
  taskTypesError: {
    type: String,
    default: null,
  },
  taskTypesErrorTraceId: {
    type: String,
    default: '',
  },
  installedImages: {
    type: Array,
    default: () => [],
  },
})

const isFieldEnabled = (key) => isCreateTaskFieldEnabled(props.fieldSettings, key)
const resolvedTaskKindOptions = computed(() => normalizeTaskKindOptions(props.taskKindOptions))
const resolvedCodeLangOptions = computed(() => normalizeCodeLangOptions(props.codeLangOptions))

/**
 * 描述 textarea 中的 @ 镜像 / 技能 mention 变化 → 同步 task.container_image 绑定。
 * 描述为权威输入源：mention 被删除时同步解除绑定（与「已安装镜像」字段共用同一绑定契约）。
 */
function onDescriptionMentionChange({ text, id }) {
  const task = props.editingTask
  if (!task || typeof task !== 'object') return
  const result = syncTaskContainerImageFromMention(task, text, props.installedImages, id)
  if (!result.mention && task.container_image) {
    task.container_image.id = null
    task.container_image.mentionText = ''
    task.container_image.skill = ''
  }
}

const selectedImage = computed(() => {
  const id = props.editingTask?.container_image?.id
  if (id == null || String(id).trim() === '') return null
  const want = String(id)
  return (props.installedImages || []).find((img) => String(img.id) === want) || null
})

const textFields = CREATE_TASK_STRUCTURED_FIELDS.filter((f) => f.kind !== 'deps')

const hasStructuredContent = () => {
  const empty = emptyCreateTaskStructuredFields()
  return CREATE_TASK_STRUCTURED_FIELDS.some((f) => {
    const v = props.editingTask?.[f.key]
    return v != null && String(v).trim() !== '' && String(v).trim() !== String(empty[f.key] || '')
  })
}

const structuredOpen = ref(true)

watch(
  () => props.editingTask,
  (task) => {
    if (!task || typeof task !== 'object') return
    const defaults = emptyCreateTaskStructuredFields()
    for (const key of Object.keys(defaults)) {
      if (task[key] == null) task[key] = defaults[key]
    }
    // 编辑且已有结构化内容时展开；新建默认展开
    structuredOpen.value = !task.id || hasStructuredContent()
  },
  { immediate: true },
)
</script>

<template>
  <div id="create-task-modal" class="app-modal-overlay bg-black bg-opacity-50 flex items-center justify-center z-9999 p-4" v-if="show" @click="closeAllDropdowns">
    <div class="bg-white rounded-lg shadow-xl w-full max-w-lg max-h-[90vh] flex flex-col p-6 z-10000 relative" @click.stop @keydown.enter="onEnterKey">
      <div class="flex justify-between items-center mb-4 shrink-0">
        <h3 class="text-xl font-bold text-gray-900">{{ editingTask?.id ? '编辑任务' : '创建任务' }}</h3>
        <button class="text-gray-500 hover:text-gray-700" @click="handleClose">
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
          </svg>
        </button>
      </div>
      <div v-if="editingTask" class="space-y-4 overflow-y-auto min-h-0 flex-1 pr-1">
        <CreateTaskBasicFields
          :editing-task="editingTask"
          :task-statuses="taskStatuses"
          :task-types="taskTypes"
          :task-kind-options="taskKindOptions"
          :code-lang-options="codeLangOptions"
          :field-settings="fieldSettings"
          :is-task-types-loading="isTaskTypesLoading"
          :task-types-error="taskTypesError"
          :task-types-error-trace-id="taskTypesErrorTraceId"
          :installed-images="installedImages"
        />
        <CreateTaskParentDeliverableField
          v-model="editingTask.parent_task"
          :task-types="taskTypes"
          :todos="todos"
          :selected-category-id="editingTask.task_type?.id"
          :exclude-task-id="editingTask?.id"
        />
        <CreateTaskProjectBranchSection
          v-if="isFieldEnabled('project_branch')"
          ref="projectBranchSectionRef"
          :editing-task="editingTask"
          :projects="projects"
          :task-statuses="taskStatuses"
          :task-types="taskTypes"
          :is-projects-loading="isProjectsLoading"
          :projects-error="projectsError"
          :projects-error-trace-id="projectsErrorTraceId"
          :tenant-id="tenantId"
          :company-user-name="companyUserName"
          :show="show"
          :repo-hint-dismiss-tick="repoHintDismissTick"
        />
        <CreateTaskMetaFields
          :editing-task="editingTask"
          :current-workspace="currentWorkspace"
          :tenant-id="tenantId"
          :show="show"
          :create-submit-blocked-reason="createSubmitBlockedReason"
          :field-settings="fieldSettings"
        />
        <CreateTaskAutoRunSection
          v-if="isFieldEnabled('auto_run')"
          :editing-task="editingTask"
          :projects="projects"
          :installed-images="installedImages"
          :on-primary-project-change="onPrimaryProjectChange"
          :tenant-id="tenantId"
          :workspace-id="currentWorkspace?.id"
          :show="show"
        />
        <CreateTaskGitIdentitySection
          :editing-task="editingTask"
          :projects="projects"
          :tenant-id="tenantId"
          :show="show"
        />
        <CreateTaskRepoOAuthSection
          :editing-task="editingTask"
          :projects="projects"
          :oauth-repo-urls="oauthRepoUrls"
          :oauth-bound-by-url="oauthBoundByUrl"
          :oauth-loading-by-url="oauthLoadingByUrl"
          :oauth-error-by-url="oauthErrorByUrl"
          :oauth-error-trace-id-by-url="oauthErrorTraceIdByUrl"
        />
        <CreateTaskPeopleFields
          v-if="isFieldEnabled('owner') || isFieldEnabled('operator') || isFieldEnabled('assignees')"
          ref="peopleFieldsRef"
          :editing-task="editingTask"
          :tenant-id="tenantId"
          :current-workspace="currentWorkspace"
          :show="show"
          :field-settings="fieldSettings"
        />
      </div>
      <div class="mt-6 flex flex-col items-end gap-2 shrink-0">
        <p
          v-if="createSubmitBlockedReason"
          class="text-sm text-amber-700"
          data-testid="create-task-submit-blocked-reason"
          role="status"
          :data-traceId="createSubmitBlockedTraceId || undefined"
        >
          {{ createSubmitBlockedReason }}
        </p>
        <div class="flex space-x-3">
          <button type="button" class="px-4 py-2 border border-gray-300 rounded-md shadow-sm text-sm font-medium text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-primary" @click="handleClose">取消</button>
          <button
            type="button"
            class="px-4 py-2 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-primary hover:bg-blue-600 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-primary disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:bg-primary"
            data-testid="create-task-submit-btn"
            :disabled="Boolean(createSubmitBlockedReason)"
            :title="createSubmitBlockedReason || undefined"
            @click="handleSubmit"
          >{{ editingTask?.id ? '保存' : '创建' }}</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import { resolveBranchNamePlaceholders } from '../utils/workPanelBranchHelpers.js'
import {
  applyComposedDescriptionToTask,
  emptyCreateTaskStructuredFields,
  hydrateStructuredFieldsFromDescription,
} from '../utils/createTaskDescriptionCompose.js'
import { applyImageMentionToTaskDescription, bindTaskContainerImageSkillId } from '../utils/createTaskImageMention.js'
import { normalizeFeatureParamsSourceForSelect, resolveCreateTaskFeatureParamsBlockedReason } from '../utils/envParamsSourceSelection.js'
import { defaultCreateTaskFieldSettings, isCreateTaskFieldEnabled } from '../utils/createTaskFieldSettings.js'
import { resolveParentDeliverableBlockedReason } from '../utils/workPanelCreateTaskParent.js'
import CreateTaskBasicFields from './CreateTaskBasicFields.vue'
import CreateTaskParentDeliverableField from './CreateTaskParentDeliverableField.vue'
import CreateTaskProjectBranchSection from './CreateTaskProjectBranchSection.vue'
import CreateTaskMetaFields from './CreateTaskMetaFields.vue'
import CreateTaskAutoRunSection from './CreateTaskAutoRunSection.vue'
import CreateTaskGitIdentitySection from './CreateTaskGitIdentitySection.vue'
import CreateTaskRepoOAuthSection from './CreateTaskRepoOAuthSection.vue'
import CreateTaskPeopleFields from './CreateTaskPeopleFields.vue'
import { useCreateTaskRepoOAuth } from '../composables/useCreateTaskRepoOAuth.js'
import { shouldRequireAutoRunOauthGate } from '../utils/createTaskOauthGate.js'
import {
  collectCreateTaskRepoUrls,
  validateCreateTaskGitIdentities,
} from '../utils/createTaskGitIdentityGate.js'

const props = defineProps({
  show: {
    type: Boolean,
    default: false,
  },
  editingTask: {
    default: null,
    validator: (val) =>
      val === null || (typeof val === 'object' && !Array.isArray(val)),
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
  /** 工作空间任务列表，用于非顶层类别时选择上层交付物 */
  todos: {
    type: Array,
    default: () => []
  },
  projects: {
    type: Array,
    default: () => [],
  },
  installedImages: {
    type: Array,
    default: () => [],
  },
  isProjectsLoading: {
    type: Boolean,
    default: false,
  },
  projectsError: {
    type: String,
    default: null,
  },
  projectsErrorTraceId: {
    type: String,
    default: '',
  },
  isInstalledImagesLoading: {
    type: Boolean,
    default: false,
  },
  installedImagesError: {
    type: String,
    default: null,
  },
  installedImagesErrorTraceId: {
    type: String,
    default: '',
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
  currentWorkspace: {
    type: Object,
    default: null,
  },
  tenantId: {
    type: [String, Number],
    default: null,
  },
  companyUserName: {
    type: String,
    default: '',
  },
})

const emit = defineEmits(['close', 'submit'])

const peopleFieldsRef = ref(null)
const projectBranchSectionRef = ref(null)
const repoHintDismissTick = ref(0)
/** 避免同一打开会话内重复 hydrate 冲掉用户输入 */
const hydratedTaskKey = ref('')

watch(
  () => ({ show: props.show, taskId: props.editingTask?.id, task: props.editingTask }),
  ({ show, taskId, task }) => {
    if (!show || !task || typeof task !== 'object') {
      if (!show) hydratedTaskKey.value = ''
      return
    }
    const key = taskId != null && taskId !== '' ? `edit:${String(taskId)}` : 'create'
    if (hydratedTaskKey.value === key) return
    const defaults = emptyCreateTaskStructuredFields()
    for (const fieldKey of Object.keys(defaults)) {
      if (task[fieldKey] == null) task[fieldKey] = defaults[fieldKey]
    }
    if (taskId != null && taskId !== '') {
      hydrateStructuredFieldsFromDescription(task)
    }
    hydratedTaskKey.value = key
  },
  { immediate: true },
)

const featureParamsSourceModel = computed(() =>
  normalizeFeatureParamsSourceForSelect(props.editingTask?.feature_params_source),
)
const selectedPersonalConfigIdModel = computed(() =>
  String(props.editingTask?.personal_feature_params_config_id || '').trim(),
)

const isFieldEnabled = (key) => isCreateTaskFieldEnabled(props.fieldSettings, key)

const {
  boundByUrl: oauthBoundByUrl,
  loadingByUrl: oauthLoadingByUrl,
  errorByUrl: oauthErrorByUrl,
  errorTraceIdByUrl: oauthErrorTraceIdByUrl,
  oauthRepoUrls,
  blockedReason: oauthBlockedReason,
  blockedTraceId: oauthBlockedTraceId,
} = useCreateTaskRepoOAuth({
  editingTask: () => props.editingTask,
  projects: () => props.projects,
  tenantId: () => props.tenantId,
  enabled: () => Boolean(
    props.show
    && isFieldEnabled('project_branch')
    && shouldRequireAutoRunOauthGate(props.editingTask?.auto_run),
  ),
})

const createSubmitBlockedReason = computed(() => {
  if (isFieldEnabled('feature_params')) {
    const featureReason = resolveCreateTaskFeatureParamsBlockedReason({
      featureParamsSource: featureParamsSourceModel.value,
      selectedPersonalConfigId: selectedPersonalConfigIdModel.value,
      isEdit: Boolean(props.editingTask?.id),
    })
    if (featureReason) return featureReason
  }
  const parentReason = resolveParentDeliverableBlockedReason({
    taskTypes: props.taskTypes,
    todos: props.todos,
    categoryId: props.editingTask?.task_type?.id,
    parentTaskId: props.editingTask?.parent_task,
  })
  if (parentReason) return parentReason
  const oauthReason = String(oauthBlockedReason.value || '')
  if (oauthReason) return oauthReason
  return validateCreateTaskGitIdentities(
    props.editingTask?.auto_run,
    props.editingTask?.repo_identities,
    collectCreateTaskRepoUrls(props.editingTask, props.projects),
  )
})

const createSubmitBlockedTraceId = computed(() => String(oauthBlockedTraceId.value || ''))

const onPrimaryProjectChange = (projectId) => {
  projectBranchSectionRef.value?.onPrimaryProjectChange?.(projectId)
}

const closeAllDropdowns = () => {
  peopleFieldsRef.value?.closePickers?.()
  repoHintDismissTick.value += 1
}

const onEnterKey = (event) => {
  const tag = (event.target?.tagName || '').toLowerCase()
  if (tag === 'textarea') return
  handleSubmit()
}

const handleClose = () => {
  peopleFieldsRef.value?.closePickers?.()
  emit('close')
}

const handleSubmit = async () => {
  if (!props.editingTask) {
    emit('submit', props.editingTask)
    return
  }
  if (createSubmitBlockedReason.value) {
    return
  }
  if (isFieldEnabled('project_branch')) {
    if (props.editingTask.workBranchPreset && props.editingTask.workBranchPreset !== 'custom') {
      await projectBranchSectionRef.value?.updateWorkBranchNameByPreset?.()
    }
    const titleSegment = projectBranchSectionRef.value?.resolveSyncTitleSegmentForBranch?.()
      ?? props.editingTask.title
    props.editingTask.workBranchName = resolveBranchNamePlaceholders(
      props.editingTask.workBranchName,
      {
        taskId: props.editingTask.id ? String(props.editingTask.id) : '',
        taskTitleSegment: titleSegment,
      },
    )
    props.editingTask.mergeTargetName = resolveBranchNamePlaceholders(
      props.editingTask.mergeTargetName,
      {
        taskId: props.editingTask.id ? String(props.editingTask.id) : '',
        taskTitleSegment: titleSegment,
      },
    )
  }
  applyImageMentionToTaskDescription(props.editingTask, props.installedImages)
  // D4 契约：技能名 → 稳定 ID（container_image.skillId → payload container_image_skill_id）。
  // 目录缺 id/技能名不在列表时置空，服务端按描述反解兜底。
  bindTaskContainerImageSkillId(props.editingTask, props.installedImages)
  applyComposedDescriptionToTask(props.editingTask)
  emit('submit', props.editingTask)
}
</script>

<style scoped>
/* 组件样式 */
</style>

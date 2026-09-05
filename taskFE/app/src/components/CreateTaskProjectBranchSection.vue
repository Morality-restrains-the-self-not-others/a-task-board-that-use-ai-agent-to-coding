<template>
  <div class="space-y-3">
    <div class="space-y-1" data-testid="create-task-project-section-heading">
      <p class="text-sm font-medium text-gray-700">项目</p>
      <p class="text-xs text-gray-500">选择关联项目与各仓基准分支</p>
    </div>
    <div v-if="isProjectsLoading" class="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm bg-gray-50 flex items-center">
      <svg class="animate-spin -ml-1 mr-2 h-4 w-4 text-gray-500" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
      </svg>
      加载中...
    </div>
    <div v-else-if="projectsError" class="mt-1 block w-full px-3 py-2 border border-red-300 rounded-md shadow-sm bg-red-50 text-red-700">
      {{ projectsError }}
    </div>
    <template v-else>
      <div class="space-y-3">
        <div
          v-for="(selection, index) in editingTask.projectSelections"
          :key="selection.selectionRowKey ?? `ps-m-${index}`"
          class="rounded-md border border-gray-200 p-3 space-y-2"
        >
          <div>
            <div class="flex items-center gap-1">
              <label :for="`task-project-${selection.selectionRowKey ?? index}`" class="text-sm font-medium text-gray-700">项目</label>
              <ProjectRepoAccessHintIcon
                v-if="index === 0"
                :dismiss-signal="repoHintDismissTick"
              />
            </div>
            <select
              :id="`task-project-${selection.selectionRowKey ?? index}`"
              v-model="selection.projectId"
              class="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-primary focus:border-primary"
              @change="onProjectSelectionChange(selection)"
            >
              <option v-if="projects.length === 0" value="">无可用项目</option>
              <option v-else-if="getProjectsForSelectionRow(index).length === 0" value="" disabled>无可用项目（已全部选择）</option>
              <option v-for="project in getProjectsForSelectionRow(index)" :key="project.id" :value="String(project.id)">{{ getProjectDisplayLabel(project) }}</option>
            </select>
          </div>
          <div v-if="!selection.projectId" class="text-xs text-gray-500">
            请先选择项目
          </div>
          <template v-else>
            <p class="text-sm font-medium text-gray-700" data-testid="create-task-branch-strategy-heading">
              分支策略
            </p>
            <div
              v-for="rb in selection.repoBranches"
              :key="`${selection.projectId}-repo-${rb.repoIndex}`"
              class="rounded border border-gray-100 bg-gray-50/80 p-2 space-y-2"
            >
              <p class="text-xs text-gray-600 break-all">
                <a
                  v-if="isHttpRepoUrl(getRepoUrl(selection.projectId, rb.repoIndex))"
                  :href="getRepoUrl(selection.projectId, rb.repoIndex)"
                  target="_blank"
                  rel="noopener noreferrer"
                  class="text-blue-600 hover:underline break-all"
                  :title="getRepoUrl(selection.projectId, rb.repoIndex) || undefined"
                  data-testid="create-task-repo-url"
                >{{ getRepoBranchRowLabel(selection.projectId, rb.repoIndex) }}</a>
                <span
                  v-else
                  class="break-all"
                  :title="getRepoUrl(selection.projectId, rb.repoIndex) || undefined"
                >{{ getRepoBranchRowLabel(selection.projectId, rb.repoIndex) }}</span>
              </p>
              <div>
                <div class="flex items-center justify-between">
                  <label
                    :for="`task-base-branch-${index}-${rb.repoIndex}`"
                    class="inline-flex items-center gap-1.5 text-sm font-medium text-gray-700"
                  >
                    <span>基准分支</span>
                    <span
                      v-if="isBaseBranchCommitChecking(selection.projectId, rb.repoIndex, rb.baseBranch)"
                      class="inline-flex items-center text-gray-500"
                      data-testid="base-branch-commit-checking"
                      aria-label="正在校验 commit"
                      title="正在校验 commit"
                    >
                      <svg class="h-4 w-4 animate-spin" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" aria-hidden="true">
                        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                      </svg>
                    </span>
                    <span
                      v-else-if="isBaseBranchCommitVerified(selection.projectId, rb.repoIndex, rb.baseBranch)"
                      class="inline-flex items-center text-green-600"
                      data-testid="base-branch-commit-ok"
                      aria-label="已确认该 commit 存在"
                      title="已确认该 commit 存在"
                    >
                      <svg class="h-4 w-4" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
                        <path fill-rule="evenodd" d="M16.704 5.29a1 1 0 010 1.42l-7.25 7.25a1 1 0 01-1.414 0l-3.25-3.25a1 1 0 011.414-1.42l2.543 2.543 6.543-6.543a1 1 0 011.414 0z" clip-rule="evenodd" />
                      </svg>
                    </span>
                  </label>
                  <button
                    type="button"
                    class="text-xs px-2 py-1 border border-gray-200 text-gray-600 rounded hover:bg-gray-50"
                    @click="refreshRepoBranches(selection.projectId, rb.repoIndex)"
                    :disabled="!selection.projectId || isProjectBranchesLoading(selection.projectId, rb.repoIndex)"
                  >
                    刷新分支
                  </button>
                </div>
                <input
                  :id="`task-base-branch-${index}-${rb.repoIndex}`"
                  v-model="rb.baseBranch"
                  type="text"
                  :list="datalistListAttr(`task-base-branch-${index}-${rb.repoIndex}`, `task-base-branch-options-${index}-${rb.repoIndex}`)"
                  class="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-primary focus:border-primary"
                  placeholder="输入或选择基准分支"
                  @focus="restoreDatalistOnFocus"
                  @input="(event) => onBaseBranchDatalistEvent(selection, rb, event)"
                  @change="(event) => onBaseBranchDatalistEvent(selection, rb, event)"
                >
                <datalist :id="`task-base-branch-options-${index}-${rb.repoIndex}`">
                  <option
                    v-for="branch in getProjectBranches(selection.projectId, rb.repoIndex)"
                    :key="`project-${selection.projectId}-r${rb.repoIndex}-branch-${branch}`"
                    :value="branch"
                  />
                </datalist>
                <p
                  v-if="getProjectBranchesError(selection.projectId, rb.repoIndex)"
                  class="mt-1 text-xs text-red-600"
                  :data-traceId="getProjectBranchesErrorTraceId(selection.projectId, rb.repoIndex) || undefined"
                >
                  {{ getProjectBranchesError(selection.projectId, rb.repoIndex) }}
                </p>
                <p
                  v-if="isBaseBranchCommitMissing(selection.projectId, rb.repoIndex, rb.baseBranch)"
                  class="mt-1 text-xs text-amber-700/80"
                  data-testid="base-branch-commit-missing"
                  :data-traceId="getBaseBranchCommitCheckTraceId(selection.projectId, rb.repoIndex, rb.baseBranch) || undefined"
                >
                  {{ getBaseBranchCommitMissingHint(selection.projectId, rb.repoIndex, rb.baseBranch) }}
                </p>
              </div>
            </div>
          </template>
        </div>
      </div>
      <div
        class="rounded-md border border-gray-200 p-3 space-y-3"
        data-testid="create-task-branch-strategy-section"
      >
        <div class="space-y-0.5" data-testid="create-task-branch-strategy-heading">
          <p class="text-sm font-medium text-gray-700">分支策略</p>
          <p class="text-xs text-gray-500">定工作分支与合并目标分支</p>
        </div>
        <div>
          <label for="work-branch-name" class="block text-sm font-medium text-gray-700">工作分支</label>
          <input
            id="work-branch-name"
            v-model="editingTask.workBranchName"
            type="text"
            :list="datalistListAttr('work-branch-name', 'work-branch-preset-options')"
            class="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-primary focus:border-primary"
            placeholder="选择模版或手动输入分支名"
            @input="(event) => { syncWorkBranchPresetFromName(); onWorkBranchDatalistChange(event) }"
            @focus="restoreDatalistOnFocus"
            @change="onWorkBranchDatalistChange"
          >
          <datalist id="work-branch-preset-options">
            <option
              v-for="opt in workBranchDatalistOptions"
              :key="`work-branch-${opt.preset}`"
              :value="opt.value"
              :label="opt.label"
            />
          </datalist>
          <p class="mt-1 text-xs text-gray-500">当前公司昵称：{{ companyUserName || '未设置' }}</p>
          <p v-if="isTaskTitleTranslating" class="mt-1 text-xs text-gray-500">正在翻译任务标题...</p>
          <p
            v-if="taskTitleTranslationError"
            class="mt-1 text-xs text-red-600"
            :data-traceId="taskTitleTranslationErrorTraceId || undefined"
          >{{ taskTitleTranslationError }}</p>
        </div>
        <div>
          <label for="merge-target-name" class="block text-sm font-medium text-gray-700">目标分支（合并目标）</label>
          <input
            id="merge-target-name"
            v-model="editingTask.mergeTargetName"
            type="text"
            :list="datalistListAttr('merge-target-name', 'merge-target-preset-options')"
            class="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-primary focus:border-primary"
            placeholder="选择各仓库共有分支或手动输入"
            @input="(event) => { markMergeTargetUserEdited(); syncMergeTargetPresetFromName(); onMergeTargetBranchChange(event) }"
            @focus="restoreDatalistOnFocus"
            @change="onMergeTargetBranchChange"
          >
          <datalist id="merge-target-preset-options">
            <option
              v-for="opt in mergeTargetDatalistOptions"
              :key="`merge-target-${opt.preset}`"
              :value="opt.value"
              :label="opt.label"
            />
          </datalist>
          <p v-if="commonMergeTargetBranchesLoading" class="mt-1 text-xs text-gray-500">正在拉取各仓库分支…</p>
          <p v-else-if="commonMergeTargetBranches.length > 0" class="mt-1 text-xs text-gray-500">
            共有分支 {{ commonMergeTargetBranches.length }} 个（来自已选项目全部仓库的交集）
          </p>
          <p v-else class="mt-1 text-xs text-amber-700">
            暂无共有分支候选，请先选择项目并等待分支加载，或手动输入
          </p>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup>
import { onBeforeUnmount } from 'vue'
import ProjectRepoAccessHintIcon from './ProjectRepoAccessHintIcon.vue'
import { createDatalistDismissController } from '../utils/datalistDismiss.js'
import { isHttpRepoUrl } from '../utils/repoUrl.js'
import { useCreateTaskProjectBranches } from '../composables/useCreateTaskProjectBranches.js'
import { useCreateTaskBranchNaming } from '../composables/useCreateTaskBranchNaming.js'

const props = defineProps({
  editingTask: {
    type: Object,
    required: true,
  },
  projects: {
    type: Array,
    default: () => [],
  },
  taskStatuses: {
    type: Array,
    default: () => [],
  },
  taskTypes: {
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
  tenantId: {
    type: [String, Number],
    default: null,
  },
  companyUserName: {
    type: String,
    default: '',
  },
  show: {
    type: Boolean,
    default: false,
  },
  repoHintDismissTick: {
    type: Number,
    default: 0,
  },
})

const datalistController = createDatalistDismissController()
const branchNamingRef = { current: null }

const {
  getProjectsForSelectionRow,
  getProjectDisplayLabel,
  getProjectBranches,
  getProjectBranchesError,
  getProjectBranchesErrorTraceId,
  isProjectBranchesLoading,
  getRepoUrl,
  getRepoBranchRowLabel,
  refreshRepoBranches,
  onProjectSelectionChange,
  onBaseBranchDatalistEvent,
  datalistListAttr,
  restoreDatalistOnFocus,
  onPrimaryProjectChange,
  resetAllCommitChecks,
  branchMapKey,
  isBaseBranchCommitChecking,
  isBaseBranchCommitVerified,
  isBaseBranchCommitMissing,
  getBaseBranchCommitCheckTraceId,
  getBaseBranchCommitMissingHint,
} = useCreateTaskProjectBranches({
  editingTask: () => props.editingTask,
  projects: () => props.projects,
  taskStatuses: () => props.taskStatuses,
  taskTypes: () => props.taskTypes,
  tenantId: () => props.tenantId,
  show: () => props.show,
  isProjectsLoading: () => props.isProjectsLoading,
  initBranchDefaults: () => branchNamingRef.current?.initBranchDefaults?.(),
  datalistController,
})

const {
  isTaskTitleTranslating,
  taskTitleTranslationError,
  taskTitleTranslationErrorTraceId,
  workBranchDatalistOptions,
  mergeTargetDatalistOptions,
  commonMergeTargetBranches,
  commonMergeTargetBranchesLoading,
  onWorkBranchDatalistChange,
  onMergeTargetBranchChange,
  syncWorkBranchPresetFromName,
  syncMergeTargetPresetFromName,
  markMergeTargetUserEdited,
  updateWorkBranchNameByPreset,
  resolveSyncTitleSegmentForBranch,
  initBranchDefaults,
} = useCreateTaskBranchNaming({
  editingTask: () => props.editingTask,
  companyUserName: () => props.companyUserName,
  tenantId: () => props.tenantId,
  show: () => props.show,
  getProjectBranches,
  isProjectBranchesLoading,
  getProjectBranchesError,
  branchMapKey,
  datalistController,
})

branchNamingRef.current = { initBranchDefaults }

onBeforeUnmount(() => {
  resetAllCommitChecks()
})

defineExpose({
  updateWorkBranchNameByPreset,
  resolveSyncTitleSegmentForBranch,
  onPrimaryProjectChange,
})
</script>

<template>
  <div>
      <p class="mb-2 text-[11px] text-gray-500 leading-snug">
        一个任务只能关联一个项目；选择项目后可为各仓库配置基准分支。
      </p>
      <div v-if="editingTask.linkedProjects && editingTask.linkedProjects.length > 0" class="space-y-2 mb-2">
        <div
          v-for="(project, index) in editingTask.linkedProjects"
          :key="index"
          class="p-2 bg-gray-50 rounded border border-gray-100/80"
        >
          <div class="flex flex-wrap items-center gap-1.5 mb-1">
            <label
              :for="'task-linked-project-' + index"
              class="text-[11px] text-gray-600 shrink-0"
            >项目</label>
            <select
              :id="'task-linked-project-' + index"
              v-model="project.project_id"
              class="flex-1 min-w-[140px] px-2 py-0.5 border border-gray-300 rounded text-xs focus:outline-none focus:ring-1 focus:ring-primary focus:border-primary"
              @change="emit('project-change', project)"
            >
              <option value="" disabled>请选择项目</option>
              <option
                v-for="p in workspaceProjects"
                :key="p.id"
                :value="String(p.id)"
              >
                {{ p.name }}
              </option>
            </select>
          </div>
          <p v-if="!project.project_id" class="ml-1 text-[11px] text-gray-400 leading-tight">
            请先选择项目
          </p>
          <div v-else-if="getProjectRepos(project.project_id).length > 0" class="ml-1 pl-1 border-l border-gray-200/60">
            <p class="text-[11px] text-gray-500 mb-1 leading-tight">仓库基准分支：</p>
            <div class="space-y-2">
              <div
                v-for="repoUrl in getProjectRepos(project.project_id)"
                :key="repoUrl"
                class="rounded border border-gray-100 bg-white/80 px-1.5 py-1"
              >
                <div class="flex flex-wrap items-center gap-1.5">
                  <!-- Anti-Replay-OK: navigation -->
                  <a
                    v-if="isHttpRepoUrl(repoUrl)"
                    :href="String(repoUrl).trim()"
                    target="_blank"
                    rel="noopener noreferrer"
                    class="text-xs text-gray-600 truncate max-w-xs hover:underline hover:text-blue-600"
                    :title="repoUrl"
                    data-testid="task-linked-repo-url"
                  >{{ repoUrl }}</a>
                  <span
                    v-else
                    class="text-xs text-gray-600 truncate max-w-xs"
                    :title="repoUrl"
                  >{{ repoUrl }}</span>
                </div>
                <div class="mt-1 flex flex-wrap items-center gap-1.5">
                  <label
                    :for="'task-edit-base-branch-' + repoCloneFieldId(repoUrl)"
                    class="text-[11px] text-gray-500 shrink-0"
                  >基准分支</label>
                  <select
                    v-if="shouldUseEditBaseBranchSelect(project, repoUrl)"
                    :id="'task-edit-base-branch-' + repoCloneFieldId(repoUrl)"
                    :value="getEditBaseBranchSelectValue(project, repoUrl)"
                    class="flex-1 min-w-[120px] px-2 py-0.5 border border-gray-300 rounded-md text-xs font-mono focus:outline-none focus:ring-primary focus:border-primary bg-white"
                    @change="onEditBaseBranchSelect(project, repoUrl, $event)"
                  >
                    <option value="" disabled>请选择基准分支</option>
                    <option
                      v-for="branch in getRepoBranches(project.project_id, repoUrl)"
                      :key="branch"
                      :value="branch"
                    >
                      {{ branch }}
                    </option>
                    <option value="__custom__">手动输入...</option>
                  </select>
                  <input
                    v-else
                    :id="'task-edit-base-branch-' + repoCloneFieldId(repoUrl)"
                    :value="getRepoBranchValue(project, repoUrl)"
                    type="text"
                    :list="datalistListAttr('task-edit-base-branch-' + repoCloneFieldId(repoUrl), editBaseBranchDatalistId(project, repoUrl))"
                    class="flex-1 min-w-[120px] px-2 py-0.5 border border-gray-300 rounded-md text-xs font-mono focus:outline-none focus:ring-primary focus:border-primary"
                    placeholder="输入或选择基准分支"
                    @focus="restoreDatalistOnFocus"
                    @input="onEditBaseBranchInput(project, repoUrl, $event)"
                    @change="onEditBaseBranchChange(project, repoUrl, $event)"
                  >
                  <datalist
                    v-if="!shouldUseEditBaseBranchSelect(project, repoUrl)"
                    :id="editBaseBranchDatalistId(project, repoUrl)"
                  >
                    <option
                      v-for="branch in getRepoBranches(project.project_id, repoUrl)"
                      :key="branch"
                      :value="branch"
                    />
                  </datalist>
                  <button
                    v-if="isEditBranchCustomInput(project.project_id, repoUrl)"
                    type="button"
                    class="px-1.5 py-0 text-[11px] text-gray-600 hover:bg-gray-100 rounded"
                    @click="exitEditBranchCustomInput(project.project_id, repoUrl)"
                  >
                    返回列表
                  </button>
                  <button
                    type="button"
                    class="px-1.5 py-0 text-[11px] text-primary hover:bg-primary/5 rounded"
                    :disabled="isLoadingRepoBranches(project.project_id, repoUrl)"
                    @click="emit('fetch-repo-branches', project.project_id, repoUrl, true)"
                  >
                    <span v-if="isLoadingRepoBranches(project.project_id, repoUrl)" class="inline-block animate-spin h-3 w-3 border-2 border-primary border-t-transparent rounded-full"></span>
                    刷新分支
                  </button>
                </div>
                <p
                  v-if="getRepoBranchError(project.project_id, repoUrl)"
                  class="mt-1 text-[11px] text-red-600 leading-snug"
                  data-testid="task-edit-repo-branch-error"
                >
                  {{ getRepoBranchError(project.project_id, repoUrl) }}
                </p>
                <div
                  v-if="shouldShowEditRepoOAuthAuthorize(project.project_id, repoUrl)"
                  class="mt-1 flex flex-wrap items-center gap-1.5"
                >
                  <button
                    type="button"
                    class="text-[11px] px-2 py-0.5 border border-primary text-primary rounded bg-white hover:bg-primary/5 disabled:opacity-50"
                    data-testid="task-edit-repo-oauth-btn"
                    :disabled="editRepoOAuthActionLoadingByUrl[normalizeRepoUrlKey(repoUrl)]"
                    @click="startEditRepoOAuthConnect(repoUrl)"
                  >
                    {{ editRepoOAuthActionLoadingByUrl[normalizeRepoUrlKey(repoUrl)] ? '跳转中…' : 'OAuth 授权' }}
                  </button>
                  <span class="text-[11px] text-gray-500">
                    完成 {{ resolveRepoOAuthAuthorizeLabel(repoUrl) }} 授权后，请点击「刷新分支」
                  </span>
                </div>
              </div>
            </div>
          </div>
          <div v-else class="ml-1 text-[11px] text-gray-400 leading-tight">
            该项目暂无仓库
          </div>
        </div>
      </div>
      <div v-else class="mb-2">
        <button
          type="button"
          class="text-xs px-2 py-1 rounded border border-primary text-primary bg-white hover:bg-primary/5"
          data-testid="task-linked-project-add-btn"
          @click="emit('add-project-association')"
        >
          选择关联项目
        </button>
      </div>
      <p v-if="workspaceProjectsLoading" class="text-[11px] text-gray-500 mt-1">正在加载项目列表...</p>
      <p
        v-else-if="!workspaceProjectsLoading && workspaceProjects.length === 0"
        class="text-[11px] text-amber-700 mt-1"
      >
        当前工作空间暂无可用项目，请先在项目列表中创建项目。
      </p>
  </div>
</template>

<script setup>
import { resolveRepoOAuthAuthorizeLabel } from '../../utils/repoOAuthAuthorizeUtils.js'
import { isHttpRepoUrl } from '../../utils/repoUrl.js'
import { useLinkedProjectsEditBranches } from '../../composables/taskDetail/useLinkedProjectsEditBranches.js'

const props = defineProps({
  editingTask: { type: Object, required: true },
  workspaceProjects: { type: Array, required: true },
  workspaceProjectsLoading: { type: Boolean, required: true },
  getProjectRepos: { type: Function, required: true },
  getRepoBranchValue: { type: Function, required: true },
  getRepoBranches: { type: Function, required: true },
  getRepoBranchError: { type: Function, required: true },
  isLoadingRepoBranches: { type: Function, required: true },
  repoCloneFieldId: { type: Function, required: true },
  gitOAuthCatalogVersion: { type: Number, default: 0 },
  editRepoOAuthActionLoadingByUrl: { type: Object, required: true },
  shouldShowEditRepoOAuthAuthorize: { type: Function, required: true },
  startEditRepoOAuthConnect: { type: Function, required: true },
  normalizeRepoUrlKey: { type: Function, required: true },
})

const emit = defineEmits([
  'project-change',
  'set-repo-branch',
  'fetch-repo-branches',
  'add-project-association',
])

const {
  editBaseBranchDatalistId,
  isEditBranchCustomInput,
  shouldUseEditBaseBranchSelect,
  getEditBaseBranchSelectValue,
  exitEditBranchCustomInput,
  onEditBaseBranchSelect,
  datalistListAttr,
  restoreDatalistOnFocus,
  onEditBaseBranchInput,
  onEditBaseBranchChange,
} = useLinkedProjectsEditBranches({ props, emit })
</script>

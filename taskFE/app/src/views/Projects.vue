<template>
  <div data-alias="view-projects-list" class="min-h-[calc(100vh-8rem)] bg-gray-50 p-6 sm:p-8">
    <div class="max-w-7xl mx-auto">
      <header class="mb-6">
        <h1 class="text-[clamp(1.5rem,3vw,2.25rem)] font-bold text-text tracking-tight">项目列表</h1>
        <p class="mt-1 text-sm text-text-light">
          在此租户下查看与管理项目；点击进入详情或创建新项目。
        </p>
        <p v-if="!loading && !error && projects.length > 0" class="mt-2 text-xs text-text-light">
          共 {{ projects.length }} 个项目
          <template v-if="hasActiveFilters && filteredProjects.length !== projects.length">
            ，筛选后 {{ filteredProjects.length }} 个
          </template>
        </p>
      </header>

      <div v-if="loading" class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6" aria-busy="true" aria-label="加载项目中">
        <div v-for="n in 6" :key="'sk-' + n" class="rounded-xl border border-gray-100 bg-white p-6 shadow-sm animate-pulse">
          <div class="flex justify-between gap-3 mb-4">
            <div class="h-6 flex-1 rounded bg-gray-200" />
            <div class="h-6 w-16 rounded-full bg-gray-200" />
          </div>
          <div class="space-y-2 mb-4">
            <div class="h-4 rounded bg-gray-100" />
            <div class="h-4 w-5/6 rounded bg-gray-100" />
          </div>
          <div class="h-12 rounded-lg bg-gray-100 mb-4" />
          <div class="flex justify-between mt-4">
            <div class="h-4 w-32 rounded bg-gray-100" />
            <div class="h-4 w-16 rounded bg-gray-100" />
          </div>
        </div>
      </div>

      <div v-else-if="error" class="text-center py-14 px-4 bg-white rounded-xl shadow-md border border-gray-100">
        <div class="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-red-50 text-red-600">
          <svg class="h-6 w-6" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
          </svg>
        </div>
        <p
          class="text-red-600 mb-6 font-medium"
          v-bind="errorTraceId ? { 'data-traceId': errorTraceId } : {}"
        >{{ error }}</p>
        <div v-if="otherAccounts.length > 0" class="mb-6 text-left max-w-sm mx-auto">
          <p class="text-sm text-gray-600 mb-3">您有已保存的其他账号，可尝试切换：</p>
          <div class="space-y-2">
            <button
              v-for="acc in otherAccounts"
              :key="acc.userId"
              type="button"
              class="w-full text-left px-3 py-2 border border-blue-200 rounded-md hover:bg-blue-50 text-sm flex items-center gap-2 disabled:opacity-50"
              :disabled="accountSwitching"
              @click="switchToAccount(acc)"
            >
              <span class="font-medium text-blue-700">{{ acc.username || '用户 ' + acc.userId }}</span>
              <span class="text-gray-400 text-xs ml-auto">{{ accountSwitching ? '切换中…' : '切换到此账号' }}</span>
            </button>
          </div>
        </div>
        <button type="button" @click="loadProjects" class="btn-primary retry-button">重试</button>
      </div>

      <div v-else-if="projects.length === 0" class="text-center py-16 px-6 bg-white rounded-xl shadow-md border border-gray-100 max-w-lg mx-auto">
        <div class="mx-auto mb-5 flex h-14 w-14 items-center justify-center rounded-2xl bg-primary/10 text-primary">
          <svg class="h-8 w-8" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
          </svg>
        </div>
        <p class="text-text font-semibold text-lg mb-2">还没有项目</p>
        <p class="text-text-light text-sm mb-8 leading-relaxed">创建第一个项目后，即可关联镜像、任务与工作区；也可从 GitLab 同步已有仓库。</p>
        <div class="flex flex-col sm:flex-row items-center justify-center gap-3">
          <button
            type="button"
            class="btn-secondary inline-flex items-center gap-2"
            data-testid="projects-gitlab-sync-btn-empty"
            @click="openGitlabSync"
          >
            从 GitLab 同步
          </button>
          <a :href="tenantPath + '/create-project/'" class="btn-primary create-first-project-button inline-flex items-center gap-2">
          <svg class="w-5 h-5 opacity-90" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
          </svg>
          创建第一个项目
        </a>
        </div>
      </div>

      <template v-else>
        <div
          v-if="batchDeleteResult?.deleted?.length"
          class="mb-4 rounded-lg border border-green-200 bg-green-50 p-3 text-sm text-green-800"
        >
          已删除 {{ batchDeleteResult.deleted.length }} 个项目
          <template v-if="batchDeleteResult.errors?.length">
            ，{{ batchDeleteResult.errors.length }} 个失败
          </template>
        </div>

        <div class="mb-6 flex flex-row flex-nowrap items-center gap-3">
          <div class="relative min-w-0 flex-1">
            <label for="projects-search" class="sr-only">搜索项目</label>
            <span class="pointer-events-none absolute inset-y-0 left-3 flex items-center text-text-light">
              <svg class="h-5 w-5 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
              </svg>
            </span>
            <input
              id="projects-search"
              v-model="searchQuery"
              type="search"
              autocomplete="off"
              placeholder="按名称或描述筛选…"
              class="w-full min-w-0 rounded-lg border border-gray-200 bg-white py-2.5 pl-10 pr-3 text-sm text-text placeholder:text-gray-400 shadow-sm focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/20"
            >
          </div>
          <button
            v-if="!selectionMode"
            type="button"
            class="btn-secondary inline-flex shrink-0 items-center justify-center gap-2 whitespace-nowrap"
            :disabled="loading"
            data-testid="projects-select-mode-btn"
            @click="toggleSelectionMode"
          >
            选择
          </button>
          <template v-else>
            <label class="inline-flex shrink-0 items-center gap-2 text-sm text-text cursor-pointer whitespace-nowrap">
              <input
                type="checkbox"
                class="rounded border-gray-300 text-primary focus:ring-primary"
                :checked="allFilteredChecked"
                :disabled="!filteredProjects.length"
                data-testid="projects-select-all-filtered"
                @change="toggleSelectAllFiltered($event.target.checked)"
              >
              全选当前列表
            </label>
            <span class="text-xs text-text-light shrink-0 whitespace-nowrap">已选 {{ selectedCount }} 个</span>
            <button
              type="button"
              class="btn-secondary inline-flex shrink-0 items-center justify-center gap-2 whitespace-nowrap"
              data-testid="projects-cancel-select-btn"
              @click="exitSelectionMode"
            >
              取消选择
            </button>
            <button
              type="button"
              class="inline-flex shrink-0 items-center justify-center gap-2 whitespace-nowrap px-4 py-2 text-sm font-medium rounded-lg border border-red-200 text-red-600 bg-white hover:bg-red-50 disabled:opacity-50"
              :disabled="selectedCount === 0"
              data-testid="projects-batch-delete-btn"
              @click="openBatchDeleteConfirm"
            >
              批量删除（{{ selectedCount }}）
            </button>
          </template>
          <button
            type="button"
            class="btn-secondary inline-flex shrink-0 items-center justify-center gap-2 whitespace-nowrap"
            :disabled="loading"
            data-testid="projects-gitlab-sync-btn"
            @click="openGitlabSync"
          >
            <svg class="h-4 w-4 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
            </svg>
            从 GitLab 同步
          </button>
          <button
            type="button"
            class="btn-secondary inline-flex shrink-0 items-center justify-center gap-2 whitespace-nowrap"
            :disabled="loading"
            @click="loadProjects"
          >
            <svg class="h-4 w-4 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
            </svg>
            刷新
          </button>
          <a
            :href="tenantPath + '/create-project/'"
            class="btn-primary create-project-button inline-flex shrink-0 items-center justify-center gap-2 whitespace-nowrap"
          >
            <svg class="h-5 w-5 shrink-0 opacity-90" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
            </svg>
            创建项目
          </a>
        </div>

        <div
          v-if="availableTags.length"
          class="mb-4 flex flex-wrap items-center gap-2"
          data-testid="projects-tag-filter"
        >
          <span class="text-xs font-medium text-text-light shrink-0">标签筛选</span>
          <button
            v-for="tag in availableTags"
            :key="tag"
            type="button"
            class="inline-flex px-2.5 py-1 rounded-full text-xs font-medium border transition-colors"
            :class="isTagSelected(tag)
              ? 'bg-primary text-white border-primary'
              : 'bg-white text-blue-700 border-blue-100 hover:bg-blue-50'"
            :data-testid="'projects-tag-filter-' + tag"
            :aria-pressed="isTagSelected(tag)"
            @click="toggleTagFilter(tag)"
          >
            {{ tag }}
          </button>
          <button
            v-if="hasActiveFilters"
            type="button"
            class="text-xs font-medium text-primary hover:text-primary-dark shrink-0"
            data-testid="projects-clear-filters-btn"
            @click="clearAllFilters"
          >
            清除筛选
          </button>
        </div>

        <div v-if="filteredProjects.length === 0" class="text-center py-14 px-4 bg-white rounded-xl border border-dashed border-gray-200">
          <p class="text-text-light">{{ emptyFilterMessage }}</p>
          <button type="button" class="mt-4 text-sm font-medium text-primary hover:text-primary-dark" @click="clearAllFilters">
            清除筛选
          </button>
        </div>

        <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          <template v-for="project in filteredProjects" :key="project.id">
            <a
              v-if="!selectionMode"
              :href="tenantPath + '/projects/' + project.id + '/'"
              :data-testid="'project-card-' + project.id"
              class="group flex flex-col rounded-xl border border-gray-100 bg-white shadow-sm overflow-hidden transition-all duration-300 hover:border-primary/25 hover:shadow-lg focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2"
            >
              <div class="p-6 flex flex-col flex-1">
                <ProjectCardBody
                  :project="project"
                  :tenant-path="tenantPath"
                  :status-badge-class="statusBadgeClass(project.status)"
                  :format-date="formatDate"
                  :format-image-label="formatProjectInstalledImageLabel"
                />
              </div>
            </a>
            <div
              v-else
              role="button"
              tabindex="0"
              :data-testid="'project-card-' + project.id"
              class="group flex flex-col rounded-xl border bg-white shadow-sm overflow-hidden transition-all duration-300 cursor-pointer focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2"
              :class="selectedProjectIds[String(project.id)] ? 'border-primary/40 ring-1 ring-primary/20' : 'border-gray-100 hover:border-primary/25 hover:shadow-lg'"
              @click="toggleProject(project.id, !selectedProjectIds[String(project.id)])"
              @keydown.enter.prevent="toggleProject(project.id, !selectedProjectIds[String(project.id)])"
            >
              <div class="p-6 flex flex-col flex-1">
                <label class="inline-flex items-center gap-2 mb-3 cursor-pointer" @click.stop>
                  <input
                    type="checkbox"
                    class="rounded border-gray-300 text-primary focus:ring-primary"
                    :checked="Boolean(selectedProjectIds[String(project.id)])"
                    @change="toggleProject(project.id, $event.target.checked)"
                  >
                  <span class="text-xs text-text-light">选择此项目</span>
                </label>
                <ProjectCardBody
                  :project="project"
                  :tenant-path="tenantPath"
                  :status-badge-class="statusBadgeClass(project.status)"
                  :format-date="formatDate"
                  :format-image-label="formatProjectInstalledImageLabel"
                  :show-detail-link="true"
                />
              </div>
            </div>
          </template>
        </div>
      </template>

      <BatchDeleteProjectsModal
        :show="batchDeleteConfirmShow"
        :project-names="selectedProjectNames"
        :deleting="batchDeleteDeleting"
        :error="batchDeleteError"
        :result="batchDeleteResult"
        @close="closeBatchDeleteConfirm"
        @confirm="executeBatchDelete"
      />

      <GitlabSyncProjectsModal
        v-bind="gitlabSyncModalProps"
        @close="closeGitlabSync"
        @refresh="refreshGitlabSync"
        @create="createGitlabSyncProjects"
        @create-combined="createGitlabSyncCombinedProject"
        @select-region="selectGitlabSyncRegion"
        @toggle-repo="toggleGitlabSyncRepo"
        @toggle-select-all="toggleGitlabSyncSelectAll"
        @update:selected-workspace-id="gitlabSyncWorkspaceId = $event"
        @update:combined-project-name="setGitlabSyncCombinedName"
      />
    </div>
  </div>
</template>

<script setup>
/* @alias:view-projects-list */
import { onMounted, computed, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import GitlabSyncProjectsModal from '../components/GitlabSyncProjectsModal.vue'
import BatchDeleteProjectsModal from '../components/BatchDeleteProjectsModal.vue'
import ProjectCardBody from '../components/ProjectCardBody.vue'
import { useGitlabProjectSync } from '../composables/useGitlabProjectSync.js'
import { useProjectsBatchDelete } from '../composables/useProjectsBatchDelete.js'
import { useProjectsListLoad } from '../composables/useProjectsListLoad.js'
import { useProjectsListFilters } from '../composables/useProjectsListFilters.js'

const route = useRoute()
const router = useRouter()

const {
  projects,
  loading,
  error,
  errorTraceId,
  otherAccounts,
  accountSwitching,
  switchToAccount,
  getTenantFromRoute,
  effectiveTenantId,
  tenantPath,
  fetchCurrentTenant,
  loadProjects,
} = useProjectsListLoad({ route, router })

const {
  searchQuery,
  selectedTagFilters,
  filteredProjects,
  availableTags,
  hasActiveFilters,
  emptyFilterMessage,
  isTagSelected,
  toggleTagFilter,
  clearAllFilters,
  applyTagsQueryFromRoute,
} = useProjectsListFilters({ route, router, projects })

const {
  showModal: gitlabSyncShow,
  loading: gitlabSyncLoading,
  creating: gitlabSyncCreating,
  error: gitlabSyncError,
  errorTraceId: gitlabSyncErrorTraceId,
  oauthBound: gitlabSyncOauthBound,
  gitlabWebsite: gitlabSyncWebsite,
  gitlabLogin: gitlabSyncLogin,
  oauthStartHref: gitlabSyncOAuthStartHref,
  purchasedRegions: gitlabSyncPurchasedRegions,
  selectedRegionSlug: gitlabSyncSelectedRegionSlug,
  regionsLoading: gitlabSyncRegionsLoading,
  repos: gitlabSyncRepos,
  selectedCount: gitlabSyncSelectedCount,
  batchEligibleCount: gitlabSyncBatchEligibleCount,
  allSelectableChecked: gitlabSyncAllChecked,
  selectedRepoKeys: gitlabSyncSelectedKeys,
  workspaces: gitlabSyncWorkspaces,
  selectedWorkspaceId: gitlabSyncWorkspaceId,
  createResult: gitlabSyncCreateResult,
  combinedProjectName: gitlabSyncCombinedName,
  openModal: openGitlabSyncModal,
  closeModal: closeGitlabSync,
  loadRemoteRepos: refreshGitlabSync,
  selectRegion: selectGitlabSyncRegion,
  toggleRepo: toggleGitlabSyncRepo,
  toggleSelectAll: toggleGitlabSyncSelectAll,
  createSelectedProjects: createGitlabSyncProjects,
  createCombinedProject: createGitlabSyncCombinedProject,
  setCombinedProjectName: setGitlabSyncCombinedName,
} = useGitlabProjectSync({
  tenantId: effectiveTenantId,
  onProjectsCreated: async () => {
    await loadProjects()
  },
})

const gitlabSyncModalProps = computed(() => ({
  show: gitlabSyncShow.value,
  loading: gitlabSyncLoading.value,
  creating: gitlabSyncCreating.value,
  error: gitlabSyncError.value,
  errorTraceId: gitlabSyncErrorTraceId.value,
  oauthBound: gitlabSyncOauthBound.value,
  gitlabWebsite: gitlabSyncWebsite.value,
  gitlabLogin: gitlabSyncLogin.value,
  oauthStartHref: gitlabSyncOAuthStartHref.value,
  purchasedRegions: gitlabSyncPurchasedRegions.value,
  selectedRegionSlug: gitlabSyncSelectedRegionSlug.value,
  regionsLoading: gitlabSyncRegionsLoading.value,
  repos: gitlabSyncRepos.value,
  selectedCount: gitlabSyncSelectedCount.value,
  batchEligibleCount: gitlabSyncBatchEligibleCount.value,
  allSelectableChecked: gitlabSyncAllChecked.value,
  selectedRepoKeys: gitlabSyncSelectedKeys.value,
  workspaces: gitlabSyncWorkspaces.value,
  selectedWorkspaceId: gitlabSyncWorkspaceId.value,
  createResult: gitlabSyncCreateResult.value,
  combinedProjectName: gitlabSyncCombinedName.value,
}))

const openGitlabSync = () => {
  openGitlabSyncModal()
}

const {
  selectionMode,
  selectedProjectIds,
  showConfirmModal: batchDeleteConfirmShow,
  deleting: batchDeleteDeleting,
  error: batchDeleteError,
  deleteResult: batchDeleteResult,
  selectedCount,
  selectedProjects,
  allFilteredChecked,
  toggleSelectionMode,
  exitSelectionMode,
  toggleProject,
  toggleSelectAllFiltered,
  openBatchDeleteConfirm,
  closeConfirmModal: closeBatchDeleteConfirm,
  executeBatchDelete,
} = useProjectsBatchDelete({
  tenantId: effectiveTenantId,
  projects,
  filteredProjects,
  onDeleted: async () => {
    await loadProjects()
  },
})

const selectedProjectNames = computed(() =>
  selectedProjects.value.map((p) => p.name || String(p.id)),
)

const statusBadgeClass = (status) => {
  const s = String(status || '进行中').trim()
  if (/完成|结束|closed|done/i.test(s)) return 'bg-slate-100 text-slate-700'
  if (/暂停|停止|pause/i.test(s)) return 'bg-amber-100 text-amber-800'
  if (/进行|活跃|active|open/i.test(s)) return 'bg-green-100 text-green-800'
  return 'bg-gray-100 text-gray-700'
}

const formatProjectInstalledImageLabel = (value) => {
  if (value == null || value === '') return ''
  if (typeof value === 'string') return value
  const name = value.name
  const tag = value.tag
  if (name != null && tag != null && tag !== '') return `${name}:${tag}`
  if (name != null) return String(name)
  return ''
}

const formatDate = (dateString) => {
  if (!dateString) return '未知'
  return new Date(dateString).toLocaleString('zh-CN')
}

watch(
  () => route.params.tenant,
  (newTenant) => {
    if (newTenant) loadProjects()
  },
)

watch(
  () => route.query.tags,
  (raw) => {
    applyTagsQueryFromRoute(raw)
  },
)

onMounted(async () => {
  if (!getTenantFromRoute()) await fetchCurrentTenant()
  loadProjects()
})
</script>



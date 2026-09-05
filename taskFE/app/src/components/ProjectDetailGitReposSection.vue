<template>
  <div class="col-span-2" data-testid="project-detail-git-repos-section">
    <div class="flex justify-between items-center mb-1">
      <label class="block text-sm font-medium text-gray-700">Git 仓库</label>
      <button
        type="button"
        class="text-xs px-3 py-1 rounded border border-blue-200 text-blue-600 hover:bg-blue-50 disabled:opacity-50 disabled:cursor-not-allowed"
        :disabled="branchPreviewLoading"
        @click="fetchProjectRepoBranchesPreview"
      >
        {{ branchPreviewLoading ? '查询中...' : '分支列表预览' }}
      </button>
    </div>
    <div class="space-y-2">
      <template v-if="projectGitRepoEntries.length">
        <div
          v-for="(entry, idx) in projectGitRepoEntries"
          :key="idx"
          class="rounded-md border border-gray-100 bg-gray-50/60 px-3 py-2"
          data-testid="project-detail-git-repo-row"
        >
          <div class="flex flex-wrap items-center gap-x-2 gap-y-1">
            <span
              class="inline-flex items-center px-2 py-0.5 text-xs font-medium rounded border shrink-0"
              :class="repoOAuthStatusBadgeClass(entry.url)"
              :data-testid="`git-repo-oauth-status-${idx}`"
            >
              {{ repoOAuthStatusLabel(entry.url) }}
            </span>
            <span
              v-if="entry.cloneAlias"
              class="inline-flex items-center px-2 py-0.5 text-xs rounded bg-white border border-gray-200 text-gray-700 shrink-0"
              data-testid="git-repo-clone-alias-display"
            >
              别名：{{ entry.cloneAlias }}
            </span>
            <span
              v-if="typeof entry.diskSizeBytes === 'number'"
              class="inline-flex items-center px-2 py-0.5 text-xs rounded bg-white border border-gray-200 text-gray-700 shrink-0"
              data-testid="git-repo-disk-size"
            >
              磁盘：{{ formatDiskSizeBytes(entry.diskSizeBytes) }}
            </span>
            <button
              v-if="shouldShowRepoOAuthButton(entry.url)"
              type="button"
              class="text-xs px-2 py-0.5 border border-primary text-primary rounded bg-white hover:bg-primary/5 disabled:opacity-50 disabled:cursor-not-allowed shrink-0"
              :disabled="isRepoOAuthActionDisabled(entry.url)"
              :data-testid="`git-repo-oauth-action-${idx}`"
              @click="startRepoOAuthConnect(entry.url)"
            >
              {{ repoOAuthButtonLabel(entry.url) }}
            </button>
          </div>
          <a
            :href="entry.url"
            target="_blank"
            rel="noopener noreferrer"
            class="mt-1 block text-sm text-blue-600 hover:underline break-all"
            data-testid="git-repo-url-link"
          >
            {{ entry.url }}
          </a>
          <p
            v-if="repoOAuthErrorByUrl(entry.url)"
            class="mt-1 text-xs text-red-600"
            role="alert"
            data-testid="git-repo-validate-error"
            :data-traceId="repoOAuthErrorTraceIdByUrl(entry.url) || undefined"
          >
            {{ repoOAuthErrorByUrl(entry.url) }}
          </p>
          <p
            v-else-if="gitRepoOAuthStatusHint(repoOAuthTokenStatus(entry.url))"
            class="mt-1 text-xs text-red-600"
            :data-testid="`git-repo-oauth-hint-${idx}`"
          >
            {{ gitRepoOAuthStatusHint(repoOAuthTokenStatus(entry.url)) }}
          </p>
        </div>
      </template>
      <p v-else class="text-gray-500">未设置</p>
    </div>
    <div v-if="branchPreviewError" class="mt-2 text-xs text-red-600">
      {{ branchPreviewError }}
    </div>
    <div v-if="repoBranchPreviews.length" class="mt-3 space-y-3">
      <div
        v-for="repoPreview in repoBranchPreviews"
        :key="repoPreview.repoUrl"
        class="p-3 bg-gray-50 rounded border border-gray-100"
      >
        <p class="text-xs text-gray-700 break-all">{{ repoPreview.repoUrl }}</p>
        <p v-if="repoPreview.error" class="mt-1 text-xs text-red-600" :data-traceId="repoPreview.traceId || undefined">
          {{ repoPreview.error }}
        </p>
        <div v-else-if="repoPreview.branches.length" class="mt-2 flex flex-wrap gap-2">
          <span
            v-for="branch in repoPreview.branches"
            :key="`${repoPreview.repoUrl}-${branch}`"
            class="inline-flex items-center px-2 py-0.5 text-xs rounded bg-white border border-gray-200 text-gray-700"
          >
            {{ branch }}
          </span>
        </div>
        <p v-else class="mt-1 text-xs text-gray-500">未查询到分支</p>
      </div>
    </div>

    <div class="mt-4" data-testid="project-detail-nested-git-repos">
      <div class="flex justify-between items-center mb-1">
        <label class="block text-sm font-medium text-gray-700">子 Git 仓库</label>
        <button
          type="button"
          class="text-xs px-3 py-1 rounded border border-gray-200 text-gray-600 hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
          :disabled="nestedLoading || !primaryRepoUrl"
          data-testid="project-detail-nested-git-repos-refresh"
          @click="fetchNestedGitRepos"
        >
          {{ nestedLoading ? '查询中...' : '刷新' }}
        </button>
      </div>
      <label
        v-if="primaryRepoUrl"
        class="mb-2 flex items-start gap-2 text-sm text-gray-700 cursor-pointer"
        data-testid="project-detail-auto-clone-nested-repos-label"
      >
        <input
          type="checkbox"
          class="mt-0.5"
          data-testid="project-detail-auto-clone-nested-repos"
          :checked="autoCloneNestedRepos"
          :disabled="autoCloneSaving || autoCloneToggleGuard.isBusy()"
          @change="onAutoCloneToggle"
        >
        <span>
          自动克隆子仓库
          <span class="block text-xs text-gray-500 font-normal">
            控制容器启动时是否根据 `.gitmodules` 克隆子仓库（不影响上方发现列表）。
          </span>
        </span>
      </label>
      <p
        v-if="autoCloneSaveError"
        class="mb-2 text-xs text-red-600"
        role="alert"
        data-testid="project-detail-auto-clone-nested-repos-error"
        :data-traceId="autoCloneSaveErrorTraceId || undefined"
      >
        {{ autoCloneSaveError }}
      </p>
      <p v-if="!primaryRepoUrl" class="text-gray-500 text-sm">未设置父仓库，无法发现子仓库</p>
      <p v-else-if="nestedLoading" class="text-gray-500 text-sm">正在发现子 Git 仓库...</p>
      <p
        v-else-if="nestedError"
        class="text-xs text-red-600"
        data-testid="project-detail-nested-git-repos-error"
        :data-traceId="nestedErrorTraceId || undefined"
      >
        {{ nestedError }}
      </p>
      <div v-else-if="sortedNestedRepos.length" class="space-y-2">
        <p
          v-if="!autoCloneNestedRepos"
          class="text-xs text-amber-700"
          data-testid="project-detail-auto-clone-nested-repos-disabled-hint"
        >
          已关闭自动克隆：容器将仅克隆父仓库，下列子仓不会在 bootstrap 时拉取。
        </p>
        <div
          v-for="(row, idx) in sortedNestedRepos"
          :key="`${row.path}-${idx}`"
          class="rounded-md border border-gray-100 bg-white px-3 py-2"
          data-testid="project-detail-nested-git-repo-row"
        >
          <div class="flex flex-wrap items-center gap-x-2 gap-y-1">
            <span
              v-if="row.url"
              class="inline-flex items-center px-2 py-0.5 text-xs font-medium rounded border shrink-0"
              :class="repoOAuthStatusBadgeClass(row.url)"
              :data-testid="`nested-git-repo-oauth-status-${idx}`"
            >
              {{ repoOAuthStatusLabel(row.url) }}
            </span>
            <span
              class="inline-flex items-center px-2 py-0.5 text-xs font-medium rounded border border-gray-200 bg-gray-50 text-gray-700 shrink-0"
            >
              {{ row.path }}
            </span>
            <span
              v-if="row.source === 'gitmodules'"
              class="inline-flex items-center px-2 py-0.5 text-xs rounded bg-gray-50 border border-gray-100 text-gray-500 shrink-0"
              data-testid="project-detail-nested-git-repo-submodule-badge"
            >
              submodule
            </span>
            <button
              v-if="row.url && shouldShowRepoOAuthButton(row.url)"
              type="button"
              class="text-xs px-2 py-0.5 border border-primary text-primary rounded bg-white hover:bg-primary/5 disabled:opacity-50 disabled:cursor-not-allowed shrink-0"
              :disabled="isRepoOAuthActionDisabled(row.url)"
              data-testid="nested-git-repo-oauth-button"
              @click="startRepoOAuthConnect(row.url)"
            >
              {{ repoOAuthButtonLabel(row.url) }}
            </button>
          </div>
          <a
            v-if="row.url"
            :href="row.url"
            target="_blank"
            rel="noopener noreferrer"
            class="mt-1 block text-sm text-blue-600 hover:underline break-all"
            :title="row.url"
          >
            {{ row.url }}
          </a>
          <span v-else class="mt-1 block text-xs text-gray-500">{{ row.resolve_error || '无法推导远程地址' }}</span>
          <p
            v-if="row.url && repoOAuthErrorByUrl(row.url)"
            class="mt-1 text-xs text-red-600"
            :data-traceId="repoOAuthErrorTraceIdByUrl(row.url) || undefined"
          >
            {{ repoOAuthErrorByUrl(row.url) }}
          </p>
          <p
            v-else-if="row.url && gitRepoOAuthStatusHint(repoOAuthTokenStatus(row.url))"
            class="mt-1 text-xs text-red-600"
            :data-testid="`nested-git-repo-oauth-hint-${idx}`"
          >
            {{ gitRepoOAuthStatusHint(repoOAuthTokenStatus(row.url)) }}
          </p>
        </div>
      </div>
      <template v-else>
        <p class="text-gray-500 text-sm" data-testid="project-detail-nested-git-repos-empty">
          未发现子仓库
        </p>
        <p
          v-if="!autoCloneNestedRepos"
          class="mt-1 text-xs text-amber-700"
          data-testid="project-detail-auto-clone-nested-repos-disabled-hint"
        >
          已关闭自动克隆子仓库，容器将仅克隆父仓库。
        </p>
      </template>
    </div>
  </div>
</template>

<script setup>
import { computed, ref, toRef, watch, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { apiFetch } from '../utils/apiUtils.js'
import { createClickGuard, mergeIdempotencyHeaders } from '../utils/clickGuard.js'
import { extractTraceId } from '../utils/traceId.js'
import { applyOAuthCallbackFromRoute } from '../utils/gitSiteOAuthCallbackUtils.js'
import { useProjectDetailGitRepos } from '../composables/useProjectDetailGitRepos.js'
import { useProjectNestedGitRepos } from '../composables/useProjectNestedGitRepos.js'
import {
  GIT_REPO_OAUTH_STATUS,
  gitRepoOAuthStatusHint,
  sortReposByOAuthAttention,
} from '../utils/gitRepoOAuthStatusUtils.js'
import {
  isAutoCloneNestedReposEnabled,
  resolveAutoRunGitAuthBlock,
} from '../utils/projectDetailAutoRunGitGate.js'
import { formatDiskSizeBytes } from '../utils/formatDiskSizeBytes.js'

const props = defineProps({
  project: { type: Object, required: true },
})

const emit = defineEmits(['oauth-callback-success', 'auto-run-git-gate', 'project-updated'])

const projectRef = toRef(props, 'project')
const route = useRoute()
const autoCloneSaving = ref(false)
const autoCloneSaveError = ref('')
const autoCloneSaveErrorTraceId = ref('')
// OPT-20260819-038: 自动克隆开关切换是写操作，防连点/超时重试双发 PUT
const autoCloneToggleGuard = createClickGuard()

const autoCloneNestedRepos = computed(() =>
  isAutoCloneNestedReposEnabled(props.project?.auto_clone_nested_repos),
)

const onAutoCloneToggle = async (event) => {
  const next = !!event?.target?.checked
  // OPT-20260819-038: 自动克隆开关切换是写操作，防连点/超时重试双发 PUT
  await autoCloneToggleGuard.run(async ({ idempotencyKey }) => {
    autoCloneSaveError.value = ''
    autoCloneSaveErrorTraceId.value = ''
    autoCloneSaving.value = true
    try {
      const tenantId = String(route.params.tenant || '')
      const projectId = String(props.project?.id || route.params.id || '')
      const response = await apiFetch(`/api/projects/tenant_id/${tenantId}/${projectId}/`, {
        method: 'PUT',
        headers: mergeIdempotencyHeaders({ 'Content-Type': 'application/json' }, idempotencyKey),
        credentials: 'include',
        body: JSON.stringify({ auto_clone_nested_repos: next }),
      })
      if (!response.ok) {
        autoCloneSaveErrorTraceId.value = extractTraceId(response) || ''
        const errData = response._errorData ?? (await response.json().catch(() => ({})))
        autoCloneSaveError.value = errData.error || errData.detail || errData.message || '保存失败'
        event.target.checked = !next
        return
      }
      const updated = await response.json().catch(() => ({}))
      if (props.project && typeof props.project === 'object') {
        props.project.auto_clone_nested_repos = updated.auto_clone_nested_repos ?? next
      }
      emit('project-updated', updated)
    } catch (err) {
      autoCloneSaveError.value = '网络错误，请重试'
      autoCloneSaveErrorTraceId.value = extractTraceId(err) || ''
      if (event?.target) event.target.checked = !next
    } finally {
      autoCloneSaving.value = false
    }
  })
}
const {
  router,
  projectGitRepoList,
  projectGitRepoEntries,
  branchPreviewLoading,
  branchPreviewError,
  repoBranchPreviews,
  fetchProjectRepoBranchesPreview,
  bootstrapGitOAuthCatalog,
  applyGitReposStatusFromApi,
  refreshGitReposOAuthStatus,
  isRepoOAuthActionDisabled,
  repoOAuthButtonLabel,
  shouldShowRepoOAuthButton,
  repoOAuthErrorByUrl,
  repoOAuthErrorTraceIdByUrl,
  repoOAuthTokenStatus,
  repoOAuthStatusLabel,
  repoOAuthStatusBadgeClass,
  startRepoOAuthConnect,
} = useProjectDetailGitRepos({ project: projectRef })

const primaryRepoUrl = computed(() => {
  const entries = projectGitRepoEntries.value
  if (entries.length) return entries[0].url
  return projectGitRepoList.value[0] || ''
})

const {
  nestedRepos,
  nestedLoading,
  nestedError,
  nestedErrorTraceId,
  fetchNestedGitRepos,
} = useProjectNestedGitRepos({
  tenantId: () => route.params.tenant,
  projectId: () => route.params.id || props.project?.id,
  repoUrl: primaryRepoUrl,
  auto: true,
})

const nestedRepoTokenStatus = (row) => {
  const url = String(row?.url || '').trim()
  if (!url) return GIT_REPO_OAUTH_STATUS.NOT_APPLICABLE
  return repoOAuthTokenStatus(url)
}

const sortedNestedRepos = computed(() =>
  sortReposByOAuthAttention(nestedRepos.value, nestedRepoTokenStatus),
)

const collectedParentOAuthTokenStatuses = computed(() => {
  const statuses = []
  for (const entry of projectGitRepoEntries.value) {
    const url = String(entry?.url || '').trim()
    if (url) statuses.push(repoOAuthTokenStatus(url))
  }
  return statuses
})

const collectedNestedOAuthTokenStatuses = computed(() => {
  const statuses = []
  for (const row of nestedRepos.value) {
    const url = String(row?.url || '').trim()
    if (url) statuses.push(repoOAuthTokenStatus(url))
  }
  return statuses
})

const autoRunGitGate = computed(() =>
  resolveAutoRunGitAuthBlock({
    tokenStatuses: collectedParentOAuthTokenStatuses.value,
    nestedTokenStatuses: collectedNestedOAuthTokenStatuses.value,
    nestedError: nestedError.value,
    nestedErrorTraceId: nestedErrorTraceId.value,
    nestedLoading: nestedLoading.value,
    autoCloneNestedRepos: autoCloneNestedRepos.value,
  }),
)

watch(
  autoRunGitGate,
  (gate) => {
    emit('auto-run-git-gate', gate)
  },
  { immediate: true, deep: true },
)

watch(
  () => ({
    repos: Array.isArray(props.project?.git_repos) ? props.project.git_repos : [],
    entries: Array.isArray(props.project?.git_repo_entries) ? props.project.git_repo_entries : [],
    status: Array.isArray(props.project?.git_repos_status) ? props.project.git_repos_status : [],
  }),
  ({ repos, status }) => {
    if (Array.isArray(status) && status.length) {
      applyGitReposStatusFromApi(status)
    }
    void refreshGitReposOAuthStatus((repos || []).filter(Boolean))
  },
  { immediate: true, deep: true },
)

watch(
  nestedRepos,
  (rows) => {
    const urls = (Array.isArray(rows) ? rows : [])
      .map((row) => String(row?.url || '').trim())
      .filter(Boolean)
    if (urls.length) {
      void refreshGitReposOAuthStatus(urls)
    }
  },
  { immediate: true, deep: true },
)

onMounted(async () => {
  await bootstrapGitOAuthCatalog()
  await refreshGitReposOAuthStatus(projectGitRepoList.value)
  const oauthResult = await applyOAuthCallbackFromRoute(route, router)
  if (oauthResult && oauthResult.severity === 'success') {
    await refreshGitReposOAuthStatus(projectGitRepoList.value, { force: true })
    emit('oauth-callback-success')
  }
})
</script>

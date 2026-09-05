<template>
  <div class="detail-header flex justify-between items-center mb-3 flex-wrap gap-2">
    <div id="task-detail-title-row" class="flex items-center gap-3 flex-wrap">
      <h1 id="task-detail-title" class="text-xl font-bold text-gray-900">任务详情</h1>
      <button
        v-if="hasTask"
        id="task-fork-btn"
        type="button"
        :disabled="isForking"
        class="inline-flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md shadow-sm hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-primary disabled:opacity-50 disabled:cursor-not-allowed"
        title="复制当前任务属性并创建新任务（进度从第一列开始，记录 fork_from）"
        @click="openForkConfirm"
      >
        <!-- Anti-Replay-OK: ui-only 仅打开确认模态，不发写请求 -->
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"
          />
        </svg>
        Fork
      </button>
      <button
        v-if="hasTask && !isEditing"
        id="task-edit-btn"
        type="button"
        class="inline-flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md shadow-sm hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-primary"
        title="编辑任务信息"
        @click="emit('start-edit')"
      >
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
        </svg>
        编辑
      </button>
      <!-- Anti-Replay-OK: GET list; fetch only when opened -->
      <EntityRevisionPanel
        v-if="hasTask && revisionListUrl"
        :list-url="revisionListUrl"
        title-field="title"
      />
    </div>
    <template v-if="!isEditing">
      <router-link id="back-to-work-panel" :to="backToWorkPanelRoute" class="px-4 py-2 bg-primary hover:bg-primary-dark text-white rounded-md text-sm font-medium">
        返回工作面板
      </router-link>
    </template>
    <div v-else class="fixed bottom-4 right-4 z-50 flex flex-col items-end gap-2">
      <div
        v-if="editError"
        class="max-w-sm rounded-lg border border-red-200 bg-red-50 px-3 py-2 shadow-lg"
        role="alert"
        aria-live="assertive"
        data-testid="task-edit-error-floating"
      >
        <p class="text-xs font-medium text-red-800 mb-0.5">保存失败</p>
        <p class="text-xs text-red-700 whitespace-pre-wrap leading-snug">{{ editError }}</p>
      </div>
      <p
        v-if="resolvedSaveBlockedReason"
        class="max-w-sm text-xs text-amber-800 bg-amber-50 border border-amber-200 rounded-lg px-3 py-2 shadow"
        data-testid="task-edit-save-blocked-reason"
      >
        {{ resolvedSaveBlockedReason }}
      </p>
      <div class="flex gap-2 shadow-lg bg-white/95 backdrop-blur-sm rounded-lg p-2 border border-gray-200">
        <button
          id="cancel-edit-btn"
          type="button"
          class="px-4 py-2 border border-gray-300 bg-white text-gray-700 rounded-md text-sm font-medium hover:bg-gray-50"
          @click="emit('cancel-edit')"
        >
          取消
        </button>
        <button
          id="save-edit-btn"
          type="button"
          :disabled="isSaving || Boolean(resolvedSaveBlockedReason)"
          class="px-4 py-2 bg-primary hover:bg-primary-dark text-white rounded-md text-sm font-medium disabled:opacity-50 disabled:cursor-not-allowed"
          @click="emit('save-edit')"
        >
          {{ isSaving ? '保存中...' : '保存' }}
        </button>
      </div>
    </div>

    <ForkAutoRunConfirmModal
      :show="showForkConfirm"
      :forking="isForking"
      :fork-progress-current="forkProgress.current"
      :fork-progress-total="forkProgress.total"
      :auto-run-blocked-reason="forkAutoRunBlockedReason"
      :auto-run-blocked-trace-id="forkAutoRunBlockedTraceId"
      :oauth-repo-rows="forkOauthRepoRows"
      :git-identity-repo-urls="forkGitIdentityRepoUrls"
      :git-identities="forkGitIdentities"
      :git-identity-settings-href="forkGitIdentitySettingsHref"
      :git-identity-id-for-url="forkGitIdentityIdForUrl"
      :git-identity-blocked-reason="forkGitIdentityBlockedReason"
      :git-identity-blocked-trace-id="forkGitIdentityLoadErrorTraceId"
      :tenant-id="tenantId"
      :workspace-id="workspaceId"
      :source-task="sourceTask"
      @close="showForkConfirm = false"
      @confirm="onForkConfirm"
      @update-git-identity="onForkGitIdentityChange"
    />
  </div>
</template>

<script setup>
import { computed, inject, reactive, ref, unref } from 'vue'
import ForkAutoRunConfirmModal from './ForkAutoRunConfirmModal.vue'
import EntityRevisionPanel from '../entity-revision/EntityRevisionPanel.vue'
import { useCreateTaskRepoOAuth } from '../../composables/useCreateTaskRepoOAuth.js'
import { useCreateTaskGitIdentities } from '../../composables/useCreateTaskGitIdentities.js'
import { createClickGuard } from '../../utils/clickGuard.js'
import { clampForkCopyCount } from '../../utils/forkCopyCount.js'
import {
  collectForkTaskOAuthRepoRows,
  collectForkTaskOAuthRepoUrls,
  resolveForkAutoRunOauthBlockedReason,
} from '../../utils/createTaskOauthGate.js'
import {
  collectForkTaskRepoUrls,
  normalizeCreateTaskRepoIdentities,
  resolveForkAutoRunGitIdentityBlockedReason,
} from '../../utils/createTaskGitIdentityGate.js'
import {
  buildRepoOAuthStartHref,
  resolveRepoOAuthAuthorizeLabel,
} from '../../utils/repoOAuthAuthorizeUtils.js'

const props = defineProps({
  hasTask: { type: Boolean, required: true },
  isEditing: { type: Boolean, required: true },
  isForking: { type: Boolean, required: true },
  isSaving: { type: Boolean, required: true },
  editError: { type: String, default: '' },
  backToWorkPanelRoute: { type: [String, Object], required: true },
  forkTask: { type: Function, required: true },
  tenantId: { type: [String, Number], default: '' },
  workspaceId: { type: [String, Number], default: '' },
  sourceTask: { type: Object, default: null },
  taskProjectsWithDetails: {
    type: Array,
    default: () => [],
  },
})

const emit = defineEmits(['start-edit', 'cancel-edit', 'save-edit'])

const showForkConfirm = ref(false)
const forkProgress = reactive({ current: 0, total: 0 })
const forkConfirmGuard = createClickGuard({ debounceMs: 300 })
const forkIdentityDraft = reactive({
  repo_identities: [],
})

const forkGitIdentityRepoUrls = computed(() => collectForkTaskRepoUrls(props.taskProjectsWithDetails))

const revisionListUrl = computed(() => {
  const tid = String(props.tenantId || '').trim()
  const wid = String(props.workspaceId || '').trim()
  const taskId = String(props.sourceTask?.id || '').trim()
  if (!tid || !wid || !taskId) return ''
  return `/api/tasks/todos/tenant_id/${tid}/workspace_id/${wid}/${taskId}/revisions/`
})

const {
  boundByUrl: forkOauthBoundByUrl,
  loadingByUrl: forkOauthLoadingByUrl,
  errorByUrl: forkOauthErrorByUrl,
  errorTraceIdByUrl: forkOauthErrorTraceIdByUrl,
  oauthRepoUrls: forkOauthRepoUrls,
  blockedReason: forkAutoRunBlockedReason,
  blockedTraceId: forkAutoRunBlockedTraceId,
} = useCreateTaskRepoOAuth({
  enabled: () => showForkConfirm.value,
  tenantId: () => props.tenantId,
  repoUrls: () => collectForkTaskOAuthRepoUrls(props.taskProjectsWithDetails),
  repoRows: () => collectForkTaskOAuthRepoRows(props.taskProjectsWithDetails),
  resolveBlockedReason: resolveForkAutoRunOauthBlockedReason,
})

const {
  gitIdentities: forkGitIdentities,
  identitiesLoading: forkGitIdentitiesLoading,
  loadError: forkGitIdentityLoadError,
  loadErrorTraceId: forkGitIdentityLoadErrorTraceId,
  settingsHref: forkGitIdentitySettingsHref,
  gitIdentityIdForUrl: forkGitIdentityIdForUrl,
  setGitIdentityIdForUrl: setForkGitIdentityIdForUrl,
} = useCreateTaskGitIdentities({
  editingTask: () => forkIdentityDraft,
  projects: () => [],
  tenantId: () => props.tenantId,
  enabled: () => showForkConfirm.value,
  repoUrls: () => forkGitIdentityRepoUrls.value,
})

const forkGitIdentityBlockedReason = computed(() => resolveForkAutoRunGitIdentityBlockedReason({
  requiredRepoUrls: forkGitIdentityRepoUrls.value,
  selections: forkIdentityDraft.repo_identities,
  loading: forkGitIdentitiesLoading.value,
  loadError: forkGitIdentityLoadError.value,
}))

const forkOauthRepoRows = computed(() => {
  const nextPath = typeof window !== 'undefined'
    ? `${window.location.pathname || ''}${window.location.search || ''}`
    : ''
  return forkOauthRepoUrls.value.map((repoUrl) => ({
    repoUrl,
    bound: forkOauthBoundByUrl.value[repoUrl] === true,
    loading: Boolean(forkOauthLoadingByUrl.value[repoUrl]),
    error: String(forkOauthErrorByUrl.value[repoUrl] || '').trim(),
    errorTraceId: String(forkOauthErrorTraceIdByUrl.value[repoUrl] || '').trim(),
    siteLabel: resolveRepoOAuthAuthorizeLabel(repoUrl),
    bindHref: buildRepoOAuthStartHref(repoUrl, nextPath ? { nextPath } : {}),
  }))
})

function openForkConfirm() {
  if (props.isForking) return
  forkIdentityDraft.repo_identities = []
  forkProgress.current = 0
  forkProgress.total = 0
  showForkConfirm.value = true
}

function onForkGitIdentityChange({ repoUrl, gitIdentityId }) {
  setForkGitIdentityIdForUrl(repoUrl, gitIdentityId)
}

async function onForkConfirm(payload) {
  if (props.isForking) return
  const autoRun = payload?.autoRun === true
  if (autoRun && (forkAutoRunBlockedReason.value || forkGitIdentityBlockedReason.value)) return
  const copyCount = clampForkCopyCount(payload?.copyCount ?? 1)
  await forkConfirmGuard.run(async ({ idempotencyKey }) => {
    forkProgress.current = 0
    forkProgress.total = copyCount
    const request = { autoRun, copyCount, batchIdempotencyKey: idempotencyKey }
    if (autoRun) {
      request.repoIdentities = normalizeCreateTaskRepoIdentities(forkIdentityDraft.repo_identities)
      request.featureParamsSource = payload?.featureParamsSource
      request.personalFeatureParamsConfigId = payload?.personalFeatureParamsConfigId
      request.agentModelProvider = payload?.agentModelProvider
      request.agents = payload?.agents
    }
    request.onForkProgress = (current, total) => {
      forkProgress.current = current
      forkProgress.total = total
    }
    const ok = await props.forkTask(request)
    if (ok) showForkConfirm.value = false
    return ok
  })
}

const injectedSaveBlocked = inject('taskDetailSaveBlockedReason', null)
const resolvedSaveBlockedReason = computed(() => {
  const raw = unref(injectedSaveBlocked)
  return raw != null ? String(raw) : ''
})
</script>

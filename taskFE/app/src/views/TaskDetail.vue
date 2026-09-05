<template>
  <div class="min-h-screen bg-gradient-to-br from-primary/5 to-background font-sans">
    <main class="embed-container">
      <div class="content-card bg-white rounded-lg shadow-xl p-3 md:p-4">
        <TaskDetailPageHeader
          :has-task="Boolean(localTask)"
          :is-editing="isEditing"
          :is-forking="isForking"
          :is-saving="isSaving"
          :edit-error="editError"
          :back-to-work-panel-route="backToWorkPanelRoute"
          :fork-task="forkTask"
          :tenant-id="String(effectiveTenantId || '')"
          :workspace-id="String(effectiveWorkspaceId || '')"
          :source-task="localTask"
          :task-projects-with-details="taskProjectsWithDetails"
          @start-edit="startEdit"
          @cancel-edit="cancelEdit"
          @save-edit="saveEdit"
        />
        <div
          v-if="!localTask && !props.task"
          class="py-8 text-center text-sm"
          data-testid="task-detail-load-state"
        >
          <p v-if="taskDetailLoading" class="text-gray-500 flex items-center justify-center gap-2">
            <span
              class="inline-block h-4 w-4 shrink-0 animate-spin rounded-full border-2 border-gray-300 border-t-primary"
            />
            正在加载任务详情…
          </p>
          <p v-else-if="taskDetailLoadError" class="text-red-600" :data-traceId="taskDetailLoadErrorTraceId || undefined">{{ taskDetailLoadError }}</p>
          <p v-else class="text-gray-500">未找到任务数据，请刷新页面或返回工作面板重试。</p>
        </div>
        <div class="space-y-4" v-if="localTask">
          <TaskDetailEditErrorBanner :message="editError" />
          <TaskDetailTaskIdentityPanel
            :task="localTask"
            :is-editing="isEditing"
            :editing-task="editingTask"
            :collaborator-name-by-id="collaboratorNameById"
            :collaborator-options="collaboratorOptions"
            :fork-source-task-route="forkSourceTaskRoute"
            :fork-source-title="forkSourceTitle"
            :fork-source-seq="forkSourceSeq"
            :is-fork-source-title-loading="isForkSourceTitleLoading"
            :progress-status-options="progressStatusOptions"
            :is-progress-statuses-loading="isProgressStatusesLoading"
            :is-updating-progress-status="isUpdatingProgressStatus"
            :progress-status-error="progressStatusError"
            :progress-status-error-trace-id="progressStatusErrorTraceId"
            :container-endpoint-registered="containerEndpointRegistered"
            :container-heartbeat-status="containerHeartbeatStatus"
            :tenant-id="String(effectiveTenantId || '')"
            :workspace-id="String(effectiveWorkspaceId || '')"
            :task-id="String(effectiveTaskId || '')"
            :comment-id="containerForwardCommentId"
            @progress-status-change="onProgressStatusChange"
            @task-updated="onServerConfigTaskUpdated"
          />

          <TaskDetailBranchStrategyPanel
            :task="localTask"
            :task-id="String(effectiveTaskId || '')"
            :is-editing="isEditing"
            :editing-task="editingTask"
            :company-user-name="companyUserName"
            :is-task-title-translating="isTaskTitleTranslating"
            :task-title-translation-error="taskTitleTranslationError"
            :common-merge-target-branches="commonMergeTargetBranches"
            :common-merge-target-branches-loading="commonMergeTargetBranchesLoading"
            :common-merge-target-branches-error="commonMergeTargetBranchesError"
            @refresh-common-merge-target-branches="() => fetchAllLinkedRepoBranches(true)"
          />

          <TaskDetailRuntimeSection
            ref="runtimeSectionRef"
            v-bind="runtimeSectionProps"
            @task-updated="onServerConfigTaskUpdated"
            @start-request-accepted="handleStartRequestAccepted"
            @repo-reclone="onRepoReclone"
            @fetch-layer-repo-git-identities="fetchLayerRepoGitIdentities"
            @sync-per-repo-git-identities-to-container="syncPerRepoGitIdentitiesToContainer"
            @sync-repo-address="syncStaleTaskRepoAddresses"
            @repo-clone-identity-change="onRepoCloneIdentityChange"
            @project-change="onProjectChange"
            @remove-project-association="removeProjectAssociation"
            @set-repo-branch="setRepoBranch"
            @fetch-repo-branches="fetchRepoBranches"
            @add-project-association="addProjectAssociation"
            @git-identity-created="fetchLayerGitIdentityOptions"
            @repo-oauth-readiness="onRepoOAuthReadiness"
            @sse-manual-reconnect="handleSSEManualReconnect"
          />
          <TaskDetailAutoRunSkipBanner
            :task="localTask"
            :tenant-id="String(effectiveTenantId)"
            :workspace-id="String(effectiveWorkspaceId)"
            :task-id="String(effectiveTaskId)"
            :repo-url="String(taskRepoRows[0]?.url || '')"
            @updated="onAutoRunSkipBannerUpdated"
          />
          <TaskDetailCommentsSection
            ref="taskLayerAssociationPanelRef"
            :server-runtime-status-panel="serverRuntimeStatusPanelProps"
            :server-content-panel="serverContentPanelProps"
            v-model:new-comment="newComment"
            v-model:layer-graph-command-kind="layerGraphCommandKind"
            v-model:layer-graph-selected-model="layerGraphSelectedModel"
            v-model:layer-graph-auto-iteration-count="layerGraphAutoIterationCount"
            v-model:layer-graph-command-text="layerGraphCommandText"
            v-bind="commentsSectionProps"
            @remove-comment-composer-chip="removeCommentComposerChip"
            @comment-textarea-keydown="onCommentTextareaKeydown"
            @repo-oauth-readiness="onRepoOAuthReadiness"
            @submit-comment="submitComment"
            @load-more-comments="loadMoreCommentFeeds"
            @task-updated="onServerConfigTaskUpdated"
            @refresh-layer-graph="refreshLayerGraph"
            @layer-graph-node-select="onLayerGraphNodeSelect"
            @layer-graph-job-redo="onLayerGraphJobRedo"
            @layer-graph-job-interrupt="onLayerGraphJobInterrupt"
            @layer-graph-job-continue="onLayerGraphJobContinue"
            @layer-graph-job-edit-run="onLayerGraphJobEditRun"
            @layer-graph-job-delete="onLayerGraphJobDelete"
            @layer-graph-layer-delete="onLayerGraphLayerDelete"
            @layer-graph-layer-submit="onLayerGraphLayerSubmit"
            @layer-graph-layer-push="onLayerGraphLayerPush"
            @layer-graph-layer-submit-and-push="onLayerGraphLayerSubmitAndPush"
            @layer-graph-layer-submit-and-merge="onLayerGraphLayerSubmitAndMerge"
            @layer-graph-layer-merge="onLayerGraphLayerMerge"
            @layer-changes-refresh="refreshSelectedLayerChanges"
            @layer-changes-staged-refresh="refreshSelectedLayerChanges"
            @layer-changes-commit-staged="onLayerChangesListCommitStaged"
            @layer-changes-load-more="loadMoreSelectedLayerChanges"
            @submit-layer-graph-command="submitLayerGraphCommand"
            @copy-layer-exec-log="copyLayerExecLog"
            @clear-layer-exec-log="clearLayerExecLog"
            @copy-agent-step-json="copyAgentStepJson"
            @agent-step-rich-interact="onAgentStepRichInteract"
            @sse-manual-reconnect="handleSSEManualReconnect"
            @repo-reclone="onRepoReclone"
            @patch-layer-panel="(id, p) => layerPanelStore.patch(id, p)"
          />
        </div>
      </div>
    </main>
  </div>
</template>

<script setup>
import { onMounted, onBeforeUnmount, watch, computed, ref, provide } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useTaskDetail } from '../composables/useTaskDetail.js'
import { isRouteRelayToTraeQuery } from '../utils/relayToTraeUtils.js'
import TaskDetailPageHeader from '../components/task-detail/TaskDetailPageHeader.vue'
import TaskDetailEditErrorBanner from '../components/task-detail/TaskDetailEditErrorBanner.vue'
import TaskDetailTaskIdentityPanel from '../components/task-detail/TaskDetailTaskIdentityPanel.vue'
import TaskDetailBranchStrategyPanel from '../components/task-detail/TaskDetailBranchStrategyPanel.vue'
import TaskDetailRuntimeSection from '../components/task-detail/TaskDetailRuntimeSection.vue'
import TaskDetailCommentsSection from '../components/task-detail/TaskDetailCommentsSection.vue'
import TaskDetailAutoRunSkipBanner from '../components/task-detail/TaskDetailAutoRunSkipBanner.vue'
import { gitCloneRefMatchKey, shortCloneRepoLabel, cloneProgressRowHasSubPhases, cloneProgressRecvPct, cloneProgressUnpackPct } from '../utils/taskDetailContainerCloneProgress.js'
import { repoCloneFieldId } from '../utils/taskDetailBranchAndRepoUtils.js'
import { useProjectGitOAuthCatalog } from '../composables/useProjectGitOAuthCatalog.js'
import { apiFetch } from '../utils/apiUtils.js'
import {
  useTaskDetailRuntimeSectionBindings,
  useTaskDetailCommentsSectionBindings,
} from './taskDetailSectionBindings.js'
import { useTaskDetailTaskSwitch } from '../composables/taskDetail/useTaskDetailTaskSwitch.js'
import { mergeTaskDetailUpdate } from '../composables/taskDetail/mergeTaskDetailUpdate.js'

const props = defineProps({
  task: { type: Object, default: null },
  tenantId: { type: [String, Number], default: null },
  taskId: { type: [String, Number], default: null },
  workspaceId: { type: [String, Number], default: null },
})
const emit = defineEmits(['close', 'task-updated'])
const route = useRoute()
const router = useRouter()
const relayToTraeEnabled = computed(() => isRouteRelayToTraeQuery(route.query))
const { gitOAuthCatalogVersion, bootstrapGitOAuthCatalog } = useProjectGitOAuthCatalog(
  apiFetch,
  () => String(props.tenantId || route.params.tenant || ''),
)

const repoOAuthReadiness = ref({
  allBound: false,
  loading: true,
  unboundRepoUrls: [],
  hasOAuthRepos: false,
  startBlocked: true,
})

const onRepoOAuthReadiness = (readiness) => {
  const next = {
    allBound: Boolean(readiness?.allBound),
    loading: Boolean(readiness?.loading),
    unboundRepoUrls: Array.isArray(readiness?.unboundRepoUrls) ? readiness.unboundRepoUrls : [],
    hasOAuthRepos: Boolean(readiness?.hasOAuthRepos),
    startBlocked: Boolean(readiness?.startBlocked),
  }
  // catalog 未 bootstrap 前忽略「无 OAuth 目标」空快照，保留 relay 悲观阻断；
  // catalog 就绪后必须接受该快照，否则 SSH/未识别仓库会永久卡在「请点击 OAuth 绑定」且无按钮。
  if (!next.hasOAuthRepos && !next.loading && !next.startBlocked) {
    if (gitOAuthCatalogVersion.value <= 0) return
  }
  repoOAuthReadiness.value = next
}

const relayStartBlockedByUnboundOAuth = computed(() => {
  if (!relayToTraeEnabled.value) return false
  const readiness = repoOAuthReadiness.value
  if (readiness.startBlocked) return true
  if (readiness.loading) return true
  return readiness.hasOAuthRepos && !readiness.allBound
})

const relayStartBlockedByOAuthCheckLoading = computed(
  () => relayToTraeEnabled.value && repoOAuthReadiness.value.loading,
)

provide('taskDetailRelayStartBlockedByUnboundOAuth', relayStartBlockedByUnboundOAuth)
provide('taskDetailRelayStartBlockedByOAuthCheckLoading', relayStartBlockedByOAuthCheckLoading)

const $ = useTaskDetail(props, { emit, route, router })

const {
  localTask, taskDetailLoading, taskDetailLoadError, taskDetailLoadErrorTraceId,
  effectiveTenantId, effectiveWorkspaceId, effectiveTaskId, workspaceId,
  backToWorkPanelRoute, forkSourceTaskRoute, forkSourceTitle, forkSourceSeq, isForkSourceTitleLoading,
  serverConfigRef, projectFileTreeRefreshNonce,
  sseConnection, sseLive, sseReconnecting, sseReconnectAttempts, ssePlatformRestartHint,
  closeSSEConnection, establishSSEConnection, handleSSEManualReconnect, handleStartRequestAccepted,
  serverStatus, statusMessage, statusCommentId, statusRuntimeStatus, statusTraceId, statusProgress, statusLogs,
  isServerRunning, isServerStarting, serverUrl, cloudRuntimeStatus,
  updateServerStatus,
  collaboratorNameById, collaboratorAvatarById, collaboratorOptions,
  progressStatusOptions, isProgressStatusesLoading, isUpdatingProgressStatus,
  progressStatusError, progressStatusErrorTraceId, onProgressStatusChange,
  isForking, forkTask,
  isEditing, editingTask, isSaving, editError,
  isTaskTitleTranslating, taskTitleTranslationError,
  companyUserName,
  startEdit, cancelEdit, saveEdit,
  addProjectAssociation, removeProjectAssociation,
  workspaceProjects, workspaceProjectsLoading, projectServerRunTemplate,
  repoCloneIdentityByUrl, repoCloneIdentitySaveError, repoCloneIdentitySaving,
  onRepoCloneIdentityChange,
  recloneLoadingByUrl, recloneStatusByUrl, recloneErrorTraceIdByUrl, repoRecloneGlobalLoading, onRepoReclone,
  taskProjectsWithDetails, taskRepoRows, getProjectRepos,
  staleRepoSyncLoading, staleRepoSyncError, staleRepoSyncNeedsReclone, syncStaleTaskRepoAddresses,
  getRepoBranchValue, setRepoBranch, getRepoBranches, getRepoBranchError,
  isLoadingRepoBranches, fetchRepoBranches, onProjectChange,
  commonMergeTargetBranches, commonMergeTargetBranchesLoading, commonMergeTargetBranchesError,
  fetchAllLinkedRepoBranches,
  newComment, commentComposerChips, displayComments,
  containerForwardCommentId,
  commentsHasMore, commentsFeedsLoadingMore, loadMoreCommentFeeds,
  removeCommentComposerChip, onCommentTextareaKeydown, submitComment,
  aiStreamBusy, aiStreamBuffer, activeContainerAgentId,
  bindingStatusFor, bindingCscIdFor, bindingContainerNameFor,
  commentHasLiveBinding, commentOwnsSharedContainer,
  bindingStartTraceIdFor, buildPerBindingServerStatusProps, markBindingReconnecting,
  cancelWaitingPreviousBinding, isCancelWaitingBusy,
  containerEndpointRegistered, containerPageUrl, containerVscodeUrl,
  displayContainerVscodeUrl, containerHttpUnreachable,
  containerHeartbeatStatus, containerHeartbeatLastSuccess,
  containerHeartbeatAttempts, containerHeartbeatError, containerHeartbeatSeqInfo,
  containerHeartbeatLogLines, appendContainerHeartbeatLogLines,
  containerActionsBlocked,
  containerCloneProgressEntries, containerCloneProgressByCommentId, cloneProgressEntryByRepoMatchKey,
  cloneProgressBarWidthTransitionClass,
  cloneProgressRowLogIsPlaceholder, cloneProgressRowLogDisplayText,
  containerPageLinkPendingReveal,
  containerBootstrapFailureMessage, containerBootstrapFailureTraceId, containerLayerGraphAuthInvalid,
  serverRuntimeNotServing, containerHeartbeatPaused,
  layerGraphZNodes, layerGraphMetaLine, layerGraphRefreshing,
  selectedLayerGraphNode, selectedLayerGraphFileTreeLayerId,
  layerGraphSelectionDismissedByUser,
  layerGraphCommandText, layerGraphCommandKind, layerGraphAutoIterationCount,
  layerGraphCmdError, layerGraphCmdErrorTraceId, layerGraphCmdSending,
  layerGraphEditRunTargetJobId, layerGraphBusyActionKey,
  layerGraphModelSelectDisabled, taskLayerAssociationPanelRef,
  layerGraphSelectedModel, layerGraphModelOptions, layerGraphDefaultModel,
  layerGraphModelLoadError, layerGraphModelLoadErrorTraceId,
  layerExecLogLoading, layerExecLogCopyable, layerExecLogClearable,
  layerExecLogCopyFeedback, layerExecLogTopError,
  layerCloneLogFetchError, layerCloneLogFetchErrorTraceId, layerCloneLogText,
  layerJobLogFetchError, layerJobLogFetchErrorTraceId, layerJobExecutionPayload,
  layerLiveOutputDisplay, layerJobCommandHead, layerJobOutputDisplay,
  layerAgentStepCopyFeedbackKey, layerAgentStepCards,
  layerChangesRefreshBusy, layerChangesRefreshEnabled, layerChangesRefreshError,
  layerChangesRefreshErrorTraceId,
  layerChangesGitStagedActionsBlocked, layerChangesGitCommitIdentityBlocked,
  selectedZTreeLayerChangesPanel, zTreeLogTargets,
  refreshLayerGraph, onLayerGraphNodeSelect,
  onLayerGraphJobRedo, onLayerGraphJobInterrupt, onLayerGraphJobContinue,
  onLayerGraphJobDelete, onLayerGraphLayerDelete, onLayerGraphJobEditRun,
  onLayerGraphLayerSubmit, onLayerGraphLayerPush, onLayerGraphLayerMerge, onLayerGraphLayerSubmitAndPush, onLayerGraphLayerSubmitAndMerge,
  refreshSelectedLayerChanges, loadMoreSelectedLayerChanges, onLayerChangesListCommitStaged,
  submitLayerGraphCommand, copyLayerExecLog, clearLayerExecLog,
  copyAgentStepJson, onAgentStepRichInteract,
  layerPanelStore,
  layerGitIdentityOptions, layerGitIdentityLoading,
  fetchLayerGitIdentityOptions,
  layerRepoGitIdentityLoading, layerRepoGitIdentityFetchError,
  perRepoGitIdentitySyncing, perRepoGitIdentitySyncError,
  fetchLayerRepoGitIdentities, syncPerRepoGitIdentitiesToContainer,
  containerGitIdentityLineForRepoUrl,
  linkedProjectsPanelRef,
  abortAiInstruct, abortLayerLog, cleanupExecLogTimers,
  clearContainerUnreachableProbeTimer, stopContainerHeartbeat,
  pauseContainerHeartbeatForRelayStop, resumeContainerHeartbeatForRelayStart,
  fetchContainerTaskUiContext, onDocumentVisibilityForTaskLayerSync,
  fetchTaskDetail, fetchWorkspaceCollaborators, fetchProgressStatusOptions,
  fetchCurrentCompanyUserName, fetchLayerGraphModelOptions,
  fetchWorkspaceProjects,
  markContainerTransportOk, resetLayerGraphFetchBackoff,
  resetServerRuntimeLayerGraphGateCache,
  layerGraphSnapshot, layerGraphSeenLayerIdSet,
  layerGraphHydrateInFlight,
  layerChangesPrefetchInFlight, layerJobLiveOutputMap,
  containerCloneProgressByKey, containerBootstrapCloneLogFull,
  containerBootstrapCloneLogSegments,
  bootstrapCloneDone,
  layerChangesByLayerId,
  CONTAINER_CLONE_PROGRESS_GLOBAL_KEY,
  _internal, getContainerCloneProgressTimer,
  cleanupContainerTimers,
  BOOTSTRAP_CLONE_LOG_INDICATES_DONE_RE,
} = $

const runtimeSectionRef = ref(null)
const { runtimeSectionProps } = useTaskDetailRuntimeSectionBindings({
  localTask, workspaceId, sseConnection, updateServerStatus, statusMessage, statusCommentId, statusRuntimeStatus, statusProgress, statusLogs,
  serverStatus, isServerRunning, isServerStarting, cloudRuntimeStatus, serverUrl, containerVscodeUrl, containerPageUrl,
  containerHeartbeatStatus, pauseContainerHeartbeatForRelayStop, resumeContainerHeartbeatForRelayStart,
  appendContainerHeartbeatLogLines, fetchTaskDetail, relayStartBlockedByUnboundOAuth,
  relayStartBlockedByOAuthCheckLoading, projectServerRunTemplate, effectiveTenantId, effectiveTaskId,
  isEditing, taskProjectsWithDetails, editingTask, containerEndpointRegistered, relayToTraeEnabled,
  repoCloneIdentitySaveError, recloneLoadingByUrl, repoRecloneGlobalLoading, recloneStatusByUrl,
  repoCloneIdentityByUrl, layerGitIdentityOptions, repoCloneIdentitySaving, layerGitIdentityLoading,
  cloneProgressEntryByRepoMatchKey, gitCloneRefMatchKey, repoCloneFieldId, cloneProgressRowHasSubPhases,
  cloneProgressRecvPct, cloneProgressUnpackPct, cloneProgressBarWidthTransitionClass,
  bootstrapCloneDone,
  bootstrapCloneLogFull: containerBootstrapCloneLogFull,
  bootstrapCloneLogSegments: containerBootstrapCloneLogSegments,
  workspaceProjects,
  workspaceProjectsLoading, getProjectRepos, getRepoBranchValue, getRepoBranches, getRepoBranchError,
  isLoadingRepoBranches, selectedLayerGraphFileTreeLayerId, layerRepoGitIdentityLoading,
  layerRepoGitIdentityFetchError, perRepoGitIdentitySyncing, perRepoGitIdentitySyncError,
  staleRepoSyncLoading, staleRepoSyncError, staleRepoSyncNeedsReclone, containerGitIdentityLineForRepoUrl, gitOAuthCatalogVersion,
  onRepoOAuthReadiness, containerHttpUnreachable, sseLive, sseReconnecting, sseReconnectAttempts,
  ssePlatformRestartHint,
  containerHeartbeatLastSuccess, containerHeartbeatAttempts, containerHeartbeatError,
  containerHeartbeatSeqInfo, containerHeartbeatLogLines, displayComments, activeContainerAgentId,
  runtimeSectionRef, serverConfigRef, linkedProjectsPanelRef,
})

const { commentsSectionProps } = useTaskDetailCommentsSectionBindings({
  localTask, taskRepoRows, repoCloneIdentityByUrl, containerCloneProgressEntries, containerCloneProgressByCommentId, repoCloneFieldId, shortCloneRepoLabel,
  cloneProgressRowHasSubPhases, cloneProgressRecvPct, cloneProgressUnpackPct,
  cloneProgressBarWidthTransitionClass, cloneProgressRowLogIsPlaceholder, cloneProgressRowLogDisplayText,
  displayComments, collaboratorNameById, collaboratorAvatarById, commentsHasMore,   commentsFeedsLoadingMore, aiStreamBusy, aiStreamBuffer,
  activeContainerAgentId,
  bindingStatusFor, bindingCscIdFor, bindingContainerNameFor,
  commentHasLiveBinding, commentOwnsSharedContainer,
  bindingStartTraceIdFor, buildPerBindingServerStatusProps, markBindingReconnecting,
  cancelWaitingPreviousBinding, isCancelWaitingBusy,
  recloneLoadingByUrl, recloneStatusByUrl, recloneErrorTraceIdByUrl, repoRecloneGlobalLoading,
  zTreeLogTargets, layerAgentStepCards, effectiveTenantId, effectiveWorkspaceId, effectiveTaskId,
  containerPageLinkPendingReveal, containerHttpUnreachable, displayContainerVscodeUrl,
  commentComposerChips,
  containerPageUrl, layerGraphZNodes, layerGraphRefreshing, layerGraphMetaLine, layerGraphBusyActionKey,
  selectedLayerGraphNode, selectedZTreeLayerChangesPanel, selectedLayerGraphFileTreeLayerId,
  layerChangesRefreshBusy, layerChangesRefreshEnabled, layerChangesRefreshError,
  layerChangesRefreshErrorTraceId,
  layerChangesGitStagedActionsBlocked, layerChangesGitCommitIdentityBlocked, containerEndpointRegistered,
  projectFileTreeRefreshNonce, layerGraphModelSelectDisabled, layerGraphModelOptions, layerGraphDefaultModel,
  layerGraphEditRunTargetJobId, layerGraphModelLoadError, layerGraphModelLoadErrorTraceId,
  layerGraphCmdError, layerGraphCmdErrorTraceId, layerGraphCmdSending,
  containerActionsBlocked, layerExecLogLoading, layerExecLogCopyable, layerExecLogClearable,
  layerExecLogCopyFeedback, layerExecLogTopError, layerCloneLogFetchError, layerCloneLogFetchErrorTraceId, layerCloneLogText,
  layerLiveOutputDisplay, layerJobLogFetchError, layerJobLogFetchErrorTraceId, layerJobExecutionPayload, layerJobCommandHead,
  layerJobOutputDisplay, layerAgentStepCopyFeedbackKey,
  serverRuntimeNotServing, containerHeartbeatPaused,
  containerBootstrapFailureMessage, containerBootstrapFailureTraceId, containerLayerGraphAuthInvalid,
  containerBootstrapCloneLogFull,
  sseLive, sseReconnecting, sseReconnectAttempts, ssePlatformRestartHint,
  projectServerRunTemplate, taskProjectsWithDetails,
  layerPanelStore, layerGitIdentityOptions, repoOAuthReadiness,
})

const onServerConfigTaskUpdated = (updatedTask) => {
  if (updatedTask && typeof updatedTask === 'object') {
    localTask.value = mergeTaskDetailUpdate(localTask.value, updatedTask)
  }
}

const onAutoRunSkipBannerUpdated = (updatedTask) => {
  onServerConfigTaskUpdated(updatedTask)
  void fetchTaskDetail()
}

/**
 * 服务器运行状态面板数据（由 ServerConfig.logic.vue 经 runtimeSectionRef 暴露）。
 * 传入评论区后注入**每条评论**各自的「执行细节」Tab —— 原页面顶部卡片已移除，
 * 此处为运行态数据源；展示层按 commentId 快照隔离，禁止单例面板。
 */
const serverRuntimeStatusPanelProps = computed(() => {
  const sc = runtimeSectionRef.value?.serverConfigRef
  // 注意：defineExpose 暴露的 ref 经模板 ref 访问会被 proxyRefs 自动解包，
  // sc.serverRuntimeStatusPanel 已是对象本身（不是 ref），勿再读 .value
  const panel = sc?.serverRuntimeStatusPanel
  return panel && typeof panel === 'object' ? panel : null
})

const serverContentPanelProps = computed(() => {
  const sc = runtimeSectionRef.value?.serverConfigRef
  const panel = sc?.serverContentPanel
  return panel && typeof panel === 'object' ? panel : null
})

// ── Task switch watchers (extracted to composable) ──
useTaskDetailTaskSwitch({
  $, props, route, effectiveTaskId, localTask, newComment, commentComposerChips,
  serverConfigRef, _internal: $._internal,
})

// ── Lifecycle ──
onMounted(async () => {
  await bootstrapGitOAuthCatalog()
  const initTasks = []
  initTasks.push(fetchTaskDetail())
  initTasks.push(fetchWorkspaceCollaborators())
  initTasks.push(fetchProgressStatusOptions())
  initTasks.push(fetchLayerGitIdentityOptions())
  initTasks.push(fetchLayerGraphModelOptions())
  initTasks.push(fetchWorkspaceProjects())
  initTasks.push(fetchCurrentCompanyUserName(effectiveTenantId.value))
  initTasks.push(fetchContainerTaskUiContext())

  const taskId = effectiveTaskId.value
  if (taskId) establishSSEConnection(taskId)

  const isInIframe = window !== window.top
  if (isInIframe) {
    const backButton = document.getElementById('back-to-work-panel')
    const titleRow = document.getElementById('task-detail-title-row')
    if (backButton) backButton.style.display = 'none'
    if (titleRow) titleRow.style.display = 'none'
  }
  document.addEventListener('visibilitychange', onDocumentVisibilityForTaskLayerSync)
})

onBeforeUnmount(() => {
  document.removeEventListener('visibilitychange', onDocumentVisibilityForTaskLayerSync)
  closeSSEConnection()
  abortAiInstruct()
  abortLayerLog()
  cleanupExecLogTimers()
  layerExecLogCopyFeedback.value = false
  layerAgentStepCopyFeedbackKey.value = ''
  clearContainerUnreachableProbeTimer()
  cleanupContainerTimers()
  stopContainerHeartbeat()
})

// ── Watchers ──
watch(() => props.task, (newTask) => {
  if (newTask == null) return
  localTask.value = mergeTaskDetailUpdate(localTask.value, newTask)
})
</script>

/**
 * Extracted from useTaskDetail.js (OPT-20260823-007 line-limit split).
 */
import { ref, computed, nextTick, watch, provide } from 'vue'
import { updateServerStatus as _updateServerStatus } from './updateServerStatus.js'
import { establishSSEConnection as _establishSSEConnection } from './establishSSEConnection.js'
import { bindEstablishSSEConnection } from './bindEstablishSSEConnection.js'
import { createServerStartupStatusPollController } from './serverStartupStatusPoll.js'
import { submitAIComment as _submitAIComment } from './submitAIComment.js'
import { submitLayerGraphCommand as _submitLayerGraphCommand } from './submitLayerGraphCommand.js'
import { onLayerGraphLayerSubmit as _onLayerGraphLayerSubmit, onLayerChangesListCommitStaged as _onLayerChangesListCommitStaged, onLayerGraphLayerPush as _onLayerGraphLayerPush, onLayerGraphLayerMerge as _onLayerGraphLayerMerge, onLayerGraphLayerSubmitAndPush as _onLayerGraphLayerSubmitAndPush, onLayerGraphLayerSubmitAndMerge as _onLayerGraphLayerSubmitAndMerge } from './taskDetailLayerActions.js'
import {
  fetchContainerTaskUiContext as _fetchContainerTaskUiContext, resolveContainerUiContextCommentId,
  ensureServerRuntimeAllowsContainerLayerGraph as _ensureServerRuntimeAllowsContainerLayerGraph, refreshLayerGraphFromServer as _refreshLayerGraphFromServer,
  fetchContainerBootstrapCloneLog as _fetchContainerBootstrapCloneLog,
} from './taskDetailContainerFns.js'
import { collectGitPrHtmlUrls } from './taskDetailGitPrReply.js'
import {
  CONTAINER_CLONE_PROGRESS_GLOBAL_KEY, BOOTSTRAP_CLONE_LOG_INDICATES_DONE_RE,
  createCloneProgressState, createCloneProgressComputeds,
  onBootstrapCloneLogUpdate as _onBootstrapCloneLogUpdate,
  rescheduleContainerCloneProgressClearTimer as _rescheduleContainerCloneProgressClearTimer,
  applyBootstrapCloneLogDoneToCloneProgress as _applyBootstrapCloneLogDoneToCloneProgress,
} from './taskDetailCloneProgress.js'
import {
  createContainerHeartbeatState,
  createLayerGraphFetchBackoffState,
  createLayerGraphHeartbeatRefresh,
  createServerRuntimeLayerGraphGateState,
  layerGraphSnapshotHasActiveJob,
  layerGraphSnapshotHasContent,
  SERVER_RUNTIME_LAYER_GRAPH_GATE_MS,
} from './taskDetailContainerHeartbeat.js'
import {
  createContainerHeartbeatSseBuffer,
  flushPendingContainerHeartbeat,
  promoteContainerHeartbeatFromIdle,
} from './applyContainerHeartbeatSse.js'
import { createSseReconnectState } from './taskDetailSseReconnect.js'
import { createApplyLayerGraphFromPayload } from './taskDetailLayerGraphPayload.js'
import {
  createTaskDetailZTreeExecLogState,
  installTaskDetailZTreeExecLogWatchers,
  resolveZTreeLogTargets,
  normalizeLayerChangesPayload,
} from './taskDetailZTreeExecLogState.js'
import { installCommentLayerPanelHydration } from './installCommentLayerPanelHydration.js'
import { fetchLayerRepoGitIdentities as _fetchLayerRepoGitIdentities, syncPerRepoGitIdentitiesToContainer as _syncPerRepoGitIdentitiesToContainer } from './taskDetailGitFns.js'
import { fetchTaskDetail as _fetchTaskDetail, loadMoreCommentFeeds as _loadMoreCommentFeeds, fetchWorkspaceCollaborators as _fetchWorkspaceCollaborators, fetchProgressStatusOptions as _fetchProgressStatusOptions, fetchDeliverableCategoryOptions as _fetchDeliverableCategoryOptions, fetchWorkspaceTodosForParent as _fetchWorkspaceTodosForParent, fetchParentDeliverableTitle as _fetchParentDeliverableTitle, fetchForkSourceTitle as _fetchForkSourceTitle, fetchTaskSubtree as _fetchTaskSubtree, onProgressStatusChange as _onProgressStatusChange, submitComment as _submitComment, onRepoReclone as _onRepoReclone, onRepoCloneIdentityChange as _onRepoCloneIdentityChange, maybeAutoApplyDefaultRepoCloneIdentities as _maybeAutoApplyDefaultRepoCloneIdentities } from './taskDetailFetchFns.js'

export function initUseTaskDetailActions(d) {
  const {
    serverStatus, statusMessage, statusCommentId, statusRuntimeStatus, statusTraceId,
    statusProgress, statusLogs, sseLive, isServerRunning, isServerStarting,
    cloudRuntimeStatus, serverUrl, pendingStartEventId, effectiveTenantId,
    effectiveWorkspaceId, effectiveTaskId, relayAccessCode, localTask,
    layerGraphPushTargetBranch, layerGraphMergeTargetBranch, serverConfigRef,
    displayComments, activeContainerAgentId, commentForwardScope,
    containerForwardCommentId, newComment, commentComposerChips,
    buildCommentBodyWithChips, bumpProjectFileTreeRefresh, workspaceProjectsLoading,
    repoCloneIdentityByUrl, repoCloneIdentitySaveError, repoCloneIdentitySaving,
    repoCloneIdentityUserTouchedByUrl, repoCloneIdentityAutoApplyInFlight, taskRepoRows,
    syncRepoCloneIdentityMapFromTask, validateLinkedProjectReposBeforeSendToAi,
    repoCloneIdentityIdForUrl, firstTaskRepoCloneIdentityId, commentOwnsSharedContainer,
    fetchTaskDetail, layerPanelStore, containerEndpointRegistered, containerPageUrl,
    containerVscodeUrl, lgBackoff, layerGraphSnapshot, cpState,
    onBootstrapCloneLogUpdate, fetchContainerBootstrapCloneLog, runtimeGate,
    selectedLayerGraphNode, layerChangesByLayerId, layerGraphCommandText, zlog,
    resetLayerGraphFetchBackoff, registerLayerGraphFetchFailure,
    containerBootstrapFailureMessage, containerBootstrapFailureTraceId,
    containerCloneProgressByKey, containerCloneProgressByCommentId,
    containerBootstrapCloneLogFull, containerPageLinkPendingReveal,
    containerHttpUnreachable, containerLayerGraphAuthInvalid, containerHeartbeatStatus,
    containerHeartbeatPaused, serverRuntimeNotServing, containerHeartbeatLastSuccess,
    containerHeartbeatAttempts, containerHeartbeatError,
    applyContainerHeartbeatSeqFromSse, markContainerTransportOk, stopContainerHeartbeat,
    pauseContainerHeartbeatForRelayStop, markServerRuntimeServing,
    enterServerNotServingUiState, formatLayerGraphCommandErrorForUser,
    markContainerTransportUnreachableIfForwardingFailed,
    resetServerRuntimeLayerGraphGateCache, layerJobExecutionPayload,
    layerJobLiveOutputMap, selectedLayerGraphFileTreeLayerId,
    refreshSelectedLayerChanges, applyLiveLayerChangesToCurrentPayload,
    refreshZTreeExecutionLog, layerGraphZNodes, layerGraphCommandKind,
    layerGraphAutoIterationCount, layerGraphModelProvider, layerGraphSelectedModel,
    layerGraphCmdError, layerGraphCmdErrorTraceId, layerGraphCmdSending,
    layerGraphEditRunTargetJobId, layerGraphBusyActionKey,
    layerGraphSelectionDismissedByUser, layerGitIdentityOptions, layerGitIdentityLoading,
    layerGitGithubAppOauthConnected, layerRepoGitIdentityLoading,
    layerRepoGitIdentityFetchError, layerRepoGitIdentityRows, perRepoGitIdentitySyncing,
    perRepoGitIdentitySyncError, onLayerGraphNodeSelect, normalizeSelectedAgentModels,
  } = d
const onRepoCloneIdentityChange = async (url, nextId) => {
  await _onRepoCloneIdentityChange(url, nextId, {
    effectiveTenantId, effectiveWorkspaceId, effectiveTaskId,
    repoCloneIdentityByUrl, repoCloneIdentitySaving, repoCloneIdentitySaveError,
    syncRepoCloneIdentityMapFromTask, repoCloneIdentityUserTouchedByUrl, localTask,
  })
  // OPT-20260724-031-4: 身份变更后，若容器已在运行，自动推送新身份到容器
  if (containerEndpointRegistered.value && !containerHttpUnreachable.value) {
    try {
      await syncPerRepoGitIdentitiesToContainer()
    } catch {
      /* best-effort: 自动推失败时用户仍可手动点击按钮 */
    }
  }
}
// 发送给 AI — delegated to extracted pure function
const submitAIComment = () => _submitAIComment({
  effectiveTenantId, effectiveWorkspaceId, effectiveTaskId,
  selectedLayerGraphNode, layerGraphSnapshot,
  layerGraphAutoIterationCount, layerGraphSelectedModel,
  layerGraphCommandKind, layerGraphModelProvider,
  newComment, commentComposerChips, activeAiInstructId,
  aiStreamBuffer, aiStreamBusy, localTask,
  normalizeSelectedAgentModels, buildCommentBodyWithChips,
  markContainerTransportUnreachableIfForwardingFailed,
  fetchTaskDetail,
  get aiInstructAbortController() { return d.aiInstructAbortController },
  set aiInstructAbortController(v) { d.aiInstructAbortController = v },
})

const aiStreamBusy = ref(false)
const aiStreamBuffer = ref('')
/** 当前一次「发送给 AI」对应的评论 id，用于只合并同一次 SSE 流式正文 */
const activeAiInstructId = ref(null)

const submitLayerGraphCommand = (commentId) => {
  const cid = String(commentId || '').trim()
  const slice = cid ? layerPanelStore.refsFor(cid) : null
  return _submitLayerGraphCommand({
  effectiveTenantId, effectiveWorkspaceId, effectiveTaskId,
  selectedLayerGraphNode: slice?.selectedLayerGraphNode || selectedLayerGraphNode,
  layerGraphSnapshot: slice?.layerGraphSnapshot || layerGraphSnapshot,
  layerGraphCommandText: slice?.layerGraphCommandText || layerGraphCommandText,
  layerGraphCommandKind: slice?.layerGraphCommandKind || layerGraphCommandKind,
  layerGraphCmdError: slice?.layerGraphCmdError || layerGraphCmdError,
  layerGraphCmdErrorTraceId: slice?.layerGraphCmdErrorTraceId || layerGraphCmdErrorTraceId,
  layerGraphCmdSending: slice?.layerGraphCmdSending || layerGraphCmdSending,
  layerGraphAutoIterationCount: slice?.layerGraphAutoIterationCount || layerGraphAutoIterationCount,
  layerGraphSelectedModel: slice?.layerGraphSelectedModel || layerGraphSelectedModel,
  layerGraphModelProvider,
  layerGraphEditRunTargetJobId,
  validateLinkedProjectReposBeforeSendToAi, normalizeSelectedAgentModels,
  formatLayerGraphCommandErrorForUser,
  markContainerTransportUnreachableIfForwardingFailed, markContainerTransportOk,
  refreshLayerGraphFromServer: d.refreshLayerGraphFromServer, bumpProjectFileTreeRefresh,
  ...commentForwardScope, localTask,
  commentId: cid,
  containerReleased: zlog.containerReleased,
  layerPanelStore,
})
}

const onLayerGraphLayerSubmit = (node) => _onLayerGraphLayerSubmit(node, {
  containerEndpointRegistered, containerHttpUnreachable, containerPageUrl,
  taskRepoRows, repoCloneIdentityIdForUrl, layerGraphBusyActionKey,
  effectiveTenantId, effectiveWorkspaceId, effectiveTaskId,
  markContainerTransportUnreachableIfForwardingFailed, markContainerTransportOk,
  layerChangesByLayerId, refreshLayerGraphFromServer: d.refreshLayerGraphFromServer, refreshZTreeExecutionLog,
  ...commentForwardScope,
})

const onLayerChangesListCommitStaged = (payload) => _onLayerChangesListCommitStaged(payload, {
  containerEndpointRegistered, containerHttpUnreachable, containerPageUrl,
  taskRepoRows, repoCloneIdentityIdForUrl, layerChangesByLayerId,
  effectiveTenantId, effectiveWorkspaceId, effectiveTaskId,
  markContainerTransportUnreachableIfForwardingFailed, markContainerTransportOk,
  refreshLayerGraphFromServer: d.refreshLayerGraphFromServer, refreshZTreeExecutionLog, refreshSelectedLayerChanges,
  bumpProjectFileTreeRefresh,
  ...commentForwardScope,
})

const onLayerGraphLayerPush = (node) => _onLayerGraphLayerPush(node, {
  containerEndpointRegistered, containerHttpUnreachable, containerPageUrl,
  taskRepoRows, repoCloneIdentityIdForUrl, firstTaskRepoCloneIdentityId,
  layerGraphPushTargetBranch, layerGitGithubAppOauthConnected, layerGraphBusyActionKey,
  effectiveTenantId, effectiveWorkspaceId, effectiveTaskId,
  markContainerTransportUnreachableIfForwardingFailed, markContainerTransportOk,
  refreshLayerGraphFromServer: d.refreshLayerGraphFromServer, refreshZTreeExecutionLog, fetchTaskDetail,
  layerGraphSnapshot,
  ...commentForwardScope,
})

const onLayerGraphLayerMerge = (node) => _onLayerGraphLayerMerge(node, {
  containerEndpointRegistered, containerHttpUnreachable, containerPageUrl,
  layerGraphMergeTargetBranch, layerGraphBusyActionKey,
  effectiveTenantId, effectiveWorkspaceId, effectiveTaskId,
  markContainerTransportUnreachableIfForwardingFailed, markContainerTransportOk,
  refreshLayerGraphFromServer: d.refreshLayerGraphFromServer, refreshZTreeExecutionLog,
  ...commentForwardScope,
})

const onLayerGraphLayerSubmitAndPush = (node) => _onLayerGraphLayerSubmitAndPush(node, {
  containerEndpointRegistered, containerHttpUnreachable, containerPageUrl,
  taskRepoRows, repoCloneIdentityIdForUrl, firstTaskRepoCloneIdentityId,
  layerGraphPushTargetBranch, layerGitGithubAppOauthConnected, layerGraphBusyActionKey,
  effectiveTenantId, effectiveWorkspaceId, effectiveTaskId,
  markContainerTransportUnreachableIfForwardingFailed, markContainerTransportOk,
  refreshLayerGraphFromServer: d.refreshLayerGraphFromServer, refreshZTreeExecutionLog, fetchTaskDetail,
  layerGraphSnapshot, layerChangesByLayerId,
  ...commentForwardScope,
})

const onLayerGraphLayerSubmitAndMerge = (node) => _onLayerGraphLayerSubmitAndMerge(node, {
  containerEndpointRegistered, containerHttpUnreachable, containerPageUrl,
  taskRepoRows, repoCloneIdentityIdForUrl,
  layerGraphMergeTargetBranch, layerGraphBusyActionKey,
  effectiveTenantId, effectiveWorkspaceId, effectiveTaskId,
  markContainerTransportUnreachableIfForwardingFailed, markContainerTransportOk,
  refreshLayerGraphFromServer: d.refreshLayerGraphFromServer, refreshZTreeExecutionLog,
  layerGraphSnapshot, layerChangesByLayerId,
  ...commentForwardScope,
})

const maybeAutoApplyDefaultRepoCloneIdentities = () => _maybeAutoApplyDefaultRepoCloneIdentities({
  effectiveTenantId, effectiveWorkspaceId, effectiveTaskId,
  layerGitIdentityOptions, layerGitIdentityLoading, workspaceProjectsLoading,
  taskRepoRows, repoCloneIdentityIdForUrl,
  repoCloneIdentityByUrl, repoCloneIdentitySaving, repoCloneIdentitySaveError,
  localTask, repoCloneIdentityUserTouchedByUrl, repoCloneIdentityAutoApplyInFlight,
})

watch(
  [layerGitIdentityOptions, taskRepoRows, layerGitIdentityLoading, workspaceProjectsLoading],
  () => { void maybeAutoApplyDefaultRepoCloneIdentities() },
  { deep: true },
)

const fetchLayerRepoGitIdentities = () => _fetchLayerRepoGitIdentities({
  effectiveTenantId, effectiveWorkspaceId, effectiveTaskId,
  containerEndpointRegistered, selectedLayerGraphFileTreeLayerId,
  layerRepoGitIdentityLoading, layerRepoGitIdentityFetchError, layerRepoGitIdentityRows,
  ...commentForwardScope,
})

const syncPerRepoGitIdentitiesToContainer = () => _syncPerRepoGitIdentitiesToContainer({
  effectiveTenantId, effectiveWorkspaceId, effectiveTaskId,
  containerEndpointRegistered, containerHeartbeatStatus,
  selectedLayerGraphFileTreeLayerId, containerPageUrl,
  taskRepoRows, repoCloneIdentityIdForUrl,
  perRepoGitIdentitySyncing, perRepoGitIdentitySyncError,
  containerLayerGraphAuthInvalid, fetchLayerRepoGitIdentities,
  ...commentForwardScope,
})

d.layerGraphHydrateInFlight = false
const rescheduleContainerCloneProgressClearTimer = () => _rescheduleContainerCloneProgressClearTimer(cpState)

d.applyLayerGraphFromPayload = createApplyLayerGraphFromPayload({
  layerGraphSnapshot,
  containerLayerGraphAuthInvalid,
})

d.ensureServerRuntimeAllowsContainerLayerGraph = (bypassCache) => _ensureServerRuntimeAllowsContainerLayerGraph(bypassCache, {
  containerEndpointRegistered, containerHeartbeatStatus,
  containerHttpUnreachable, effectiveTenantId, effectiveWorkspaceId, effectiveTaskId,
  get serverRuntimeLayerGraphGateCheckedAt() { return runtimeGate.serverRuntimeLayerGraphGateCheckedAt },
  set serverRuntimeLayerGraphGateCheckedAt(v) { runtimeGate.serverRuntimeLayerGraphGateCheckedAt = v },
  get serverRuntimeLayerGraphGateAllowed() { return runtimeGate.serverRuntimeLayerGraphGateAllowed },
  set serverRuntimeLayerGraphGateAllowed(v) { runtimeGate.serverRuntimeLayerGraphGateAllowed = v },
  SERVER_RUNTIME_LAYER_GRAPH_GATE_MS,
  onServerRuntimeNotServing: () => enterServerNotServingUiState({ fromRuntime: true }),
  onServerRuntimeServing: (rs) => updateServerStatus({
    status: 'runtime_hydrate',
    runtime_status: rs || 'Running',
  }),
})

/** 容器已注册 server_url 时，经 Django 转发拉取层级（与 SSE container_layer_graph 结构一致） */
d.refreshLayerGraphFromServer = (force, opts) => _refreshLayerGraphFromServer(force, opts, {
  containerEndpointRegistered, containerLayerGraphAuthInvalid,
  effectiveTenantId, effectiveWorkspaceId, effectiveTaskId,
  ...commentForwardScope,
  layerPanelStore,
  ensureServerRuntimeAllowsContainerLayerGraph: d.ensureServerRuntimeAllowsContainerLayerGraph,
  applyLayerGraphFromPayload: d.applyLayerGraphFromPayload,
  registerLayerGraphFetchFailure, resetLayerGraphFetchBackoff,
  markContainerTransportOk, markContainerTransportUnreachableIfForwardingFailed,
  selectedLayerGraphNode, layerGraphZNodes, onLayerGraphNodeSelect,
  layerGraphSelectionDismissedByUser,
  get layerGraphSnapshot() { return layerGraphSnapshot },
  layerGraphSnapshotHasContent, layerGraphSnapshotHasActiveJob,
  get layerGraphFetchBackoffUntil() { return lgBackoff.layerGraphFetchBackoffUntil },
  set layerGraphFetchBackoffUntil(v) { lgBackoff.layerGraphFetchBackoffUntil = v },
  get layerGraphHydrateInFlight() { return d.layerGraphHydrateInFlight },
  set layerGraphHydrateInFlight(v) { d.layerGraphHydrateInFlight = v },
  nextTick,
  // OPT-20260903-002：层图快照补写 git_pr 子评论后，评论 Feed 若缺该 PR URL 则再拉一次。
  getExistingGitPrHtmlUrls: () => collectGitPrHtmlUrls(displayComments.value),
  onLayerPrBackfillDetected: () => {
    if (typeof d.fetchTaskDetail === 'function') void d.fetchTaskDetail()
  },
})

const lgHeartbeatRefresh = createLayerGraphHeartbeatRefresh({
  containerEndpointRegistered,
  containerHttpUnreachable,
  containerLayerGraphAuthInvalid,
  containerPageLinkPendingReveal: cpState.containerPageLinkPendingReveal,
  layerGraphSnapshot,
  refreshLayerGraphFromServer: (force, opts) => d.refreshLayerGraphFromServer(force, opts),
  fetchContainerBootstrapCloneLog,
})
const { maybeRefreshLayerGraphOnContainerHeartbeatOk } = lgHeartbeatRefresh

/**
 * 引导克隆日志已写出「完成」时，将 SSE 维护的 `containerCloneProgressByKey` 收束到 100%，
 * 避免折叠区内全文已结束而横幅 `message`/百分比仍被滞后事件拉到旧状态。
 */
const applyBootstrapCloneLogDoneToCloneProgress = () => _applyBootstrapCloneLogDoneToCloneProgress(cpState, { refreshLayerGraphFromServer: d.refreshLayerGraphFromServer, bumpProjectFileTreeRefresh })

watch(
  [containerBootstrapCloneLogFull, containerCloneProgressByKey, containerCloneProgressByCommentId],
  () => {
    applyBootstrapCloneLogDoneToCloneProgress()
  },
  { deep: true },
)

const containerHeartbeatSseBuffer = createContainerHeartbeatSseBuffer()
const syncContainerHeartbeatAfterServingHint = () => {
  promoteContainerHeartbeatFromIdle({
    containerHeartbeatStatus,
    containerHeartbeatPaused,
    containerHeartbeatError,
    containerEndpointRegistered,
    isServerRunning,
    isServerStarting,
    serverRuntimeNotServing,
  })
  flushPendingContainerHeartbeat(containerHeartbeatSseBuffer, {
    containerHeartbeatStatus,
    containerHeartbeatAttempts,
    containerHeartbeatLastSuccess,
    containerHeartbeatError,
    applyContainerHeartbeatSeqFromSse,
    markContainerTransportOk,
    resetServerRuntimeLayerGraphGateCache,
    maybeRefreshLayerGraphOnContainerHeartbeatOk,
    containerHeartbeatPaused,
    isServerRunning,
    isServerStarting,
    serverRuntimeNotServing,
    containerEndpointRegistered,
  })
}

const fetchContainerTaskUiContext = (commentId) => _fetchContainerTaskUiContext({
  effectiveTenantId, effectiveWorkspaceId, effectiveTaskId,
  ...commentForwardScope,
  layerPanelStore,
  containerEndpointRegistered, containerPageUrl, containerVscodeUrl,
  containerPageLinkPendingReveal, markContainerTransportOk, refreshLayerGraphFromServer: d.refreshLayerGraphFromServer,
  ensureServerRuntimeAllowsContainerLayerGraph: d.ensureServerRuntimeAllowsContainerLayerGraph,
  syncContainerHeartbeatAfterServingHint,
  containerHeartbeatSseBuffer,
  containerHeartbeatStatus,
  containerHeartbeatPaused,
  isServerRunning,
  isServerStarting,
  serverRuntimeNotServing,
})

watch(
  containerForwardCommentId,
  (id, prev) => {
    if (id && id !== prev) void fetchContainerTaskUiContext(id)
  },
)

installCommentLayerPanelHydration(layerPanelStore, {
  listCscCommentIds: () => (displayComments.value || [])
    .map((c) => c?.id)
    .filter((id) => commentOwnsSharedContainer(id)),
  fetchContainerTaskUiContext,
})

const sse = createSseReconnectState({
  sseLive,
  effectiveTaskId,
  get serverStartupStatusPoll() { return d.serverStartupStatusPoll },
  establishSSEConnection: (taskId) => d.establishSSEConnection(taskId),
})
const {
  sseConnection,
  sseReconnectAttempts,
  sseReconnecting,
  ssePlatformRestartHint,
  closeSSEConnection,
  scheduleSSEReconnect,
  handleSSEManualReconnect,
} = sse

// 更新服务器状态 — delegated to extracted pure function
const updateServerStatus = (statusData) => {
  _updateServerStatus(statusData, {
    serverConfigRef, activeAiInstructId, aiStreamBuffer, aiStreamBusy,
    activeContainerAgentId,
    containerBootstrapCloneLogFull, containerCloneProgressByKey,
    containerCloneProgressByCommentId,
    containerEndpointRegistered, containerPageUrl, containerVscodeUrl,
    containerPageLinkPendingReveal, layerGraphSnapshot, layerChangesByLayerId,
    containerBootstrapFailureMessage, containerBootstrapFailureTraceId,
    layerPanelStore,
    ...commentForwardScope,
    layerJobLiveOutputMap, layerJobExecutionPayload, isServerRunning, isServerStarting, serverUrl,
    serverStatus, statusMessage, statusEventCommentId: statusCommentId, statusEventRuntimeStatus: statusRuntimeStatus, statusTraceId, statusProgress, statusLogs,
    cloudRuntimeStatus, markServerRuntimeServing, serverRuntimeNotServing,
    fetchTaskDetail, markContainerTransportOk, onBootstrapCloneLogUpdate,
    refreshLayerGraphFromServer: d.refreshLayerGraphFromServer, bumpProjectFileTreeRefresh,
    rescheduleContainerCloneProgressClearTimer, normalizeLayerChangesPayload,
    applyLiveLayerChangesToCurrentPayload, refreshZTreeExecutionLog,
    stopContainerHeartbeat, fetchContainerTaskUiContext, pauseContainerHeartbeatForRelayStop,
    syncContainerHeartbeatAfterServingHint,
    get containerCloneProgressTimer() { return cpState.containerCloneProgressTimer },
    set containerCloneProgressTimer(v) { cpState.containerCloneProgressTimer = v },
    CONTAINER_CLONE_PROGRESS_GLOBAL_KEY,
  })
}

bindEstablishSSEConnection(d, {
  _establishSSEConnection,
  sseConnection, sseLive, sseReconnecting, sseReconnectAttempts, ssePlatformRestartHint,
  containerHeartbeatStatus, containerHeartbeatAttempts,
  containerHeartbeatLastSuccess, containerHeartbeatError,
  closeSSEConnection, scheduleSSEReconnect, markContainerTransportOk,
  resetServerRuntimeLayerGraphGateCache, maybeRefreshLayerGraphOnContainerHeartbeatOk,
  updateServerStatus, applyContainerHeartbeatSeqFromSse,
  effectiveTenantId, effectiveWorkspaceId, relayAccessCode, sse,
  containerHeartbeatPaused, isServerRunning, isServerStarting, serverRuntimeNotServing,
  containerEndpointRegistered, containerHeartbeatSseBuffer,
})

d.serverStartupStatusPoll = createServerStartupStatusPollController({
  effectiveTenantId,
  effectiveWorkspaceId,
  effectiveTaskId,
  isServerStarting,
  sseLive,
  statusProgress,
  updateServerStatus,
})

const handleStartRequestAccepted = (result) => {
  pendingStartEventId.value = result?.event_id != null ? String(result.event_id) : ''
  handleSSEManualReconnect()
  d.serverStartupStatusPoll.start(result)
}

watch([isServerStarting, sseLive], ([starting, live]) => {
  if (!starting) {
    pendingStartEventId.value = ''
    d.serverStartupStatusPoll.stop()
    return
  }
  // SSE 连接正常时也启动 REST 兜底轮询（慢速），防止 SSE success 事件丢失导致永久卡「启动中」
  if (pendingStartEventId.value) {
    d.serverStartupStatusPoll.start({ event_id: pendingStartEventId.value })
  }
  // SSE 状态变化时自适应快/慢轮询间隔
  d.serverStartupStatusPoll.reschedule?.()
})

/** 从其他标签页/窗口回到本页时强制同步容器层图与执行日志，避免仅依赖已错过的 SSE 块 */
const onDocumentVisibilityForTaskLayerSync = () => {
  if (document.visibilityState !== 'visible') return
  void fetchContainerTaskUiContext()
  if (!containerEndpointRegistered.value) return
  if (containerPageLinkPendingReveal.value) return
  void d.refreshLayerGraphFromServer(true)
  void refreshZTreeExecutionLog()
  void fetchContainerBootstrapCloneLog()
}
  Object.assign(d, {
    onRepoCloneIdentityChange, submitAIComment, aiStreamBusy, aiStreamBuffer,
    activeAiInstructId, submitLayerGraphCommand, onLayerGraphLayerSubmit,
    onLayerChangesListCommitStaged, onLayerGraphLayerPush, onLayerGraphLayerMerge,
    onLayerGraphLayerSubmitAndPush, onLayerGraphLayerSubmitAndMerge,
    maybeAutoApplyDefaultRepoCloneIdentities, fetchLayerRepoGitIdentities,
    syncPerRepoGitIdentitiesToContainer, rescheduleContainerCloneProgressClearTimer,
    lgHeartbeatRefresh, applyBootstrapCloneLogDoneToCloneProgress,
    containerHeartbeatSseBuffer, syncContainerHeartbeatAfterServingHint,
    fetchContainerTaskUiContext, sse, updateServerStatus, handleStartRequestAccepted,
    onDocumentVisibilityForTaskLayerSync, maybeRefreshLayerGraphOnContainerHeartbeatOk,
    sseConnection, sseReconnectAttempts, sseReconnecting, ssePlatformRestartHint,
    closeSSEConnection, scheduleSSEReconnect, handleSSEManualReconnect,
  })
}

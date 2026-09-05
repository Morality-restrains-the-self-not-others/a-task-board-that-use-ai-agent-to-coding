/**
 * Extracted from useTaskDetail.js (OPT-20260823-007 line-limit split).
 */
import { isLoopbackHttpUrl } from '../../utils/httpUrlHost.js';
import { ref, computed, nextTick, watch, provide } from 'vue'
import {
  LAYER_TREE_NODE_PREFIX,
  normalizeLayerGitDirty,
} from '../../utils/layerZtreeNodes.js'
import {
  fetchContainerTaskUiContext as _fetchContainerTaskUiContext,
  resolveContainerUiContextCommentId,
  ensureServerRuntimeAllowsContainerLayerGraph as _ensureServerRuntimeAllowsContainerLayerGraph,
  refreshLayerGraphFromServer as _refreshLayerGraphFromServer,
  fetchContainerBootstrapCloneLog as _fetchContainerBootstrapCloneLog,
} from './taskDetailContainerFns.js'
import {
  CONTAINER_CLONE_PROGRESS_GLOBAL_KEY,
  BOOTSTRAP_CLONE_LOG_INDICATES_DONE_RE,
  createCloneProgressState,
  createCloneProgressComputeds,
  onBootstrapCloneLogUpdate as _onBootstrapCloneLogUpdate,
  rescheduleContainerCloneProgressClearTimer as _rescheduleContainerCloneProgressClearTimer,
  applyBootstrapCloneLogDoneToCloneProgress as _applyBootstrapCloneLogDoneToCloneProgress,
} from './taskDetailCloneProgress.js'
import { createContainerBootstrapFailureState } from './containerBootstrapFailureState.js'
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
  createTaskDetailLayerGraphState,
  installTaskDetailLayerGraphWatchers,
} from './taskDetailLayerGraphState.js'
import {
  createTaskDetailZTreeExecLogState,
  installTaskDetailZTreeExecLogWatchers,
  resolveZTreeLogTargets,
  normalizeLayerChangesPayload,
} from './taskDetailZTreeExecLogState.js'
import { createCommentLayerPanelStore } from './commentLayerPanelStore.js'
import { runtimeStatusFromCommentPanel } from './bindCommentRuntimePanel.js'
import { fetchTaskDetail as _fetchTaskDetail, loadMoreCommentFeeds as _loadMoreCommentFeeds, fetchWorkspaceCollaborators as _fetchWorkspaceCollaborators, fetchProgressStatusOptions as _fetchProgressStatusOptions, fetchDeliverableCategoryOptions as _fetchDeliverableCategoryOptions, fetchWorkspaceTodosForParent as _fetchWorkspaceTodosForParent, fetchParentDeliverableTitle as _fetchParentDeliverableTitle, fetchForkSourceTitle as _fetchForkSourceTitle, fetchTaskSubtree as _fetchTaskSubtree, onProgressStatusChange as _onProgressStatusChange, submitComment as _submitComment, onRepoReclone as _onRepoReclone, onRepoCloneIdentityChange as _onRepoCloneIdentityChange, maybeAutoApplyDefaultRepoCloneIdentities as _maybeAutoApplyDefaultRepoCloneIdentities } from './taskDetailFetchFns.js'

export function initUseTaskDetailContainer(d) {
  const {
    isServerRunning, isServerStarting, serverUrl, effectiveTenantId,
    effectiveWorkspaceId, effectiveTaskId, localTask, layerGraphPushTargetBranch,
    layerGraphMergeTargetBranch, serverConfigRef, progressStatusOptions,
    isUpdatingProgressStatus, progressStatusError, progressStatusErrorTraceId,
    commentForwardScope, containerForwardCommentId, commentComposerChips,
    bumpProjectFileTreeRefresh, taskRepoRows, repoCloneIdentityIdForUrl,
    firstTaskRepoCloneIdentityId, fetchTaskDetail, fetchTaskSubtree,
  } = d
/** 与 cloud_cloudserverconfig：容器 exchange-refresh 上报 business_api_endpoint 一致 */
const layerPanelStore = createCommentLayerPanelStore()
const containerEndpointRegistered = ref(false)
const containerPageUrl = ref('')
/** 容器内 code-server（VS Code Web）；仅在有映射 URL 时显示「打开容器开发页面」 */
const containerVscodeUrl = ref('')
/** 不展示指向环回地址的链接，避免注册完成前误开本机端口 */
const displayContainerVscodeUrl = computed(() => {
  const u = String(containerVscodeUrl.value || '').trim()
  if (!u) return ''
  return isLoopbackHttpUrl(u) ? '' : u
})
const lgBackoff = createLayerGraphFetchBackoffState()
const {
  resetLayerGraphFetchBackoff,
  registerLayerGraphFetchFailure,
} = lgBackoff

/** 容器经 server-container-token/layer-graph-push → SSE container_layer_graph 推送的层级快照（评论区可写层串行列表） */
const layerGraphSnapshot = ref(null)
const { containerBootstrapFailureMessage, containerBootstrapFailureTraceId } = createContainerBootstrapFailureState()

const onBootstrapCloneLogFailure = (message) => {
  const msg = String(message || '').trim()
  if (!msg) return
  containerBootstrapFailureMessage.value = msg
}

// Clone Progress — delegated to taskDetailCloneProgress.js
const cpState = createCloneProgressState()
const {
  containerCloneProgressByKey, containerCloneProgressByCommentId, containerBootstrapCloneLogFull,
  containerBootstrapCloneLogSegments,
  containerPageLinkPendingReveal,
} = cpState

/** 容器引导克隆经 SaaS SSE 上报的进度（可按 repo_url 多行并列），展示在「仓库列表与克隆身份」各仓（非持久化评论） */
const cpComputeds = createCloneProgressComputeds(cpState, { taskRepoRows })
const {
  bootstrapCloneDone,
  cloneProgressBarWidthTransitionClass,
  cloneProgressRowLogText, cloneProgressRowLogDisplayText,
  cloneProgressRowLogIsPlaceholder,
  containerCloneProgressEntries, cloneProgressEntryByRepoMatchKey,
} = cpComputeds

const onBootstrapCloneLogUpdate = (payload) => _onBootstrapCloneLogUpdate(payload, cpState)

const fetchContainerBootstrapCloneLog = () =>
  _fetchContainerBootstrapCloneLog({
    effectiveTenantId,
    effectiveWorkspaceId,
    effectiveTaskId,
    containerEndpointRegistered,
    onBootstrapCloneLogUpdate,
    onBootstrapCloneLogFailure,
    ...commentForwardScope,
  })

// 容器就绪后拉取引导克隆日志：进度 Map 在全局完成后会被清空，子仓状态需日志回落
watch(
  () => Boolean(containerEndpointRegistered.value),
  (ready) => {
    if (ready) void fetchContainerBootstrapCloneLog()
  },
  { immediate: true },
)

d.applyLayerGraphFromPayload = undefined
d.ensureServerRuntimeAllowsContainerLayerGraph = undefined
d.refreshLayerGraphFromServer = undefined
d.establishSSEConnection = undefined

const hb = createContainerHeartbeatState({
  effectiveTenantId,
  effectiveWorkspaceId,
  effectiveTaskId,
  ...commentForwardScope,
  containerEndpointRegistered,
  containerPageUrl,
  containerVscodeUrl,
  containerPageLinkPendingReveal: cpState.containerPageLinkPendingReveal,
  layerGraphSnapshot,
  serverUrl,
  isServerRunning,
  isServerStarting,
  resetLayerGraphFetchBackoff,
  ensureServerRuntimeAllowsContainerLayerGraph: (bypassCache) => d.ensureServerRuntimeAllowsContainerLayerGraph(bypassCache),
  refreshLayerGraphFromServer: (force, opts) => d.refreshLayerGraphFromServer(force, opts),
  applyLayerGraphFromPayload: (data, opts) => d.applyLayerGraphFromPayload(data, opts),
  establishSSEConnection: (taskId) => d.establishSSEConnection(taskId),
})

const {
  containerHttpUnreachable,
  containerLayerGraphAuthInvalid,
  containerHeartbeatStatus,
  containerHeartbeatPaused,
  serverRuntimeNotServing,
  containerHeartbeatLastSuccess,
  containerHeartbeatAttempts,
  containerHeartbeatError,
  containerHeartbeatSeqInfo,
  containerHeartbeatLogLines,
  appendContainerHeartbeatLogLines,
  applyContainerHeartbeatSeqFromSse,
  markContainerTransportOk,
  startContainerHeartbeat,
  stopContainerHeartbeat,
  pauseContainerHeartbeatForRelayStop,
  resumeContainerHeartbeatForRelayStart,
  markServerRuntimeServing,
  enterServerNotServingUiState,
  isContainerAccessTokenInvalidDetail,
  formatLayerGraphCommandErrorForUser,
  markContainerTransportUnreachableIfForwardingFailed,
  probeContainerReachability,
  scheduleContainerUnreachableProbe,
  clearContainerUnreachableProbeTimer,
} = hb

const runtimeGate = createServerRuntimeLayerGraphGateState({
  containerEndpointRegistered,
  containerHeartbeatStatus: hb.containerHeartbeatStatus,
  containerHttpUnreachable: hb.containerHttpUnreachable,
})
const {
  resetServerRuntimeLayerGraphGateCache,
} = runtimeGate

/** 层图 / zTree 执行日志：跨模块共享 refs */
const selectedLayerGraphNode = ref(null)
const layerChangesByLayerId = ref({})
const layerGraphCommandText = ref('')
const taskLayerAssociationPanelRef = ref(null)

// OPT-20260823-049: 与徽章同一判定——binding 仍 running 但评论运行时快照 Released 时，
// zlog skip 也能命中（避免释放后首屏短暂打 clone-log 拿 409）。
const runtimeStatusFor = (commentId) => {
  const panel = serverConfigRef.value?.serverRuntimeStatusPanel
  return runtimeStatusFromCommentPanel(panel, commentId)
}

const zlog = createTaskDetailZTreeExecLogState({
  effectiveTenantId,
  effectiveWorkspaceId,
  effectiveTaskId,
  commentId: containerForwardCommentId,
  ...commentForwardScope,
  selectedLayerGraphNode,
  layerGraphSnapshot,
  layerChangesByLayerId,
  containerEndpointRegistered,
  containerHttpUnreachable,
  taskRepoRows,
  repoCloneIdentityIdForUrl,
  markContainerTransportUnreachableIfForwardingFailed,
  markContainerTransportOk,
  bumpProjectFileTreeRefresh,
  commentComposerChips,
  layerGraphCommandText,
  taskLayerAssociationPanelRef,
  layerPanelStore,
  serverRuntimeNotServing,
  runtimeStatusFor,
})
const {
  layerExecLogLoading,
  layerExecLogTopError,
  layerCloneLogText,
  layerCloneLogFetchError,
  layerCloneLogFetchErrorTraceId,
  layerJobLogFetchError,
  layerJobLogFetchErrorTraceId,
  layerJobExecutionPayload,
  layerJobLiveOutputMap,
  layerChangesRefreshBusy,
  layerChangesRefreshError,
  layerChangesRefreshErrorTraceId,
  selectedZTreeLayerChangesPanel,
  selectedLayerGraphFileTreeLayerId,
  selectedLayerGraphNodeLogKey,
  zTreeLogTargets,
  layerChangesRefreshEnabled,
  layerChangesGitStagedActionsBlocked,
  layerChangesGitCommitIdentityBlocked,
  ingestLayerChangesFromExecutionPayload,
  refreshSelectedLayerChanges,
  loadMoreSelectedLayerChanges,
  applyLiveLayerChangesToCurrentPayload,
  layerJobOutputDisplay,
  layerLiveOutputDisplay,
  layerAgentSteps,
  layerAgentStepCards,
  onAgentStepRichInteract,
  copyAgentStepJson,
  layerJobCommandHead,
  layerExecLogCopyText,
  layerExecLogCopyable,
  layerExecLogClearable,
  layerExecLogCopyFeedback,
  layerAgentStepCopyFeedbackKey,
  copyLayerExecLog,
  clearLayerExecLog,
  prefetchLayerChangeSummariesForDirtyLayers,
  refreshZTreeExecutionLog,
  activeJobExecLogPoller,
  layerChangesPrefetchInFlight,
} = zlog

installTaskDetailZTreeExecLogWatchers(zlog, {
  layerGraphSnapshot,
  containerEndpointRegistered,
  containerHttpUnreachable,
})

const lg = createTaskDetailLayerGraphState({
  effectiveTenantId,
  effectiveWorkspaceId,
  effectiveTaskId,
  ...commentForwardScope,
  localTask,
  layerGraphSnapshot,
  layerGraphMergeTargetBranch,
  layerChangesByLayerId,
  selectedLayerGraphNode,
  layerGraphCommandText,
  taskLayerAssociationPanelRef,
  containerEndpointRegistered,
  containerHttpUnreachable,
  markContainerTransportUnreachableIfForwardingFailed,
  markContainerTransportOk,
  containerReleased: zlog.containerReleased,
  refreshLayerGraphFromServer: (force, opts) => d.refreshLayerGraphFromServer(force, opts),
  fetchTaskDetail,
  taskRepoRows,
  repoCloneIdentityIdForUrl,
  firstTaskRepoCloneIdentityId,
  layerGraphPushTargetBranch,
  containerPageUrl,
  startContainerHeartbeat,
  stopContainerHeartbeat,
  prefetchLayerChangeSummariesForDirtyLayers,
  refreshZTreeExecutionLog,
  activeJobExecLogPoller,
  zTreeLogTargets,
  layerChangesRefreshError,
  layerPanelStore,
  resolveActiveLayerCommentId: () => resolveContainerUiContextCommentId(commentForwardScope),
})
const {
  layerGraphSeenLayerIdSet,
  layerGraphRefreshing,
  refreshLayerGraph,
  layerGraphZNodes,
  layerGraphMetaLine,
  layerGraphLayerIdsKey,
  layerGraphCommandKind,
  layerGraphAutoIterationCount,
  layerGraphModelProvider,
  layerGraphDefaultModel,
  layerGraphModelOptions,
  layerGraphSelectedModel,
  layerGraphModelLoading,
  layerGraphModelLoadError,
  layerGraphModelLoadErrorTraceId,
  layerGraphCmdError,
  layerGraphCmdErrorTraceId,
  layerGraphCmdSending,
  layerGraphEditRunTargetJobId,
  layerGraphBusyActionKey,
  layerGraphSelectionDismissedByUser,
  layerGitIdentityOptions,
  layerGitIdentityLoading,
  layerGitGithubAppOauthConnected,
  layerRepoGitIdentityLoading,
  layerRepoGitIdentityFetchError,
  layerRepoGitIdentityRows,
  perRepoGitIdentitySyncing,
  perRepoGitIdentitySyncError,
  fetchLayerGraphModelOptions,
  fetchLayerGitIdentityOptions,
  profileGitIdentitiesApiPath,
  containerGitIdentityRowForRepoUrl,
  containerGitIdentityLineForRepoUrl,
  onLayerGraphNodeSelect,
  callLayerGraphJobAction,
  callLayerGraphLayerDelete,
  onLayerGraphJobRedo,
  onLayerGraphJobInterrupt,
  onLayerGraphJobContinue,
  onLayerGraphJobDelete,
  onLayerGraphLayerDelete,
  onLayerGraphJobEditRun,
  pickNewestAddedLayerIdFromSnapshot,
  normalizeSelectedAgentModels,
} = lg

installTaskDetailLayerGraphWatchers(lg, {
  layerGraphSnapshot,
  effectiveTenantId,
  effectiveWorkspaceId,
  localTask,
  containerEndpointRegistered,
  selectedLayerGraphFileTreeLayerId,
  refreshZTreeExecutionLog,
  activeJobExecLogPoller,
  prefetchLayerChangeSummariesForDirtyLayers,
  startContainerHeartbeat,
  stopContainerHeartbeat,
  refreshLayerGraphFromServer: (force, opts) => d.refreshLayerGraphFromServer(force, opts),
  containerHeartbeatStatus,
  containerHeartbeatPaused,
  serverRuntimeNotServing,
  isServerRunning,
  isServerStarting,
  zTreeLogTargets,
  selectedLayerGraphNodeLogKey,
})

/** 未注册端点，或已注册但 HTTP 转发不可达时，禁用依赖容器链路的按钮 */
const containerActionsBlocked = computed(
  () => !containerEndpointRegistered.value || containerHttpUnreachable.value
)
const layerGraphModelSelectDisabled = computed(
  () => layerGraphCmdSending.value || layerGraphModelLoading.value || layerGraphCommandKind.value !== 'trae'
)

const onProgressStatusChange = (event) => {
  // OPT-20260724-029: 标记为终态（完成/已关闭）前检查未提交变更
  const nextStatusId = event?.target?.value ? String(event.target.value) : ''
  const selected = progressStatusOptions.value.find((item) => String(item.id) === nextStatusId)
  const isTerminal = selected?.name && /完成|completed|done|已结束|已关闭|关闭/i.test(selected.name)
  if (isTerminal && Array.isArray(layerGraphZNodes.value) && layerGraphZNodes.value.length > 0) {
    const dirtyLayers = layerGraphZNodes.value.filter(
      (n) => normalizeLayerGitDirty(n?.git_worktree_dirty) === true
    )
    if (dirtyLayers.length > 0) {
      const confirmed = window.confirm(
        `容器中仍有 ${dirtyLayers.length} 个层级存在未提交的代码变更。\n\n` +
        '标记为完成将触发容器释放，未提交的变更可能丢失。\n\n' +
        '建议先提交变更后再标记完成。是否仍要继续？'
      )
      if (!confirmed) return
    }
  }
  return _onProgressStatusChange(event, {
    effectiveTenantId, effectiveWorkspaceId, effectiveTaskId,
    progressStatusOptions, isUpdatingProgressStatus, progressStatusError,
    progressStatusErrorTraceId, localTask, fetchTaskSubtree,
  })
}
  Object.assign(d, {
    layerPanelStore, containerEndpointRegistered, containerPageUrl, containerVscodeUrl,
    displayContainerVscodeUrl, lgBackoff, layerGraphSnapshot, onBootstrapCloneLogFailure,
    cpState, cpComputeds, onBootstrapCloneLogUpdate, fetchContainerBootstrapCloneLog, hb,
    runtimeGate, selectedLayerGraphNode, layerChangesByLayerId, layerGraphCommandText,
    taskLayerAssociationPanelRef, runtimeStatusFor, zlog, lg, containerActionsBlocked,
    layerGraphModelSelectDisabled, onProgressStatusChange, resetLayerGraphFetchBackoff,
    registerLayerGraphFetchFailure, containerBootstrapFailureMessage,
    containerBootstrapFailureTraceId, containerCloneProgressByKey,
    containerCloneProgressByCommentId, containerBootstrapCloneLogFull,
    containerBootstrapCloneLogSegments, containerPageLinkPendingReveal,
    bootstrapCloneDone, cloneProgressBarWidthTransitionClass, cloneProgressRowLogText,
    cloneProgressRowLogDisplayText, cloneProgressRowLogIsPlaceholder,
    containerCloneProgressEntries, cloneProgressEntryByRepoMatchKey,
    containerHttpUnreachable, containerLayerGraphAuthInvalid, containerHeartbeatStatus,
    containerHeartbeatPaused, serverRuntimeNotServing, containerHeartbeatLastSuccess,
    containerHeartbeatAttempts, containerHeartbeatError, containerHeartbeatSeqInfo,
    containerHeartbeatLogLines, appendContainerHeartbeatLogLines,
    applyContainerHeartbeatSeqFromSse, markContainerTransportOk, startContainerHeartbeat,
    stopContainerHeartbeat, pauseContainerHeartbeatForRelayStop,
    resumeContainerHeartbeatForRelayStart, markServerRuntimeServing,
    enterServerNotServingUiState, isContainerAccessTokenInvalidDetail,
    formatLayerGraphCommandErrorForUser,
    markContainerTransportUnreachableIfForwardingFailed, probeContainerReachability,
    scheduleContainerUnreachableProbe, clearContainerUnreachableProbeTimer,
    resetServerRuntimeLayerGraphGateCache, layerExecLogLoading, layerExecLogTopError,
    layerCloneLogText, layerCloneLogFetchError, layerCloneLogFetchErrorTraceId,
    layerJobLogFetchError, layerJobLogFetchErrorTraceId, layerJobExecutionPayload,
    layerJobLiveOutputMap, layerChangesRefreshBusy, layerChangesRefreshError,
    layerChangesRefreshErrorTraceId, selectedZTreeLayerChangesPanel,
    selectedLayerGraphFileTreeLayerId, selectedLayerGraphNodeLogKey, zTreeLogTargets,
    layerChangesRefreshEnabled, layerChangesGitStagedActionsBlocked,
    layerChangesGitCommitIdentityBlocked, ingestLayerChangesFromExecutionPayload,
    refreshSelectedLayerChanges, loadMoreSelectedLayerChanges,
    applyLiveLayerChangesToCurrentPayload, layerJobOutputDisplay, layerLiveOutputDisplay,
    layerAgentSteps, layerAgentStepCards, onAgentStepRichInteract, copyAgentStepJson,
    layerJobCommandHead, layerExecLogCopyText, layerExecLogCopyable,
    layerExecLogClearable, layerExecLogCopyFeedback, layerAgentStepCopyFeedbackKey,
    copyLayerExecLog, clearLayerExecLog, prefetchLayerChangeSummariesForDirtyLayers,
    refreshZTreeExecutionLog, activeJobExecLogPoller, layerChangesPrefetchInFlight,
    layerGraphSeenLayerIdSet, layerGraphRefreshing, refreshLayerGraph, layerGraphZNodes,
    layerGraphMetaLine, layerGraphLayerIdsKey, layerGraphCommandKind,
    layerGraphAutoIterationCount, layerGraphModelProvider, layerGraphDefaultModel,
    layerGraphModelOptions, layerGraphSelectedModel, layerGraphModelLoading,
    layerGraphModelLoadError, layerGraphModelLoadErrorTraceId, layerGraphCmdError,
    layerGraphCmdErrorTraceId, layerGraphCmdSending, layerGraphEditRunTargetJobId,
    layerGraphBusyActionKey, layerGraphSelectionDismissedByUser, layerGitIdentityOptions,
    layerGitIdentityLoading, layerGitGithubAppOauthConnected,
    layerRepoGitIdentityLoading, layerRepoGitIdentityFetchError,
    layerRepoGitIdentityRows, perRepoGitIdentitySyncing, perRepoGitIdentitySyncError,
    fetchLayerGraphModelOptions, fetchLayerGitIdentityOptions,
    profileGitIdentitiesApiPath, containerGitIdentityRowForRepoUrl,
    containerGitIdentityLineForRepoUrl, onLayerGraphNodeSelect, callLayerGraphJobAction,
    callLayerGraphLayerDelete, onLayerGraphJobRedo, onLayerGraphJobInterrupt,
    onLayerGraphJobContinue, onLayerGraphJobDelete, onLayerGraphLayerDelete,
    onLayerGraphJobEditRun, pickNewestAddedLayerIdFromSnapshot,
    normalizeSelectedAgentModels,
  })
}

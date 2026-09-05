#!/usr/bin/env python3
"""One-shot patch: extract project/repo + layer graph domains from useTaskDetail.js."""
from pathlib import Path

path = Path(__file__).resolve().parents[1] / 'src/composables/useTaskDetail.js'
text = path.read_text()

IMPORTS = """
import {
  createTaskDetailProjectRepoState,
  installTaskDetailProjectRepoWatchers,
} from './taskDetail/taskDetailProjectRepoState.js'
import {
  createTaskDetailLayerGraphState,
  installTaskDetailLayerGraphWatchers,
} from './taskDetail/taskDetailLayerGraphState.js'
import {
  createTaskDetailZTreeExecLogState,
  resolveZTreeLogTargets,
  normalizeLayerChangesPayload,
} from './taskDetail/taskDetailZTreeExecLogState.js'
"""

if 'createTaskDetailProjectRepoState' not in text:
    anchor = "import { resolveParentDeliverableBlockedReason } from '../utils/workPanelCreateTaskParent.js'\n"
    text = text.replace(anchor, anchor + IMPORTS + "\n", 1)

# --- project/repo slice ---
pr_start = text.index('const linkedProjectsPanelRef = ref(null)')
pr_end = text.index('const removeProjectAssociation = (index) =>')
pr_block = '''const pr = createTaskDetailProjectRepoState({
  effectiveTenantId,
  effectiveWorkspaceId,
  effectiveTaskId,
  localTask,
  isEditing,
  editingTask,
})
const {
  linkedProjectsPanelRef,
  workspaceProjects,
  workspaceProjectsLoading,
  projectServerRunTemplate,
  repoCloneIdentityByUrl,
  repoCloneIdentitySaveError,
  repoCloneIdentitySaving,
  repoCloneIdentityUserTouchedByUrl,
  repoCloneIdentityAutoApplyInFlight,
  recloneLoadingByUrl,
  recloneStatusByUrl,
  repoRecloneGlobalLoading,
  staleRepoSyncLoading,
  staleRepoSyncError,
  repoBranchesCache,
  repoBranchesLoading,
  repoBranchesErrors,
  taskProjectsWithDetails,
  taskRepoRows,
  syncStaleTaskRepoAddresses,
  syncRepoCloneIdentityMapFromTask,
  fetchWorkspaceProjects,
  savedRepoCloneIdentityIdForUrl,
  validateLinkedProjectReposBeforeSendToAi,
  repoCloneIdentityIdForUrl,
  firstTaskRepoCloneIdentityId,
  getProjectRepos,
  buildProjectsApiPayload,
  getRepoBranchValue,
  setRepoBranch,
  getRepoBranches,
  getRepoBranchError,
  isLoadingRepoBranches,
  fetchRepoBranches,
  onProjectChange,
  linkedRepoBranchTargets,
  fetchAllLinkedRepoBranches,
  commonMergeTargetBranchesLoading,
  commonMergeTargetBranchesError,
  commonMergeTargetBranches,
} = pr
installTaskDetailProjectRepoWatchers(pr, { localTask, isEditing, effectiveTaskId })

'''
text = text[:pr_start] + pr_block + text[pr_end:]

# thin wrappers still needed after title translation block
for old, new in [
    (
        '''const syncRepoCloneIdentityMapFromTask = () => {
  const raw = localTask.value?.parameters?.repo_clone_git_identities
''',
        '''const onRepoCloneIdentityChange = (url, nextId) => _onRepoCloneIdentityChange(url, nextId, {
  effectiveTenantId, effectiveWorkspaceId, effectiveTaskId,
  repoCloneIdentityByUrl, repoCloneIdentitySaving, repoCloneIdentitySaveError,
  syncRepoCloneIdentityMapFromTask, repoCloneIdentityUserTouchedByUrl, localTask,
})

/*REMOVE_SYNC*/const syncRepoCloneIdentityMapFromTask = () => {
  const raw = localTask.value?.parameters?.repo_clone_git_identities
''',
    ),
]:
    if old in text:
        text = text.replace(old, new, 1)
        break

# remove duplicate project/repo function bodies between sync and onRepoReclone
if '/*REMOVE_SYNC*/' in text:
    rs = text.index('/*REMOVE_SYNC*/')
    re = text.index('const onRepoReclone = (repoUrl)')
    text = text[:rs] + text[re:]

# remove workspaceProjects watch duplicate
text = text.replace(
    "watch([workspaceProjects, localTask], () => syncRepoCloneIdentityMapFromTask(), { deep: true })\n\n",
    "",
)

# --- layer graph + ztree slice ---
lg_start = text.index('/** 已见过的可写层 id，与首次/增量快照配合自动选「最后一层」 */')
lg_end = text.index('let layerGraphHydrateInFlight = false')

early_start = text.index('/** 未注册端点，或已注册但 HTTP 转发不可达时，禁用依赖容器链路的按钮 */')
early_end = text.index('// Clone Progress — delegated to taskDetailCloneProgress.js')
early_block = text[early_start:early_end]
text = text[:early_start] + text[early_end:]

factory = r'''
/** 层图 / zTree 执行日志：跨模块共享 refs */
const selectedLayerGraphNode = ref(null)
const layerChangesByLayerId = ref({})
const layerGraphCommandText = ref('')
const taskLayerAssociationPanelRef = ref(null)

const zlog = createTaskDetailZTreeExecLogState({
  effectiveTenantId,
  effectiveWorkspaceId,
  effectiveTaskId,
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
})
const {
  layerExecLogLoading,
  layerExecLogTopError,
  layerCloneLogText,
  layerCloneLogFetchError,
  layerJobLogFetchError,
  layerJobExecutionPayload,
  layerJobLiveOutputMap,
  layerChangesRefreshBusy,
  layerChangesRefreshError,
  selectedZTreeLayerChangesPanel,
  selectedLayerGraphFileTreeLayerId,
  selectedLayerGraphNodeLogKey,
  zTreeLogTargets,
  layerChangesRefreshEnabled,
  layerChangesGitStagedActionsBlocked,
  layerChangesGitCommitIdentityBlocked,
  ingestLayerChangesFromExecutionPayload,
  refreshSelectedLayerChanges,
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

const lg = createTaskDetailLayerGraphState({
  effectiveTenantId,
  effectiveWorkspaceId,
  effectiveTaskId,
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
  refreshLayerGraphFromServer,
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
  layerGraphCmdError,
  layerGraphCmdSending,
  layerGraphEditRunTargetJobId,
  layerGraphBusyActionKey,
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
  containerEndpointRegistered,
  selectedLayerGraphFileTreeLayerId,
  refreshZTreeExecutionLog,
  activeJobExecLogPoller,
  prefetchLayerChangeSummariesForDirtyLayers,
  startContainerHeartbeat,
  stopContainerHeartbeat,
  refreshLayerGraphFromServer,
  containerHeartbeatStatus,
  containerHeartbeatPaused,
  serverRuntimeNotServing,
  isServerRunning,
  isServerStarting,
  zTreeLogTargets,
  selectedLayerGraphNodeLogKey,
})

''' + early_block + r'''
const submitLayerGraphCommand = () => _submitLayerGraphCommand({
  effectiveTenantId, effectiveWorkspaceId, effectiveTaskId,
  selectedLayerGraphNode, layerGraphSnapshot,
  layerGraphCommandText, layerGraphCommandKind, layerGraphCmdError, layerGraphCmdSending,
  layerGraphAutoIterationCount, layerGraphSelectedModel, layerGraphModelProvider,
  layerGraphEditRunTargetJobId,
  validateLinkedProjectReposBeforeSendToAi, normalizeSelectedAgentModels,
  formatLayerGraphCommandErrorForUser,
  markContainerTransportUnreachableIfForwardingFailed, markContainerTransportOk,
  refreshLayerGraphFromServer, bumpProjectFileTreeRefresh,
})

const onLayerGraphLayerSubmit = (node) => _onLayerGraphLayerSubmit(node, {
  containerEndpointRegistered, containerHttpUnreachable, containerPageUrl,
  taskRepoRows, repoCloneIdentityIdForUrl, layerGraphBusyActionKey,
  effectiveTenantId, effectiveWorkspaceId, effectiveTaskId,
  markContainerTransportUnreachableIfForwardingFailed, markContainerTransportOk,
  layerChangesByLayerId, refreshLayerGraphFromServer, refreshZTreeExecutionLog,
})

const onLayerChangesListCommitStaged = (payload) => _onLayerChangesListCommitStaged(payload, {
  containerEndpointRegistered, containerHttpUnreachable, containerPageUrl,
  taskRepoRows, repoCloneIdentityIdForUrl, layerChangesByLayerId,
  effectiveTenantId, effectiveWorkspaceId, effectiveTaskId,
  markContainerTransportUnreachableIfForwardingFailed, markContainerTransportOk,
  refreshLayerGraphFromServer, refreshZTreeExecutionLog, refreshSelectedLayerChanges,
  bumpProjectFileTreeRefresh,
})

const onLayerGraphLayerPush = (node) => _onLayerGraphLayerPush(node, {
  containerEndpointRegistered, containerHttpUnreachable, containerPageUrl,
  taskRepoRows, repoCloneIdentityIdForUrl, firstTaskRepoCloneIdentityId,
  layerGraphPushTargetBranch, layerGitGithubAppOauthConnected, layerGraphBusyActionKey,
  effectiveTenantId, effectiveWorkspaceId, effectiveTaskId,
  markContainerTransportUnreachableIfForwardingFailed, markContainerTransportOk,
  refreshLayerGraphFromServer, refreshZTreeExecutionLog, fetchTaskDetail,
  layerGraphSnapshot,
})

const onLayerGraphLayerMerge = (node) => _onLayerGraphLayerMerge(node, {
  containerEndpointRegistered, containerHttpUnreachable, containerPageUrl,
  layerGraphMergeTargetBranch, layerGraphBusyActionKey,
  effectiveTenantId, effectiveWorkspaceId, effectiveTaskId,
  markContainerTransportUnreachableIfForwardingFailed, markContainerTransportOk,
  refreshLayerGraphFromServer, refreshZTreeExecutionLog,
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
})

const syncPerRepoGitIdentitiesToContainer = () => _syncPerRepoGitIdentitiesToContainer({
  effectiveTenantId, effectiveWorkspaceId, effectiveTaskId,
  containerEndpointRegistered, containerHeartbeatStatus,
  selectedLayerGraphFileTreeLayerId, containerPageUrl,
  taskRepoRows, repoCloneIdentityIdForUrl,
  perRepoGitIdentitySyncing, perRepoGitIdentitySyncError,
  containerLayerGraphAuthInvalid, fetchLayerRepoGitIdentities,
})

'''

insert_after = "const onBootstrapCloneLogUpdate = (payload) => _onBootstrapCloneLogUpdate(payload, cpState)\n\n"
# keep layerGraphSnapshot line before factory
snapshot_line = "const layerGraphSnapshot = ref(null)\n"
if snapshot_line in text[lg_start - 200:lg_start]:
    pass

text = text[:lg_start] + text[lg_end:]
text = text.replace(insert_after, insert_after + factory + "\n", 1)

# cleanup timers in return
text = text.replace(
    "if (typeof layerLogAbortController !== 'undefined' && layerLogAbortController) { layerLogAbortController.abort(); layerLogAbortController = null; }",
    "if (zlog.layerLogAbortController) { zlog.layerLogAbortController.abort(); zlog.layerLogAbortController = null; }",
)
text = text.replace(
    "if (typeof layerExecLogCopyFeedbackTimer !== 'undefined' && layerExecLogCopyFeedbackTimer) { clearTimeout(layerExecLogCopyFeedbackTimer); layerExecLogCopyFeedbackTimer = null; }",
    "if (zlog.layerExecLogCopyFeedbackTimer) { clearTimeout(zlog.layerExecLogCopyFeedbackTimer); zlog.layerExecLogCopyFeedbackTimer = null; }",
)
text = text.replace(
    "if (typeof layerAgentStepCopyFeedbackTimer !== 'undefined' && layerAgentStepCopyFeedbackTimer) { clearTimeout(layerAgentStepCopyFeedbackTimer); layerAgentStepCopyFeedbackTimer = null; }",
    "if (zlog.layerAgentStepCopyFeedbackTimer) { clearTimeout(zlog.layerAgentStepCopyFeedbackTimer); zlog.layerAgentStepCopyFeedbackTimer = null; }",
)

# header line count hint
import re
lines = text.count('\n') + 1
text = re.sub(
    r'This file \(~[0-9]+ lines\)',
    f'This file (~{lines} lines)',
    text,
    count=1,
)

path.write_text(text)
print(f'patched useTaskDetail.js -> {lines} lines')

/**
 * Extracted from useTaskDetail.js (OPT-20260823-007 line-limit split).
 */
import { resolveTaskRouteIds } from '../../utils/resolveTaskRouteIds.js'
import { ref, computed, nextTick, watch, provide } from 'vue'
import {
  inferWorkBranchPreset,
  inferMergeTargetPreset,
  resolveLayerGraphPushTargetBranch,
  resolveLayerGraphMergeTargetBranch,
} from '../../utils/taskDetailBranchAndRepoUtils.js'
import { buildDisplayComments } from './buildDisplayComments.js'
import { saveEdit as _saveEdit, forkTask as _forkTask, buildTaskDetailRouteQuery } from './taskDetailEditing.js'
import {
  fetchContainerTaskUiContext as _fetchContainerTaskUiContext,
  resolveContainerUiContextCommentId,
  ensureServerRuntimeAllowsContainerLayerGraph as _ensureServerRuntimeAllowsContainerLayerGraph,
  refreshLayerGraphFromServer as _refreshLayerGraphFromServer,
  fetchContainerBootstrapCloneLog as _fetchContainerBootstrapCloneLog,
} from './taskDetailContainerFns.js'
import { useCommentContainerBindings } from './useCommentContainerBindings.js'
import { createSseReconnectState } from './taskDetailSseReconnect.js'
import {
  createTaskDetailProjectRepoState,
  installTaskDetailProjectRepoWatchers,
} from './taskDetailProjectRepoState.js'
import {
  createTaskDetailEditBranchState,
  installTaskDetailEditBranchWatchers,
} from './taskDetailEditBranchState.js'
import { fetchTaskDetail as _fetchTaskDetail, loadMoreCommentFeeds as _loadMoreCommentFeeds, fetchWorkspaceCollaborators as _fetchWorkspaceCollaborators, fetchProgressStatusOptions as _fetchProgressStatusOptions, fetchDeliverableCategoryOptions as _fetchDeliverableCategoryOptions, fetchWorkspaceTodosForParent as _fetchWorkspaceTodosForParent, fetchParentDeliverableTitle as _fetchParentDeliverableTitle, fetchForkSourceTitle as _fetchForkSourceTitle, fetchTaskSubtree as _fetchTaskSubtree, onProgressStatusChange as _onProgressStatusChange, submitComment as _submitComment, onRepoReclone as _onRepoReclone, onRepoCloneIdentityChange as _onRepoCloneIdentityChange, maybeAutoApplyDefaultRepoCloneIdentities as _maybeAutoApplyDefaultRepoCloneIdentities } from './taskDetailFetchFns.js'
import { resolveTodoParentTaskId } from '../../utils/workPanelDeliverableAggregation.js'

export function initUseTaskDetailCore(d, props, { emit, route, router }) {
/** 用户重复点击「发送给 AI」或离开页面时中止流，触发后端 notify abort */
d.aiInstructAbortController = null
d.serverStartupStatusPoll = null

// SSE reconnect state created later via createSseReconnectState (after d.establishSSEConnection forward decl)

// 服务器状态
const serverStatus = ref('')
const statusMessage = ref('')
/** 最近一条启动/停止 SSE 的 comment_id；Describe 刷新禁止回落到 forPoll 单槽 */
const statusCommentId = ref('')
/** 最近一条启动/停止 SSE 的 runtime_status；前端只 apply，不自动查云 */
const statusRuntimeStatus = ref('')
/** SSE/HTTP 启动链路的 trace_id（透传至错误 data-traceId 与执行细节「启动 TraceId」） */
const statusTraceId = ref('')
const statusProgress = ref(0)
const statusLogs = ref([])
/** 与启动日志并列：绿点表示 EventSource 已连通（含收到心跳） */
const sseLive = ref(false)
const isServerRunning = ref(false)
const isServerStarting = ref(false)
/** 云 Describe runtime_status（供启动状态展示层冷打开回退） */
const cloudRuntimeStatus = ref('')
const serverUrl = ref('')
const pendingStartEventId = ref('')




const routeIds = computed(() => resolveTaskRouteIds({
  tenantId: props.tenantId,
  workspaceId: props.workspaceId,
  taskId: props.taskId,
  task: props.task,
  routeParams: route.params,
}))
const effectiveTenantId = computed(() => routeIds.value.tenantId)
const effectiveWorkspaceId = computed(() => routeIds.value.workspaceId)
const effectiveTaskId = computed(() => routeIds.value.taskId)
const relayAccessCode = computed(() => {
  const raw = route.query?.accessCode
  return raw == null ? '' : String(Array.isArray(raw) ? raw[0] || '' : raw)
})
/** 供模板传给 ServerConfig；与 effectiveWorkspaceId 一致，避免未定义变量导致子组件拿不到 workspace */
const workspaceId = effectiveWorkspaceId
const backToWorkPanelRoute = computed(() => {
  const tenantId = effectiveTenantId.value
  if (!tenantId) {
    return '/'
  }
  const accessCode = route.query.accessCode
  return {
    name: 'work_panel',
    params: { tenant: String(tenantId) },
    query: accessCode ? { accessCode: String(accessCode) } : {}
  }
})

const detailQuery = computed(() => buildTaskDetailRouteQuery(route.query))

const forkSourceTaskRoute = computed(() => {
  const tenantId = effectiveTenantId.value
  const workspaceId = effectiveWorkspaceId.value
  const forkFrom = localTask.value?.fork_from
  if (!tenantId || !workspaceId || forkFrom == null || forkFrom === '') {
    return { path: '/' }
  }
  return {
    name: 'task_detail',
    params: {
      tenant: String(tenantId),
      workspaceId: String(workspaceId),
      taskId: String(forkFrom)
    },
    query: detailQuery.value
  }
})

const parentDeliverableId = computed(() => resolveTodoParentTaskId(localTask.value))

const parentDeliverableRoute = computed(() => {
  const tenantId = effectiveTenantId.value
  const workspaceId = effectiveWorkspaceId.value
  const parentId = parentDeliverableId.value
  if (!tenantId || !workspaceId || !parentId) {
    return { path: '/' }
  }
  return {
    name: 'task_detail',
    params: {
      tenant: String(tenantId),
      workspaceId: String(workspaceId),
      taskId: String(parentId),
    },
    query: detailQuery.value,
  }
})

// 响应式数据
const localTask = ref(props.task || null)
const parentTaskTitle = ref('')
const parentTaskSeq = ref(0)
const isParentTaskTitleLoading = ref(false)
const forkSourceTitle = ref('')
const forkSourceSeq = ref(0)
const isForkSourceTitleLoading = ref(false)
const forkFromId = computed(() => {
  const raw = localTask.value?.fork_from
  return raw != null && String(raw).trim() !== '' ? String(raw).trim() : ''
})
const taskDetailLoading = ref(false)
const taskDetailLoadError = ref('')
const taskDetailLoadErrorTraceId = ref('')
const isEditing = ref(false)
const editingTask = ref(null)
const isSaving = ref(false)
const editError = ref('')
const workspaceTodos = ref([])
const isWorkspaceTodosLoading = ref(false)

const layerGraphPushTargetBranch = computed(() =>
  resolveLayerGraphPushTargetBranch(localTask.value, effectiveTaskId.value || ''),
)
const layerGraphMergeTargetBranch = computed(() =>
  resolveLayerGraphMergeTargetBranch(localTask.value, effectiveTaskId.value || ''),
)

/** 模拟启动日志：SSE 增量需写入 ServerConfig（defineExpose） */
const serverConfigRef = ref(null)
const collaboratorNameById = ref({})
const collaboratorAvatarById = ref({})
/** 从 collaboratorNameById 导出 { id, name } 数组，供编辑模式多选 */
const collaboratorOptions = computed(() => {
  const map = collaboratorNameById.value || {}
  return Object.keys(map).map((id) => ({ id, name: map[id] }))
})
const progressStatusOptions = ref([])
const isProgressStatusesLoading = ref(false)
const isUpdatingProgressStatus = ref(false)
const progressStatusError = ref('')
const progressStatusErrorTraceId = ref('')
const taskSubtreeSummary = ref(null)
const taskSubtreeNodes = ref([])
const isTaskSubtreeLoading = ref(false)
const taskSubtreeError = ref('')
const taskSubtreeErrorTraceId = ref('')
const deliverableCategoryOptions = ref([])
const isDeliverableCategoriesLoading = ref(false)
const deliverableCategoryError = ref('')
const isForking = ref(false)

const repoRecloneBridge = { fn: null }
const pr = createTaskDetailProjectRepoState({
  effectiveTenantId,
  effectiveWorkspaceId,
  effectiveTaskId,
  localTask,
  isEditing,
  editingTask,
  repoRecloneBridge,
  containerEndpointRegistered: d.containerEndpointRegistered,
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
  recloneErrorTraceIdByUrl,
  repoRecloneGlobalLoading,
  staleRepoSyncLoading,
  staleRepoSyncError,
  staleRepoSyncNeedsReclone,
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

const editBranch = createTaskDetailEditBranchState({
  effectiveTenantId,
  effectiveTaskId,
  localTask,
  editingTask,
  isEditing,
  editError,
  workspaceProjects,
  getProjectRepos,
  inferWorkBranchPreset,
  inferMergeTargetPreset,
  fetchWorkspaceTodosForParent: () => d.fetchWorkspaceTodosForParent(),
  fetchAllLinkedRepoBranches: (force) => fetchAllLinkedRepoBranches(force),
})
const {
  companyUserName,
  isTaskTitleTranslating,
  taskTitleTranslationError,
  translatedTaskTitleSegmentMap,
  buildWorkBranchName,
  buildMergeTargetBranchName,
  fetchCurrentCompanyUserName,
  fetchTranslatedTaskTitleSegment,
  resolveTaskTitleSegment,
  updateWorkBranchNameByPreset,
  scheduleWorkBranchNameUpdate,
  updateMergeTargetBranchNameByPreset,
  applyWorkBranchPreset,
  applyMergeTargetPreset,
  startEdit,
  cancelEdit,
  addProjectAssociation,
  removeProjectAssociation,
} = editBranch
installTaskDetailEditBranchWatchers(editBranch, { editingTask })

const displayComments = computed(() => buildDisplayComments(localTask.value))
/** 容器 Agent（$镜像）流式：与 ai_instruct_stream 分槽过滤，展示复用 aiStream* */
const activeContainerAgentId = ref(null)
// OPT-20260816-002: 评论容器绑定提升到 useTaskDetail（单一实例，评论区不再重复创建），
// 使页面级 containerForwardCommentId 与执行面板一致：有 CSC 的 running 评论优先于最后一条 AI 评论。
const commentBindings = useCommentContainerBindings({
  tenantId: effectiveTenantId,
  workspaceId: effectiveWorkspaceId,
  taskId: effectiveTaskId,
  displayComments,
  taskStatusLogs: statusLogs,
})
const {
  bindingStatusFor,
  bindingCscIdFor,
  bindingContainerNameFor,
  commentHasLiveBinding,
  commentOwnsSharedContainer,
  bindingStartTraceIdFor,
  buildPerBindingServerStatusProps,
  markBindingReconnecting,
  cancelWaitingPreviousBinding,
  isCancelWaitingBusy,
} = commentBindings
const commentForwardScope = { displayComments, activeContainerAgentId, bindingStatusFor, bindingCscIdFor }
const containerForwardCommentId = computed(() =>
  resolveContainerUiContextCommentId(commentForwardScope),
)

const onRepoReclone = (payload) => _onRepoReclone(payload, {
  effectiveTenantId, effectiveWorkspaceId, effectiveTaskId,
  containerEndpointRegistered: d.containerEndpointRegistered, recloneLoadingByUrl, recloneStatusByUrl,
  recloneErrorTraceIdByUrl,
  repoRecloneGlobalLoading, repoCloneIdentityIdForUrl,
  containerForwardCommentId,
})
// OPT-20260829-025: syncStaleTaskRepoAddresses 成功后经桥接触发重新克隆
repoRecloneBridge.fn = (payload) => onRepoReclone(payload)

const commentsHasMore = computed(() => {
  const t = localTask.value
  if (!t) return false
  return Boolean(t.comments_has_more || t.ai_comments_has_more || t.container_agent_comments_has_more)
})

// 评论相关（须在 submitComment / d.submitAIComment 之前定义）
const newComment = ref('')
const commentsFeedsLoadingMore = ref(false)
/** Composer 引用芯片（写入评论/AI 时拼入正文前缀） */
const commentComposerChips = ref([])

function removeCommentComposerChip(idx) {
  if (idx < 0 || idx >= commentComposerChips.value.length) return
  commentComposerChips.value = commentComposerChips.value.filter((_, i) => i !== idx)
}

function buildCommentBodyWithChips() {
  const chips = commentComposerChips.value
  const base = newComment.value.trim()
  if (!chips.length) return base
  const prefix = `${chips.map((c) => `[引用 ${c}]`).join(' ')}\n`
  return (prefix + base).trim()
}

const onCommentTextareaKeydown = (e) => {
  if (e.key !== 'Enter') return
  if (e.metaKey || e.ctrlKey) {
    e.preventDefault()
    void submitComment()
  }
}

const submitComment = () => _submitComment({
  effectiveTenantId, effectiveTaskId, newComment, commentComposerChips,
  buildCommentBodyWithChips, fetchTaskDetail: d.fetchTaskDetail, taskProjectsWithDetails,
})

/** 克隆完成等场景下层图会 refresh，但项目文件树仅依赖 layerId/容器就绪，需单独 bump 以重新拉取 container-layer-files */
const projectFileTreeRefreshNonce = ref(0)
function bumpProjectFileTreeRefresh() {
  projectFileTreeRefreshNonce.value += 1
}

  Object.assign(d, {
    serverStatus, statusMessage, statusCommentId, statusRuntimeStatus, statusTraceId,
    statusProgress, statusLogs, sseLive, isServerRunning, isServerStarting,
    cloudRuntimeStatus, serverUrl, pendingStartEventId, routeIds, effectiveTenantId,
    effectiveWorkspaceId, effectiveTaskId, relayAccessCode, workspaceId,
    backToWorkPanelRoute, detailQuery, forkSourceTaskRoute, parentDeliverableId,
    parentDeliverableRoute, localTask, parentTaskTitle, parentTaskSeq,
    isParentTaskTitleLoading, forkSourceTitle, forkSourceSeq, isForkSourceTitleLoading,
    forkFromId, taskDetailLoading, taskDetailLoadError, taskDetailLoadErrorTraceId,
    isEditing, editingTask, isSaving, editError, workspaceTodos, isWorkspaceTodosLoading,
    layerGraphPushTargetBranch, layerGraphMergeTargetBranch, serverConfigRef,
    collaboratorNameById, collaboratorAvatarById, collaboratorOptions,
    progressStatusOptions, isProgressStatusesLoading, isUpdatingProgressStatus,
    progressStatusError, progressStatusErrorTraceId, taskSubtreeSummary,
    taskSubtreeNodes, isTaskSubtreeLoading, taskSubtreeError, taskSubtreeErrorTraceId,
    deliverableCategoryOptions, isDeliverableCategoriesLoading, deliverableCategoryError,
    isForking, pr, editBranch, displayComments, activeContainerAgentId, commentBindings,
    commentForwardScope, containerForwardCommentId, onRepoReclone, commentsHasMore,
    newComment, commentsFeedsLoadingMore, commentComposerChips,
    removeCommentComposerChip, buildCommentBodyWithChips, onCommentTextareaKeydown,
    submitComment, projectFileTreeRefreshNonce, bumpProjectFileTreeRefresh,
    linkedProjectsPanelRef, workspaceProjects, workspaceProjectsLoading,
    projectServerRunTemplate, repoCloneIdentityByUrl, repoCloneIdentitySaveError,
    repoCloneIdentitySaving, repoCloneIdentityUserTouchedByUrl,
    repoCloneIdentityAutoApplyInFlight, recloneLoadingByUrl, recloneStatusByUrl,
    recloneErrorTraceIdByUrl, repoRecloneGlobalLoading, staleRepoSyncLoading,
    staleRepoSyncError, staleRepoSyncNeedsReclone, repoBranchesCache, repoBranchesLoading, repoBranchesErrors,
    taskProjectsWithDetails, taskRepoRows, syncStaleTaskRepoAddresses,
    syncRepoCloneIdentityMapFromTask, fetchWorkspaceProjects,
    savedRepoCloneIdentityIdForUrl, validateLinkedProjectReposBeforeSendToAi,
    repoCloneIdentityIdForUrl, firstTaskRepoCloneIdentityId, getProjectRepos,
    buildProjectsApiPayload, getRepoBranchValue, setRepoBranch, getRepoBranches,
    getRepoBranchError, isLoadingRepoBranches, fetchRepoBranches, onProjectChange,
    linkedRepoBranchTargets, fetchAllLinkedRepoBranches,
    commonMergeTargetBranchesLoading, commonMergeTargetBranchesError,
    commonMergeTargetBranches, companyUserName, isTaskTitleTranslating,
    taskTitleTranslationError, translatedTaskTitleSegmentMap, buildWorkBranchName,
    buildMergeTargetBranchName, fetchCurrentCompanyUserName,
    fetchTranslatedTaskTitleSegment, resolveTaskTitleSegment,
    updateWorkBranchNameByPreset, scheduleWorkBranchNameUpdate,
    updateMergeTargetBranchNameByPreset, applyWorkBranchPreset, applyMergeTargetPreset,
    startEdit, cancelEdit, addProjectAssociation, removeProjectAssociation,
    bindingStatusFor, bindingCscIdFor, bindingContainerNameFor, commentHasLiveBinding,
    commentOwnsSharedContainer, bindingStartTraceIdFor, buildPerBindingServerStatusProps,
    markBindingReconnecting, cancelWaitingPreviousBinding, isCancelWaitingBusy, emit,
    route, router, props,
  })
}

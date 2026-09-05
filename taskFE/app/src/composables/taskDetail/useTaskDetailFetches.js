/**
 * Extracted from useTaskDetail.js (OPT-20260823-007 line-limit split).
 */
import { ref, computed, nextTick, watch, provide } from 'vue'
import { saveEdit as _saveEdit, forkTask as _forkTask, buildTaskDetailRouteQuery } from './taskDetailEditing.js'
import { fetchTaskDetail as _fetchTaskDetail, loadMoreCommentFeeds as _loadMoreCommentFeeds, fetchWorkspaceCollaborators as _fetchWorkspaceCollaborators, fetchProgressStatusOptions as _fetchProgressStatusOptions, fetchDeliverableCategoryOptions as _fetchDeliverableCategoryOptions, fetchWorkspaceTodosForParent as _fetchWorkspaceTodosForParent, fetchParentDeliverableTitle as _fetchParentDeliverableTitle, fetchForkSourceTitle as _fetchForkSourceTitle, fetchTaskSubtree as _fetchTaskSubtree, onProgressStatusChange as _onProgressStatusChange, submitComment as _submitComment, onRepoReclone as _onRepoReclone, onRepoCloneIdentityChange as _onRepoCloneIdentityChange, maybeAutoApplyDefaultRepoCloneIdentities as _maybeAutoApplyDefaultRepoCloneIdentities } from './taskDetailFetchFns.js'
import { resolveParentDeliverableBlockedReason } from '../../utils/workPanelCreateTaskParent.js'

export function initUseTaskDetailFetches(d) {
  const {
    effectiveTenantId, effectiveWorkspaceId, effectiveTaskId, detailQuery,
    parentDeliverableId, parentDeliverableRoute, localTask, parentTaskTitle,
    parentTaskSeq, isParentTaskTitleLoading, forkSourceTitle, forkSourceSeq,
    isForkSourceTitleLoading, forkFromId, taskDetailLoading, taskDetailLoadError,
    taskDetailLoadErrorTraceId, isEditing, editingTask, isSaving, editError,
    workspaceTodos, isWorkspaceTodosLoading, collaboratorNameById,
    collaboratorAvatarById, progressStatusOptions, isProgressStatusesLoading,
    progressStatusError, taskSubtreeSummary, taskSubtreeNodes, isTaskSubtreeLoading,
    taskSubtreeError, taskSubtreeErrorTraceId, deliverableCategoryOptions,
    isDeliverableCategoriesLoading, deliverableCategoryError, isForking,
    commentsFeedsLoadingMore, syncRepoCloneIdentityMapFromTask, getProjectRepos,
    buildProjectsApiPayload, cancelEdit, emit, router,
  } = d
// Task editing & forking — delegated to extracted pure functions
const saveEdit = () => _saveEdit({
  effectiveTenantId, effectiveWorkspaceId, effectiveTaskId,
  editingTask, editError, isSaving, localTask, cancelEdit,
  getProjectRepos, buildProjectsApiPayload,
  deliverableCategoryOptions, workspaceTodos,
  onTaskUpdated: (updatedTask) => {
    if (typeof emit === 'function') emit('task-updated', updatedTask)
  },
})

const parentDeliverableEditBlockedReason = computed(() => {
  if (!isEditing.value || !editingTask.value) return ''
  return resolveParentDeliverableBlockedReason({
    taskTypes: deliverableCategoryOptions.value,
    todos: workspaceTodos.value,
    categoryId: editingTask.value.deliverable_obj_id,
    parentTaskId: editingTask.value.parent_task,
  })
})

const forkTask = (opts = {}) => _forkTask({
  effectiveTenantId, effectiveWorkspaceId, effectiveTaskId,
  localTask, isForking, router, detailQuery,
  progressStatusOptions,
  fetchProgressStatusOptions,
  autoRun: opts?.autoRun === true,
  copyCount: opts?.copyCount,
  batchIdempotencyKey: opts?.batchIdempotencyKey,
  repoIdentities: opts?.repoIdentities,
  featureParamsSource: opts?.featureParamsSource,
  personalFeatureParamsConfigId: opts?.personalFeatureParamsConfigId,
  agentModelProvider: opts?.agentModelProvider,
  agents: opts?.agents,
  onForkProgress: opts?.onForkProgress,
  onForked: (created) => {
    if (typeof emit === 'function') emit('task-updated', created)
  },
})

const fetchTaskDetail = () => _fetchTaskDetail({
  effectiveTenantId, effectiveWorkspaceId, effectiveTaskId, localTask,
  taskDetailLoading, taskDetailLoadError, taskDetailLoadErrorTraceId,
  syncRepoCloneIdentityMapFromTask,
})

const loadMoreCommentFeeds = () => _loadMoreCommentFeeds({
  effectiveTenantId, effectiveWorkspaceId, effectiveTaskId, localTask,
  commentsFeedsLoadingMore,
})

const fetchWorkspaceCollaborators = () => _fetchWorkspaceCollaborators({
  effectiveTenantId, effectiveWorkspaceId, collaboratorNameById, collaboratorAvatarById,
})

const fetchProgressStatusOptions = () => _fetchProgressStatusOptions({
  effectiveTenantId, effectiveWorkspaceId, progressStatusOptions,
  isProgressStatusesLoading, progressStatusError,
})

const fetchTaskSubtree = () => _fetchTaskSubtree({
  effectiveTenantId, effectiveWorkspaceId, effectiveTaskId,
  taskSubtreeSummary, taskSubtreeNodes, isTaskSubtreeLoading,
  taskSubtreeError, taskSubtreeErrorTraceId,
})

const fetchDeliverableCategoryOptions = () => _fetchDeliverableCategoryOptions({
  effectiveTenantId, effectiveWorkspaceId, deliverableCategoryOptions,
  isDeliverableCategoriesLoading, deliverableCategoryError,
})

const fetchWorkspaceTodosForParent = () => _fetchWorkspaceTodosForParent({
  effectiveTenantId, effectiveWorkspaceId, workspaceTodos, isWorkspaceTodosLoading,
})

const fetchParentDeliverableTitle = () => _fetchParentDeliverableTitle({
  effectiveTenantId,
  effectiveWorkspaceId,
  parentTaskId: parentDeliverableId.value,
  parentTaskTitle,
  parentTaskSeq,
  isParentTaskTitleLoading,
})

const fetchForkSourceTitle = () => _fetchForkSourceTitle({
  effectiveTenantId,
  effectiveWorkspaceId,
  forkFromId: forkFromId.value,
  forkSourceTitle,
  forkSourceSeq,
  isForkSourceTitleLoading,
})

watch(
  parentDeliverableId,
  (parentId) => {
    if (!parentId) {
      parentTaskTitle.value = ''
      parentTaskSeq.value = 0
      isParentTaskTitleLoading.value = false
      return
    }
    void fetchParentDeliverableTitle()
  },
  { immediate: true },
)

watch(
  forkFromId,
  (id) => {
    if (!id) {
      forkSourceTitle.value = ''
      forkSourceSeq.value = 0
      isForkSourceTitleLoading.value = false
      return
    }
    void fetchForkSourceTitle()
  },
  { immediate: true },
)

watch(
  [effectiveTenantId, effectiveWorkspaceId],
  () => {
    void fetchDeliverableCategoryOptions()
  },
  { immediate: true },
)

watch(
  [effectiveTenantId, effectiveWorkspaceId, effectiveTaskId],
  () => {
    void fetchTaskSubtree()
  },
  { immediate: true },
)

provide('taskDetailParentDeliverable', {
  parentDeliverableRoute,
  parentTaskTitle,
  parentTaskSeq,
  isParentTaskTitleLoading,
})
provide('taskDetailDeliverableCategory', {
  deliverableCategoryOptions,
  isDeliverableCategoriesLoading,
  deliverableCategoryError,
})
provide('taskDetailWorkspaceTodos', workspaceTodos)
provide('taskDetailSaveBlockedReason', parentDeliverableEditBlockedReason)
provide('taskDetailSubtree', {
  summary: taskSubtreeSummary,
  nodes: taskSubtreeNodes,
  loading: isTaskSubtreeLoading,
  error: taskSubtreeError,
  errorTraceId: taskSubtreeErrorTraceId,
})

  Object.assign(d, {
    saveEdit, parentDeliverableEditBlockedReason, forkTask, fetchTaskDetail,
    loadMoreCommentFeeds, fetchWorkspaceCollaborators, fetchProgressStatusOptions,
    fetchTaskSubtree, fetchDeliverableCategoryOptions, fetchWorkspaceTodosForParent,
    fetchParentDeliverableTitle, fetchForkSourceTitle,
  })
}

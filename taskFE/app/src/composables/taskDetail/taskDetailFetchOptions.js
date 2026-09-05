/**
 * TaskDetail 各类下拉/选项数据拉取与变更。
 * Split from taskDetailFetchFns.js (OPT-20260816-005): 协作人 / 进度状态 / 交付物
 * / 任务子树等「选项类」fetch + 变更。
 */
import { apiFetch } from '../../utils/apiUtils.js'
import { extractTraceId } from '../../utils/traceId.js'
import {
  buildCollaboratorAvatarById,
  buildCollaboratorNameById,
} from '../../utils/taskCardPeopleDisplay.js'
import { sortProgressColumns } from '../../utils/workPanelKanbanUtils.js'

export async function fetchWorkspaceCollaborators(deps) {
  const {
    effectiveTenantId,
    effectiveWorkspaceId,
    collaboratorNameById,
    collaboratorAvatarById,
  } = deps
  try {
    const tenantId = effectiveTenantId.value; const workspaceId = effectiveWorkspaceId.value
    if (!tenantId || !workspaceId) {
      collaboratorNameById.value = {}
      if (collaboratorAvatarById) collaboratorAvatarById.value = {}
      return
    }
    const response = await apiFetch(`/api/projects/workspace-access/workspace-collaborators/tenant_id/${tenantId}/?workspace_id=${workspaceId}`, { credentials: 'include', headers: { 'Accept': 'application/json' } })
    if (!response.ok) {
      collaboratorNameById.value = {}
      if (collaboratorAvatarById) collaboratorAvatarById.value = {}
      return
    }
    const data = await response.json()
    const list = Array.isArray(data) ? data : []
    collaboratorNameById.value = buildCollaboratorNameById(list)
    if (collaboratorAvatarById) {
      collaboratorAvatarById.value = buildCollaboratorAvatarById(list)
    }
  } catch (error) {
    console.error('获取协作人员映射失败:', error)
    collaboratorNameById.value = {}
    if (collaboratorAvatarById) collaboratorAvatarById.value = {}
  }
}

export async function fetchProgressStatusOptions(deps) {
  const { effectiveTenantId, effectiveWorkspaceId, progressStatusOptions, isProgressStatusesLoading, progressStatusError } = deps
  const tenantId = effectiveTenantId.value; const workspaceId = effectiveWorkspaceId.value
  if (!tenantId || !workspaceId) { progressStatusOptions.value = []; return }
  isProgressStatusesLoading.value = true; progressStatusError.value = ''
  try {
    const response = await apiFetch(`/api/projects/workspaces/tenant_id/${tenantId}/${workspaceId}/progress-system/`, { credentials: 'include', headers: { 'Accept': 'application/json' } })
    if (!response.ok) { progressStatusOptions.value = []; progressStatusError.value = '获取进度状态失败'; return }
    const data = await response.json(); const columns = Array.isArray(data?.columns) ? data.columns : []
    const sorted = sortProgressColumns(columns)
    progressStatusOptions.value = sorted.map((col) => ({ id: String(col.id), name: col.name }))
  } catch (error) { console.error('获取进度状态失败:', error); progressStatusOptions.value = []; progressStatusError.value = '获取进度状态失败' }
  finally { isProgressStatusesLoading.value = false }
}

/**
 * 按任务 id 拉取同工作空间 todo 标题（上层交付物 / fork 源等）。
 * 无 id、缺路由参数或请求失败时清空标题，不抛错阻断详情页。
 */
function applyWorkspaceSeq(seqRef, raw) {
  if (!seqRef) return
  const n = Number(raw)
  seqRef.value = Number.isInteger(n) && n > 0 ? n : 0
}

export async function fetchWorkspaceTodoTitle(deps) {
  const {
    effectiveTenantId,
    effectiveWorkspaceId,
    todoId,
    titleRef,
    loadingRef,
    seqRef,
    logLabel = '任务标题',
  } = deps
  const tenantId = effectiveTenantId.value
  const workspaceId = effectiveWorkspaceId.value
  const id = todoId != null ? String(todoId).trim() : ''
  if (!id || !tenantId || !workspaceId || workspaceId === 'default') {
    titleRef.value = ''
    applyWorkspaceSeq(seqRef, 0)
    loadingRef.value = false
    return
  }
  loadingRef.value = true
  try {
    const response = await apiFetch(
      `/api/tasks/todos/tenant_id/${tenantId}/workspace_id/${workspaceId}/${id}/`,
      { credentials: 'include', headers: { Accept: 'application/json' } },
    )
    if (!response.ok) {
      titleRef.value = ''
      applyWorkspaceSeq(seqRef, 0)
      console.error(`获取${logLabel}失败`, response.status)
      return
    }
    const data = await response.json()
    const title = data?.title != null ? String(data.title).trim() : ''
    titleRef.value = title
    applyWorkspaceSeq(seqRef, data?.workspace_seq)
  } catch (error) {
    console.error(`获取${logLabel}出错:`, error)
    titleRef.value = ''
    applyWorkspaceSeq(seqRef, 0)
  } finally {
    loadingRef.value = false
  }
}

/**
 * 按 parent_task 拉取上层交付物标题（用于详情身份区对齐展示）。
 */
export async function fetchParentDeliverableTitle(deps) {
  const {
    effectiveTenantId,
    effectiveWorkspaceId,
    parentTaskId,
    parentTaskTitle,
    isParentTaskTitleLoading,
    parentTaskSeq,
  } = deps
  return fetchWorkspaceTodoTitle({
    effectiveTenantId,
    effectiveWorkspaceId,
    todoId: parentTaskId,
    titleRef: parentTaskTitle,
    loadingRef: isParentTaskTitleLoading,
    seqRef: parentTaskSeq,
    logLabel: '上层交付物标题',
  })
}

/**
 * 按 fork_from 拉取派生源任务标题（用于详情身份区链接文案）。
 */
export async function fetchForkSourceTitle(deps) {
  const {
    effectiveTenantId,
    effectiveWorkspaceId,
    forkFromId,
    forkSourceTitle,
    isForkSourceTitleLoading,
    forkSourceSeq,
  } = deps
  return fetchWorkspaceTodoTitle({
    effectiveTenantId,
    effectiveWorkspaceId,
    todoId: forkFromId,
    titleRef: forkSourceTitle,
    loadingRef: isForkSourceTitleLoading,
    seqRef: forkSourceSeq,
    logLabel: '派生源任务标题',
  })
}

/** 工作空间当前交付物体系下的交付物类别（current_deliverable_objs） */
export async function fetchDeliverableCategoryOptions(deps) {
  const {
    effectiveTenantId,
    effectiveWorkspaceId,
    deliverableCategoryOptions,
    isDeliverableCategoriesLoading,
    deliverableCategoryError,
  } = deps
  const tenantId = effectiveTenantId.value
  const workspaceId = effectiveWorkspaceId.value
  if (!tenantId || !workspaceId || workspaceId === 'default') {
    deliverableCategoryOptions.value = []
    return
  }
  isDeliverableCategoriesLoading.value = true
  deliverableCategoryError.value = ''
  try {
    const response = await apiFetch(
      `/api/projects/manage-deliverable-system/tenant_id/${tenantId}?workspace_id=${workspaceId}`,
      { credentials: 'include', headers: { Accept: 'application/json' } },
    )
    if (!response.ok) {
      deliverableCategoryOptions.value = []
      deliverableCategoryError.value = '获取交付物类别失败'
      return
    }
    const data = await response.json()
    const list = Array.isArray(data?.current_deliverable_objs) ? data.current_deliverable_objs : []
    deliverableCategoryOptions.value = list.map((item) => ({
      id: String(item.id),
      name: item.name != null ? String(item.name) : String(item.id),
      order: item.order,
      color: item.color,
    }))
  } catch (error) {
    console.error('获取交付物类别失败:', error)
    deliverableCategoryOptions.value = []
    deliverableCategoryError.value = '获取交付物类别失败'
  } finally {
    isDeliverableCategoriesLoading.value = false
  }
}

/**
 * 拉取工作空间任务列表，供详情编辑时过滤「上层交付物」候选。
 */
export async function fetchWorkspaceTodosForParent(deps) {
  const {
    effectiveTenantId,
    effectiveWorkspaceId,
    workspaceTodos,
    isWorkspaceTodosLoading,
  } = deps
  const tenantId = effectiveTenantId.value
  const workspaceId = effectiveWorkspaceId.value
  if (!tenantId || !workspaceId || workspaceId === 'default') {
    workspaceTodos.value = []
    return
  }
  if (isWorkspaceTodosLoading) isWorkspaceTodosLoading.value = true
  try {
    const response = await apiFetch(
      `/api/tasks/todos/tenant_id/${tenantId}/workspace_id/${workspaceId}`,
      { credentials: 'include', headers: { Accept: 'application/json' } },
    )
    if (!response.ok) {
      console.error('获取工作空间任务列表失败', response.status)
      workspaceTodos.value = []
      return
    }
    const data = await response.json()
    const list = Array.isArray(data)
      ? data
      : (Array.isArray(data?.results) ? data.results : (Array.isArray(data?.todos) ? data.todos : []))
    workspaceTodos.value = list
  } catch (error) {
    console.error('获取工作空间任务列表出错:', error)
    workspaceTodos.value = []
  } finally {
    if (isWorkspaceTodosLoading) isWorkspaceTodosLoading.value = false
  }
}

export async function fetchTaskSubtree(deps) {
  const {
    effectiveTenantId, effectiveWorkspaceId, effectiveTaskId,
    taskSubtreeSummary, taskSubtreeNodes, isTaskSubtreeLoading,
    taskSubtreeError, taskSubtreeErrorTraceId,
  } = deps
  const tenantId = effectiveTenantId.value
  const workspaceId = effectiveWorkspaceId.value
  const taskId = effectiveTaskId.value
  if (!tenantId || !workspaceId || !taskId) {
    taskSubtreeSummary.value = null
    taskSubtreeNodes.value = []
    return
  }
  isTaskSubtreeLoading.value = true
  taskSubtreeError.value = ''
  if (taskSubtreeErrorTraceId) taskSubtreeErrorTraceId.value = ''
  try {
    const response = await apiFetch(
      `/api/tasks/todos/tenant_id/${tenantId}/workspace_id/${workspaceId}/${taskId}/subtree/?max_depth=2`,
      { credentials: 'include', headers: { Accept: 'application/json' } },
    )
    if (!response.ok) {
      taskSubtreeSummary.value = null
      taskSubtreeNodes.value = []
      taskSubtreeError.value = '获取下级交付物失败'
      if (taskSubtreeErrorTraceId) taskSubtreeErrorTraceId.value = extractTraceId(response) || ''
      return
    }
    const data = await response.json()
    taskSubtreeSummary.value = data?.summary && typeof data.summary === 'object' ? data.summary : null
    taskSubtreeNodes.value = Array.isArray(data?.nodes) ? data.nodes : []
  } catch (error) {
    console.error('获取下级交付物失败:', error)
    taskSubtreeSummary.value = null
    taskSubtreeNodes.value = []
    taskSubtreeError.value = '获取下级交付物失败'
    if (taskSubtreeErrorTraceId) taskSubtreeErrorTraceId.value = extractTraceId(error) || ''
  } finally {
    isTaskSubtreeLoading.value = false
  }
}

export async function fetchDeliverableTypeOptions(deps) {
  const {
    effectiveTenantId, effectiveWorkspaceId,
    deliverableTypeOptions, isDeliverableTypesLoading, deliverableTypeError,
  } = deps
  const tenantId = effectiveTenantId.value
  const workspaceId = effectiveWorkspaceId.value
  if (!tenantId || !workspaceId) {
    deliverableTypeOptions.value = []
    return
  }
  isDeliverableTypesLoading.value = true
  deliverableTypeError.value = ''
  try {
    const response = await apiFetch(
      `/api/projects/manage-deliverable-system/tenant_id/${tenantId}?workspace_id=${workspaceId}`,
      { credentials: 'include', headers: { Accept: 'application/json' } },
    )
    if (!response.ok) {
      deliverableTypeOptions.value = []
      deliverableTypeError.value = '获取交付物类别失败'
      return
    }
    const data = await response.json()
    const objs = Array.isArray(data?.current_deliverable_objs) ? data.current_deliverable_objs : []
    deliverableTypeOptions.value = objs.map((obj) => ({
      id: String(obj.id),
      name: obj.name,
    }))
  } catch (error) {
    console.error('获取交付物类别失败:', error)
    deliverableTypeOptions.value = []
    deliverableTypeError.value = '获取交付物类别失败'
  } finally {
    isDeliverableTypesLoading.value = false
  }
}

export async function onDeliverableTypeChange(event, deps) {
  const {
    effectiveTenantId, effectiveWorkspaceId, effectiveTaskId,
    deliverableTypeOptions, isUpdatingDeliverableType, deliverableTypeError, localTask,
  } = deps
  const raw = event?.target?.value
  const nextId = raw != null && String(raw) !== '' ? String(raw) : null
  const tenantId = effectiveTenantId.value
  const workspaceId = effectiveWorkspaceId.value
  const taskId = effectiveTaskId.value
  if (!tenantId || !workspaceId || !taskId) {
    deliverableTypeError.value = '更新交付物类别失败'
    return
  }
  isUpdatingDeliverableType.value = true
  deliverableTypeError.value = ''
  try {
    const response = await apiFetch(
      `/api/tasks/todos/tenant_id/${tenantId}/workspace_id/${workspaceId}/${taskId}/`,
      {
        method: 'PATCH',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ deliverable_obj_id: nextId }),
      },
    )
    if (!response.ok) {
      deliverableTypeError.value = '更新交付物类别失败'
      return
    }
    if (localTask.value) {
      localTask.value.deliverable_obj_id = nextId
      const selected = nextId
        ? deliverableTypeOptions.value.find((item) => item.id === nextId)
        : null
      if (selected) {
        localTask.value.deliverable_obj = { id: selected.id, name: selected.name }
      } else {
        localTask.value.deliverable_obj = null
      }
    }
  } catch (error) {
    console.error('更新交付物类别失败:', error)
    deliverableTypeError.value = '更新交付物类别失败'
  } finally {
    isUpdatingDeliverableType.value = false
  }
}

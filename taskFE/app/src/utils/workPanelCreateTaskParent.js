/**
 * 创建任务：非顶层交付物类别时的上层交付物（parent_task）辅助。
 */
import {
  isTopLevelDeliverableCategory,
  listParentDeliverableCandidates,
  readFilterFromPath,
} from './workPanelDeliverableAggregation.js'
import { emptyCreateTaskStructuredFields } from './createTaskDescriptionCompose.js'
import { pickCreateTaskDefaultProjectId } from './createTaskPreferredProject.js'

/** @param {unknown} raw */
export function normalizeParentTaskId(raw) {
  if (raw == null) return ''
  if (typeof raw === 'object' && raw.id != null) return String(raw.id)
  return String(raw)
}

/**
 * 非顶层类别创建时未选上层交付物的提交门禁文案；可提交则返回空串。
 * @param {{
 *   taskTypes?: unknown[],
 *   todos?: unknown[],
 *   categoryId?: unknown,
 *   parentTaskId?: unknown,
 * }} opts
 */
export function resolveParentDeliverableBlockedReason(opts = {}) {
  const { taskTypes, todos, categoryId } = opts
  if (categoryId == null || categoryId === '') return ''
  if (isTopLevelDeliverableCategory(taskTypes, categoryId)) return ''
  const candidates = listParentDeliverableCandidates(todos, taskTypes, categoryId)
  if (!candidates.length) {
    return '暂无上一层交付物任务，请先创建顶层/上一层交付物'
  }
  const parent = normalizeParentTaskId(opts.parentTaskId).trim()
  if (!parent || !candidates.some((c) => c.id === parent)) {
    return '请先选择上层交付物后再提交'
  }
  return ''
}


/** @param {Array<{ path?: unknown }>|null|undefined} bars */
export function resolvePreferredParentTaskIdFromFilterBars(bars) {
  const list = Array.isArray(bars) ? bars : []
  for (let i = list.length - 1; i >= 0; i -= 1) {
    const { taskId } = readFilterFromPath(list[i]?.path)
    if (taskId) return String(taskId)
  }
  return ''
}

/**
 * @param {{
 *   taskTypes?: unknown[],
 *   todos?: unknown[],
 *   defaultTypeId?: unknown,
 *   preferredParentTaskId?: unknown,
 * }} opts
 */
export function buildInitialParentTaskId(opts = {}) {
  const { taskTypes, todos, defaultTypeId } = opts
  const preferred = opts.preferredParentTaskId != null ? String(opts.preferredParentTaskId).trim() : ''
  if (defaultTypeId == null || defaultTypeId === '') return ''
  if (isTopLevelDeliverableCategory(taskTypes, defaultTypeId)) return ''
  const candidates = listParentDeliverableCandidates(todos, taskTypes, defaultTypeId)
  if (preferred && candidates.some((c) => c.id === preferred)) return preferred
  return ''
}

/**
 * 写入创建/更新 payload 的 parent_task。
 * 顶层类别显式置空串，确保编辑时清空旧上层关系。
 * @param {Record<string, unknown>} payload
 * @param {{ task_type?: { id?: unknown }, deliverable_obj_id?: unknown, parent_task?: unknown }} task
 * @param {unknown[]} taskTypes
 */
export function appendParentTaskToCreatePayload(payload, task, taskTypes) {
  const out = payload && typeof payload === 'object' ? payload : {}
  const typeId = task?.task_type?.id ?? task?.deliverable_obj_id
  if (typeId == null || typeId === '' || isTopLevelDeliverableCategory(taskTypes, typeId)) {
    out.parent_task = ''
    return out
  }
  const raw = task?.parent_task
  const parentTaskId =
    raw != null && raw !== ''
      ? String(typeof raw === 'object' && raw != null && 'id' in raw ? raw.id : raw).trim()
      : ''
  out.parent_task = parentTaskId
  return out
}

/**
 * 创建任务模态框初始 draft（含 parent_task 预填）。
 * @param {{
 *   taskStatuses?: Array<{id?: unknown}>,
 *   taskTypes?: unknown[],
 *   todos?: unknown[],
 *   projects?: Array<{id?: unknown}>,
 *   preferredProjectId?: string,
 *   filterBars?: unknown[],
 *   ownerId?: string,
 *   operatorId?: string,
 *   dueDate?: string,
 *   workBranchName?: string,
 * }} opts
 */
export function buildCreateTaskDraft(opts = {}) {
  const taskTypes = opts.taskTypes || []
  const defaultTypeId = taskTypes.length > 0 ? taskTypes[0].id : null
  const statuses = opts.taskStatuses || []
  const projects = opts.projects || []
  return {
    title: '',
    description: '',
    task_kind: '',
    code_lang: '',
    ...emptyCreateTaskStructuredFields(),
    progressColumn: { id: statuses.length > 0 ? statuses[0].id : 0 },
    task_type: { id: defaultTypeId },
    parent_task: buildInitialParentTaskId({
      taskTypes,
      todos: opts.todos,
      defaultTypeId,
      preferredParentTaskId: resolvePreferredParentTaskIdFromFilterBars(opts.filterBars),
    }),
    container_image: { id: null },
    priority: 1,
    due_date: opts.dueDate || '',
    projectSelections: [{
      projectId: pickCreateTaskDefaultProjectId(projects, new Set(), opts.preferredProjectId),
      repoBranches: [],
    }],
    workBranchPreset: 'feature',
    workBranchName: opts.workBranchName || '',
    mergeTargetPreset: 'custom',
    mergeTargetName: '',
    assignees: [],
    owner: opts.ownerId || '',
    operator: opts.operatorId || '',
    auto_run: false,
    queued_auto_run: false,
    force_auto_run: false,
    auto_commit_after_agent_complete: false,
    feature_params_source: '',
    personal_feature_params_config_id: '',
  }
}

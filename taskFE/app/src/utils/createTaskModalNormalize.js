/**
 * CreateTaskModal：规范化 editingTask 嵌套字段。
 */
import { normalizeFeatureParamsSourceForSelect } from './envParamsSourceSelection.js'
import { normalizeParentTaskId } from './workPanelCreateTaskParent.js'

/**
 * @param {object|null|undefined} task
 * @param {{ taskStatuses?: unknown[], taskTypes?: unknown[] }} ctx
 */
export function normalizeEditingTaskNestedObjects(task, ctx = {}) {
  const t = task
  if (!t || typeof t !== 'object') return

  const taskStatuses = ctx.taskStatuses || []
  const taskTypes = ctx.taskTypes || []

  if (!t.progressColumn || typeof t.progressColumn !== 'object') {
    const firstId = taskStatuses?.[0]?.id
    t.progressColumn = { id: firstId !== undefined && firstId !== null ? firstId : 0 }
  } else if (t.progressColumn.id === undefined || t.progressColumn.id === null) {
    const firstId = taskStatuses?.[0]?.id
    if (firstId !== undefined && firstId !== null) {
      t.progressColumn.id = firstId
    }
  }

  if (!t.task_type || typeof t.task_type !== 'object') {
    const fromTask = t.deliverable_obj_id ?? t.deliverable_obj?.id ?? null
    const firstTypeId = taskTypes?.[0]?.id
    t.task_type = {
      id: fromTask != null && fromTask !== ''
        ? fromTask
        : (firstTypeId !== undefined && firstTypeId !== null ? firstTypeId : null),
    }
  } else if (t.task_type.id == null || t.task_type.id === '') {
    const fromTask = t.deliverable_obj_id ?? t.deliverable_obj?.id
    if (fromTask != null && fromTask !== '') {
      t.task_type.id = fromTask
    }
  }

  t.parent_task = normalizeParentTaskId(t.parent_task)

  if (!t.container_image || typeof t.container_image !== 'object') {
    t.container_image = { id: null }
  }

  if (typeof t.auto_run !== 'boolean') {
    t.auto_run = false
  }
  if (typeof t.force_auto_run !== 'boolean') {
    t.force_auto_run = false
  }

  t.feature_params_source = normalizeFeatureParamsSourceForSelect(t.feature_params_source)
  if (t.personal_feature_params_config_id == null) {
    t.personal_feature_params_config_id = ''
  } else {
    t.personal_feature_params_config_id = String(t.personal_feature_params_config_id).trim()
  }

  if (!Array.isArray(t.projectSelections)) {
    t.projectSelections = []
  }
}

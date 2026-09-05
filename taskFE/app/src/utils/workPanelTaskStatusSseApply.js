/**
 * Apply a work-panel SSE task_status_changed payload onto a todos array (immutable).
 * @param {Array<object>} todos
 * @param {object} data SSE status_data
 * @returns {{ todos: Array<object>, matched: boolean, changed: boolean }}
 */
export function applyWorkPanelTaskStatusChanged(todos, data) {
  const list = Array.isArray(todos) ? todos : []
  if (!data || typeof data !== 'object') {
    return { todos: list, matched: false, changed: false }
  }
  const taskId = String(data.task_id || data.taskId || '').trim()
  if (!taskId) {
    return { todos: list, matched: false, changed: false }
  }
  const nextColumn =
    data.progress_column_id != null ? String(data.progress_column_id) : null
  const hasCompleted = Object.prototype.hasOwnProperty.call(data, 'completed')
  let matched = false
  let changed = false
  const next = list.map((todo) => {
    if (String(todo?.id ?? '') !== taskId) return todo
    matched = true
    const patch = { ...todo }
    if (nextColumn != null) {
      const prev =
        todo.progress_column_id ??
        todo.progressColumn?.id ??
        todo.progress_column?.id
      if (String(prev ?? '') !== nextColumn) {
        patch.progress_column_id = nextColumn
        if (patch.progressColumn && typeof patch.progressColumn === 'object') {
          patch.progressColumn = { ...patch.progressColumn, id: nextColumn }
          if (data.progress_column_name) {
            patch.progressColumn.name = String(data.progress_column_name)
          }
        }
        if (patch.progress_column && typeof patch.progress_column === 'object') {
          patch.progress_column = { ...patch.progress_column, id: nextColumn }
          if (data.progress_column_name) {
            patch.progress_column.name = String(data.progress_column_name)
          }
        }
        changed = true
      }
    }
    if (hasCompleted && Boolean(todo.completed) !== Boolean(data.completed)) {
      patch.completed = Boolean(data.completed)
      changed = true
    }
    return changed ? patch : todo
  })
  return { todos: changed ? next : list, matched, changed }
}

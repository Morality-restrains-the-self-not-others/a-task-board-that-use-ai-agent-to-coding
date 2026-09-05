/**
 * 工作面板看板：进度列解析与任务分列（纯函数，便于单测）。
 */

export const DEFAULT_TASK_STATUSES = [
  { id: 0, name: '待处理', color: '#ff6b6b' },
  { id: 1, name: '进行中', color: '#4dabf7' },
  { id: 2, name: '已完成', color: '#51cf66' },
]

const STATUS_COLORS = [
  '#ff6b6b',
  '#4dabf7',
  '#51cf66',
  '#fcc419',
  '#9c27b0',
  '#20c997',
  '#17a2b8',
  '#6c757d',
]

/** 从任务对象读取原始 progress 列 ID（不含未分配回落）。 */
export function readTodoProgressColumnId(todo) {
  if (!todo) return null
  const raw =
    todo.progressColumn?.id ??
    todo.progress_column_id ??
    todo.progress_column?.id ??
    todo.progress_column ??
    todo.status?.id ??
    todo.status
  if (raw == null || raw === '') return null
  return String(raw)
}

/**
 * 看板分列用：未设置 progress 列的任务归入第一列，避免 API 有数据但界面全空。
 */
export function resolveTodoKanbanColumnId(todo, taskStatuses) {
  const explicit = readTodoProgressColumnId(todo)
  const columns = Array.isArray(taskStatuses) ? taskStatuses : []
  if (explicit != null) {
    const known = columns.some((s) => String(s.id) === explicit)
    if (known) return explicit
    if (columns.length > 0) return String(columns[0].id)
    return explicit
  }
  const first = columns[0]?.id
  if (first == null || first === '') return null
  return String(first)
}

export function isTodoInKanbanColumn(todo, statusId, taskStatuses) {
  const col = resolveTodoKanbanColumnId(todo, taskStatuses)
  if (col == null) return false
  return col === String(statusId)
}

/** 进度列按 order_num/order 升序、同值按 id 兜底排序（看板分列与创建任务下拉共用）。 */
export function sortProgressColumns(columns) {
  if (!Array.isArray(columns)) return []
  return columns.slice().sort((a, b) => {
    const oa = Number(a?.order_num ?? a?.order)
    const ob = Number(b?.order_num ?? b?.order)
    const na = Number.isFinite(oa) ? oa : 0
    const nb = Number.isFinite(ob) ? ob : 0
    if (na !== nb) return na - nb
    return String(a?.id ?? '').localeCompare(String(b?.id ?? ''))
  })
}

export function mapProgressColumnsToTaskStatuses(columns) {
  if (!Array.isArray(columns) || columns.length === 0) return null
  return sortProgressColumns(columns).map((column, index) => ({
    id: column.id,
    name: column.name,
    color: STATUS_COLORS[index % STATUS_COLORS.length],
  }))
}

export function todoOrderNum(todo) {
  const n = Number(todo?.order)
  return Number.isFinite(n) ? n : 0
}

export function todoTimeMs(todo) {
  const raw = todo?.updated_at ?? todo?.updatedAt ?? todo?.created_at ?? todo?.createdAt
  if (raw == null || raw === '') return 0
  const ms = Date.parse(raw)
  return Number.isFinite(ms) ? ms : 0
}

export function sortTodosForKanbanColumn(todos) {
  return todos.slice().sort((a, b) => {
    const oa = todoOrderNum(a)
    const ob = todoOrderNum(b)
    if (oa !== ob) return oa - ob
    const ta = todoTimeMs(a)
    const tb = todoTimeMs(b)
    if (tb !== ta) return tb - ta
    try {
      const ba = BigInt(String(a.id ?? '0'))
      const bb = BigInt(String(b.id ?? '0'))
      if (ba < bb) return 1
      if (ba > bb) return -1
      return 0
    } catch {
      return String(b.id).localeCompare(String(a.id))
    }
  })
}

export function filterTodosByKanbanColumn(todos, statusId, taskStatuses) {
  return sortTodosForKanbanColumn(
    (Array.isArray(todos) ? todos : []).filter((todo) =>
      isTodoInKanbanColumn(todo, statusId, taskStatuses),
    ),
  )
}

/**
 * 按进度列统计任务数（顺序与 taskStatuses 一致），供分区标题展示各列数量。
 * @returns {{ id: *, name: string, color: string, count: number }[]}
 */
export function countTodosByKanbanColumns(todos, taskStatuses) {
  const columns = Array.isArray(taskStatuses) ? taskStatuses : []
  return columns.map((status) => ({
    id: status.id,
    name: status.name || '',
    color: status.color || '#94a3b8',
    count: filterTodosByKanbanColumn(todos, status.id, columns).length,
  }))
}

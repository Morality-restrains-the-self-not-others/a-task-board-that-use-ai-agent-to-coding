/**
 * 工作面板：多条交付物过滤栏的创建与分区。
 */
import {
  filterTodosByDeliverablePath,
  readFilterFromPath,
  rootDeliverablePath,
} from './workPanelDeliverableAggregation.js'

let _filterBarSeq = 0

/** @returns {{ id: string, path: ReturnType<rootDeliverablePath> }} */
export function createDeliverableFilterBar(id) {
  _filterBarSeq += 1
  return {
    id: id != null && String(id) !== '' ? String(id) : `bar-${_filterBarSeq}`,
    path: rootDeliverablePath(),
  }
}

/** @returns {Array<{ id: string, path: ReturnType<rootDeliverablePath> }>} */
export function defaultDeliverableFilterBars() {
  return [createDeliverableFilterBar('bar-0')]
}

/**
 * 在指定过滤栏上方插入一条新过滤栏；找不到 beforeBarId 时追加到末尾。
 *
 * @param {Array<{ id: string, path?: unknown }>|null|undefined} bars
 * @param {string|number|null|undefined} beforeBarId
 * @param {{ id: string, path: ReturnType<rootDeliverablePath> }|null|undefined} [newBar]
 * @returns {Array<{ id: string, path: ReturnType<rootDeliverablePath> }>}
 */
export function insertDeliverableFilterBarBefore(bars, beforeBarId, newBar) {
  const list = Array.isArray(bars) ? [...bars] : []
  const inserted = newBar != null ? newBar : createDeliverableFilterBar()
  const idx =
    beforeBarId == null || beforeBarId === ''
      ? -1
      : list.findIndex((b) => String(b.id) === String(beforeBarId))
  if (idx < 0) {
    list.push(inserted)
    return list
  }
  list.splice(idx, 0, inserted)
  return list
}

/**
 * @param {unknown[]|null|undefined} path
 */
export function filterBarSectionLabel(path) {
  const segments = Array.isArray(path) ? path : []
  for (let i = segments.length - 1; i >= 0; i -= 1) {
    const seg = segments[i]
    if (seg?.type === 'task') {
      const label = String(seg.label ?? '').trim()
      if (label) return label
      if (seg.id != null && seg.id !== '') return String(seg.id)
    }
  }
  return '未命名过滤'
}

/**
 * 按多条过滤栏分区：命中任一过滤条件的任务进入对应分区；其余进入「其他」。
 *
 * @param {unknown[]} todos
 * @param {unknown[]} categories
 * @param {Array<{ id?: string, path?: unknown }>|null|undefined} bars
 * @returns {{
 *   other: unknown[],
 *   sections: Array<{ barId: string, label: string, path: unknown, todos: unknown[] }>,
 * }}
 */
export function partitionTodosByFilterBars(todos, categories, bars) {
  const list = Array.isArray(todos) ? todos : []
  const barList = Array.isArray(bars) ? bars : []
  const sections = []
  const matchedIds = new Set()

  for (const bar of barList) {
    if (!bar || typeof bar !== 'object') continue
    const path = bar.path
    const { taskId } = readFilterFromPath(path)
    if (taskId == null || taskId === '') continue
    const filtered = filterTodosByDeliverablePath(list, categories, path)
    const barId = bar.id != null && String(bar.id) !== '' ? String(bar.id) : `bar-${sections.length}`
    sections.push({
      barId,
      label: filterBarSectionLabel(path),
      path,
      todos: filtered,
    })
    for (const todo of filtered) {
      if (todo?.id != null && todo.id !== '') matchedIds.add(String(todo.id))
    }
  }

  const other = list.filter((todo) => {
    const id = todo?.id != null ? String(todo.id) : null
    if (!id) return false
    return !matchedIds.has(id)
  })

  return { other, sections }
}

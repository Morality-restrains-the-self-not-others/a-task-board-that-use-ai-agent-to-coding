import { formatTaskIdTitleLabel } from './taskIdDisplay.js'

/**
 * 读取工作空间人读序号；无效时返回 undefined（不回退技术 ID）。
 * @param {unknown} todo
 * @returns {number|undefined}
 */
export function readTodoWorkspaceSeq(todo) {
  if (!todo || typeof todo !== 'object') return undefined
  const n = Number(todo.workspace_seq ?? todo.workspaceSeq)
  if (!Number.isInteger(n) || n <= 0) return undefined
  return n
}

/**
 * 过滤栏 / 上层交付物下拉用的任务内容项。
 * @param {unknown} todo
 * @returns {{ id: string, title: string, workspace_seq?: number }|null}
 */
export function todoToDeliverableContent(todo) {
  const id = todo?.id != null && todo.id !== '' ? String(todo.id) : null
  if (!id) return null
  const title = String(todo.title ?? todo.name ?? id).trim() || id
  const workspace_seq = readTodoWorkspaceSeq(todo)
  return workspace_seq != null ? { id, title, workspace_seq } : { id, title }
}

/**
 * 下拉 option 文案：有序号为 `#N 标题`，否则为标题。
 * @param {{ id?: unknown, title?: unknown, name?: unknown, workspace_seq?: unknown }|null|undefined} content
 * @returns {string}
 */
export function deliverableContentOptionLabel(content) {
  if (!content || content.id == null || content.id === '') return ''
  return formatTaskIdTitleLabel(
    content.id,
    content.title ?? content.name,
    content.workspace_seq ?? content.workspaceSeq,
  )
}

/**
 * 看板卡片展示用任务编号：取 id 字符串后 6 位（不足则原样）。
 * 人读路径请用 formatTaskDisplayNo / formatTaskIdTitleLabel（workspace_seq）。
 * @param {unknown} id
 * @returns {string}
 */
export function taskIdLast6(id) {
  if (id == null || id === '') return ''
  const s = String(id)
  return s.length <= 6 ? s : s.slice(-6)
}

/**
 * 工作空间人读序号：`#12`。无效序号返回空串（不回退技术 ID 后六位）。
 * @param {unknown} workspaceSeq
 * @returns {string}
 */
export function formatTaskDisplayNo(workspaceSeq) {
  const n = Number(workspaceSeq)
  if (!Number.isInteger(n) || n <= 0) return ''
  return `#${n}`
}

/**
 * 导航/派生自/上层交付物统一文案：优先 `#序号` 或 `#序号 标题`。
 * 无序号时不展示技术 ID 后六位。
 * @param {unknown} id
 * @param {unknown} [title]
 * @param {unknown} [workspaceSeq]
 * @returns {string}
 */
export function formatTaskIdTitleLabel(id, title, workspaceSeq) {
  const num = formatTaskDisplayNo(workspaceSeq)
  const name = title != null ? String(title).trim() : ''
  if (num) return name ? `${num} ${name}` : num
  if (id == null || id === '') return ''
  return name
}

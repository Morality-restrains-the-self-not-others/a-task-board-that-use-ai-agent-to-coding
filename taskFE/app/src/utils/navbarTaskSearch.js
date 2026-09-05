/**
 * 导航栏任务搜索：成员名匹配、search API 查询串、跳转路径。
 */

import { formatTaskDisplayNo } from './taskIdDisplay.js'

/**
 * 规范化导航栏搜索关键字：去空白、去 #，并把常见误粘贴形态还原为规范任务 id：
 * - `task-task_<digits>`（库名/服务名叠前缀）→ `task_<digits>`
 * - 评论容器名 / 云实例名 `task_<digits>_cmt_<commentId>`（含 `task_task_` 双前缀）→ `task_<digits>`
 * 规范评论 id `cmt_<id>` 原样保留，由服务端按评论表匹配所属任务。
 * @param {unknown} raw
 * @returns {string}
 */
export function normalizeSearchText(raw) {
  let q = String(raw ?? '').trim()
  if (!q) return q
  if (q.startsWith('#')) q = q.slice(1).trim()
  if (q.startsWith('task-task_')) {
    q = `task_${q.slice('task-task_'.length)}`
  }
  const taskID = taskIDFromCommentContainerQuery(q)
  if (taskID) return taskID
  return q
}

/**
 * @param {string} q
 * @returns {string}
 */
function taskIDFromCommentContainerQuery(q) {
  const marker = '_cmt_'
  const i = q.indexOf(marker)
  if (i <= 0) return ''
  const head = q.slice(0, i)
  if (head.startsWith('task_task_')) {
    const digits = head.slice('task_task_'.length)
    if (/^\d+$/.test(digits)) return `task_${digits}`
    return ''
  }
  if (head.startsWith('task_')) {
    const digits = head.slice('task_'.length)
    if (/^\d+$/.test(digits)) return `task_${digits}`
  }
  return ''
}

/**
 * @param {{ id?: unknown, name?: unknown }[]} members
 * @param {string} q
 * @param {number} [limit=20]
 * @returns {string[]} company_member / user ids matching name or id
 */
export function matchMemberIdsByQuery(members, q, limit = 20) {
  const needle = normalizeSearchText(q).toLowerCase()
  if (!needle || !Array.isArray(members)) return []
  const out = []
  const seen = new Set()
  for (const m of members) {
    const id = String(m?.id ?? '').trim()
    const name = String(m?.name ?? '').trim().toLowerCase()
    if (!id) continue
    if (name.includes(needle) || id.toLowerCase().includes(needle)) {
      if (seen.has(id)) continue
      seen.add(id)
      out.push(id)
      if (out.length >= limit) break
    }
  }
  return out
}

/**
 * @param {{ q?: string, assigneeIds?: string[], workspaceId?: string, limit?: number }} opts
 * @returns {string} query string without leading ?
 */
export function buildTaskSearchQuery(opts = {}) {
  const params = new URLSearchParams()
  const q = normalizeSearchText(opts.q)
  if (q) params.set('q', q)
  const ids = Array.isArray(opts.assigneeIds)
    ? opts.assigneeIds.map((x) => String(x).trim()).filter(Boolean)
    : []
  if (ids.length) params.set('assignee_ids', ids.join(','))
  const ws = String(opts.workspaceId || '').trim()
  if (ws) params.set('workspace_id', ws)
  const limit = Number(opts.limit)
  params.set('limit', String(Number.isFinite(limit) && limit > 0 ? Math.min(limit, 100) : 20))
  return params.toString()
}

/**
 * @param {string} tenantId
 * @param {{ workspaceId?: string, taskId?: string, accessCode?: string }} opts
 */
export function buildWorkPanelHref(tenantId, opts = {}) {
  const tid = String(tenantId || '').trim()
  if (!tid) return '#'
  const params = new URLSearchParams()
  const ws = String(opts.workspaceId || '').trim()
  if (ws) params.set('workspace_id', ws)
  const taskId = String(opts.taskId || '').trim()
  if (taskId) params.set('task_id', taskId)
  const accessCode = String(opts.accessCode || '').trim()
  if (accessCode) params.set('accessCode', accessCode)
  const qs = params.toString()
  return `/tenant/${tid}/work-panel/${qs ? `?${qs}` : ''}`
}

/**
 * @param {string} tenantId
 * @param {{ workspaceId: string, taskId: string, accessCode?: string, commentId?: string }} opts
 * commentId 来自搜索命中评论（comment-container / cmt_ 查询），任务详情页挂载后
 * 据此滚到该评论并高亮（OPT-20260817-013 前端部分）。
 */
export function buildTaskDetailHref(tenantId, opts) {
  const tid = String(tenantId || '').trim()
  const ws = String(opts?.workspaceId || '').trim()
  const taskId = String(opts?.taskId || '').trim()
  if (!tid || !ws || !taskId) return '#'
  const params = new URLSearchParams()
  const accessCode = String(opts?.accessCode || '').trim()
  if (accessCode) params.set('accessCode', accessCode)
  const commentId = String(opts?.commentId || '').trim()
  if (commentId) params.set('comment', commentId)
  const qs = params.toString()
  return `/tenant/${tid}/workspace/${ws}/task-detail/${taskId}/${qs ? `?${qs}` : ''}`
}

/**
 * @param {Record<string, unknown>} hit
 * @param {Map<string, string>|Record<string, string>} [nameById]
 */
export function formatSearchHitLabel(hit, nameById) {
  const title = String(hit?.title || hit?.name || '').trim() || '(无标题)'
  const shortId = formatTaskDisplayNo(hit?.workspace_seq)
  const owner = String(hit?.owner ?? '').trim()
  const operator = String(hit?.operator ?? '').trim()
  const assignees = Array.isArray(hit?.assignees) ? hit.assignees.map(String) : []
  const lookup = nameById instanceof Map
    ? (k) => nameById.get(k) || k
    : (k) => (nameById && nameById[k]) || k
  const people = []
  const pushPerson = (raw) => {
    if (!raw) return
    const n = lookup(raw)
    if (n && !people.includes(n)) people.push(n)
  }
  pushPerson(owner)
  pushPerson(operator)
  for (const a of assignees) pushPerson(a)
  const peopleText = people.length ? ` · ${people.join(', ')}` : ''
  return shortId ? `${shortId} ${title}${peopleText}` : `${title}${peopleText}`
}

/**
 * 从 todos 中按完整 id 或 workspace_seq 匹配任务。
 * @param {Array<{ id?: unknown }>} todos
 * @param {string} taskIdOrSuffix
 */
export function findTodoByTaskIdQuery(todos, taskIdOrSuffix) {
  const q = normalizeSearchText(taskIdOrSuffix)
  if (!q || !Array.isArray(todos)) return null
  const exact = todos.find((t) => String(t?.id ?? '') === q)
  if (exact) return exact
  const bySeq = todos.find((t) => {
    const n = Number(t?.workspace_seq)
    return Number.isInteger(n) && n > 0 && String(n) === q
  })
  if (bySeq) return bySeq
  return todos.find((t) => String(t?.id ?? '').endsWith(q)) || null
}

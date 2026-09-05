/**
 * WorkPanel：工作空间机器节点汇总（启动/闲置/回收间隙）与摘要点击过滤。
 */

/** @typedef {'started' | 'idle'} MachineStatusFilter */

export const MACHINE_STATUS_FILTER = Object.freeze({
  STARTED: 'started',
  IDLE: 'idle',
})

/**
 * @typedef {{
 *   startedCount: number,
 *   startingCount: number,
 *   idleCount: number,
 *   busyCount: number,
 *   idleRecycleMinutes: number,
 * }} WorkspaceMachineSummary
 */

/**
 * @param {unknown} payload
 * @returns {WorkspaceMachineSummary | null}
 */
export function normalizeWorkspaceMachineSummary(payload) {
  if (payload == null || typeof payload !== 'object') {
    return null
  }
  const started = Number(payload.started_count)
  const idle = Number(payload.idle_count)
  const busy = Number(payload.busy_count)
  const minutes = Number(payload.idle_recycle_minutes)
  if (![started, idle, busy, minutes].every((n) => Number.isFinite(n))) {
    return null
  }
  const startingRaw = payload.starting_count
  const starting =
    startingRaw == null || startingRaw === ''
      ? 0
      : Number(startingRaw)
  if (!Number.isFinite(starting) || starting < 0) {
    return null
  }
  return {
    startedCount: started,
    startingCount: starting,
    idleCount: idle,
    busyCount: busy,
    idleRecycleMinutes: minutes,
  }
}

/**
 * @param {WorkspaceMachineSummary | null | undefined} summary
 * @returns {string}
 */
export function formatWorkspaceMachineSummaryLabel(summary) {
  if (!summary) {
    return '机器节点：—'
  }
  const recycle =
    summary.idleRecycleMinutes <= 0
      ? '闲置回收：已关闭'
      : `闲置回收：${summary.idleRecycleMinutes} 分钟`
  const starting =
    summary.startingCount > 0 ? ` · 启动中 ${summary.startingCount}` : ''
  return `机器节点：已启动 ${summary.startedCount}${starting} · 闲置 ${summary.idleCount} · ${recycle}`
}

/**
 * 摘要点击：同一项再次点击清除；另一项切换。
 * @param {MachineStatusFilter | null | undefined} current
 * @param {MachineStatusFilter} next
 * @returns {MachineStatusFilter | null}
 */
export function toggleMachineStatusFilter(current, next) {
  if (next !== MACHINE_STATUS_FILTER.STARTED && next !== MACHINE_STATUS_FILTER.IDLE) {
    throw new Error(`unknown machine status filter: ${String(next)}`)
  }
  return current === next ? null : next
}

/**
 * @param {MachineStatusFilter | null | undefined} filter
 * @returns {string}
 */
export function machineStatusFilterChipLabel(filter) {
  if (filter === MACHINE_STATUS_FILTER.STARTED) {
    return '机器节点·已启动'
  }
  if (filter === MACHINE_STATUS_FILTER.IDLE) {
    return '机器节点·闲置'
  }
  return ''
}

/**
 * @param {{ id?: string|number } | null | undefined} todo
 * @param {Record<string, { machineRunning?: boolean, containerRunning?: boolean }>} indicators
 * @param {MachineStatusFilter | null | undefined} filter
 * @returns {boolean}
 */
export function todoMatchesMachineStatusFilter(todo, indicators, filter) {
  if (filter == null || filter === '') {
    return true
  }
  if (filter !== MACHINE_STATUS_FILTER.STARTED && filter !== MACHINE_STATUS_FILTER.IDLE) {
    throw new Error(`unknown machine status filter: ${String(filter)}`)
  }
  const taskId = String(todo?.id ?? '').trim()
  if (!taskId) {
    return false
  }
  const ind = indicators && typeof indicators === 'object' ? indicators[taskId] : null
  const machineRunning = Boolean(ind?.machineRunning)
  const containerRunning = Boolean(ind?.containerRunning)
  if (filter === MACHINE_STATUS_FILTER.STARTED) {
    return machineRunning
  }
  return machineRunning && !containerRunning
}

/**
 * @param {Array<{ id?: string|number }>} todos
 * @param {Record<string, { machineRunning?: boolean, containerRunning?: boolean }>} indicators
 * @param {MachineStatusFilter | null | undefined} filter
 * @returns {Array<{ id?: string|number }>}
 */
export function filterTodosByMachineStatus(todos, indicators, filter) {
  const list = Array.isArray(todos) ? todos : []
  if (filter == null || filter === '') {
    return list
  }
  return list.filter((todo) => todoMatchesMachineStatusFilter(todo, indicators, filter))
}

/**
 * @param {{ apiFetch: typeof fetch, tenantId: string|number, workspaceId: string|number }} opts
 * @returns {Promise<WorkspaceMachineSummary | null>}
 */
export async function fetchWorkspaceMachineSummary(opts) {
  const { apiFetch, tenantId, workspaceId } = opts
  const tid = String(tenantId ?? '').trim()
  const wid = String(workspaceId ?? '').trim()
  if (!tid || !wid || wid === 'default') {
    return null
  }
  const url = `/api/cloud/compute/workspace-machine-summary/tenant_id/${encodeURIComponent(tid)}/workspace_id/${encodeURIComponent(wid)}`
  const response = await apiFetch(url, {
    credentials: 'include',
    headers: { Accept: 'application/json' },
  })
  if (!response.ok) {
    const err = new Error(`workspace-machine-summary HTTP ${response.status}`)
    err.status = response.status
    // apiFetch 已把失败请求 traceId 注入 response.traceId；透传至错误对象，
    // 供 useWorkPanelMachineSummary 捕获后注入 header-summary-row 的 data-traceId（OPT-20260809-012）。
    err.traceId = response.traceId || ''
    throw err
  }
  const data = await response.json()
  return normalizeWorkspaceMachineSummary(data)
}

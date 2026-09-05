/**
 * WorkPanel 任务卡片：机器运行圆圈 / 容器运行原点 的数据归一与展示判定。
 */

/**
 * @typedef {{ task_id: string, machine_running?: boolean, container_running?: boolean }} RuntimeIndicatorRow
 * @typedef {{ machineRunning: boolean, containerRunning: boolean }} TaskRuntimeIndicator
 */

/** 空心环：机器节点运行中 */
export const RUNTIME_MACHINE_TIP = '机器节点运行中'

/** 实心点：容器运行中 */
export const RUNTIME_CONTAINER_TIP = '容器运行中'

/**
 * 将批量 API 响应归一为 taskId → 指示状态 map。
 * @param {unknown} payload
 * @returns {Record<string, TaskRuntimeIndicator>}
 */
export function normalizeWorkspaceRuntimeIndicators(payload) {
  const out = Object.create(null)
  if (payload == null || typeof payload !== 'object') {
    return out
  }
  const rows = Array.isArray(payload.indicators) ? payload.indicators : []
  for (const row of rows) {
    if (row == null || typeof row !== 'object') {
      continue
    }
    const taskId = String(row.task_id ?? '').trim()
    if (!taskId) {
      continue
    }
    out[taskId] = {
      machineRunning: Boolean(row.machine_running),
      containerRunning: Boolean(row.container_running),
    }
  }
  return out
}

/**
 * @param {TaskRuntimeIndicator | null | undefined} indicator
 */
export function resolveTaskRuntimeIndicatorFlags(indicator) {
  const machineRunning = Boolean(indicator?.machineRunning)
  const containerRunning = Boolean(indicator?.containerRunning)
  return {
    machineRunning,
    containerRunning,
    showMachineRing: machineRunning,
    showContainerDot: containerRunning,
    showRuntimeIndicators: machineRunning || containerRunning,
  }
}

/**
 * GET 工作区批量运行态指示。
 * @param {{ apiFetch: typeof fetch, tenantId: string|number, workspaceId: string|number }} opts
 * @returns {Promise<Record<string, TaskRuntimeIndicator>>}
 */
export async function fetchWorkspaceRuntimeIndicators(opts) {
  const { apiFetch, tenantId, workspaceId } = opts
  const tid = String(tenantId ?? '').trim()
  const wid = String(workspaceId ?? '').trim()
  if (!tid || !wid || wid === 'default') {
    return Object.create(null)
  }
  const url = `/api/cloud/compute/workspace-runtime-indicators/tenant_id/${encodeURIComponent(tid)}/workspace_id/${encodeURIComponent(wid)}`
  const response = await apiFetch(url, {
    credentials: 'include',
    headers: { Accept: 'application/json' },
  })
  if (!response.ok) {
    const err = new Error(`workspace-runtime-indicators HTTP ${response.status}`)
    err.status = response.status
    // apiFetch 已把失败请求 traceId 注入 response.traceId；透传至错误对象，
    // 供 useWorkPanelMachineSummary 捕获后注入 header-summary-row 的 data-traceId（OPT-20260809-012）。
    err.traceId = response.traceId || ''
    throw err
  }
  const data = await response.json()
  return normalizeWorkspaceRuntimeIndicators(data)
}

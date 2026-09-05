/**
 * 工作空间自动调度是否启用（与 join_queue_requires_workspace_schedule 判定一致）。
 * @param {unknown} snapshot GET /queue-schedule/ 响应体
 */
export function isWorkspaceAutoScheduleEnabled(snapshot) {
  return snapshot?.schedule_rhythm?.enabled === true
}

/**
 * 读取工作空间自动调度启用态。失败时抛出带 optional traceId 的 Error。
 * @param {{ tenantId: string, workspaceId: string, apiFetch?: Function }} opts
 */
export async function fetchWorkspaceAutoScheduleEnabled({ tenantId, workspaceId, apiFetch } = {}) {
  const tid = String(tenantId || '').trim()
  const wid = String(workspaceId || '').trim()
  if (!tid || !wid) {
    const err = new Error('tenantId and workspaceId required')
    throw err
  }
  const fetchFn = apiFetch || (typeof window !== 'undefined' ? window.apiFetch : null)
  if (typeof fetchFn !== 'function') {
    throw new Error('apiFetch unavailable')
  }
  const path = `/api/tenant/${tid}/workspace/${wid}/queue-schedule/`
  const resp = await fetchFn(path, {
    method: 'GET',
    credentials: 'include',
    headers: { Accept: 'application/json' },
  })
  if (!resp || typeof resp.json !== 'function') {
    throw new Error('queue-schedule response invalid')
  }
  const data = await resp.json().catch(() => ({}))
  if (!resp.ok) {
    const err = new Error(data?.error || data?.message || `HTTP ${resp.status}`)
    err.traceId = resp.headers?.get?.('X-Trace-Id') || data?.trace_id || ''
    throw err
  }
  return {
    enabled: isWorkspaceAutoScheduleEnabled(data),
    snapshot: data,
  }
}

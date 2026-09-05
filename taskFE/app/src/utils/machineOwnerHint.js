/**
 * 任务详情镜像区：机器节点所属任务提示（纯函数）
 */

/**
 * @param {object|null|undefined} runtimeStatus server-runtime-status 响应
 * @param {{ serverUrl?: string, containerPageUrl?: string, mockContainerRunning?: boolean }} frontendFlags
 * @returns {boolean}
 */
export function shouldShowMachineOwnerHint(runtimeStatus, frontendFlags = {}) {
  const flags = frontendFlags || {}
  const frontendRunning = Boolean(
    String(flags.serverUrl || '').trim()
      || String(flags.containerPageUrl || '').trim()
      || flags.mockContainerRunning,
  )
  const apiRunning = Boolean(runtimeStatus?.container_running)
  if (!frontendRunning && !apiRunning) {
    return false
  }
  const owners = normalizeOwnerTaskIds(runtimeStatus)
  const instanceId = String(runtimeStatus?.instance_id || '').trim()
  return owners.length > 0 && Boolean(instanceId)
}

/**
 * @param {object|null|undefined} runtimeStatus
 * @returns {string[]}
 */
export function normalizeOwnerTaskIds(runtimeStatus) {
  const raw = runtimeStatus?.machine_owner_task_ids
  if (!Array.isArray(raw)) {
    return []
  }
  const out = []
  const seen = new Set()
  for (const id of raw) {
    const tid = String(id || '').trim()
    if (!tid || seen.has(tid)) continue
    seen.add(tid)
    out.push(tid)
  }
  return out
}

/**
 * @param {string[]} ownerTaskIds
 * @param {string} viewerTaskId
 * @param {{ tenantId: string, workspaceId: string }} ctx
 * @returns {{ taskId: string, isViewer: boolean, href: string|null }[]}
 */
export function buildMachineOwnerEntries(ownerTaskIds, viewerTaskId, ctx) {
  const viewer = String(viewerTaskId || '').trim()
  const tenantId = String(ctx?.tenantId || '').trim()
  const workspaceId = String(ctx?.workspaceId || '').trim()
  return (ownerTaskIds || []).map((taskId) => {
    const tid = String(taskId || '').trim()
    const isViewer = Boolean(viewer && tid === viewer)
    let href = null
    if (!isViewer && tid && tenantId && workspaceId) {
      href = `/tenant/${encodeURIComponent(tenantId)}/workspace/${encodeURIComponent(workspaceId)}/task-detail/${encodeURIComponent(tid)}/`
    }
    return { taskId: tid, isViewer, href }
  }).filter((e) => e.taskId)
}

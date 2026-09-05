/**
 * SSE 断连时不再 REST 轮询启动进度（直播靠推送；对账靠「刷新状态」）。
 * 保留 URL/fetch 纯函数供测试与未来手动同步复用。
 */
import { getApiUrl } from '../../utils/config.js'
import { apiFetch } from '../../utils/apiUtils.js'
import { appendCommentIdPath } from '../../utils/containerForwardCommentId.js'

export const SERVER_STARTUP_STATUS_POLL_MS = 5000
export const SERVER_STARTUP_STATUS_POLL_SLOW_MS = 15000
export const SERVER_STARTUP_STATUS_POLL_MAX_TICKS = 120

export function shouldPollServerStartupStatus({ isServerStarting }) {
  return Boolean(isServerStarting)
}

export function shouldApplyPolledStartupStatus(currentProgress, polledProgress) {
  const cur = Number(currentProgress)
  const next = Number(polledProgress)
  if (!Number.isFinite(next)) return false
  if (!Number.isFinite(cur)) return true
  return next >= cur
}

export function buildServerStartupStatusPollUrl({ tenantId, workspaceId, taskId, eventId, commentId }) {
  const tid = String(tenantId || '').trim()
  const wid = String(workspaceId || '').trim()
  const tk = String(taskId || '').trim()
  if (!tid || !wid || !tk) return ''
  const params = new URLSearchParams({ task_id: tk })
  const eid = String(eventId || '').trim()
  if (eid) params.set('event_id', eid)
  const cid = String(commentId || '').trim()
  return getApiUrl(
    appendCommentIdPath(
      `/api/cloud/compute/server-startup-status/tenant_id/${tid}/workspace_id/${wid}?${params.toString()}`,
      cid,
    ),
  )
}

export async function fetchServerStartupStatusPoll({ tenantId, workspaceId, taskId, eventId, commentId }) {
  const url = buildServerStartupStatusPollUrl({ tenantId, workspaceId, taskId, eventId, commentId })
  if (!url) return null
  const response = await apiFetch(url, {
    method: 'GET',
    credentials: 'include',
    headers: { Accept: 'application/json' },
  })
  const data = await response.json().catch(() => ({}))
  if (!response.ok) {
    return { ok: false, data }
  }
  return { ok: true, data }
}

export function createServerStartupStatusPollController() {
  const stop = () => {}
  const start = () => {}
  const tick = async () => {}
  const reschedule = () => {}
  return { start, stop, tick, reschedule }
}

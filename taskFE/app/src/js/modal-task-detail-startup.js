/**
 * 遗留看板任务详情模态：启动服务器后的 DOM 状态反馈与 REST 轮询兜底。
 */
import { appendCommentIdPath } from '../utils/containerForwardCommentId.js'

export function getWorkspaceIdFromLegacyContext() {
  const fromUser = window.currentUser?.current_workspace?.id
  if (fromUser != null && String(fromUser).trim()) {
    return String(fromUser).trim()
  }
  const m = window.location.pathname.match(/\/workspace\/([^/]+)\//)
  return m ? m[1] : ''
}

export function applyLegacyStartupStatusDom({ message, progress, status }) {
  const statusMessageEl = document.getElementById('status-message')
  const progressEl = document.getElementById('status-progress')
  const progressPctEl = document.getElementById('progress-percentage')
  const statusPanel = document.getElementById('server-startup-status')
  const indicator = document.getElementById('status-indicator')

  if (statusMessageEl && message) {
    statusMessageEl.textContent = message
  }
  const pct = Number(progress)
  if (Number.isFinite(pct)) {
    if (progressEl) {
      progressEl.style.width = `${Math.min(100, Math.max(0, pct))}%`
    }
    if (progressPctEl) {
      progressPctEl.textContent = `${Math.min(100, Math.max(0, pct))}%`
    }
  }
  if (statusPanel) {
    statusPanel.classList.remove('hidden')
    statusPanel.style.display = ''
  }
  if (indicator) {
    indicator.classList.remove('bg-gray-400', 'bg-green-500', 'bg-red-500')
    if (status === 'error') {
      indicator.classList.add('bg-red-500')
    } else if (status === 'success') {
      indicator.classList.add('bg-green-500')
    } else {
      indicator.classList.add('bg-yellow-400')
    }
  }
}

export function buildLegacyStartupStatusPollUrl({ tenantId, workspaceId, taskId, eventId, commentId }) {
  const tid = String(tenantId || '').trim()
  const wid = String(workspaceId || '').trim()
  const tk = String(taskId || '').trim()
  if (!tid || !wid || !tk) return ''
  const params = new URLSearchParams({ task_id: tk })
  const eid = String(eventId || '').trim()
  if (eid) params.set('event_id', eid)
  const base = `/api/cloud/compute/server-startup-status/tenant_id/${tid}/workspace_id/${wid}?${params.toString()}`
  // 评论级 CSC：comment_id 走 path（/comment_id/{cid}/），避免 query 回落到任务级启动事件。
  return appendCommentIdPath(base, commentId)
}

const legacyPollState = {
  timerId: null,
  eventId: '',
}

export function stopLegacyStartupStatusPoll() {
  if (legacyPollState.timerId !== null) {
    clearInterval(legacyPollState.timerId)
    legacyPollState.timerId = null
  }
  legacyPollState.eventId = ''
}

/** v84: 禁止 REST 轮询启动进度。直播走 SSE；对账走刷新按钮。保留函数以免遗留看板调用崩。 */
export function startLegacyStartupStatusPoll() {
  stopLegacyStartupStatusPoll()
}

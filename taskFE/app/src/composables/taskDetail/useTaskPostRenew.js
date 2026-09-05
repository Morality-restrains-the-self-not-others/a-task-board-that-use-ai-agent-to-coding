import { ref } from 'vue'
import { apiFetch } from '../../utils/apiUtils.js'

/**
 * 任务帖存续期展示与续存（+12 个月）。
 */
export function formatPostExpiry(iso) {
  if (!iso) return ''
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) return String(iso)
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')} 到期`
}

/**
 * @param {{
 *   task: import('vue').Ref|object|{value?: object},
 *   tenantId: () => string,
 *   workspaceId: () => string,
 *   taskId: () => string,
 *   onUpdated: (payload: object) => void,
 * }} opts
 */
export function useTaskPostRenew({ task, tenantId, workspaceId, taskId, onUpdated }) {
  const renewing = ref(false)
  const renewError = ref('')
  const renewErrorTraceId = ref('')

  const resolveTask = () => (task && typeof task === 'object' && 'value' in task ? task.value : task)

  const renewPost = async () => {
    if (!window.confirm('续存该任务帖将消耗 1 个创建帖次数，存续期延长 12 个月。确认续存？')) return
    renewing.value = true
    renewError.value = ''
    renewErrorTraceId.value = ''
    try {
      const t = resolveTask()
      const id = String(t?.id || taskId() || '')
      const url = `/api/tasks/todos/tenant_id/${encodeURIComponent(String(tenantId() || ''))}/workspace_id/${encodeURIComponent(String(workspaceId() || ''))}/${encodeURIComponent(id)}/renew/`
      const resp = await apiFetch(url, {
        method: 'POST',
        credentials: 'include',
        headers: { Accept: 'application/json' },
      })
      const requestTraceId = resp?.traceId || ''
      let errBody = {}
      if (!resp.ok) {
        try { errBody = await resp.json() } catch (e) { /* body 可能非 JSON */ }
        renewErrorTraceId.value = errBody?._traceId || requestTraceId
        renewError.value = errBody?.detail || errBody?.error || `续存失败（HTTP ${resp.status}）`
        return
      }
      let updated = null
      try { updated = await resp.json() } catch (e) { /* ignore */ }
      onUpdated(updated || { post_expired: false })
    } catch (e) {
      renewErrorTraceId.value = e?.traceId || ''
      renewError.value = `续存出错：${e?.message || '请稍后重试'}`
    } finally {
      renewing.value = false
    }
  }

  return { renewing, renewError, renewErrorTraceId, renewPost, formatPostExpiry }
}

import { ref } from 'vue'

/**
 * 工作空间级排队调度（OPT-20260824-005 配套）。
 * 快照加载 GET /api/tenant/{tid}/workspace/{wid}/queue-schedule/
 * 保存     PUT /api/tenant/{tid}/workspace/{wid}/queue-schedule/
 * @param {{ tenantId: string|() => string, workspaceId: string|() => string }} props
 *         tenantId/workspaceId 可传函数（页面切换工作空间时复用同一实例）。
 */
export function useWorkspaceQueueSchedule({ tenantId, workspaceId }) {
  const snapshot = ref(null)
  const loading = ref(false)
  const saving = ref(false)
  const error = ref('')
  const errorTraceId = ref('')
  const historyItems = ref([])
  const historyHasMore = ref(false)
  const historyCursor = ref('')
  const historyLoading = ref(false)
  const historyError = ref('')
  const historyErrorTraceId = ref('')

  function resolveId(v) {
    return typeof v === 'function' ? v() : v
  }

  function queuePath() {
    return `/api/tenant/${resolveId(tenantId)}/workspace/${resolveId(workspaceId)}/queue-schedule/`
  }

  function historyPath() {
    return `${queuePath()}history/`
  }

  function seedHistoryFromSnapshot(data) {
    const recent = Array.isArray(data?.recent_history) ? data.recent_history : []
    historyItems.value = recent
    // 后端快照已带分页标识，不再用 length>=8 启发式
    historyHasMore.value = !!data?.recent_history_has_more
    const last = recent[recent.length - 1]
    historyCursor.value = last ? `${last.created_at}|${last.id}` : ''
    historyError.value = ''
    historyErrorTraceId.value = ''
  }

  async function request(path, options = {}) {
    const apiFetch = window.apiFetch
    if (typeof apiFetch !== 'function') {
      throw new Error('apiFetch unavailable')
    }
    const { headers: extraHeaders, ...rest } = options
    const resp = await apiFetch(path, {
      credentials: 'include',
      ...rest,
      headers: { Accept: 'application/json', ...(extraHeaders || {}) },
    })
    const data = await resp.json().catch(() => ({}))
    if (!resp.ok) {
      const err = new Error(data?.error || data?.message || `HTTP ${resp.status}`)
      err.traceId = resp.headers?.get?.('X-Trace-Id') || data?.trace_id || ''
      throw err
    }
    return data
  }

  async function loadSnapshot() {
    loading.value = true
    error.value = ''
    errorTraceId.value = ''
    try {
      snapshot.value = await request(queuePath(), { method: 'GET' })
      seedHistoryFromSnapshot(snapshot.value)
    } catch (e) {
      error.value = e?.message || '加载工作空间排队调度失败'
      errorTraceId.value = e?.traceId || ''
      snapshot.value = null
      historyItems.value = []
      historyHasMore.value = false
      historyCursor.value = ''
    } finally {
      loading.value = false
    }
  }

  /**
   * 保存节奏并刷新快照。
   * @param {{ enabled: boolean, timezone: string, windows: Array<object> }} rhythm
   */
  async function saveSchedule(rhythm, { idempotencyKey } = {}) {
    saving.value = true
    error.value = ''
    errorTraceId.value = ''
    try {
      snapshot.value = await request(queuePath(), {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          ...(idempotencyKey ? { 'Idempotency-Key': idempotencyKey } : {}),
        },
        body: JSON.stringify(rhythm),
      })
      seedHistoryFromSnapshot(snapshot.value)
      return true
    } catch (e) {
      error.value = e?.message || '保存工作空间排队调度失败'
      errorTraceId.value = e?.traceId || ''
      return false
    } finally {
      saving.value = false
    }
  }

  /**
   * 加入/离开自动调度队列（复用 todos PATCH queued_auto_run），成功后刷新快照。
   * @param {string} taskId
   * @param {boolean} queued
   * @param {string} idempotencyKey
   */
  async function patchQueuedAutoRun(taskId, queued, idempotencyKey) {
    const tid = resolveId(tenantId)
    const wid = resolveId(workspaceId)
    const id = String(taskId || '').trim()
    if (!tid || !wid || !id) {
      error.value = '缺少租户、工作空间或任务 ID'
      errorTraceId.value = ''
      return false
    }
    error.value = ''
    errorTraceId.value = ''
    try {
      await request(`/api/tenant/${tid}/workspace/${wid}/todos/${id}/`, {
        method: 'PATCH',
        headers: {
          'Content-Type': 'application/json',
          ...(idempotencyKey ? { 'Idempotency-Key': idempotencyKey } : {}),
        },
        body: JSON.stringify({ queued_auto_run: !!queued }),
      })
      console.info('[useWorkspaceQueueSchedule] queued_auto_run_patched', {
        task_id: id,
        queued: !!queued,
      })
      await loadSnapshot()
      return true
    } catch (e) {
      error.value = e?.message || '更新排队任务失败'
      errorTraceId.value = e?.traceId || ''
      console.warn('[useWorkspaceQueueSchedule] queued_auto_run_patch_error', {
        task_id: id,
        queued: !!queued,
      })
      return false
    }
  }

  /**
   * 分页加载调度历史。首次快照已带 recent_history；本方法用于「加载更多」。
   */
  async function loadMoreHistory() {
    if (historyLoading.value) return false
    historyLoading.value = true
    historyError.value = ''
    historyErrorTraceId.value = ''
    try {
      const q = new URLSearchParams({ limit: '20' })
      if (historyCursor.value) q.set('cursor', historyCursor.value)
      const data = await request(`${historyPath()}?${q.toString()}`, { method: 'GET' })
      const items = Array.isArray(data?.items) ? data.items : []
      const seen = new Set(historyItems.value.map((r) => r.id))
      historyItems.value = historyItems.value.concat(items.filter((r) => r && !seen.has(r.id)))
      historyHasMore.value = !!data?.has_more
      historyCursor.value = data?.next_cursor || historyCursor.value
      return true
    } catch (e) {
      historyError.value = e?.message || '加载调度历史失败'
      historyErrorTraceId.value = e?.traceId || ''
      return false
    } finally {
      historyLoading.value = false
    }
  }

  return {
    snapshot,
    loading,
    saving,
    error,
    errorTraceId,
    historyItems,
    historyHasMore,
    historyLoading,
    historyError,
    historyErrorTraceId,
    loadSnapshot,
    saveSchedule,
    patchQueuedAutoRun,
    loadMoreHistory,
  }
}

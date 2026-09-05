import { computed, ref, watch } from 'vue'

/**
 * 排队自动执行：入队/出队 + 队列快照加载。
 * @param {{ task: object, tenantId: string, workspaceId: string }} props
 * @param {{ patchTask: Function, modalOpen: import('vue').Ref<boolean> }} ctx
 */
export function useQueuedAutoRunPanel(props, { patchTask, modalOpen }) {
  const queueSnapshot = ref(null)
  const queueLoading = ref(false)
  const queueError = ref('')
  const queueErrorTraceId = ref('')

  const queuedOn = computed(() => props.task?.queued_auto_run === true)

  const topTaskId = computed(() => {
    const top = props.task?.queued_top_task_id
    if (top) return String(top)
    if (props.task?.parent_task) return String(props.task.parent_task)
    return props.task?.id ? String(props.task.id) : ''
  })

  const aheadCount = computed(() => {
    const n = Number(props.task?.queued_ahead_count)
    return Number.isFinite(n) && n > 0 ? Math.floor(n) : 0
  })

  const statusText = computed(() => {
    if (!queuedOn.value) return ''
    if (aheadCount.value > 0) return `前方还有 ${aheadCount.value} 个任务在等待`
    const st = props.task?.queued_auto_run_status
    if (st === 'deferred') return props.task?.queued_deferred_reason || '等待时段'
    if (st === 'starting') return '调度启服中'
    if (st === 'queued') return '排队中，即将执行'
    return '已入队'
  })

  const queueMembers = computed(() => {
    const members = queueSnapshot.value?.members
    return Array.isArray(members) ? members : []
  })

  const queueMetaText = computed(() => {
    const snap = queueSnapshot.value
    if (!snap) return ''
    const used = Number(snap.queued_slots_used) || 0
    const max = Number(snap.max_queued_machines) || 0
    const msg = snap.window_message || ''
    return `占用 ${used}/${max}${msg ? ` · ${msg}` : ''}`
  })

  async function loadQueueSnapshot() {
    const tid = topTaskId.value
    if (!tid || !props.tenantId || !props.workspaceId) return
    const apiFetch = window.apiFetch
    if (typeof apiFetch !== 'function') return
    queueLoading.value = true
    queueError.value = ''
    queueErrorTraceId.value = ''
    try {
      const path = `/api/tenant/${props.tenantId}/workspace/${props.workspaceId}/todos/${tid}/queued-auto-run/`
      const resp = await apiFetch(path, {
        method: 'GET',
        credentials: 'include',
        headers: { Accept: 'application/json' },
      })
      const data = await resp.json().catch(() => ({}))
      if (!resp.ok) {
        const err = new Error(data?.error || data?.message || `HTTP ${resp.status}`)
        err.traceId = resp.headers?.get?.('X-Trace-Id') || data?.trace_id || ''
        throw err
      }
      queueSnapshot.value = data
    } catch (e) {
      queueError.value = e?.message || '加载队列失败'
      queueErrorTraceId.value = e?.traceId || ''
      queueSnapshot.value = null
    } finally {
      queueLoading.value = false
    }
  }

  watch(
    () => [modalOpen.value, topTaskId.value, props.task?.queued_auto_run],
    ([open]) => {
      if (open) loadQueueSnapshot()
    },
  )

  async function joinQueue() {
    const ok = await patchTask({ queued_auto_run: true })
    if (ok) await loadQueueSnapshot()
    return ok
  }

  async function leaveQueue() {
    const ok = await patchTask({ queued_auto_run: false })
    if (ok) await loadQueueSnapshot()
    return ok
  }

  function memberStatusText(m) {
    const st = m?.status
    const reason = m?.deferred_reason
    if (st === 'deferred') return reason || '等待时段'
    if (st === 'starting') return '调度启服中'
    if (st === 'queued') return '排队中'
    return '已入队'
  }

  return {
    queuedOn,
    topTaskId,
    aheadCount,
    statusText,
    queueMembers,
    queueMetaText,
    queueLoading,
    queueError,
    queueErrorTraceId,
    loadQueueSnapshot,
    joinQueue,
    leaveQueue,
    memberStatusText,
  }
}

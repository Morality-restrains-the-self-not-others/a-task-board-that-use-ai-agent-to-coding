import { getCurrentInstance, onUnmounted } from 'vue'
import { getApiUrl } from '../utils/config.js'
import { applyWorkPanelTaskStatusChanged } from '../utils/workPanelTaskStatusSseApply.js'

/** Work-panel SSE reconnect: max attempts */
export const WORK_PANEL_SSE_MAX_RECONNECT_ATTEMPTS = 10
/** Work-panel SSE reconnect: initial delay (ms) */
export const WORK_PANEL_SSE_INITIAL_RECONNECT_DELAY = 1000
/** Work-panel SSE reconnect: max delay (ms) */
export const WORK_PANEL_SSE_MAX_RECONNECT_DELAY = 30000

/**
 * Open work-panel task status SSE (taskSSE: /work-panel-events-sse/).
 * Auth via Cookie + gateway forward-auth; do not pass user_id in query.
 * On error: exponential backoff reconnect; calls onNeedResync after each close and on successful reopen.
 *
 * @param {{
 *   tenantId: string,
 *   workspaceId: string,
 *   onStatusChanged: (data: object) => void,
 *   onNeedResync?: () => void,
 *   onMachineRuntimeHint?: () => void,
 * }} opts
 */
export function openWorkPanelTaskStatusSse({
  tenantId,
  workspaceId,
  onStatusChanged,
  onNeedResync,
  onMachineRuntimeHint,
}) {
  const tid = String(tenantId || '').trim()
  const ws = String(workspaceId || '').trim()
  if (!tid || !ws || typeof EventSource === 'undefined') {
    return { close: () => {} }
  }

  const url = getApiUrl(`/api/sse/work-panel-events/tenant_id/${tid}/workspace_id/${ws}/`)
  let closed = false
  let es = null
  let reconnectAttempts = 0
  let reconnectTimer = null
  let openedOnce = false

  const clearReconnectTimer = () => {
    if (reconnectTimer != null) {
      clearTimeout(reconnectTimer)
      reconnectTimer = null
    }
  }

  const connect = () => {
    if (closed) return
    try {
      es?.close()
    } catch {
      /* ignore */
    }
    es = new EventSource(url, { withCredentials: true })

    es.onopen = () => {
      if (closed) return
      const wasReconnect = openedOnce && reconnectAttempts > 0
      reconnectAttempts = 0
      openedOnce = true
      if (wasReconnect) {
        onNeedResync?.()
        onMachineRuntimeHint?.()
      }
    }

    es.onmessage = (event) => {
      let data
      try {
        data = JSON.parse(event.data)
      } catch {
        return
      }
      if (!data || typeof data !== 'object') return
      if (data.type === 'heartbeat' || data.event_name === 'work_panel_sse_heartbeat') return
      if (data.event_name === 'work_panel_sse_connected') return
      // Cross-tab / Chrome plugin create&delete: full board resync (no local patch payload).
      if (data.event_name === 'task_created' || data.event_name === 'task_deleted') {
        onNeedResync?.()
        onMachineRuntimeHint?.()
        return
      }
      if (data.event_name === 'server_status_update' || data.event_name === 'container_heartbeat') {
        onMachineRuntimeHint?.()
        return
      }
      if (data.event_name !== 'task_status_changed') return
      onStatusChanged?.(data)
      onMachineRuntimeHint?.()
    }

    es.onerror = () => {
      if (closed) return
      try {
        es?.close()
      } catch {
        /* ignore */
      }
      es = null
      onNeedResync?.()
      onMachineRuntimeHint?.()
      if (reconnectAttempts >= WORK_PANEL_SSE_MAX_RECONNECT_ATTEMPTS) {
        return
      }
      const delay = Math.min(
        WORK_PANEL_SSE_INITIAL_RECONNECT_DELAY * 2 ** reconnectAttempts,
        WORK_PANEL_SSE_MAX_RECONNECT_DELAY,
      )
      reconnectAttempts += 1
      clearReconnectTimer()
      reconnectTimer = setTimeout(() => {
        reconnectTimer = null
        connect()
      }, delay)
    }
  }

  const close = () => {
    if (closed) return
    closed = true
    clearReconnectTimer()
    try {
      es?.close()
    } catch {
      /* ignore */
    }
    es = null
  }

  connect()

  if (getCurrentInstance()) {
    onUnmounted(close)
  }

  return { close }
}

/**
 * Bind SSE to a Vue todos ref: patch in place; resync via fetchTodos when unknown task.
 *
 * @param {{
 *   tenantId: import('vue').Ref<string> | (() => string),
 *   workspaceId: import('vue').Ref<string|null|undefined> | (() => string),
 *   todos: import('vue').Ref<Array<object>>,
 *   fetchTodos: () => Promise<void> | void,
 *   refreshMachineSummary?: () => Promise<void> | void,
 * }} opts
 */
export function useWorkPanelTaskStatusSse({ tenantId, workspaceId, todos, fetchTodos, refreshMachineSummary }) {
  let closer = { close: () => {} }

  const read = (v) => (typeof v === 'function' ? v() : v?.value != null ? v.value : v)

  const start = () => {
    closer.close()
    const tid = String(read(tenantId) || '').trim()
    const ws = String(read(workspaceId) || '').trim()
    if (!tid || !ws) return
    closer = openWorkPanelTaskStatusSse({
      tenantId: tid,
      workspaceId: ws,
      onStatusChanged: (data) => {
        const { todos: next, matched, changed } = applyWorkPanelTaskStatusChanged(
          todos.value,
          data,
        )
        if (changed) {
          todos.value = next
          return
        }
        if (!matched) {
          void fetchTodos?.()
        }
      },
      onNeedResync: () => {
        void fetchTodos?.()
      },
      onMachineRuntimeHint: () => {
        void refreshMachineSummary?.()
      },
    })
  }

  const stop = () => {
    closer.close()
    closer = { close: () => {} }
  }

  /** Call after initial/workspace todos fetch so SSE follows pull-then-push. */
  const notifyBoardReady = () => start()

  if (getCurrentInstance()) {
    onUnmounted(stop)
  }

  return { notifyBoardReady, stop }
}

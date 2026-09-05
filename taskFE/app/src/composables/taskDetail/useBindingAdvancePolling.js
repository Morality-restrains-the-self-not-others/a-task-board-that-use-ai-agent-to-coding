import { watch } from 'vue'
import { advanceCommentContainerBindings } from '../../utils/commentExecutionApi.js'
import { latestBindingAdvanced, latestContainerRunning } from './perContainerHeartbeatBus.js'

/**
 * SSE binding_advanced / 容器 running 信号驱动一次 advance。
 * 禁止 30s 定时器兜底（v84：前后端都不适合轮询）。
 */
export function useBindingAdvancePolling({
  bindings,
  ids,
  bindingStatusFor,
  refreshBindings,
  appendBindingStatusLog,
  appendBindingStageLog,
}) {
  function applyBindingAdvancedFromSse(eventData) {
    if (!eventData || typeof eventData !== 'object') return false
    const cid = String(eventData.comment_id || '').trim()
    if (!cid) return false
    const newStatus = String(eventData.status || eventData.new_status || '').trim()
    if (!newStatus) return false
    const cscId = String(eventData.csc_id || eventData.cscId || '').trim()
    const containerName = String(eventData.container_name || eventData.containerName || '').trim()
    const arr = bindings.value || []
    const oldStatus = (() => {
      for (let i = 0; i < arr.length; i++) {
        const b = arr[i]
        if (String(b?.comment_id || b?.commentId || '') === cid) {
          return String(b?.status || '')
        }
      }
      return ''
    })()
    let updated = false
    for (let i = 0; i < arr.length; i++) {
      const b = arr[i]
      const bCid = String(b?.comment_id || b?.commentId || '')
      if (bCid === cid) {
        const next = { ...b, status: newStatus }
        if (cscId) next.csc_id = cscId
        if (containerName) next.container_name = containerName
        arr[i] = next
        updated = true
        break
      }
    }
    if (updated) {
      bindings.value = [...arr]
      if (newStatus && newStatus !== oldStatus) {
        appendBindingStatusLog(cid, newStatus)
        if (newStatus === 'starting' && String(cscId || '').trim()) {
          appendBindingStageLog(cid, 'cscAllocated')
        }
      }
    }
    return updated
  }

  let _autoAdvancePending = false
  async function autoAdvanceOnContainerRunning(heartbeatData) {
    if (_autoAdvancePending) return
    const cid = String(heartbeatData?.comment_id || '').trim()
    if (!cid) return
    const { tid, wid, tk } = ids()
    if (!tid || !wid || !tk) return
    if (bindingStatusFor(cid) !== 'starting') return
    _autoAdvancePending = true
    try {
      await advanceCommentContainerBindings({ tenantId: tid, workspaceId: wid, taskId: tk })
      await refreshBindings()
    } catch (e) {
      console.warn('autoAdvanceOnContainerRunning failed', cid, e)
    } finally {
      _autoAdvancePending = false
    }
  }

  watch(latestBindingAdvanced, (data) => {
    if (data && typeof data === 'object') {
      applyBindingAdvancedFromSse(data)
    }
  })

  watch(latestContainerRunning, (data) => {
    if (data && typeof data === 'object') {
      autoAdvanceOnContainerRunning(data)
    }
  })

  return {
    applyBindingAdvancedFromSse,
    autoAdvanceOnContainerRunning,
    startPolling: () => {},
    stopPolling: () => {},
  }
}

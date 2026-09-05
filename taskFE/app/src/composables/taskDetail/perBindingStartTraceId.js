import { ref } from 'vue'
import {
  independentStartTraceId,
  resolveBindingPersistedStartTraceId,
} from '../../utils/bindingStartTraceIdFromLogs.js'

export function createPerBindingStartTraceIdApi(getTaskId) {
  const perBindingStartTraceId = ref({})

  function recordBindingStartTraceId(commentId, traceId) {
    const cid = String(commentId || '').trim()
    const taskId = typeof getTaskId === 'function' ? getTaskId() : ''
    const tid = independentStartTraceId(taskId, traceId)
    if (!cid || !tid) return
    perBindingStartTraceId.value = { ...(perBindingStartTraceId.value || {}), [cid]: tid }
  }

  function bindingStartTraceIdFor(commentId) {
    const cid = String(commentId || '').trim()
    return String((perBindingStartTraceId.value || {})[cid] || '').trim()
  }

  function hydrateFromBinding(binding) {
    const cid = String(binding?.comment_id || binding?.commentId || '').trim()
    if (!cid) return
    const taskId = typeof getTaskId === 'function' ? getTaskId() : ''
    const tid = resolveBindingPersistedStartTraceId(binding, taskId)
    if (tid) recordBindingStartTraceId(cid, tid)
  }

  return {
    perBindingStartTraceId,
    recordBindingStartTraceId,
    bindingStartTraceIdFor,
    hydrateFromBinding,
  }
}

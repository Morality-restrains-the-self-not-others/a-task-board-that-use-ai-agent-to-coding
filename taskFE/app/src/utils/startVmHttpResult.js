/**
 * 将 start-vm / start-vm-auto HTTP 响应映射为 TaskDetail 服务器状态更新。
 * API 200 + status success 仅表示「请求已受理」，不等于虚拟机已 Running。
 */

import { enrichStartupStatusUpdate } from './serverStartupErrorDisplay.js'

function pickStartVmTraceId(result) {
  const fromBody =
    (typeof result?.trace_id === 'string' && result.trace_id.trim()) ||
    (typeof result?.traceId === 'string' && result.traceId.trim()) ||
    ''
  const fromReq =
    typeof result?._traceId === 'string' && result._traceId.trim() ? result._traceId.trim() : ''
  return fromBody || fromReq
}

export function isAsyncStartVmSubmissionAck(result) {
  if (!result || typeof result !== 'object') {
    return false
  }
  const msg = typeof result.message === 'string' ? result.message : ''
  if (msg.includes('已提交') || msg.includes('请求已提交')) {
    return true
  }
  if (result.event_id != null && result.event_id !== '' && !result.vm_info) {
    return true
  }
  return false
}

export function buildStartVmAcceptedStatusUpdate(result) {
  const message =
    (typeof result?.message === 'string' && result.message.trim()) ||
    '启动请求已提交，正在创建资源…'
  const update = {
    status: 'processing',
    message,
    progress: 5,
    event_id: result?.event_id != null ? String(result.event_id) : undefined,
  }
  // 响应体 trace_id（服务端分配）优先于 apiFetch 注入的入站 _traceId
  const tid = pickStartVmTraceId(result)
  if (tid) update.trace_id = tid
  return update
}

export function buildStartVmErrorStatusUpdate(result, fallbackMessage) {
  const tid = pickStartVmTraceId(result)
  return enrichStartupStatusUpdate({
    status: 'error',
    message: (typeof result?.message === 'string' && result.message) || fallbackMessage,
    progress: 0,
    code: typeof result?.code === 'string' ? result.code : undefined,
    ...(tid ? { trace_id: tid } : {}),
  })
}

/** @returns {'accepted' | 'error' | 'ignored'} */
export function applyStartVmHttpResult(result, updateServerStatus) {
  if (typeof updateServerStatus !== 'function') {
    return 'ignored'
  }
  if (!result || typeof result !== 'object') {
    updateServerStatus(buildStartVmErrorStatusUpdate(null, '启动服务器失败'))
    return 'error'
  }
  if (result.status === 'error') {
    updateServerStatus(buildStartVmErrorStatusUpdate(result, '启动服务器失败'))
    return 'error'
  }
  if (result.status === 'success' && isAsyncStartVmSubmissionAck(result)) {
    updateServerStatus(buildStartVmAcceptedStatusUpdate(result))
    return 'accepted'
  }
  if (result.status === 'success' && result.vm_info) {
    updateServerStatus({
      status: 'success',
      message: typeof result.message === 'string' ? result.message : '服务器启动成功',
      progress: 100,
      vm_info: result.vm_info,
    })
    return 'accepted'
  }
  if (result.status === 'success') {
    updateServerStatus(buildStartVmAcceptedStatusUpdate(result))
    return 'accepted'
  }
  updateServerStatus(buildStartVmErrorStatusUpdate(result, '启动服务器失败'))
  return 'error'
}

/**
 * 容器 BOOTSTRAP_FAILED / COMPLETE SSE：失败文案与请求级 trace_id。
 * 不复用任务级 statusTraceId（启动链路常把 trace_id 写成 task_id）。
 */

export function traceIdFromSseStatusData(statusData) {
  if (typeof statusData?.trace_id === 'string' && statusData.trace_id.trim()) {
    return statusData.trace_id.trim()
  }
  if (typeof statusData?.traceId === 'string' && statusData.traceId.trim()) {
    return statusData.traceId.trim()
  }
  return ''
}

export function clearContainerBootstrapFailure(deps) {
  if (deps?.containerBootstrapFailureMessage) {
    deps.containerBootstrapFailureMessage.value = ''
  }
  if (deps?.containerBootstrapFailureTraceId) {
    deps.containerBootstrapFailureTraceId.value = ''
  }
}

export function applyContainerBootstrapFailure(statusData, deps) {
  const msg = typeof statusData?.message === 'string' ? statusData.message.trim() : ''
  if (deps?.containerBootstrapFailureMessage) {
    deps.containerBootstrapFailureMessage.value = msg || '引导克隆失败，请查看容器启动日志'
  }
  if (deps?.containerBootstrapFailureTraceId) {
    deps.containerBootstrapFailureTraceId.value = traceIdFromSseStatusData(statusData)
  }
}

/** @returns {boolean} 已消费该 SSE */
export function handleContainerBootstrapSse(statusData, deps) {
  if (statusData?.status === 'container_bootstrap_failed') {
    applyContainerBootstrapFailure(statusData, deps)
    return true
  }
  if (statusData?.status === 'container_bootstrap_complete') {
    clearContainerBootstrapFailure(deps)
    if (typeof deps?.refreshLayerGraphFromServer === 'function') {
      void deps.refreshLayerGraphFromServer(true)
    }
    if (typeof deps?.bumpProjectFileTreeRefresh === 'function') {
      deps.bumpProjectFileTreeRefresh()
    }
    return true
  }
  if (statusData?.status === 'container_bootstrap_progress') {
    const phase = String(statusData?.phase || '').trim()
    if (phase === 'clone_begin') {
      clearContainerBootstrapFailure(deps)
    }
    const msg = typeof statusData?.message === 'string' ? statusData.message.trim() : ''
    if (msg && typeof deps?.onBootstrapCloneLogUpdate === 'function') {
      const cur = String(deps.containerBootstrapCloneLogFull?.value || '').trim()
      if (!cur || /开始拉取任务详情|正在拉取任务详情/.test(cur)) {
        deps.onBootstrapCloneLogUpdate({ text: msg })
      }
    }
    return true
  }
  return false
}

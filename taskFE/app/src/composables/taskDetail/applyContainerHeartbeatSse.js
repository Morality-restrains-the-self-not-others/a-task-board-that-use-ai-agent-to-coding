/**
 * 容器心跳 SSE 应用与冷打开门禁。
 *
 * 冷打开时 isServerRunning 可能尚未经 runtime_hydrate 回填；若 DB 已登记
 * container_endpoint_registered，应接受心跳，否则状态会永久停在 idle（「等待连接」）。
 */

/**
 * @param {{
 *   containerHeartbeatPaused?: { value: boolean },
 *   serverRuntimeNotServing?: { value: boolean },
 *   isServerRunning?: { value: boolean },
 *   isServerStarting?: { value: boolean },
 *   containerEndpointRegistered?: { value: boolean },
 * }} deps
 */
export function isContainerHeartbeatServing(deps) {
  if (deps.containerHeartbeatPaused?.value === true) {
    return false
  }
  if (deps.serverRuntimeNotServing?.value === true) {
    return false
  }
  return (
    deps.isServerRunning?.value === true ||
    deps.isServerStarting?.value === true ||
    deps.containerEndpointRegistered?.value === true
  )
}

/**
 * @param {Record<string, unknown>} statusData
 * @param {{
 *   containerHeartbeatStatus: { value: string },
 *   containerHeartbeatAttempts: { value: number },
 *   containerHeartbeatLastSuccess: { value: Date | null },
 *   containerHeartbeatError: { value: string },
 *   applyContainerHeartbeatSeqFromSse: (d: Record<string, unknown>) => void,
 *   markContainerTransportOk: () => void,
 *   resetServerRuntimeLayerGraphGateCache: () => void,
 *   maybeRefreshLayerGraphOnContainerHeartbeatOk: () => void,
 * }} deps
 */
export function applyContainerHeartbeatSsePayload(statusData, deps) {
  const {
    containerHeartbeatStatus,
    containerHeartbeatAttempts,
    containerHeartbeatLastSuccess,
    containerHeartbeatError,
    applyContainerHeartbeatSeqFromSse,
    markContainerTransportOk,
    resetServerRuntimeLayerGraphGateCache,
    maybeRefreshLayerGraphOnContainerHeartbeatOk,
  } = deps

  containerHeartbeatStatus.value = 'connecting'
  containerHeartbeatAttempts.value++
  applyContainerHeartbeatSeqFromSse(statusData)
  const bidirectionalOk =
    statusData.bidirectional_ok === true ||
    (statusData.uplink_ok === true && statusData.downlink_ok === true)
  if (statusData.status === 'ok' && bidirectionalOk) {
    containerHeartbeatStatus.value = 'connected'
    containerHeartbeatLastSuccess.value = new Date()
    containerHeartbeatError.value = ''
    markContainerTransportOk()
    resetServerRuntimeLayerGraphGateCache()
    maybeRefreshLayerGraphOnContainerHeartbeatOk()
  } else if (statusData.uplink_ok === true && statusData.downlink_ok !== true) {
    containerHeartbeatStatus.value = 'connecting'
    containerHeartbeatError.value =
      statusData.message || '容器→SaaS 可达，SaaS→容器未确认（等待下行探测）'
  } else if (statusData.downlink_ok === true && statusData.uplink_ok !== true) {
    containerHeartbeatStatus.value = 'connecting'
    containerHeartbeatError.value = statusData.message || 'SaaS→容器可达，容器上行 seq 未确认'
  } else {
    containerHeartbeatStatus.value = 'disconnected'
    containerHeartbeatError.value = statusData.message || '双向心跳未就绪'
  }
}

/**
 * 早到心跳缓冲：门禁未放行时暂存最近一条，serving 后 flush。
 */
export function createContainerHeartbeatSseBuffer() {
  let pending = null

  return {
    stash(payload) {
      if (payload && typeof payload === 'object') {
        pending = payload
      }
    },
    take() {
      const p = pending
      pending = null
      return p
    },
    peek() {
      return pending
    },
  }
}

/**
 * 端点已登记且处于服务态时，将 idle 提升为 connecting，避免冷打开长期「等待连接」。
 * @param {{
 *   containerHeartbeatStatus: { value: string },
 *   containerHeartbeatPaused?: { value: boolean },
 *   containerEndpointRegistered?: { value: boolean },
 * }} deps
 * @param {(deps: object) => boolean} servingCheck
 */
export function promoteContainerHeartbeatFromIdle(deps, servingCheck = isContainerHeartbeatServing) {
  if (deps.containerHeartbeatPaused?.value === true) {
    return false
  }
  if (deps.containerHeartbeatStatus.value !== 'idle') {
    return false
  }
  const serverUp =
    deps.isServerRunning?.value === true || deps.isServerStarting?.value === true
  // VM 已 Running 但尚未 register-reachability：离开 idle，避免「等待连接」+ 空白 zTree
  if (
    serverUp &&
    !deps.containerEndpointRegistered?.value &&
    deps.serverRuntimeNotServing?.value !== true
  ) {
    deps.containerHeartbeatStatus.value = 'connecting'
    if (deps.containerHeartbeatError) {
      deps.containerHeartbeatError.value =
        '服务器已启动，等待容器镜像初始化并登记业务地址（register-reachability）…'
    }
    return true
  }
  if (!deps.containerEndpointRegistered?.value) {
    return false
  }
  if (!servingCheck(deps)) {
    return false
  }
  deps.containerHeartbeatStatus.value = 'connecting'
  return true
}

/**
 * @param {ReturnType<typeof createContainerHeartbeatSseBuffer>} buffer
 * @param {object} deps
 */
export function flushPendingContainerHeartbeat(buffer, deps) {
  if (!buffer || typeof buffer.take !== 'function') {
    return false
  }
  if (!isContainerHeartbeatServing(deps)) {
    return false
  }
  const pending = buffer.take()
  if (!pending) {
    return false
  }
  applyContainerHeartbeatSsePayload(pending, deps)
  return true
}

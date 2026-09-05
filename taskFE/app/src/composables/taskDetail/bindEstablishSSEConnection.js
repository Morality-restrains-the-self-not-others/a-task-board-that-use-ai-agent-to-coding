/**
 * Bind TaskDetail establishSSEConnection with shared deps + git_pr reply refresh hook.
 */
export function bindEstablishSSEConnection(d, {
  _establishSSEConnection,
  sseConnection, sseLive, sseReconnecting, sseReconnectAttempts, ssePlatformRestartHint,
  containerHeartbeatStatus, containerHeartbeatAttempts,
  containerHeartbeatLastSuccess, containerHeartbeatError,
  closeSSEConnection, scheduleSSEReconnect, markContainerTransportOk,
  resetServerRuntimeLayerGraphGateCache, maybeRefreshLayerGraphOnContainerHeartbeatOk,
  updateServerStatus, applyContainerHeartbeatSeqFromSse,
  effectiveTenantId, effectiveWorkspaceId, relayAccessCode, sse,
  containerHeartbeatPaused, isServerRunning, isServerStarting, serverRuntimeNotServing,
  containerEndpointRegistered, containerHeartbeatSseBuffer,
}) {
  d.establishSSEConnection = (taskId) => {
    _establishSSEConnection(taskId, {
      sseConnection, sseLive, sseReconnecting, sseReconnectAttempts, ssePlatformRestartHint,
      containerHeartbeatStatus, containerHeartbeatAttempts,
      containerHeartbeatLastSuccess, containerHeartbeatError,
      closeSSEConnection, scheduleSSEReconnect, markContainerTransportOk,
      resetServerRuntimeLayerGraphGateCache, maybeRefreshLayerGraphOnContainerHeartbeatOk,
      updateServerStatus, applyContainerHeartbeatSeqFromSse,
      effectiveTenantId, effectiveWorkspaceId,
      relayAccessCode,
      get currentSseTaskId() { return sse.currentSseTaskId },
      set currentSseTaskId(v) { sse.currentSseTaskId = v },
      get sseReconnectTimer() { return sse.sseReconnectTimer },
      set sseReconnectTimer(v) { sse.sseReconnectTimer = v },
      containerHeartbeatPaused,
      isServerRunning,
      isServerStarting,
      serverRuntimeNotServing,
      containerEndpointRegistered,
      containerHeartbeatSseBuffer,
      establishSSEConnectionSelf: d.establishSSEConnection,
      onTaskGitPrReplyCreated: () => {
        if (typeof d.fetchTaskDetail === 'function') void d.fetchTaskDetail()
      },
    })
  }
}

/**
 * OPT-20260719-035: Extracted heartbeat wiring from useTaskDetail.js.
 *
 * Wraps createContainerHeartbeatState with the dependency injection needed by
 * the task detail page. Callers pass refs and callback functions; this composable
 * wires them to the heartbeat state machine and returns the public API.
 *
 * Usage:
 *   const hb = useContainerHeartbeat({
 *     effectiveTenantId, effectiveWorkspaceId, effectiveTaskId,
 *     containerEndpointRegistered, containerPageUrl, containerVscodeUrl,
 *     containerPageLinkPendingReveal, layerGraphSnapshot, serverUrl,
 *     isServerRunning, isServerStarting, resetLayerGraphFetchBackoff,
 *     ensureServerRuntimeAllowsContainerLayerGraph,
 *     refreshLayerGraphFromServer,
 *     applyLayerGraphFromPayload,
 *     establishSSEConnection,
 *   })
 *   const { containerHeartbeatStatus, startContainerHeartbeat, ... } = hb
 */
import {
  createContainerHeartbeatState,
} from './taskDetailContainerHeartbeat.js'

export function useContainerHeartbeat(deps) {
  return createContainerHeartbeatState({
    effectiveTenantId: deps.effectiveTenantId,
    effectiveWorkspaceId: deps.effectiveWorkspaceId,
    effectiveTaskId: deps.effectiveTaskId,
    containerEndpointRegistered: deps.containerEndpointRegistered,
    containerPageUrl: deps.containerPageUrl,
    containerVscodeUrl: deps.containerVscodeUrl,
    containerPageLinkPendingReveal: deps.containerPageLinkPendingReveal,
    layerGraphSnapshot: deps.layerGraphSnapshot,
    serverUrl: deps.serverUrl,
    isServerRunning: deps.isServerRunning,
    isServerStarting: deps.isServerStarting,
    resetLayerGraphFetchBackoff: deps.resetLayerGraphFetchBackoff,
    ensureServerRuntimeAllowsContainerLayerGraph:
      (bypassCache) => deps.ensureServerRuntimeAllowsContainerLayerGraph(bypassCache),
    refreshLayerGraphFromServer:
      (force, opts) => deps.refreshLayerGraphFromServer(force, opts),
    applyLayerGraphFromPayload:
      (data, opts) => deps.applyLayerGraphFromPayload(data, opts),
    establishSSEConnection:
      (taskId) => deps.establishSSEConnection(taskId),
  })
}

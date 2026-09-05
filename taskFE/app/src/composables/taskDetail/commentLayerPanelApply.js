import { resolveZTreeLogTargets } from './taskDetailZTreeExecLogDerived.js'
import { mergeLayerChangesIntoPayload } from './taskDetailZTreeExecLogLiveOutput.js'
import { commentIdFromStatusEvent } from './runtimeStatusRefreshPolicy.js'

export function resolveLayerChangesOwnerCommentId(store, statusData, layerId) {
  const fromEvent = commentIdFromStatusEvent(statusData)
  if (fromEvent) return fromEvent
  return store && typeof store.commentIdOwningLayer === 'function'
    ? store.commentIdOwningLayer(layerId)
    : ''
}

export function applyNormalizedLayerChangesToStore(store, commentId, normalized) {
  if (!store || typeof store.patch !== 'function' || !commentId || !normalized) return
  const slot = store.get(commentId)
  const layerChangesByLayerId = {
    ...(slot.layerChangesByLayerId || {}),
    [normalized.layer_id]: normalized,
  }
  const targets = resolveZTreeLogTargets(slot.selectedNode, slot.snapshot?.jobs)
  const jobExecutionPayload = mergeLayerChangesIntoPayload(
    slot.jobExecutionPayload,
    normalized,
    targets.layerId,
  )
  store.patch(commentId, { layerChangesByLayerId, jobExecutionPayload })
}

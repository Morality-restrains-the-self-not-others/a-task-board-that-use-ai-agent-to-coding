import { snapshotHasRealWritableLayer } from '../../utils/layerZtreeBootstrapAnchor.js'
import { commentIdFromStatusEvent } from './runtimeStatusRefreshPolicy.js'
import { resolveContainerUiContextCommentId } from './resolveContainerUiContextCommentId.js'
import { clearContainerBootstrapFailure } from './containerBootstrapSse.js'

export function applyContainerLayerGraphPayload(targetRef, statusData) {
  if (!targetRef) return
  const nextLayers = Array.isArray(statusData.layers) ? statusData.layers : []
  if (nextLayers.length === 0) {
    const prev = targetRef.value
    if (prev && Array.isArray(prev.layers) && prev.layers.length > 0) {
      return
    }
  }
  targetRef.value = {
    layers: nextLayers,
    jobs: Array.isArray(statusData.jobs) ? statusData.jobs : [],
    layers_root: typeof statusData.layers_root === 'string' ? statusData.layers_root : '',
    bootstrap_layer_id:
      statusData.bootstrap_layer_id != null && statusData.bootstrap_layer_id !== ''
        ? String(statusData.bootstrap_layer_id)
        : '',
  }
}

/** @returns {boolean} 已消费该 SSE */
export function handleContainerLayerGraphSse(statusData, deps) {
  if (statusData?.status !== 'container_layer_graph') return false
  const { markContainerTransportOk, layerGraphSnapshot } = deps
  markContainerTransportOk()
  const cid = commentIdFromStatusEvent(statusData)
  const store = deps.layerPanelStore
  if (cid && store && typeof store.refsFor === 'function') {
    applyContainerLayerGraphPayload(store.refsFor(cid).layerGraphSnapshot, statusData)
  }
  const activeId = resolveContainerUiContextCommentId(deps)
  if (!cid || !activeId || cid === activeId || !store) {
    applyContainerLayerGraphPayload(layerGraphSnapshot, statusData)
  }
  const appliedLayers = Array.isArray(statusData.layers) ? statusData.layers : []
  // 仅引导空层锚点不算克隆成功，不得清掉 BOOTSTRAP_FAILED 文案。
  if (snapshotHasRealWritableLayer(appliedLayers)) {
    clearContainerBootstrapFailure(deps)
  }
  return true
}

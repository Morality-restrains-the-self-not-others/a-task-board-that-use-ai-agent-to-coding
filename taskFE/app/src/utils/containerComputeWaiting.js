/**
 * 容器转发尚未就绪 / 层目录尚未落地时的用户可见文案。
 * 这类状态会随 heartbeat、layer-graph-push、克隆完成自动恢复，不得画成阻断红错。
 */

export const CONTAINER_ENDPOINT_WAITING_HINT =
  '容器正在注册业务地址，就绪后将自动加载'

export const LAYER_DIR_PENDING_HINT =
  '层目录尚未就绪，克隆或叠层完成后将自动加载'

export function isLayerNotFoundDetail(detail) {
  const s = String(detail || '').trim().toLowerCase()
  return s === 'layer not found' || s.startsWith('layer not found:')
}

export function isContainerEndpointNotReadyDetail(detail) {
  const s = String(detail || '')
  if (!s) return false
  if (s === CONTAINER_ENDPOINT_WAITING_HINT) return true
  if (s.includes('容器 server_url 未就绪')) return true
  if (s.includes('尚未注册可用业务地址')) return true
  if (s.includes('exchange-refresh')) return true
  return false
}

export function isContainerComputeWaitingMessage(message) {
  const s = String(message || '')
  if (!s) return false
  if (s === LAYER_DIR_PENDING_HINT || s === CONTAINER_ENDPOINT_WAITING_HINT) return true
  return isLayerNotFoundDetail(s) || isContainerEndpointNotReadyDetail(s)
}

/**
 * @param {unknown} detail
 * @param {number} [httpStatus]
 * @returns {{ tone: 'waiting'|'error', message: string }}
 */
export function userFacingContainerComputeFailure(detail, httpStatus) {
  const d = typeof detail === 'string'
    ? detail
    : detail != null
      ? JSON.stringify(detail)
      : ''
  if (isLayerNotFoundDetail(d)) {
    return { tone: 'waiting', message: LAYER_DIR_PENDING_HINT }
  }
  const status = Number(httpStatus)
  if (status === 409 || isContainerEndpointNotReadyDetail(d)) {
    return { tone: 'waiting', message: CONTAINER_ENDPOINT_WAITING_HINT }
  }
  return { tone: 'error', message: d }
}

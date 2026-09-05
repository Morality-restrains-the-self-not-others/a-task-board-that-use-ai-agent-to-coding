/**
 * Pure functions: job/layer action handlers for TaskDetail layer graph.
 */
import { postContainerCompute, containerComputeFuncFirstUrl } from './containerComputeRequest.js'
import { jsonPostWithCommentId } from '../../utils/containerForwardCommentId.js'
import { resolveContainerUiContextCommentId } from './resolveContainerUiContextCommentId.js'
import { apiFetch } from '../../utils/apiUtils.js'
import { messageFromFailedResponse } from '../../utils/httpError.js'
import { showRequestError } from '../../utils/requestErrorDisplay.js'
import { isReleasedFlag } from './taskDetailReleasedFlag.js'

export async function callLayerGraphJobAction(jobId, action, failLabel, deps) {
  const {
    effectiveTenantId, effectiveWorkspaceId, effectiveTaskId,
    layerGraphBusyActionKey, markContainerTransportUnreachableIfForwardingFailed,
    markContainerTransportOk,
  } = deps
  const jid = String(jobId || '').trim()
  if (!jid) return false
  if (isReleasedFlag(deps.containerReleased) || isReleasedFlag(deps.serverRuntimeNotServing)) return false
  const tenantId = effectiveTenantId.value
  const workspaceId = effectiveWorkspaceId.value
  const taskId = effectiveTaskId.value
  const commentId = resolveContainerUiContextCommentId(deps)
  if (!tenantId || !workspaceId || !taskId || !commentId) return false
  const safeAction = String(action || '').trim()
  if (!safeAction) return false
  layerGraphBusyActionKey.value = `${safeAction}:job:${jid}`
  try {
    const response = await apiFetch(
      containerComputeFuncFirstUrl(tenantId, workspaceId, taskId, `container-job-${safeAction}`, commentId, `job_id=${encodeURIComponent(jid)}`),
      jsonPostWithCommentId({}, commentId),
    )
    if (!response.ok) {
      const msg = messageFromFailedResponse(response, `HTTP ${response.status}`)
      markContainerTransportUnreachableIfForwardingFailed(response.status, msg)
      showRequestError(`${failLabel}失败：${msg}`, response)
      return false
    }
    markContainerTransportOk()
    await deps.refreshZTreeExecutionLog()
    return true
  } catch (err) { console.error(`${safeAction}失败`, err); showRequestError('网络错误，请稍后重试', err); return false }
  finally { layerGraphBusyActionKey.value = '' }
}

export async function callLayerGraphLayerDelete(layerId, deps) {
  const {
    effectiveTenantId, effectiveWorkspaceId, effectiveTaskId,
    layerGraphBusyActionKey, markContainerTransportUnreachableIfForwardingFailed,
    markContainerTransportOk,
  } = deps
  const lid = String(layerId || '').trim()
  if (!lid) return false
  if (isReleasedFlag(deps.containerReleased) || isReleasedFlag(deps.serverRuntimeNotServing)) return false
  const tenantId = effectiveTenantId.value
  const workspaceId = effectiveWorkspaceId.value
  const taskId = effectiveTaskId.value
  const commentId = resolveContainerUiContextCommentId(deps)
  if (!tenantId || !workspaceId || !taskId || !commentId) return false
  layerGraphBusyActionKey.value = `delete:layer:${lid}`
  try {
    const response = await postContainerCompute(deps, 'container-layer-delete', { layer_id: lid })
    if (!response.ok) {
      const msg = messageFromFailedResponse(response, `HTTP ${response.status}`)
      markContainerTransportUnreachableIfForwardingFailed(response.status, msg)
      showRequestError(`删除失败：${msg}`, response)
      return false
    }
    markContainerTransportOk()
    await deps.refreshLayerGraphFromServer(true)
    await deps.refreshZTreeExecutionLog()
    return true
  } catch (err) { console.error('删除层级失败', err); showRequestError('网络错误，请稍后重试', err); return false }
  finally { layerGraphBusyActionKey.value = '' }
}

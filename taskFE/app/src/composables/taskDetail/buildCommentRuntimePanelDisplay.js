import { isServerRuntimeNoInstanceMessage, isServerRuntimeStartFailedMessage } from '../../utils/serverRuntimeAbsent.js'
import { buildServerRuntimeStatusDetails } from '../../utils/serverRuntimeStatusDetails.js'
import { buildJumpToServerUrl, serverJumpDefaultPort } from '../../utils/serverConfigJumpUrl.js'

const STATUS_LABEL = {
  Running: '运行中',
  Stopped: '已停止',
  Starting: '启动中',
  Stopping: '停止中',
  Rebooting: '重启中',
  Pending: '创建中',
  Initializing: '初始化中',
  Released: '已释放',
  Failed: '启动失败',
}

export function runtimeStatusDisplayText(status, message, extras = {}) {
  if (!status) {
    if (extras.startFailed || isServerRuntimeStartFailedMessage(message, extras.response)) return '启动失败'
    if (isServerRuntimeNoInstanceMessage(message, extras.response)) return '未创建'
    return '未知'
  }
  return STATUS_LABEL[status] || status
}

export function runtimePublicIpFromResponse(response) {
  const ips = response?.instance_attribute?.body?.PublicIpAddress?.IpAddress
  if (!Array.isArray(ips) || ips.length === 0) return ''
  return String(ips[0] || '').trim()
}

function rawJson(response) {
  if (!response) return ''
  try {
    return JSON.stringify(response, null, 2)
  } catch {
    return ''
  }
}

/**
 * 把单条评论的 runtime 快照编成 ServerConfigRuntimeStatusSection 展示字段。
 * @param {object} [slot]
 * @param {{ tick?: number, imageDisplay?: string, serverUrl?: string }} [opts]
 */
export function buildCommentRuntimePanelDisplay(slot, opts = {}) {
  const status = String(slot?.status || '')
  const message = String(slot?.message || '')
  const response = slot?.response || null
  const pip = runtimePublicIpFromResponse(response)
  return {
    isServerRuntimeStatusLoading: Boolean(slot?.loading),
    serverRuntimeStatus: status,
    serverRuntimeStatusDisplayText: runtimeStatusDisplayText(status, message, {
      startFailed: Boolean(response?.start_failed),
      response,
    }),
    showRuntimeActionButtons: status === 'Running',
    serverRuntimeStatusMessage: message,
    serverRuntimeStatusTraceId: String(slot?.traceId || ''),
    serverRuntimeStatusDetails: buildServerRuntimeStatusDetails(
      response,
      Number(opts.tick) || 0,
      String(opts.imageDisplay || ''),
      String(opts.commentCreatedAt || ''),
    ),
    serverRuntimeStatusRawJson: rawJson(response),
    serverJumpUrl: buildJumpToServerUrl(opts.serverUrl, pip, serverJumpDefaultPort),
  }
}

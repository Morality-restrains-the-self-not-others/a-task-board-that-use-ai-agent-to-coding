import { formatBindingLogClock } from './bindingServerStartupLogs.js'

/**
 * Build task-detail server status log lines from an SSE status_data payload.
 * Backend prefixes with [实例ID、容器名]; sdk_call detail lines reuse log_label.
 */

export function resolveStartupLogLabel(statusData) {
  if (statusData && typeof statusData.log_label === 'string' && statusData.log_label.trim()) {
    return statusData.log_label.trim()
  }
  return ''
}

export function pushServerStatusLogLines(statusLogs, statusData, displayMessage, now = new Date()) {
  if (!statusLogs || typeof statusLogs.push !== 'function') return
  const ts = formatBindingLogClock('', now)
  const msg =
    typeof displayMessage === 'string' && displayMessage
      ? displayMessage
      : statusData?.message
  if (msg != null && String(msg).trim() !== '') {
    statusLogs.push(`[${ts}] ${msg}`)
  }

  if (statusData?.status === 'sdk_call' && statusData.sdk_call) {
    const sdkCall = statusData.sdk_call
    const logLabel = resolveStartupLogLabel(statusData)
    const sdkPrefix = logLabel ? `${logLabel} ` : ''
    statusLogs.push(`[${ts}] ${sdkPrefix}SDK调用: ${sdkCall.method}`)
    if (sdkCall.request_ids) {
      statusLogs.push(`[${ts}] ${sdkPrefix}Request IDs: ${sdkCall.request_ids.join(', ')}`)
    }
  }
}

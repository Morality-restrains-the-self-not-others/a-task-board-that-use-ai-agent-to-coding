import {
  filterTaskStatusLogsForBinding,
  mergeBindingAndServerStartupLogs,
} from '../../utils/bindingServerStartupLogs.js'
import { mapBindingLifecycleToServerStatus } from './bindingLifecycleMaps.js'

function unwrap(v) {
  return v && typeof v === 'object' && 'value' in v ? v.value : v
}

export function isSoleActiveBinding(bindings, commentId) {
  const cid = String(commentId || '').trim()
  const live = (Array.isArray(bindings) ? bindings : []).filter((b) => {
    const st = String(b?.status || '')
    return st === 'starting' || st === 'running'
  })
  return live.length === 1 &&
    String(live[0]?.comment_id || live[0]?.commentId || '') === cid
}

/**
 * 组装评论级启动面板 props（生命周期 + 日志 + heartbeat）。
 * SSE 连接态由调用方覆盖。
 */
export const AWAITING_CONTAINER_STATUS_MESSAGE = '云主机已运行，等待容器登记可达地址'

export function assemblePerBindingServerStatusProps({
  commentId,
  running,
  starting,
  lifecycle,
  heartbeat,
  bindingLogs,
  bindings,
  taskStatusLogs,
  containerName,
  startTraceId,
  runtimeStatus = '',
  hasServerUrl = false,
  errorReason = '',
}) {
  const cid = String(commentId || '').trim()
  const hb = heartbeat && typeof heartbeat === 'object' ? heartbeat : {}
  const taskLogs = unwrap(taskStatusLogs)
  const serverLogs = filterTaskStatusLogsForBinding(
    Array.isArray(taskLogs) ? taskLogs : [],
    {
      commentId: cid,
      containerName,
      soleActiveBinding: isSoleActiveBinding(bindings, cid),
    },
  )
  const rs = String(runtimeStatus || '').trim()
  const awaitingContainer = Boolean(starting) && !running && /^running$/i.test(rs) && !hasServerUrl
  const reason = String(errorReason || '').trim()
  let statusMessage = lifecycle
  if (awaitingContainer) {
    statusMessage = AWAITING_CONTAINER_STATUS_MESSAGE
  } else if (lifecycle === '启动失败' && reason) {
    statusMessage = reason
  }
  return {
    serverStatus: mapBindingLifecycleToServerStatus(lifecycle, running, starting),
    isServerRunning: running,
    isServerStarting: starting,
    runtimeStatus: rs,
    statusProgress: running ? 100 : (starting ? 60 : 0),
    statusMessage,
    statusLogs: mergeBindingAndServerStartupLogs(
      Array.isArray(bindingLogs) ? bindingLogs : [],
      serverLogs,
    ),
    startTraceId: startTraceId || '',
    heartbeatStatus: hb.status || '',
    heartbeatSeqInfo: hb.seqInfo || null,
    heartbeatError: hb.error || '',
    heartbeatLastSuccess: hb.lastSuccess || null,
    heartbeatAttempts: Number(hb.attempts) || (Array.isArray(hb.logLines) ? hb.logLines.length : 0),
    heartbeatLogLines: Array.isArray(hb.logLines) ? hb.logLines : [],
  }
}

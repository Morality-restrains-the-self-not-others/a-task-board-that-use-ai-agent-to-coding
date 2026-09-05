import { ref } from 'vue'
import { latestPerContainerHeartbeat } from './perContainerHeartbeatBus.js'

/** SSE 断线重连：最大重连次数 */
export const SSE_MAX_RECONNECT_ATTEMPTS = 10
/** SSE 断线重连：初始重连延迟（毫秒） */
export const SSE_INITIAL_RECONNECT_DELAY = 1000
/** SSE 断线重连：最大重连延迟（毫秒） */
export const SSE_MAX_RECONNECT_DELAY = 30000

/**
 * Factory: SSE connection ref + exponential backoff reconnect helpers.
 * @param {{ sseLive: import('vue').Ref<boolean>, serverStartupStatusPoll?: { stop?: () => void }, establishSSEConnection: (taskId: string) => void }} deps
 */
export function createSseReconnectState(deps) {
  const sseConnection = ref(null)
  let currentSseTaskId = null
  /** SSE 断线重连：重连次数 */
  const sseReconnectAttempts = ref(0)
  /** SSE 断线重连：重连定时器 */
  let sseReconnectTimer = null
  const sseReconnecting = ref(false)
  /** nginx HTML 502/5xx 探测确认后提示「状态推送服务暂不可用」（与云服务器启停无关） */
  const ssePlatformRestartHint = ref(false)

  const calculateReconnectDelay = () => {
    const delay = SSE_INITIAL_RECONNECT_DELAY * Math.pow(2, sseReconnectAttempts.value)
    return Math.min(delay, SSE_MAX_RECONNECT_DELAY)
  }

  const closeSSEConnection = () => {
    deps.sseLive.value = false
    sseReconnecting.value = false
    ssePlatformRestartHint.value = false
    deps.serverStartupStatusPoll?.stop?.()
    if (sseReconnectTimer) {
      clearTimeout(sseReconnectTimer)
      sseReconnectTimer = null
    }
    sseReconnectAttempts.value = 0
    if (sseConnection.value) {
      sseConnection.value.close()
      sseConnection.value = null
    }
    currentSseTaskId = null
    // OPT-20260724-024: 清空模块级 per-container heartbeat 总线，
    // 确保 onBeforeUnmount + 任务切换两条路径都清理脏数据。
    latestPerContainerHeartbeat.value = null
  }

  const scheduleSSEReconnect = (taskId) => {
    if (sseReconnectTimer) {
      clearTimeout(sseReconnectTimer)
      sseReconnectTimer = null
    }
    if (sseReconnectAttempts.value >= SSE_MAX_RECONNECT_ATTEMPTS) {
      console.error('SSE 重连次数已达上限，停止重连')
      sseReconnecting.value = false
      return
    }
    sseReconnecting.value = true
    const delay = calculateReconnectDelay()
    sseReconnectAttempts.value++
    console.log(`SSE 断线重连：第 ${sseReconnectAttempts.value} 次，延迟 ${delay}ms`)
    sseReconnectTimer = setTimeout(() => {
      sseReconnectTimer = null
      if (currentSseTaskId === String(taskId)) {
        deps.establishSSEConnection(taskId)
      }
    }, delay)
  }

  const handleSSEManualReconnect = () => {
    const taskId = deps.effectiveTaskId?.value
    if (!taskId) return
    sseReconnectAttempts.value = 0
    sseReconnecting.value = false
    if (sseReconnectTimer) {
      clearTimeout(sseReconnectTimer)
      sseReconnectTimer = null
    }
    if (sseConnection.value) {
      sseConnection.value.close()
      sseConnection.value = null
    }
    console.log('手动触发 SSE 重连')
    deps.establishSSEConnection(taskId)
  }

  return {
    sseConnection,
    sseReconnectAttempts,
    sseReconnecting,
    ssePlatformRestartHint,
    closeSSEConnection,
    scheduleSSEReconnect,
    handleSSEManualReconnect,
    get currentSseTaskId() { return currentSseTaskId },
    set currentSseTaskId(v) { currentSseTaskId = v },
    get sseReconnectTimer() { return sseReconnectTimer },
    set sseReconnectTimer(v) { sseReconnectTimer = v },
  }
}

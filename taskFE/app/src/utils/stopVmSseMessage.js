/**
 * SSE success copy for stop-vm / CLOUD_SERVER_STOPPED.
 * Real ECS: 「停止虚拟机成功」; Mock: 「Mock 实例已停止」.
 */

export function isCloudServerStopSuccessMessage(message) {
  if (typeof message !== 'string' || !message) {
    return false
  }
  return message.includes('停止虚拟机成功') || message.includes('Mock 实例已停止')
}

/** 停机进度/成功行（含无 comment 标签的任务级 SSE），不得吸入启动日志。 */
export function isCloudServerStopLogLine(message) {
  if (typeof message !== 'string' || !message) {
    return false
  }
  if (isCloudServerStopSuccessMessage(message)) {
    return true
  }
  return /正在初始化服务器停止|准备停止|调用\S*API停止服务器/.test(message)
}

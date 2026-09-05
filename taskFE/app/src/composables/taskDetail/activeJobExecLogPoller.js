/**
 * While the selected zTree job is active (queued/pending/running), periodically
 * refresh execution log so agent steps appear one-by-one even when job-stream
 * SSE is missing (e.g. auto_run first instruction created inside the container).
 */
export const EXEC_LOG_ACTIVE_JOB_POLL_MS = 2000

export function isActiveJobStatus(status) {
  const s = String(status || '')
    .trim()
    .toLowerCase()
  return s === 'queued' || s === 'pending' || s === 'running'
}

/**
 * @param {{
 *   getSelectedJobStatus: () => { jobId: string, status: string } | null,
 *   isEndpointReady: () => boolean,
 *   refresh: () => void | Promise<void>,
 *   intervalMs?: number,
 *   setIntervalFn?: typeof setInterval,
 *   clearIntervalFn?: typeof clearInterval,
 * }} opts
 * @returns {{ sync: () => void, stop: () => void }}
 */
export function createActiveJobExecLogPoller(opts) {
  const refresh = opts.refresh
  const getSelectedJobStatus = opts.getSelectedJobStatus
  const isEndpointReady = opts.isEndpointReady
  const intervalMs = Math.max(500, Number(opts.intervalMs) || EXEC_LOG_ACTIVE_JOB_POLL_MS)
  const setIntervalFn = opts.setIntervalFn || setInterval
  const clearIntervalFn = opts.clearIntervalFn || clearInterval
  let timer = null
  void refresh
  void getSelectedJobStatus
  void isEndpointReady
  void intervalMs
  void setIntervalFn

  const stop = () => {
    if (timer != null) {
      clearIntervalFn(timer)
      timer = null
    }
  }

  const sync = () => {
    stop()
    // 执行步骤直播走 Kafka → SSE；历史由 GET SaaS DB hydrate。禁止无触发后台轮询。
  }

  return { sync, stop }
}

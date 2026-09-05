/** 与 layerLiveOutputDisplay 展示上限对齐，避免 map 内字符串无限增长 */
export const LAYER_JOB_LIVE_OUTPUT_MAX_CHARS = 100000

/** job-stream 终态：gateway 发 done，onlineServiceJS events 发 completed/failed/interrupted */
export const CONTAINER_JOB_STREAM_TERMINAL_PHASES = new Set([
  'done',
  'error',
  'completed',
  'failed',
  'interrupted',
])

export function appendCappedLiveOutput(prev, chunk) {
  const next = `${prev || ''}${chunk || ''}`
  if (next.length <= LAYER_JOB_LIVE_OUTPUT_MAX_CHARS) return next
  return next.slice(-LAYER_JOB_LIVE_OUTPUT_MAX_CHARS)
}

/**
 * 从 job-stream phase / job_status 解析可写层展示用终态；error（轮询失败）不臆造 job.status。
 * @param {string} phase
 * @param {object} statusData
 * @returns {string} completed|failed|interrupted|''
 */
export function resolveContainerJobStreamTerminalStatus(phase, statusData) {
  const p = String(phase || '').trim().toLowerCase()
  if (p === 'completed' || p === 'failed' || p === 'interrupted') return p
  if (p === 'done') {
    const js = String(statusData?.job_status || '').trim().toLowerCase()
    if (js === 'completed' || js === 'failed' || js === 'interrupted') return js
    return 'completed'
  }
  return ''
}

/**
 * 乐观更新层图 jobs/layers，避免 GET job 超时导致 zTree 长期停在 running。
 * @param {{ value: object|null }} snapshotRef
 * @param {string} jobId
 * @param {string} status
 */
export function patchLayerGraphJobStatus(snapshotRef, jobId, status) {
  const jid = String(jobId || '').trim()
  const st = String(status || '').trim().toLowerCase()
  if (!jid || !st || !snapshotRef) return
  const snap = snapshotRef.value
  if (!snap || typeof snap !== 'object') return
  const prevJobs = Array.isArray(snap.jobs) ? snap.jobs : []
  let layerId = ''
  const jobs = prevJobs.map((j) => {
    if (!j || String(j.id) !== jid) return j
    layerId = j.layer_id != null ? String(j.layer_id) : ''
    return { ...j, status: st }
  })
  let layers = Array.isArray(snap.layers) ? snap.layers : []
  if (layerId) {
    const mind = st === 'running' || st === 'pending' ? 'running' : 'idle_done'
    layers = layers.map((L) => {
      if (!L || String(L.layer_id) !== layerId) return L
      return { ...L, job_status: st, mind_state: mind, job_id: L.job_id || jid }
    })
  }
  snapshotRef.value = { ...snap, jobs, layers }
}

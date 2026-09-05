/**
 * 从评论 binding 持久化启动日志中提取 start TraceId。
 * 后端 logServerSchedulingToBindingBestEffort 在文案末尾追加 `trace_id=<id>`。
 *
 * @param {Array<{ message?: string }|string>|null|undefined} logs
 * @returns {string}
 */
export function extractStartTraceIdFromBindingLogs(logs) {
  if (!Array.isArray(logs) || !logs.length) return ''
  // 自后向前：取最近一次调度失败/进度上的 trace
  for (let i = logs.length - 1; i >= 0; i -= 1) {
    const raw = logs[i]
    const msg =
      typeof raw === 'string'
        ? raw
        : String(raw?.message || '').trim()
    if (!msg) continue
    const m = msg.match(/\btrace_id=([A-Za-z0-9._:-]{2,128})\b/)
    if (m && m[1]) return m[1].trim()
  }
  return ''
}

/** 等于当前 taskId 的值不是独立启机 TraceId，禁止展示/回填。 */
export function isTaskIdUsedAsStartTraceId(taskId, traceId) {
  const t = String(taskId || '').trim()
  const id = String(traceId || '').trim()
  if (!t || !id) return false
  return id === t
}

export function independentStartTraceId(taskId, traceId) {
  const id = String(traceId || '').trim()
  if (!id || isTaskIdUsedAsStartTraceId(taskId, id)) return ''
  return id
}

/** 列值优先于日志后缀；丢弃等于 taskId 的值后再回退日志。 */
export function resolveBindingPersistedStartTraceId(binding, taskId) {
  const fromCol = independentStartTraceId(
    taskId,
    binding?.start_trace_id || binding?.startTraceId,
  )
  if (fromCol) return fromCol
  return independentStartTraceId(taskId, extractStartTraceIdFromBindingLogs(binding?.logs))
}

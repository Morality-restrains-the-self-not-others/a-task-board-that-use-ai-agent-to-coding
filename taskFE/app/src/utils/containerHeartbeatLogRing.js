/** 心跳探测日志环形缓冲默认容量（超出丢弃最旧） */
export const CONTAINER_HEARTBEAT_LOG_RING_MAX = 24

/**
 * 将新日志行追加到环形缓冲，保留最近 max 条。
 * @param {unknown[]} existing
 * @param {unknown[]} lines
 * @param {number} [max]
 * @returns {string[]}
 */
export function appendHeartbeatLogRing(existing, lines, max = CONTAINER_HEARTBEAT_LOG_RING_MAX) {
  const cap = Number.isFinite(max) && max > 0 ? Math.floor(max) : CONTAINER_HEARTBEAT_LOG_RING_MAX
  const base = Array.isArray(existing) ? existing.map((l) => String(l)) : []
  if (!Array.isArray(lines) || lines.length === 0) {
    return base.length > cap ? base.slice(base.length - cap) : base
  }
  const next = [...base, ...lines.map((l) => String(l))]
  return next.length > cap ? next.slice(next.length - cap) : next
}

/**
 * 压缩结构化心跳 JSON，便于在窄面板阅读。
 * @param {unknown} line
 * @returns {string}
 */
export function formatHeartbeatLogLine(line) {
  const text = String(line || '')
  try {
    const trimmed = text.trim()
    if (trimmed.startsWith('{') && trimmed.endsWith('}')) {
      const obj = JSON.parse(trimmed)
      if (obj && typeof obj === 'object') {
        const ts = obj.ts ? String(obj.ts).slice(11, 19) : ''
        const method = obj.method || ''
        const path = obj.path || obj.url || ''
        const status = obj.status != null ? obj.status : ''
        const ms = obj.duration_ms != null ? `${obj.duration_ms}ms` : ''
        const parts = [ts, method, path, status, ms].filter(Boolean)
        if (parts.length >= 2) {
          return parts.join(' ')
        }
      }
    }
  } catch {
    // keep raw
  }
  return text.length > 220 ? `${text.slice(0, 220)}…` : text
}

/**
 * 折叠态摘要：条数 + 最新一行预览。
 * @param {unknown[]} lines
 * @returns {{ count: number, latestPreview: string }}
 */
export function summarizeHeartbeatLogRing(lines) {
  const list = Array.isArray(lines) ? lines : []
  const count = list.length
  if (count === 0) {
    return { count: 0, latestPreview: '' }
  }
  const latestPreview = formatHeartbeatLogLine(list[count - 1])
  return { count, latestPreview }
}

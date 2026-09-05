/**
 * 工作面板 / 创建任务：可选租户接口的失败日志与 JSON 解析（不抛错，便于降级展示）。
 */

/**
 * @param {Response} response
 * @returns {Promise<object|array|null>}
 */
export async function parseJsonSafe(response) {
  try {
    return await response.json()
  } catch {
    return null
  }
}

/**
 * @param {string} context 简短说明，如「任务列表」
 * @param {Response} [response]
 */
export function warnOptionalApiFailure(context, response) {
  const status = response?.status ?? '?'
  const url = typeof response?.url === 'string' ? response.url : ''
  const detail = response?._errorData
  const head = `[WorkPanel:${context}] HTTP ${status}${url ? ` ${url}` : ''}`
  if (detail && typeof detail === 'object' && Object.keys(detail).length > 0) {
    console.warn(head, detail)
  } else {
    console.warn(head)
  }
}

/**
 * 网络失败告警去抖窗口（OPT-20260809-030）。
 * 15s 轮询连续失败时，同一 (context, 错误摘要) 在窗口内只告警一次，
 * 避免控制台/指标被同类信号刷屏；窗口过后仍失败则再次告警（状态未恢复的降频心跳）。
 */
const NETWORK_FAILURE_WARN_DEBOUNCE_MS = 30_000

/** @type {Map<string, number>} 模块级去抖表：key → 上次告警时间戳（Date.now） */
const networkFailureWarnAt = new Map()

function networkFailureWarnKey(context, message) {
  return `${context}|${String(message || '').trim().toLowerCase()}`
}

/**
 * @param {string} context
 * @param {unknown} error
 */
export function warnNetworkFailure(context, error) {
  const msg = error instanceof Error ? error.message : String(error ?? '')
  const key = networkFailureWarnKey(context, msg)
  const now = Date.now()
  const lastWarn = networkFailureWarnAt.get(key)
  // 同信号窗口内去抖；不同信号（异错误/异端点）各自独立、即时透传
  if (typeof lastWarn === 'number' && now - lastWarn < NETWORK_FAILURE_WARN_DEBOUNCE_MS) {
    return
  }
  // 惰性清理窗口外旧条目，避免 Map 无限增长
  if (networkFailureWarnAt.size > 128) {
    for (const [k, ts] of networkFailureWarnAt) {
      if (now - ts >= NETWORK_FAILURE_WARN_DEBOUNCE_MS) networkFailureWarnAt.delete(k)
    }
  }
  networkFailureWarnAt.set(key, now)
  console.warn(`[WorkPanel:${context}] ${msg || '网络异常'}`)
}

/**
 * DRF 分页 `{ results: [] }` 与裸数组兼容。
 * @param {unknown} data
 * @returns {array}
 */
export function normalizeListPayload(data) {
  if (Array.isArray(data)) return data
  if (data && Array.isArray(data.results)) return data.results
  return []
}

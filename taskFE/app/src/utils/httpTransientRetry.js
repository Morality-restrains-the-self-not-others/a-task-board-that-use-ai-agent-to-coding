/**
 * 只读 GET 在网关/上游短不可用（502/503/504 或网络断开）时的有限退避重试。
 * 用户已触发请求（打开 Fork 弹窗 / 切智能体资源），不是无触发轮询。
 */
import { isTransientHttpStatus } from './httpError.js'

const inVitest = Boolean(typeof process !== 'undefined' && process.env && process.env.VITEST)

export const transientHttpRetry = {
  extraAttempts: 3,
  delaysMs: inVitest ? [0, 0, 0] : [800, 1600, 3200],
  sleep(ms) {
    if (!ms) return Promise.resolve()
    return new Promise((resolve) => setTimeout(resolve, ms))
  },
}

function delayForAttempt(attemptIndex) {
  const delays = transientHttpRetry.delaysMs || []
  if (!delays.length) return 0
  return Number(delays[Math.min(attemptIndex, delays.length - 1)]) || 0
}

function isRetryableFailure(response, thrown) {
  if (thrown) return true
  if (!response || response.ok) return false
  return isTransientHttpStatus(response.status)
}

/**
 * @template T
 * @param {() => Promise<T>} fn
 * @param {{ url?: string, onRetry?: (info: object) => void }} [opts]
 * @returns {Promise<T>}
 */
export async function retryTransientHttp(fn, opts = {}) {
  const extra = Number(transientHttpRetry.extraAttempts)
  const attempts = 1 + (Number.isFinite(extra) && extra > 0 ? extra : 0)
  let lastThrown
  let lastResponse
  for (let i = 0; i < attempts; i += 1) {
    try {
      lastResponse = await fn()
      lastThrown = undefined
    } catch (error) {
      lastThrown = error
      lastResponse = undefined
    }
    const retryable = isRetryableFailure(lastResponse, lastThrown)
    const hasMore = i < attempts - 1
    if (!retryable || !hasMore) {
      if (lastThrown) throw lastThrown
      return lastResponse
    }
    const info = {
      attempt: i + 1,
      attempts,
      status: lastResponse?.status,
      url: opts.url,
      traceId: lastResponse?.traceId || lastThrown?.traceId,
    }
    if (typeof opts.onRetry === 'function') opts.onRetry(info)
    else {
      console.warn('[httpTransientRetry] event=transient_retry', info)
    }
    await transientHttpRetry.sleep(delayForAttempt(i))
  }
  if (lastThrown) throw lastThrown
  return lastResponse
}

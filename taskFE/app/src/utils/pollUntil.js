const defaultSleep = (ms) => new Promise((r) => setTimeout(r, ms))

/**
 * 通用轮询：在时限内反复执行 tick，直到完成 / 中止 / 超时。
 *
 * 仅供脚本/单测；页面业务禁止用本工具做长轮询（应走 SSE 或单次拉取/手动查询）。
 *
 * @param {object} options
 * @param {() => Promise<'continue'|'done'|'error'>} options.tick
 *   - continue：继续下一轮
 *   - done：成功结束
 *   - error：业务失败结束（不再重试）
 * @param {number} [options.maxMs=120000] 可用 `Infinity` 表示仅依赖 shouldAbort / tick 结束
 * @param {number} [options.intervalMs=2000]
 * @param {() => boolean} [options.shouldAbort] 返回 true 时立即停止
 * @param {boolean} [options.delayFirst=false] true 时首 tick 前先等待 intervalMs（对齐 setInterval）
 * @param {(ms: number) => Promise<void>} [options.sleep] 可注入以便单测
 * @returns {Promise<'done'|'error'|'timeout'|'aborted'>}
 */
export async function pollUntil({
  tick,
  maxMs = 120000,
  intervalMs = 2000,
  shouldAbort,
  delayFirst = false,
  sleep = defaultSleep
} = {}) {
  if (typeof tick !== 'function') {
    throw new Error('pollUntil requires a tick function')
  }
  const t0 = Date.now()
  let first = true
  while (Date.now() - t0 < maxMs) {
    if (shouldAbort?.()) return 'aborted'
    if (delayFirst && first) {
      first = false
      await sleep(intervalMs)
      if (shouldAbort?.()) return 'aborted'
    } else {
      first = false
    }
    const result = await tick()
    if (result === 'done' || result === 'error') return result
    if (shouldAbort?.()) return 'aborted'
    await sleep(intervalMs)
  }
  if (shouldAbort?.()) return 'aborted'
  return 'timeout'
}

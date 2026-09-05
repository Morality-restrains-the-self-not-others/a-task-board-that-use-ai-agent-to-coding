// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] usePaymentPoll.test.js requires vitest runtime')
} else {
const { describe, expect, it, vi, beforeEach, afterEach } = await import('vitest')

const { usePaymentPoll } = await import('./usePaymentPoll.js')

/**
 * OPT-20260808-015: OrderCreate 支付状态轮询链重构（单链 + 可取消）。
 * 原实现 fire-and-forget 60×2s：关闭二维码弹窗 / 离开页面后轮询链仍持续
 * 每 2s 打一次 GET，最多 2 分钟 —— 浏览器网络/CPU 挂起的源头之一。
 * 本 composable 承诺：
 *  - 单链：新轮询开始前取消旧轮询（重复点「确认支付」不产生并发链）
 *  - 弹窗关闭（isActive=false）即停止
 *  - 组件卸载（stopPoll）即停止
 *  - 订单 id 在 start 时捕获，轮询期间订单变化立即停止（不回写旧单状态）
 *  - maxAttempts 硬上限兜底
 *  - setTimeout 链式调度：慢网下不会堆积并发 GET
 */
function makeHarness(overrides = {}) {
  const state = {
    modalOpen: true,
    orderId: 'ORD-1',
    ...overrides.state,
  }
  const calls = []
  const opts = {
    getOrderId: () => state.orderId,
    isActive: (orderId) => state.modalOpen && orderId === state.orderId,
    onPoll: vi.fn(async (orderId) => {
      calls.push(orderId)
      return false
    }),
    intervalMs: 1000,
    maxAttempts: 60,
    ...overrides.opts,
  }
  const poll = usePaymentPoll(opts)
  return { poll, opts, calls, state }
}

describe('usePaymentPoll', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('polls every intervalMs until onPoll reports paid, then stops', async () => {
    const { poll, opts, calls } = makeHarness({
      opts: {
        onPoll: vi.fn(async (orderId) => {
          calls.push(orderId)
          return calls.length >= 3
        }),
      },
    })
    poll.startPoll()
    await vi.advanceTimersByTimeAsync(1000)
    await vi.advanceTimersByTimeAsync(1000)
    await vi.advanceTimersByTimeAsync(1000)
    expect(opts.onPoll).toHaveBeenCalledTimes(3)
    // 已支付 → 不再调度下一轮
    await vi.advanceTimersByTimeAsync(10_000)
    expect(opts.onPoll).toHaveBeenCalledTimes(3)
    expect(calls.every((id) => id === 'ORD-1')).toBe(true)
  })

  it('stops polling as soon as isActive turns false (QR modal closed)', async () => {
    const { poll, opts, state } = makeHarness()
    poll.startPoll()
    await vi.advanceTimersByTimeAsync(1000)
    expect(opts.onPoll).toHaveBeenCalledTimes(1)
    state.modalOpen = false // 用户点击「关闭」→ 弹窗关闭
    await vi.advanceTimersByTimeAsync(20_000)
    expect(opts.onPoll).toHaveBeenCalledTimes(1)
  })

  it('single chain: starting a new poll cancels the previous chain', async () => {
    const { poll, opts, state, calls } = makeHarness()
    poll.startPoll() // 订单 ORD-1 的轮询链
    await vi.advanceTimersByTimeAsync(1000)
    state.orderId = 'ORD-2' // 用户重新创建订单并再次支付
    poll.startPoll() // 新链（单链：先取消旧链）
    await vi.advanceTimersByTimeAsync(5000)
    // 旧链 ORD-1 不再被轮询，只有新链 ORD-2 的 tick
    expect(calls.filter((id) => id === 'ORD-1').length).toBeLessThanOrEqual(1)
    expect(calls.filter((id) => id === 'ORD-2').length).toBeGreaterThan(0)
    expect(opts.onPoll).toHaveBeenCalled()
  })

  it('order change mid-poll stops the chain (stale order guard)', async () => {
    const { poll, opts, state } = makeHarness()
    poll.startPoll()
    await vi.advanceTimersByTimeAsync(1000)
    state.orderId = 'ORD-NEW' // 轮询期间订单变化 → isActive(orderId) 变 false
    await vi.advanceTimersByTimeAsync(20_000)
    expect(opts.onPoll).toHaveBeenCalledTimes(1)
  })

  it('caps polling at maxAttempts', async () => {
    const { poll, opts } = makeHarness({ opts: { maxAttempts: 3 } })
    poll.startPoll()
    await vi.advanceTimersByTimeAsync(10_000)
    expect(opts.onPoll).toHaveBeenCalledTimes(3)
  })

  it('continues polling across transient network errors', async () => {
    const { poll, opts } = makeHarness({
      opts: {
        onPoll: vi
          .fn()
          .mockRejectedValueOnce(new Error('network down'))
          .mockResolvedValue(false),
      },
    })
    poll.startPoll()
    await vi.advanceTimersByTimeAsync(1000)
    await vi.advanceTimersByTimeAsync(1000)
    expect(opts.onPoll).toHaveBeenCalledTimes(2)
  })

  it('startPoll without an order id does not poll', async () => {
    const { poll, opts, state } = makeHarness()
    state.orderId = null
    poll.startPoll()
    await vi.advanceTimersByTimeAsync(10_000)
    expect(opts.onPoll).not.toHaveBeenCalled()
  })

  it('stopPoll cancels the pending chain immediately', async () => {
    const { poll, opts } = makeHarness()
    poll.startPoll()
    poll.stopPoll()
    await vi.advanceTimersByTimeAsync(20_000)
    expect(opts.onPoll).not.toHaveBeenCalled()
  })
})
}

// @vitest-environment jsdom
/**
 * OPT-20260809-030: warnNetworkFailure 按 (context, 错误摘要) 去抖。
 *
 * 背景：refreshMachineSummary 每 15s 调用两次 warnNetworkFailure（summary + indicators），
 * 网络瞬时故障持续数分钟时控制台/指标被同类告警刷屏。失败本身已由 pair-atomic 保留旧快照
 * 兜底，告警只需在信号变化时触发。去抖层：同一 (context, 错误摘要) 在窗口内只告警一次，
 * 不同信号（异错误/异端点）即时透传，窗口过后仍失败则再次告警（降频心跳）。
 */
import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest'
import { warnNetworkFailure } from './workPanelApiUtils.js'

describe('warnNetworkFailure 去抖（OPT-20260809-030）', () => {
  let warnSpy

  beforeEach(() => {
    vi.useFakeTimers()
    warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {})
  })

  afterEach(() => {
    vi.restoreAllMocks()
    vi.useRealTimers()
  })

  it('同 context + 同错误：窗口内只告警一次（去抖）', () => {
    const err = new Error('HTTP 502')
    warnNetworkFailure('debounce-same', err)
    warnNetworkFailure('debounce-same', err)
    warnNetworkFailure('debounce-same', err)
    expect(warnSpy).toHaveBeenCalledTimes(1)
  })

  it('同 context + 异错误消息：各自透传告警', () => {
    warnNetworkFailure('debounce-diff-msg', new Error('HTTP 502'))
    warnNetworkFailure('debounce-diff-msg', new Error('HTTP 503'))
    expect(warnSpy).toHaveBeenCalledTimes(2)
  })

  it('异 context + 同错误：各自透传告警（异端点独立）', () => {
    const err = new Error('network down')
    warnNetworkFailure('workspace-machine-summary', err)
    warnNetworkFailure('workspace-runtime-indicators', err)
    expect(warnSpy).toHaveBeenCalledTimes(2)
  })

  it('窗口过后同错误再次告警（降频心跳，非永久沉默）', () => {
    const err = new Error('HTTP 502')
    warnNetworkFailure('machine-summary-heartbeat', err)
    expect(warnSpy).toHaveBeenCalledTimes(1)

    vi.advanceTimersByTime(30_000)
    warnNetworkFailure('machine-summary-heartbeat', err)
    expect(warnSpy).toHaveBeenCalledTimes(2)
  })

  it('错误消息大小写/空白差异视为同信号去抖', () => {
    warnNetworkFailure('machine-summary-case', new Error('  HTTP 502  '))
    warnNetworkFailure('machine-summary-case', new Error('http 502'))
    expect(warnSpy).toHaveBeenCalledTimes(1)
  })
})

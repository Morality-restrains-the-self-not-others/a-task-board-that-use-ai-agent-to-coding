// @vitest-environment node
import { describe, expect, it, vi } from 'vitest'
import { pollUntil } from './pollUntil.js'

describe('pollUntil', () => {
  it('returns done when tick succeeds', async () => {
    const tick = vi.fn().mockResolvedValueOnce('done')
    await expect(pollUntil({ tick, intervalMs: 1, maxMs: 1000 })).resolves.toBe('done')
    expect(tick).toHaveBeenCalledTimes(1)
  })

  it('retries on continue until done', async () => {
    const tick = vi
      .fn()
      .mockResolvedValueOnce('continue')
      .mockResolvedValueOnce('continue')
      .mockResolvedValueOnce('done')
    await expect(pollUntil({ tick, intervalMs: 1, maxMs: 1000 })).resolves.toBe('done')
    expect(tick).toHaveBeenCalledTimes(3)
  })

  it('returns error without further retries', async () => {
    const tick = vi.fn().mockResolvedValue('error')
    await expect(pollUntil({ tick, intervalMs: 1, maxMs: 1000 })).resolves.toBe('error')
    expect(tick).toHaveBeenCalledTimes(1)
  })

  it('returns timeout when never done', async () => {
    const tick = vi.fn().mockResolvedValue('continue')
    await expect(pollUntil({ tick, intervalMs: 5, maxMs: 20 })).resolves.toBe('timeout')
    expect(tick.mock.calls.length).toBeGreaterThan(1)
  })

  it('returns aborted when shouldAbort becomes true', async () => {
    let aborted = false
    const tick = vi.fn().mockImplementation(async () => {
      aborted = true
      return 'continue'
    })
    await expect(
      pollUntil({
        tick,
        intervalMs: 1,
        maxMs: 1000,
        shouldAbort: () => aborted
      })
    ).resolves.toBe('aborted')
  })

  it('delayFirst waits before the first tick', async () => {
    const sleep = vi.fn().mockResolvedValue(undefined)
    const tick = vi.fn().mockResolvedValue('done')
    await expect(
      pollUntil({ tick, intervalMs: 7, maxMs: 1000, delayFirst: true, sleep })
    ).resolves.toBe('done')
    expect(sleep).toHaveBeenCalledWith(7)
    expect(tick).toHaveBeenCalledTimes(1)
  })

  it('supports Infinity maxMs until tick returns done', async () => {
    const sleep = vi.fn().mockResolvedValue(undefined)
    const tick = vi
      .fn()
      .mockResolvedValueOnce('continue')
      .mockResolvedValueOnce('done')
    await expect(
      pollUntil({ tick, intervalMs: 1, maxMs: Number.POSITIVE_INFINITY, sleep })
    ).resolves.toBe('done')
    expect(tick).toHaveBeenCalledTimes(2)
  })
})
import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest'
import {
  AVAILABLE_INSTANCES_DEBOUNCE_MS,
  createAvailableInstancesFetchScheduler,
} from './availableInstancesFetchCoordinator.js'

describe('availableInstancesFetchCoordinator', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('debounce 期间多次 schedule 仅触发最后一次 onRun', () => {
    const runs = []
    const scheduler = createAvailableInstancesFetchScheduler({
      debounceMs: 300,
      onRun: (generation) => runs.push(generation),
    })

    scheduler.schedule()
    scheduler.schedule()
    scheduler.schedule()

    expect(runs).toEqual([])
    vi.advanceTimersByTime(AVAILABLE_INSTANCES_DEBOUNCE_MS)
    expect(runs).toHaveLength(1)
    expect(scheduler.currentGeneration()).toBe(runs[0])
  })

  it('schedule 时 onAbort 会被调用以取消 in-flight', () => {
    const onAbort = vi.fn()
    const scheduler = createAvailableInstancesFetchScheduler({
      onRun: () => {},
    })

    scheduler.schedule({ immediate: true, onAbort })
    scheduler.schedule({ immediate: true, onAbort })

    expect(onAbort).toHaveBeenCalledTimes(2)
  })

  it('beginFetch 使过期代际 isStale，新代际仍有效', () => {
    const scheduler = createAvailableInstancesFetchScheduler({ onRun: () => {} })
    const first = scheduler.beginFetch()
    expect(first.isStale()).toBe(false)

    scheduler.schedule({ immediate: true, onAbort: () => {} })
    expect(first.isStale()).toBe(true)

    const second = scheduler.beginFetch(scheduler.currentGeneration())
    expect(second.isStale()).toBe(false)
  })
})

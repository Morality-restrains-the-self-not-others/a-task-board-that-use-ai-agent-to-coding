import { describe, expect, it, vi } from 'vitest'
import {
  notifyRuntimeHydrate,
  notifyRuntimeAbsentHydrate,
  shouldRematchRuntimeHydrate,
  RUNTIME_ABSENT_HYDRATE_STATUS,
} from './serverRuntimeHydrate.js'

describe('notifyRuntimeHydrate', () => {
  it('no-ops without updater or empty status', () => {
    const fn = vi.fn()
    notifyRuntimeHydrate(null, 'Running')
    notifyRuntimeHydrate(fn, '')
    expect(fn).not.toHaveBeenCalled()
  })

  it('emits runtime_hydrate payload', () => {
    const fn = vi.fn()
    notifyRuntimeHydrate(fn, 'Running')
    expect(fn).toHaveBeenCalledWith({
      status: 'runtime_hydrate',
      runtime_status: 'Running',
    })
  })
})

describe('notifyRuntimeAbsentHydrate', () => {
  it('emits Stopped hydrate for no-instance rematch', () => {
    const fn = vi.fn()
    notifyRuntimeAbsentHydrate(fn)
    expect(fn).toHaveBeenCalledWith({
      status: 'runtime_hydrate',
      runtime_status: RUNTIME_ABSENT_HYDRATE_STATUS,
    })
  })
})

describe('shouldRematchRuntimeHydrate', () => {
  it('true when local false and cloud Running', () => {
    expect(shouldRematchRuntimeHydrate(false, 'Running')).toBe(true)
    expect(shouldRematchRuntimeHydrate(true, 'Running')).toBe(false)
    expect(shouldRematchRuntimeHydrate(false, 'Stopped')).toBe(false)
  })

  it('true when local still running but cloud not serving', () => {
    expect(shouldRematchRuntimeHydrate(true, 'Stopped')).toBe(true)
    expect(shouldRematchRuntimeHydrate(true, 'Released')).toBe(true)
    expect(shouldRematchRuntimeHydrate(false, '')).toBe(false)
  })

  it('true when local running and API marks runtime absent', () => {
    expect(shouldRematchRuntimeHydrate(true, '', { runtimeAbsent: true })).toBe(true)
    expect(shouldRematchRuntimeHydrate(false, '', { runtimeAbsent: true })).toBe(false)
  })
})

describe('background Describe poll removed', () => {
  it('does not export createRuntimeStatusPollController', async () => {
    const mod = await import('./serverRuntimeHydrate.js')
    expect(mod.createRuntimeStatusPollController).toBeUndefined()
  })
})

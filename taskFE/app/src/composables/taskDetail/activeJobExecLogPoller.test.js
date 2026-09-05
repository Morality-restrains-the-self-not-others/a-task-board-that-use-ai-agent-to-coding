// @vitest-environment node
import { describe, expect, it, vi } from 'vitest'
import {
  createActiveJobExecLogPoller,
  isActiveJobStatus,
} from './activeJobExecLogPoller.js'

describe('activeJobExecLogPoller', () => {
  it('isActiveJobStatus covers queued/pending/running only', () => {
    expect(isActiveJobStatus('running')).toBe(true)
    expect(isActiveJobStatus('queued')).toBe(true)
    expect(isActiveJobStatus('pending')).toBe(true)
    expect(isActiveJobStatus('completed')).toBe(false)
    expect(isActiveJobStatus('failed')).toBe(false)
  })

  it('does not start interval; live steps come from SSE and history from DB', () => {
    const setIntervalFn = vi.fn()
    const poller = createActiveJobExecLogPoller({
      isEndpointReady: () => true,
      getSelectedJobStatus: () => ({ jobId: 'j1', status: 'running' }),
      refresh: vi.fn(),
      intervalMs: 1000,
      setIntervalFn,
      clearIntervalFn: vi.fn(),
    })
    poller.sync()
    expect(setIntervalFn).not.toHaveBeenCalled()
  })

  it('does not poll when endpoint not ready', () => {
    const setIntervalFn = vi.fn()
    const poller = createActiveJobExecLogPoller({
      isEndpointReady: () => false,
      getSelectedJobStatus: () => ({ jobId: 'j1', status: 'running' }),
      refresh: vi.fn(),
      setIntervalFn,
      clearIntervalFn: vi.fn(),
    })
    poller.sync()
    expect(setIntervalFn).not.toHaveBeenCalled()
  })
})

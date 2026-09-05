// @vitest-environment node
import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest'
import { ref } from 'vue'
import {
  SSE_MAX_RECONNECT_ATTEMPTS,
  SSE_INITIAL_RECONNECT_DELAY,
  createSseReconnectState,
} from './taskDetailSseReconnect.js'

describe('createSseReconnectState', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('closeSSEConnection resets live and reconnect state', () => {
    const sseLive = ref(true)
    const stop = vi.fn()
    const sse = createSseReconnectState({
      sseLive,
      effectiveTaskId: ref('task-1'),
      serverStartupStatusPoll: { stop },
      establishSSEConnection: vi.fn(),
    })
    sse.sseReconnectAttempts.value = 2
    sse.sseReconnecting.value = true
    sse.closeSSEConnection()
    expect(sseLive.value).toBe(false)
    expect(sse.sseReconnectAttempts.value).toBe(0)
    expect(sse.sseReconnecting.value).toBe(false)
    expect(stop).toHaveBeenCalled()
  })

  it('scheduleSSEReconnect stops after max attempts', () => {
    const establishSSEConnection = vi.fn()
    const sse = createSseReconnectState({
      sseLive: ref(false),
      effectiveTaskId: ref('task-1'),
      establishSSEConnection,
    })
    sse.currentSseTaskId = 'task-1'
    sse.sseReconnectAttempts.value = SSE_MAX_RECONNECT_ATTEMPTS
    sse.scheduleSSEReconnect('task-1')
    vi.runAllTimers()
    expect(establishSSEConnection).not.toHaveBeenCalled()
    expect(sse.sseReconnecting.value).toBe(false)
  })

  it('scheduleSSEReconnect uses exponential delay', () => {
    const establishSSEConnection = vi.fn()
    const sse = createSseReconnectState({
      sseLive: ref(false),
      effectiveTaskId: ref('task-1'),
      establishSSEConnection,
    })
    sse.currentSseTaskId = 'task-1'
    sse.scheduleSSEReconnect('task-1')
    expect(sse.sseReconnectAttempts.value).toBe(1)
    vi.advanceTimersByTime(SSE_INITIAL_RECONNECT_DELAY)
    expect(establishSSEConnection).toHaveBeenCalledWith('task-1')
  })

  it('handleSSEManualReconnect resets attempts and reconnects', () => {
    const establishSSEConnection = vi.fn()
    const sse = createSseReconnectState({
      sseLive: ref(false),
      effectiveTaskId: ref('task-42'),
      establishSSEConnection,
    })
    sse.sseReconnectAttempts.value = 3
    sse.handleSSEManualReconnect()
    expect(sse.sseReconnectAttempts.value).toBe(0)
    expect(establishSSEConnection).toHaveBeenCalledWith('task-42')
  })
})

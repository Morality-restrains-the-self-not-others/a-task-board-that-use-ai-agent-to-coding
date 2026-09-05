// @vitest-environment node
import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import {
  applyContainerHeartbeatSsePayload,
  createContainerHeartbeatSseBuffer,
  flushPendingContainerHeartbeat,
  isContainerHeartbeatServing,
  promoteContainerHeartbeatFromIdle,
} from './applyContainerHeartbeatSse.js'

function makeApplyDeps(overrides = {}) {
  return {
    containerHeartbeatStatus: ref('idle'),
    containerHeartbeatAttempts: ref(0),
    containerHeartbeatLastSuccess: ref(null),
    containerHeartbeatError: ref(''),
    applyContainerHeartbeatSeqFromSse: vi.fn(),
    markContainerTransportOk: vi.fn(),
    resetServerRuntimeLayerGraphGateCache: vi.fn(),
    maybeRefreshLayerGraphOnContainerHeartbeatOk: vi.fn(),
    ...overrides,
  }
}

describe('isContainerHeartbeatServing', () => {
  it('rejects when paused or notServing', () => {
    expect(
      isContainerHeartbeatServing({
        containerHeartbeatPaused: ref(true),
        isServerRunning: ref(true),
        containerEndpointRegistered: ref(true),
      }),
    ).toBe(false)
    expect(
      isContainerHeartbeatServing({
        serverRuntimeNotServing: ref(true),
        isServerRunning: ref(true),
        containerEndpointRegistered: ref(true),
      }),
    ).toBe(false)
  })

  it('accepts endpoint registered even when isServerRunning is still false (cold open)', () => {
    expect(
      isContainerHeartbeatServing({
        isServerRunning: ref(false),
        isServerStarting: ref(false),
        containerEndpointRegistered: ref(true),
        serverRuntimeNotServing: ref(false),
        containerHeartbeatPaused: ref(false),
      }),
    ).toBe(true)
  })

  it('accepts isServerRunning', () => {
    expect(
      isContainerHeartbeatServing({
        isServerRunning: ref(true),
        containerEndpointRegistered: ref(false),
      }),
    ).toBe(true)
  })
})

describe('applyContainerHeartbeatSsePayload', () => {
  it('sets connected when bidirectional ok', () => {
    const deps = makeApplyDeps()
    applyContainerHeartbeatSsePayload(
      {
        status: 'ok',
        bidirectional_ok: true,
        uplink_ok: true,
        downlink_ok: true,
      },
      deps,
    )
    expect(deps.containerHeartbeatStatus.value).toBe('connected')
    expect(deps.containerHeartbeatAttempts.value).toBe(1)
    expect(deps.markContainerTransportOk).toHaveBeenCalled()
  })
})

describe('buffer + promote + flush', () => {
  it('stashes then flushes when serving', () => {
    const buffer = createContainerHeartbeatSseBuffer()
    const deps = {
      ...makeApplyDeps(),
      isServerRunning: ref(false),
      containerEndpointRegistered: ref(false),
      serverRuntimeNotServing: ref(false),
      containerHeartbeatPaused: ref(false),
    }
    buffer.stash({
      status: 'ok',
      bidirectional_ok: true,
      uplink_ok: true,
      downlink_ok: true,
    })
    expect(flushPendingContainerHeartbeat(buffer, deps)).toBe(false)
    expect(deps.containerHeartbeatStatus.value).toBe('idle')

    deps.containerEndpointRegistered.value = true
    expect(flushPendingContainerHeartbeat(buffer, deps)).toBe(true)
    expect(deps.containerHeartbeatStatus.value).toBe('connected')
  })

  it('promotes idle to connecting when endpoint registered and serving', () => {
    const status = ref('idle')
    const ok = promoteContainerHeartbeatFromIdle({
      containerHeartbeatStatus: status,
      containerEndpointRegistered: ref(true),
      isServerRunning: ref(true),
      containerHeartbeatPaused: ref(false),
      serverRuntimeNotServing: ref(false),
    })
    expect(ok).toBe(true)
    expect(status.value).toBe('connecting')
  })

  it('promotes idle to connecting when server running but endpoint not registered yet', () => {
    const status = ref('idle')
    const err = ref('')
    const ok = promoteContainerHeartbeatFromIdle({
      containerHeartbeatStatus: status,
      containerHeartbeatError: err,
      containerEndpointRegistered: ref(false),
      isServerRunning: ref(true),
      serverRuntimeNotServing: ref(false),
    })
    expect(ok).toBe(true)
    expect(status.value).toBe('connecting')
    expect(err.value).toContain('登记')
  })

  it('does not promote when idle without endpoint and server not up', () => {
    const status = ref('idle')
    const ok = promoteContainerHeartbeatFromIdle({
      containerHeartbeatStatus: status,
      containerEndpointRegistered: ref(false),
      isServerRunning: ref(false),
      isServerStarting: ref(false),
    })
    expect(ok).toBe(false)
    expect(status.value).toBe('idle')
  })
})

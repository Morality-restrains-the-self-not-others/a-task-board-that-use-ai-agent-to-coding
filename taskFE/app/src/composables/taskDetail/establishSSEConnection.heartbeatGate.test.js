// @vitest-environment node
if (!process.env.VITEST) {
  // pre-commit 会以 node 直跑已暂存 *.test.js，非 Vitest 环境下跳过。
  console.log('[skip] establishSSEConnection.heartbeatGate.test.js requires vitest runtime')
} else {
  const { describe, expect, it, vi, beforeEach } = await import('vitest')
  const { ref } = await import('vue')
  const { establishSSEConnection } = await import('./establishSSEConnection.js')
  const { createContainerHeartbeatSseBuffer } = await import('./applyContainerHeartbeatSse.js')

describe('establishSSEConnection — heartbeat gate when not serving', () => {
  let deps
  let onmessage

  beforeEach(() => {
    vi.stubGlobal(
      'EventSource',
      class MockEventSource {
        constructor() {
          this.readyState = 1
          this.withCredentials = true
          this.onmessage = null
          this.onerror = null
          this.onclose = null
          this.addEventListener = vi.fn((type, cb) => {
            if (type === 'open') cb()
          })
          this.close = vi.fn()
          queueMicrotask(() => {
            onmessage = this.onmessage
          })
        }
      },
    )
    vi.mock('../../utils/config.js', () => ({
      getApiUrl: (p) => `http://test${p}`,
    }))

    deps = {
      sseConnection: ref(null),
      sseLive: ref(false),
      sseReconnecting: ref(false),
      sseReconnectAttempts: ref(0),
      containerHeartbeatStatus: ref('idle'),
      containerHeartbeatAttempts: ref(0),
      containerHeartbeatLastSuccess: ref(null),
      containerHeartbeatError: ref(''),
      containerHeartbeatPaused: ref(false),
      isServerRunning: ref(false),
      isServerStarting: ref(false),
      serverRuntimeNotServing: ref(false),
      containerEndpointRegistered: ref(false),
      containerHeartbeatSseBuffer: createContainerHeartbeatSseBuffer(),
      closeSSEConnection: vi.fn(),
      scheduleSSEReconnect: vi.fn(),
      markContainerTransportOk: vi.fn(),
      resetServerRuntimeLayerGraphGateCache: vi.fn(),
      maybeRefreshLayerGraphOnContainerHeartbeatOk: vi.fn(),
      updateServerStatus: vi.fn(),
      applyContainerHeartbeatSeqFromSse: vi.fn(),
      effectiveTenantId: ref('t1'),
      effectiveWorkspaceId: ref('w1'),
      relayAccessCode: ref(''),
      currentSseTaskId: '',
      sseReconnectTimer: null,
      establishSSEConnectionSelf: vi.fn(),
    }
  })

  it('stashes container_heartbeat when not serving (no endpoint / not running)', async () => {
    establishSSEConnection('task_1', deps)
    await Promise.resolve()
    onmessage = deps.sseConnection.value.onmessage
    onmessage({
      data: JSON.stringify({
        event_name: 'container_heartbeat',
        status: 'ok',
        uplink_ok: true,
        downlink_ok: false,
      }),
    })
    expect(deps.containerHeartbeatStatus.value).toBe('idle')
    expect(deps.applyContainerHeartbeatSeqFromSse).not.toHaveBeenCalled()
    expect(deps.containerHeartbeatSseBuffer.peek()).toBeTruthy()
  })

  it('applies container_heartbeat when server is running', async () => {
    deps.isServerRunning.value = true
    establishSSEConnection('task_1', deps)
    await Promise.resolve()
    onmessage = deps.sseConnection.value.onmessage
    onmessage({
      data: JSON.stringify({
        event_name: 'container_heartbeat',
        status: 'partial',
        uplink_ok: true,
        downlink_ok: false,
        message: '等待下行探测',
      }),
    })
    expect(deps.containerHeartbeatStatus.value).toBe('connecting')
    expect(deps.applyContainerHeartbeatSeqFromSse).toHaveBeenCalled()
  })

  it('applies container_heartbeat when endpoint registered even if isServerRunning false', async () => {
    deps.containerEndpointRegistered.value = true
    establishSSEConnection('task_1', deps)
    await Promise.resolve()
    onmessage = deps.sseConnection.value.onmessage
    onmessage({
      data: JSON.stringify({
        event_name: 'container_heartbeat',
        status: 'ok',
        bidirectional_ok: true,
        uplink_ok: true,
        downlink_ok: true,
      }),
    })
    expect(deps.containerHeartbeatStatus.value).toBe('connected')
    expect(deps.applyContainerHeartbeatSeqFromSse).toHaveBeenCalled()
  })
})
}

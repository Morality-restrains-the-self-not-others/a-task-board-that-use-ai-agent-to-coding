// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] establishSSEConnection.platformRestart.test.js requires vitest runtime')
} else {
const { describe, expect, it, vi, beforeEach, afterEach } = await import('vitest')
const { ref } = await import('vue')

vi.mock('../../utils/config.js', () => ({
  getApiUrl: (p) => `http://test${p}`,
}))

const { establishSSEConnection, isPlatformRestartSseProbeResponse, } = await import('./establishSSEConnection.js')

describe('isPlatformRestartSseProbeResponse', () => {
  it('true for nginx HTML 502', () => {
    expect(
      isPlatformRestartSseProbeResponse({
        status: 502,
        headers: { get: () => 'text/html' },
      }),
    ).toBe(true)
  })

  it('true for HTML 503', () => {
    expect(
      isPlatformRestartSseProbeResponse({
        status: 503,
        headers: { get: () => 'text/html; charset=utf-8' },
      }),
    ).toBe(true)
  })

  it('false for JSON 503 / 401 / 200', () => {
    expect(
      isPlatformRestartSseProbeResponse({
        status: 503,
        headers: { get: () => 'application/json' },
      }),
    ).toBe(false)
    expect(
      isPlatformRestartSseProbeResponse({
        status: 401,
        headers: { get: () => 'application/json' },
      }),
    ).toBe(false)
    expect(
      isPlatformRestartSseProbeResponse({
        status: 200,
        headers: { get: () => 'text/event-stream' },
      }),
    ).toBe(false)
  })
})

describe('establishSSEConnection — platform restart hint', () => {
  let lastEs
  let fetchMock

  beforeEach(() => {
    vi.useFakeTimers()
    lastEs = null
    fetchMock = vi.fn()
    vi.stubGlobal('fetch', fetchMock)
    vi.stubGlobal(
      'EventSource',
      class MockEventSource {
        constructor() {
          this.readyState = 0
          this.withCredentials = true
          this.onmessage = null
          this.onerror = null
          this.onclose = null
          this.addEventListener = vi.fn()
          this.close = vi.fn()
          lastEs = this
        }
      },
    )
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.unstubAllGlobals()
  })

  function makeDeps(overrides = {}) {
    return {
      sseConnection: ref(null),
      sseLive: ref(false),
      sseReconnecting: ref(false),
      sseReconnectAttempts: ref(0),
      ssePlatformRestartHint: ref(false),
      closeSSEConnection: vi.fn(),
      scheduleSSEReconnect: vi.fn(),
      updateServerStatus: vi.fn(),
      effectiveTenantId: ref('t1'),
      effectiveWorkspaceId: ref('w1'),
      relayAccessCode: ref(''),
      currentSseTaskId: '',
      sseReconnectTimer: null,
      establishSSEConnectionSelf: vi.fn(),
      ...overrides,
    }
  }

  it('短耗时 onerror 且探测为 HTML 502 时置位 hint', async () => {
    fetchMock.mockResolvedValue({
      status: 502,
      headers: { get: () => 'text/html' },
    })
    const deps = makeDeps()
    establishSSEConnection('task_1', deps)
    expect(lastEs).toBeTruthy()
    lastEs.onerror(new Error('fail'))
    await vi.runAllTimersAsync()
    expect(deps.ssePlatformRestartHint.value).toBe(true)
    expect(deps.scheduleSSEReconnect).toHaveBeenCalledWith('task_1')
  })

  it('短耗时 onerror 但探测非 502 时不置位（避免与已停止混淆）', async () => {
    fetchMock.mockResolvedValue({
      status: 401,
      headers: { get: () => 'application/json' },
    })
    const deps = makeDeps()
    deps.ssePlatformRestartHint.value = true
    establishSSEConnection('task_1', deps)
    lastEs.onerror(new Error('fail'))
    await vi.runAllTimersAsync()
    expect(deps.ssePlatformRestartHint.value).toBe(false)
  })

  it('短耗时 onerror 探测失败时清除 hint，不保留假阳性', async () => {
    fetchMock.mockRejectedValue(new Error('network'))
    const deps = makeDeps()
    deps.ssePlatformRestartHint.value = true
    establishSSEConnection('task_1', deps)
    lastEs.onerror(new Error('fail'))
    await vi.runAllTimersAsync()
    expect(deps.ssePlatformRestartHint.value).toBe(false)
  })

  it('已 open 过后再 onerror 不触发平台重启探测', async () => {
    const deps = makeDeps()
    establishSSEConnection('task_1', deps)
    const openCb = lastEs.addEventListener.mock.calls.find((c) => c[0] === 'open')?.[1]
    openCb?.()
    lastEs.onerror(new Error('fail'))
    await vi.runAllTimersAsync()
    expect(fetchMock).not.toHaveBeenCalled()
    expect(deps.ssePlatformRestartHint.value).toBe(false)
  })
})

}

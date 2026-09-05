// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] serverStartupStatusPoll.controller.test.js requires vitest runtime')
} else {
const { describe, expect, it, vi, beforeEach, afterEach } = await import('vitest')
const { ref } = await import('vue')

const hoisted = vi.hoisted(() => ({
  fetchMock: vi.fn()
}))

vi.mock('../../utils/apiUtils.js', () => ({
  apiFetch: hoisted.fetchMock
}))

vi.mock('../../utils/config.js', () => ({
  getApiUrl: (path) => `http://test${path}`
}))

const { createServerStartupStatusPollController } = await import('./serverStartupStatusPoll.js')

describe('createServerStartupStatusPollController', () => {
  beforeEach(() => {
    hoisted.fetchMock.mockReset()
    vi.useFakeTimers()
  })
  afterEach(() => {
    vi.useRealTimers()
  })

  it('does not REST-poll startup status — live sync is SSE push', async () => {
    const updateServerStatus = vi.fn()
    const ctl = createServerStartupStatusPollController({
      effectiveTenantId: ref('t1'),
      effectiveWorkspaceId: ref('w1'),
      effectiveTaskId: ref('task1'),
      isServerStarting: ref(true),
      sseLive: ref(false),
      statusProgress: ref(0),
      updateServerStatus,
    })
    ctl.start({ event_id: 'ev1', comment_id: 'C1' })
    await vi.advanceTimersByTimeAsync(20000)
    expect(hoisted.fetchMock).not.toHaveBeenCalled()
    expect(updateServerStatus).not.toHaveBeenCalled()
    ctl.stop()
  })
})

}

// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] useBaseBranchCommitCheck.unit.test.js requires vitest runtime')
} else {
const { afterEach, beforeEach, describe, expect, it, vi } = await import('vitest')

// vi.mock 工厂引用的变量必须经 vi.hoisted() 定义（Vitest 会把工厂 hoist 到模块顶部）。
const hoisted = vi.hoisted(() => ({
  apiFetchMock: vi.fn(),
}))
const apiFetchMock = hoisted.apiFetchMock

vi.mock('../utils/apiUtils.js', () => ({
  apiFetch: (...args) => hoisted.apiFetchMock(...args),
}))

const { useBaseBranchCommitCheck } = await import('../composables/useBaseBranchCommitCheck.js')

describe('useBaseBranchCommitCheck', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('debounces and marks verified when resolve-ref exists', async () => {
    apiFetchMock.mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({ exists: true, sha: 'abc1234deadbeef' }),
    })

    const { scheduleBaseBranchCommitCheck, isBaseBranchCommitVerified } = useBaseBranchCommitCheck({
      tenantId: () => 't1',
      getRepoUrl: () => 'http://localhost:8012/g/r.git',
    })

    scheduleBaseBranchCommitCheck('100', 0, 'abc1234')
    expect(isBaseBranchCommitVerified('100', 0, 'abc1234')).toBe(false)

    await vi.advanceTimersByTimeAsync(400)
    await Promise.resolve()
    await Promise.resolve()

    expect(apiFetchMock).toHaveBeenCalledTimes(1)
    expect(String(apiFetchMock.mock.calls[0][0])).toContain('/resolve-ref/')
    expect(isBaseBranchCommitVerified('100', 0, 'abc1234')).toBe(true)
  })

  it('clears state for non-hash input', async () => {
    const { scheduleBaseBranchCommitCheck, isBaseBranchCommitVerified, getCommitCheckState } = useBaseBranchCommitCheck({
      tenantId: () => 't1',
      getRepoUrl: () => 'http://localhost:8012/g/r.git',
    })
    scheduleBaseBranchCommitCheck('100', 0, 'main')
    await vi.advanceTimersByTimeAsync(400)
    expect(apiFetchMock).not.toHaveBeenCalled()
    expect(getCommitCheckState('100', 0)).toBe(null)
    expect(isBaseBranchCommitVerified('100', 0, 'main')).toBe(false)
  })

  it('exposes checking then missing with traceId when commit absent', async () => {
    apiFetchMock.mockResolvedValue({
      ok: true,
      status: 200,
      traceId: 'tid-missing-1',
      json: async () => ({ exists: false }),
    })

    const {
      scheduleBaseBranchCommitCheck,
      isBaseBranchCommitChecking,
      isBaseBranchCommitMissing,
      getBaseBranchCommitCheckTraceId,
      getBaseBranchCommitMissingHint,
    } = useBaseBranchCommitCheck({
      tenantId: () => 't1',
      getRepoUrl: () => 'http://localhost:8012/g/r.git',
    })

    scheduleBaseBranchCommitCheck('100', 0, 'deadbee')
    expect(isBaseBranchCommitChecking('100', 0, 'deadbee')).toBe(true)

    await vi.advanceTimersByTimeAsync(400)
    await Promise.resolve()
    await Promise.resolve()

    expect(isBaseBranchCommitChecking('100', 0, 'deadbee')).toBe(false)
    expect(isBaseBranchCommitMissing('100', 0, 'deadbee')).toBe(true)
    expect(getBaseBranchCommitMissingHint('100', 0, 'deadbee')).toBe('未找到该 commit')
    expect(getBaseBranchCommitCheckTraceId('100', 0, 'deadbee')).toBe('tid-missing-1')
  })
})

}

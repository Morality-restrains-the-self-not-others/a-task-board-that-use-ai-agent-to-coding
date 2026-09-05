// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] taskDetailFetchFns.loadError.traceId.test.js requires vitest runtime')
} else {
const { describe, expect, it, vi, beforeEach, afterEach } = await import('vitest')
const { ref } = await import('vue')

// vi.mock 工厂引用的变量必须经 vi.hoisted() 定义（Vitest 会把工厂 hoist 到模块顶部）。
const hoisted = vi.hoisted(() => ({
  apiFetch: vi.fn(),
}))
const apiFetch = hoisted.apiFetch

vi.mock('../../utils/apiUtils.js', () => ({
  apiFetch: (...args) => hoisted.apiFetch(...args),
}))

const { fetchTaskDetail } = await import('./taskDetailFetchFns.js')

describe('fetchTaskDetail data-traceId', () => {
  beforeEach(() => {
    apiFetch.mockReset()
  })
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('sets taskDetailLoadErrorTraceId from failed HTTP response', async () => {
    apiFetch.mockResolvedValue({
      ok: false,
      status: 500,
      traceId: 'tid-task-detail-500',
      _errorData: { detail: '服务不可用' },
      headers: { get: () => null },
    })

    const taskDetailLoadError = ref('')
    const taskDetailLoadErrorTraceId = ref('')
    await fetchTaskDetail({
      effectiveTenantId: ref('t1'),
      effectiveWorkspaceId: ref('ws1'),
      effectiveTaskId: ref('task1'),
      localTask: ref(null),
      taskDetailLoading: ref(false),
      taskDetailLoadError,
      taskDetailLoadErrorTraceId,
      syncRepoCloneIdentityMapFromTask: vi.fn(),
    })

    expect(taskDetailLoadError.value).toContain('服务不可用')
    expect(taskDetailLoadErrorTraceId.value).toBe('tid-task-detail-500')
  })

  it('omits TraceId on client-side missing-id validation', async () => {
    const taskDetailLoadError = ref('')
    const taskDetailLoadErrorTraceId = ref('stale')
    await fetchTaskDetail({
      effectiveTenantId: ref(''),
      effectiveWorkspaceId: ref('ws1'),
      effectiveTaskId: ref('task1'),
      localTask: ref(null),
      taskDetailLoading: ref(false),
      taskDetailLoadError,
      taskDetailLoadErrorTraceId,
      syncRepoCloneIdentityMapFromTask: vi.fn(),
    })

    expect(taskDetailLoadError.value).toContain('缺少租户')
    expect(taskDetailLoadErrorTraceId.value).toBe('')
    expect(apiFetch).not.toHaveBeenCalled()
  })
})
}

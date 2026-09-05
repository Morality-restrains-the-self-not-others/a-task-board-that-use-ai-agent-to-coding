// @vitest-environment node
if (!process.env.VITEST) {
  // pre-commit 会以 node 直跑已暂存 *.test.js，非 Vitest 环境下跳过。
  console.log('[skip] taskDetailFetchFns.parentDeliverable.test.js requires vitest runtime')
} else {
  const { beforeEach, describe, expect, it, vi } = await import('vitest')
  const { fetchParentDeliverableTitle } = await import('./taskDetailFetchFns.js')

vi.mock('../../utils/apiUtils.js', () => ({
  apiFetch: vi.fn(),
}))

const { apiFetch } = await import('../../utils/apiUtils.js')

describe('fetchParentDeliverableTitle', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('loads parent task title', async () => {
    apiFetch.mockResolvedValue({
      ok: true,
      json: async () => ({ id: 'p1', title: '价值流-核心链路', workspace_seq: 7 }),
    })
    const parentTaskTitle = { value: '' }
    const parentTaskSeq = { value: 0 }
    const isParentTaskTitleLoading = { value: false }
    await fetchParentDeliverableTitle({
      effectiveTenantId: { value: 't1' },
      effectiveWorkspaceId: { value: 'w1' },
      parentTaskId: 'p1',
      parentTaskTitle,
      parentTaskSeq,
      isParentTaskTitleLoading,
    })
    expect(apiFetch).toHaveBeenCalledWith(
      '/api/tasks/todos/tenant_id/t1/workspace_id/w1/p1/',
      expect.objectContaining({ credentials: 'include' }),
    )
    expect(parentTaskTitle.value).toBe('价值流-核心链路')
    expect(parentTaskSeq.value).toBe(7)
    expect(isParentTaskTitleLoading.value).toBe(false)
  })

  it('clears title when parent id empty', async () => {
    const parentTaskTitle = { value: 'stale' }
    await fetchParentDeliverableTitle({
      effectiveTenantId: { value: 't1' },
      effectiveWorkspaceId: { value: 'w1' },
      parentTaskId: '',
      parentTaskTitle,
      isParentTaskTitleLoading: { value: true },
    })
    expect(apiFetch).not.toHaveBeenCalled()
    expect(parentTaskTitle.value).toBe('')
  })

  it('clears title on HTTP failure', async () => {
    apiFetch.mockResolvedValue({ ok: false, status: 404 })
    const parentTaskTitle = { value: 'stale' }
    const isParentTaskTitleLoading = { value: false }
    await fetchParentDeliverableTitle({
      effectiveTenantId: { value: 't1' },
      effectiveWorkspaceId: { value: 'w1' },
      parentTaskId: 'missing',
      parentTaskTitle,
      isParentTaskTitleLoading,
    })
    expect(parentTaskTitle.value).toBe('')
    expect(isParentTaskTitleLoading.value).toBe(false)
  })
})
}

if (!process.env.VITEST) {
  // pre-commit 会以 node 直跑已暂存 *.test.js，非 Vitest 环境下跳过。
  console.log('[skip] taskDetailSubtreeFetch.test.js requires vitest runtime')
} else {
  const { describe, it, expect, vi, beforeEach } = await import('vitest')
  const { ref } = await import('vue')
  const { apiFetch } = await import('../../utils/apiUtils.js')
  const { onProgressStatusChange } = await import('./taskDetailFetchFns.js')

vi.mock('../../utils/apiUtils.js', () => ({
  apiFetch: vi.fn(),
}))


describe('onProgressStatusChange descendant gate', () => {
  beforeEach(() => {
    apiFetch.mockReset()
  })

  it('409 DESCENDANTS_NOT_TERMINAL 写入错误文案与 traceId', async () => {
    const progressStatusError = ref('')
    const progressStatusErrorTraceId = ref('')
    const isUpdatingProgressStatus = ref(false)
    apiFetch.mockResolvedValue({
      ok: false,
      status: 409,
      headers: { get: (k) => (String(k).toLowerCase() === 'x-trace-id' ? 'trace-gate-1' : null) },
      json: async () => ({
        code: 'DESCENDANTS_NOT_TERMINAL',
        error: '存在未完成或未取消的子任务，无法关闭当前任务',
        detail: '存在未完成或未取消的子任务，无法关闭当前任务',
      }),
    })

    await onProgressStatusChange(
      { target: { value: 'col-done' } },
      {
        effectiveTenantId: ref('t1'),
        effectiveWorkspaceId: ref('w1'),
        effectiveTaskId: ref('task1'),
        progressStatusOptions: ref([{ id: 'col-done', name: '已完成' }]),
        isUpdatingProgressStatus,
        progressStatusError,
        progressStatusErrorTraceId,
        localTask: ref({ id: 'task1', progress_column_id: 'col-wip' }),
      },
    )

    expect(progressStatusError.value).toContain('子任务')
    expect(progressStatusErrorTraceId.value).toBe('trace-gate-1')
  })

  it('PATCH 只发送 progress_column_id，不含 auto_run', async () => {
    apiFetch.mockResolvedValue({
      ok: true,
      json: async () => ({}),
      headers: { get: () => null },
    })
    await onProgressStatusChange(
      { target: { value: 'col-done' } },
      {
        effectiveTenantId: ref('t1'),
        effectiveWorkspaceId: ref('w1'),
        effectiveTaskId: ref('task1'),
        progressStatusOptions: ref([{ id: 'col-done', name: '已完成' }]),
        isUpdatingProgressStatus: ref(false),
        progressStatusError: ref(''),
        progressStatusErrorTraceId: ref(''),
        localTask: ref({ id: 'task1', progress_column_id: 'col-wip', auto_run: true }),
      },
    )
    expect(apiFetch).toHaveBeenCalledTimes(1)
    const opts = apiFetch.mock.calls[0][1]
    const body = JSON.parse(opts.body)
    expect(body).toEqual({ progress_column_id: 'col-done' })
    expect(body).not.toHaveProperty('auto_run')
  })
})
}

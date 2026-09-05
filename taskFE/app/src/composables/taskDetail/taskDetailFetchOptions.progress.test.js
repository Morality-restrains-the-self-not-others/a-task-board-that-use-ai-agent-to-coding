// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] taskDetailFetchOptions.progress.test.js requires vitest runtime')
} else {
  const { beforeEach, describe, expect, it, vi } = await import('vitest')
  const { fetchProgressStatusOptions } = await import('./taskDetailFetchOptions.js')

  vi.mock('../../utils/apiUtils.js', () => ({
    apiFetch: vi.fn(),
  }))

  const { apiFetch } = await import('../../utils/apiUtils.js')

  describe('fetchProgressStatusOptions', () => {
    beforeEach(() => {
      vi.clearAllMocks()
    })

    it('sorts columns by order_num so first option is the first progress column', async () => {
      apiFetch.mockResolvedValue({
        ok: true,
        json: async () => ({
          columns: [
            { id: 'col-wip', name: '进行中', order_num: 1 },
            { id: 'col-todo', name: '待处理', order_num: 0 },
          ],
        }),
      })
      const progressStatusOptions = { value: [] }
      await fetchProgressStatusOptions({
        effectiveTenantId: { value: 't1' },
        effectiveWorkspaceId: { value: 'w1' },
        progressStatusOptions,
        isProgressStatusesLoading: { value: false },
        progressStatusError: { value: '' },
      })
      expect(progressStatusOptions.value.map((c) => c.id)).toEqual(['col-todo', 'col-wip'])
    })
  })
}

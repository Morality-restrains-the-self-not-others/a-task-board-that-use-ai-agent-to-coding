// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] useProjectsListLoad.test.js requires vitest runtime')
} else {
  const { beforeEach, describe, expect, it, vi } = await import('vitest')

  const apiFetchMock = vi.hoisted(() => vi.fn())

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch: (...args) => apiFetchMock(...args),
  }))

  vi.mock('../utils/cookieUtils.js', () => ({
    getCookie: () => '',
  }))

  const { useProjectsListLoad } = await import('./useProjectsListLoad.js')

  describe('useProjectsListLoad', () => {
    beforeEach(() => {
      apiFetchMock.mockReset()
    })

    it('maps 401 to 请先登录 and keeps response traceId', async () => {
      apiFetchMock.mockResolvedValue({
        ok: false,
        status: 401,
        traceId: 'tid-list-401',
        json: async () => ({ error: '请先登录', message: '请先登录' }),
      })
      const route = { params: { tenant: '882297276515512320' }, query: {} }
      const { error, errorTraceId, loadProjects } = useProjectsListLoad({
        route,
        router: { replace: vi.fn() },
      })
      await loadProjects()
      expect(error.value).toBe('请先登录')
      expect(errorTraceId.value).toBe('tid-list-401')
    })
  })
}

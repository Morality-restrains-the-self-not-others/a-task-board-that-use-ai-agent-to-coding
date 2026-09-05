// @vitest-environment jsdom
// 工作空间列表加载失败时上抛 traceId（header-title-row data-traceId 数据源）。
// 约定：apiFetch 失败响应/错误均携带 .traceId，加载失败时组件不得静默吞掉。
if (!process.env.VITEST) {
  console.log('[skip] WorkspaceSwitcher.loadError.traceId.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { describe, expect, it, vi, beforeEach } = await import('vitest')

  const { apiFetchMock } = vi.hoisted(() => ({ apiFetchMock: vi.fn() }))
  vi.mock('vue-router', () => ({
    useRouter: () => ({ push: vi.fn() }),
  }))
  vi.mock('../utils/apiUtils', () => ({
    apiFetch: (...args) => apiFetchMock(...args),
  }))

  const { default: WorkspaceSwitcher } = await import('./WorkspaceSwitcher.vue')

  describe('WorkspaceSwitcher 工作空间加载失败 traceId 上抛', () => {
    beforeEach(() => {
      apiFetchMock.mockReset()
    })

    it('HTTP 失败（!ok）时 emit workspace-load-error 携带 response.traceId', async () => {
      apiFetchMock.mockResolvedValue({ ok: false, traceId: 'trace-http-001' })
      const wrapper = mount(WorkspaceSwitcher, { props: { tenant: 123 } })
      await flushPromises()

      const errEvents = wrapper.emitted('workspace-load-error')
      expect(errEvents).toBeTruthy()
      expect(errEvents[0][0]).toBe('trace-http-001')
    })

    it('网络异常时 emit workspace-load-error 携带 error.traceId', async () => {
      const networkError = new Error('network down')
      networkError.traceId = 'trace-net-002'
      apiFetchMock.mockRejectedValue(networkError)
      const wrapper = mount(WorkspaceSwitcher, { props: { tenant: 123 } })
      await flushPromises()

      const errEvents = wrapper.emitted('workspace-load-error')
      expect(errEvents).toBeTruthy()
      expect(errEvents[0][0]).toBe('trace-net-002')
    })

    it('加载成功时 emit workspace-loaded 且不 emit workspace-load-error', async () => {
      apiFetchMock.mockResolvedValue({
        ok: true,
        traceId: 'trace-ok-003',
        json: async () => [{ id: 1, name: '用户的工作空间', is_current: true }],
      })
      const wrapper = mount(WorkspaceSwitcher, { props: { tenant: 123 } })
      await flushPromises()

      expect(wrapper.emitted('workspace-loaded')).toBeTruthy()
      expect(wrapper.emitted('workspace-load-error')).toBeUndefined()
      expect(wrapper.emitted('workspace-switched')[0][0]).toEqual({ id: 1, name: '用户的工作空间' })
    })
  })
}

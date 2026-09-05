// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] WorkspaceSwitcher.click-guard.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { describe, expect, it, vi, beforeEach } = await import('vitest')

  const { apiFetchMock } = vi.hoisted(() => ({ apiFetchMock: vi.fn() }))
  vi.mock('vue-router', () => ({
    useRouter: () => ({ push: vi.fn() }),
    useRoute: () => ({ query: {} }),
  }))
  vi.mock('../utils/apiUtils', () => ({
    apiFetch: (...args) => apiFetchMock(...args),
  }))
  // OPT-20260819-038: 让 guard 直接放行并把幂等键透传给 mergeIdempotencyHeaders，
  // 断言切换工作空间 POST 携带 Idempotency-Key 头。
  vi.mock('../utils/clickGuard.js', () => ({
    createClickGuard: () => ({
      run: async (fn) => fn({ idempotencyKey: 'ik-switch-1' }),
    }),
    mergeIdempotencyHeaders: (headers, key) => ({ ...(headers || {}), 'Idempotency-Key': key }),
  }))

  const { default: WorkspaceSwitcher } = await import('./WorkspaceSwitcher.vue')

  describe('WorkspaceSwitcher 切换工作空间实施 clickGuard 接线', () => {
    beforeEach(() => {
      apiFetchMock.mockReset()
    })

    it('点工作空间选项发起一次 POST 且携带 Idempotency-Key', async () => {
      apiFetchMock.mockImplementation(async (url, opts) => {
        if (opts?.method === 'POST') {
          return { ok: true, json: async () => ({ success: true }) }
        }
        return { ok: true, json: async () => [{ id: 1, name: '工作空间A', is_current: true }] }
      })
      const wrapper = mount(WorkspaceSwitcher, { props: { tenant: 123 } })
      await flushPromises()

      await wrapper.find('#workspace-toggle').trigger('click')
      await flushPromises()
      // 下拉菜单内的工作空间选项（索引 0 是 toggle 按钮）
      const optionBtn = wrapper.findAll('button')[1]
      expect(optionBtn).toBeTruthy()
      await optionBtn.trigger('click')
      await flushPromises()

      const postCall = apiFetchMock.mock.calls.find(([, opts]) => opts?.method === 'POST')
      expect(postCall).toBeTruthy()
      expect(postCall[0]).toContain('/api/projects/switch/tenant_id/123/')
      expect(postCall[1].headers['Idempotency-Key']).toBe('ik-switch-1')
      expect(postCall[1].body).toContain('"workspace_id":1')
      wrapper.unmount()
    })
  })
}

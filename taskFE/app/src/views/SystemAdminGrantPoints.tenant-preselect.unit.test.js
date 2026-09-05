// @vitest-environment jsdom
// OPT-20260825-026: 从 /system-admin/grant-points/?tenant_id= 深链自动选中目标租户。
if (!process.env.VITEST) {
  console.log('[skip] SystemAdminGrantPoints.tenant-preselect.unit.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { describe, expect, it, vi, beforeEach } = await import('vitest')

  const mocks = vi.hoisted(() => ({
    apiFetch: vi.fn(),
  }))

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch: (...args) => mocks.apiFetch(...args),
  }))
  vi.mock('../utils/traceId.js', () => ({
    extractTraceId: (v) => (v && v.traceId) || '',
  }))
  vi.mock('vue-router', () => ({
    useRoute: () => mocks.routeMock,
  }))

  const { default: SystemAdminGrantPoints } = await import('./SystemAdminGrantPoints.vue')

  const TENANTS = [
    { id: '877397588196749312', name: '我的公司', phone: '13800138000', email: 'owner@example.com' },
    { id: '2', name: '空联系', phone: '', email: '' },
  ]

  function jsonOk(body) {
    return Promise.resolve({
      ok: true,
      status: 200,
      json: () => Promise.resolve(body),
      traceId: '',
    })
  }

  describe('SystemAdminGrantPoints 深链预选租户', () => {
    beforeEach(() => {
      mocks.apiFetch.mockReset()
      mocks.apiFetch.mockImplementation((url) => {
        const u = String(url)
        if (u.includes('/tenant-options/')) return jsonOk(TENANTS)
        if (u.includes('/gitlab-regions/')) return jsonOk({ regions: [] })
        if (u.includes('/billing/membership/')) return jsonOk({ membership: { tier: 'normal' } })
        return Promise.resolve({ ok: false, status: 404, json: async () => ({}), traceId: '' })
      })
      mocks.routeMock = { query: { tenant_id: '877397588196749312' }, params: {}, path: '/system-admin/grant-points/' }
    })

    it('挂载时按 tenant_id 搜索并自动选中目标租户', async () => {
      const wrapper = mount(SystemAdminGrantPoints)
      await flushPromises()
      await flushPromises()
      const selected = wrapper.get('[data-testid="tenant-selected-summary"]')
      expect(selected.text()).toContain('我的公司')
      expect(selected.text()).toContain('877397588196749312')
      const optionCall = mocks.apiFetch.mock.calls.find((c) => String(c[0]).includes('/tenant-options/') && String(c[0]).includes('search='))
      expect(String(optionCall[0])).toContain('search=877397588196749312')
      wrapper.unmount()
    })

    it('tenant_id 不存在时不选中任何租户', async () => {
      mocks.routeMock = { query: { tenant_id: 'nope' }, params: {}, path: '/system-admin/grant-points/' }
      const wrapper = mount(SystemAdminGrantPoints)
      await flushPromises()
      await flushPromises()
      expect(wrapper.find('[data-testid="tenant-selected-summary"]').exists()).toBe(false)
      wrapper.unmount()
    })
  })
}

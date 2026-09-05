// @vitest-environment jsdom
// 赠送页租户下拉展示创建者手机号与邮箱（TP-TENANT-OPT-1/2/4）。
if (!process.env.VITEST) {
  console.log('[skip] SystemAdminGrantPoints.tenant-options.unit.test.js requires vitest runtime')
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

  function mockApi() {
    mocks.apiFetch.mockImplementation((url) => {
      const u = String(url)
      if (u.includes('/tenant-options/')) return jsonOk(TENANTS)
      if (u.includes('/gitlab-regions/')) return jsonOk({ regions: [] })
      if (u.includes('/billing/membership/')) return jsonOk({ membership: { tier: 'normal' } })
      return Promise.resolve({ ok: false, status: 404, json: async () => ({}), traceId: '' })
    })
  }

  describe('SystemAdminGrantPoints 租户下拉联系方式', () => {
    beforeEach(() => {
      mocks.apiFetch.mockReset()
      mockApi()
    })

    it('下拉同时显示手机号和邮箱', async () => {
      const wrapper = mount(SystemAdminGrantPoints)
      await flushPromises()
      await wrapper.find('input[type="text"]').trigger('focus')
      await flushPromises()
      const first = wrapper.findAll('li')[0]
      expect(first.text()).toContain('我的公司')
      expect(first.text()).toContain('13800138000')
      expect(first.text()).toContain('owner@example.com')
      wrapper.unmount()
    })

    it('无联系方式时不渲染空白联系行', async () => {
      const wrapper = mount(SystemAdminGrantPoints)
      await flushPromises()
      await wrapper.find('input[type="text"]').trigger('focus')
      await flushPromises()
      const emptyRow = wrapper.findAll('li')[1]
      expect(emptyRow.text()).toContain('空联系')
      expect(emptyRow.find('[data-testid="tenant-option-contact"]').exists()).toBe(false)
      wrapper.unmount()
    })

    it('已选摘要复述联系方式', async () => {
      const wrapper = mount(SystemAdminGrantPoints)
      await flushPromises()
      await wrapper.find('input[type="text"]').trigger('focus')
      await flushPromises()
      await wrapper.findAll('li')[0].trigger('mousedown')
      await flushPromises()
      const selected = wrapper.get('[data-testid="tenant-selected-summary"]')
      expect(selected.text()).toContain('我的公司')
      expect(selected.text()).toContain('13800138000')
      expect(selected.text()).toContain('owner@example.com')
      wrapper.unmount()
    })
  })
}

// @vitest-environment jsdom
// 赠送数量必须显式填写；只改 VIP 不得因默认 100 帖误建赠送订单。
if (!process.env.VITEST) {
  console.log('[skip] SystemAdminGrantPoints.explicit-quantity.unit.test.js requires vitest runtime')
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

  const TENANTS = [{ id: '873472655125147648', name: 'Acme', phone: '无', email: '无' }]
  const MEMBERSHIP = { membership: { tier: 'normal', admin_tier_locked: false } }

  function jsonOk(body) {
    return Promise.resolve({
      ok: true,
      status: 200,
      json: () => Promise.resolve(body),
      traceId: '',
    })
  }

  function mockApi() {
    mocks.apiFetch.mockImplementation((url, options = {}) => {
      const u = String(url)
      if (u.includes('/tenant-options/')) return jsonOk(TENANTS)
      if (u.includes('/gitlab-regions/')) return jsonOk({ regions: [] })
      if (u.includes('/billing/membership/')) return jsonOk(MEMBERSHIP)
      if (u.includes('admin_grant_points')) {
        return jsonOk({
          status: 'ok',
          grants: [],
          membership: { tier: 'vip1', admin_tier_locked: true },
        })
      }
      return Promise.resolve({ ok: false, status: 404, json: async () => ({}), traceId: '' })
    })
  }

  async function selectFirstTenant(wrapper) {
    await wrapper.find('input[type="text"]').trigger('focus')
    await flushPromises()
    await wrapper.find('li').trigger('mousedown')
    await flushPromises()
  }

  describe('SystemAdminGrantPoints 显式数量', () => {
    beforeEach(() => {
      mocks.apiFetch.mockReset()
      mockApi()
    })

    it('数量输入默认留空而不是 100', async () => {
      const wrapper = mount(SystemAdminGrantPoints)
      await flushPromises()
      const qty = wrapper.find('[data-testid="grant-quantity-input"]')
      expect(qty.exists()).toBe(true)
      const raw = qty.element.value
      expect(raw === '' || raw === undefined).toBe(true)
      expect(raw).not.toBe('100')
      wrapper.unmount()
    })

    it('仅选 VIP1 且不填数量时 POST resources 为空数组', async () => {
      const wrapper = mount(SystemAdminGrantPoints)
      await flushPromises()
      await selectFirstTenant(wrapper)
      await wrapper.find('[data-testid="grant-membership-tier"]').setValue('vip1')
      await wrapper.find('form').trigger('submit')
      await flushPromises()
      const post = mocks.apiFetch.mock.calls.find((c) => String(c[1]?.method || '').toUpperCase() === 'POST')
      expect(post).toBeTruthy()
      const body = JSON.parse(post[1].body)
      expect(body.resources).toEqual([])
      expect(body.membership_tier).toBe('vip1')
      expect(body.idempotency_key).toBeTruthy()
      const headers = post[1].headers || {}
      expect(headers['Idempotency-Key'] || headers['idempotency-key']).toBe(body.idempotency_key)
      wrapper.unmount()
    })

    it('未填数量且不改 VIP 时不 POST', async () => {
      const wrapper = mount(SystemAdminGrantPoints)
      await flushPromises()
      await selectFirstTenant(wrapper)
      await wrapper.find('form').trigger('submit')
      await flushPromises()
      const post = mocks.apiFetch.mock.calls.find((c) => String(c[1]?.method || '').toUpperCase() === 'POST')
      expect(post).toBeFalsy()
      expect(wrapper.text()).toMatch(/请至少填写一项有效的资源数量|或选择要修改的 VIP/)
      wrapper.unmount()
    })

    it('仅改 VIP 时预览说明不会生成赠送订单', async () => {
      const wrapper = mount(SystemAdminGrantPoints)
      await flushPromises()
      await selectFirstTenant(wrapper)
      await wrapper.find('[data-testid="grant-membership-tier"]').setValue('vip1')
      await flushPromises()
      expect(wrapper.find('[data-testid="grant-submit-preview"]').text()).toContain('不会生成赠送订单')
      wrapper.unmount()
    })
  })
}

// @vitest-environment jsdom
// 赠送页可修改租户 VIP 等级；「不修改」不传 membership_tier。
if (!process.env.VITEST) {
  console.log('[skip] SystemAdminGrantPoints.membership.unit.test.js requires vitest runtime')
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

  function mockApi(overrides = {}) {
    mocks.apiFetch.mockImplementation((url, options = {}) => {
      const method = String(options.method || 'GET').toUpperCase()
      const u = String(url)
      if (overrides[`${method} ${u}`] !== undefined) {
        return jsonOk(overrides[`${method} ${u}`])
      }
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

  describe('SystemAdminGrantPoints VIP 等级', () => {
    beforeEach(() => {
      mocks.apiFetch.mockReset()
      mockApi()
    })

    it('选中租户后展示当前等级和下拉', async () => {
      const wrapper = mount(SystemAdminGrantPoints)
      await flushPromises()
      await selectFirstTenant(wrapper)
      expect(wrapper.find('[data-testid="grant-membership-tier"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="grant-membership-current"]').text()).toContain('普通会员')
      wrapper.unmount()
    })

    it('选择 VIP1 后 POST 含 membership_tier', async () => {
      const wrapper = mount(SystemAdminGrantPoints)
      await flushPromises()
      await selectFirstTenant(wrapper)
      await wrapper.find('[data-testid="grant-membership-tier"]').setValue('vip1')
      await wrapper.find('form').trigger('submit')
      await flushPromises()
      const post = mocks.apiFetch.mock.calls.find((c) => String(c[1]?.method || '').toUpperCase() === 'POST')
      expect(post).toBeTruthy()
      expect(post[1].body).toContain('"membership_tier":"vip1"')
      expect(wrapper.text()).toContain('VIP1')
      wrapper.unmount()
    })

    it('不修改等级时 POST 不含 membership_tier', async () => {
      mockApi({
        'POST /api/tenant/873472655125147648/billing/accounts/admin_grant_points/': {
          status: 'ok',
          task_post_quota_after: 100,
          grants: [{ resource_type: 'task_post', quantity: 100 }],
        },
      })
      const wrapper = mount(SystemAdminGrantPoints)
      await flushPromises()
      await selectFirstTenant(wrapper)
      // 数量须显式填写（默认留空，否则不发 POST）
      await wrapper.find('[data-testid="grant-quantity-input"]').setValue('50')
      await wrapper.find('form').trigger('submit')
      await flushPromises()
      const post = mocks.apiFetch.mock.calls.find((c) => String(c[1]?.method || '').toUpperCase() === 'POST')
      expect(post).toBeTruthy()
      expect(post[1].body).not.toContain('membership_tier')
      expect(post[1].body).toContain('"quantity":50')
      wrapper.unmount()
    })

    it('仅改 VIP 时可移除全部资源行后提交空 resources', async () => {
      const wrapper = mount(SystemAdminGrantPoints)
      await flushPromises()
      await selectFirstTenant(wrapper)
      await wrapper.find('[data-testid="grant-membership-tier"]').setValue('vip1')
      const removeBtn = wrapper.findAll('button').find((b) => b.text().trim() === '移除')
      expect(removeBtn).toBeTruthy()
      await removeBtn.trigger('click')
      await wrapper.find('form').trigger('submit')
      await flushPromises()
      const post = mocks.apiFetch.mock.calls.find((c) => String(c[1]?.method || '').toUpperCase() === 'POST')
      expect(post).toBeTruthy()
      const body = JSON.parse(post[1].body)
      expect(body.resources).toEqual([])
      expect(body.membership_tier).toBe('vip1')
      wrapper.unmount()
    })
  })
}

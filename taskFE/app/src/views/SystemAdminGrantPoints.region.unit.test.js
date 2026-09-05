// @vitest-environment jsdom
// 赠送 GitLab 磁盘/流量必须选择区域，POST body 带 region。
if (!process.env.VITEST) {
  console.log('[skip] SystemAdminGrantPoints.region.unit.test.js requires vitest runtime')
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
  const REGIONS = {
    regions: [
      { slug: 'tencent-sh-1', name: '腾讯上海一区', is_active: true },
      { slug: 'tencent-shanghai-5', name: '腾讯上海五区', is_active: true },
    ],
  }

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
        const body = overrides[`${method} ${u}`]
        if (body && body.__fail) {
          return Promise.resolve({
            ok: false,
            status: 400,
            json: () => Promise.resolve(body),
            traceId: body.trace_id || '',
          })
        }
        return jsonOk(body)
      }
      if (u.includes('/tenant-options/')) return jsonOk(TENANTS)
      if (u.includes('/gitlab-regions/')) return jsonOk(REGIONS)
      if (u.includes('/billing/membership/')) return jsonOk({ membership: { tier: 'normal', admin_tier_locked: false } })
      if (u.includes('admin_grant_points')) return jsonOk({ status: 'ok', task_post_quota_after: 1, grants: [] })
      return Promise.resolve({ ok: false, status: 404, json: async () => ({}), traceId: '' })
    })
  }

  describe('SystemAdminGrantPoints GitLab 区域', () => {
    beforeEach(() => {
      mocks.apiFetch.mockReset()
      mockApi()
    })

    it('选择 GitLab 磁盘后显示区域下拉', async () => {
      const wrapper = mount(SystemAdminGrantPoints)
      await flushPromises()
      expect(wrapper.find('[data-testid="grant-gitlab-region"]').exists()).toBe(false)
      const typeSelect = wrapper.findAll('select').at(0)
      await typeSelect.setValue('gitlab_disk')
      await flushPromises()
      const region = wrapper.find('[data-testid="grant-gitlab-region"]')
      expect(region.exists()).toBe(true)
      expect(wrapper.text()).toContain('腾讯上海一区')
      wrapper.unmount()
    })

    it('未选区域时不提交赠送', async () => {
      const wrapper = mount(SystemAdminGrantPoints)
      await flushPromises()
      await wrapper.find('input[type="text"]').trigger('focus')
      await flushPromises()
      await wrapper.find('li').trigger('mousedown')
      await wrapper.findAll('select').at(0).setValue('gitlab_disk')
      await flushPromises()
      // 数量须显式填写（默认留空）
      await wrapper.find('[data-testid="grant-quantity-input"]').setValue('1')
      await wrapper.find('form').trigger('submit')
      await flushPromises()
      const posts = mocks.apiFetch.mock.calls.filter((c) => String(c[1]?.method || '').toUpperCase() === 'POST')
      expect(posts.length).toBe(0)
      expect(wrapper.text()).toContain('请选择 GitLab 区域')
      wrapper.unmount()
    })

    it('赠送磁盘时 POST body 含所选 region', async () => {
      mockApi({
        'POST /api/tenant/873472655125147648/billing/accounts/admin_grant_points/': {
          status: 'ok',
          task_post_quota_after: 1,
          grants: [{ resource_type: 'gitlab_disk', quantity: 1, region: 'tencent-sh-1' }],
        },
      })
      const wrapper = mount(SystemAdminGrantPoints)
      await flushPromises()
      await wrapper.find('input[type="text"]').trigger('focus')
      await flushPromises()
      await wrapper.find('li').trigger('mousedown')
      await wrapper.findAll('select').at(0).setValue('gitlab_disk')
      await flushPromises()
      await wrapper.find('[data-testid="grant-gitlab-region"]').setValue('tencent-sh-1')
      // 数量须显式填写（默认留空）
      await wrapper.find('[data-testid="grant-quantity-input"]').setValue('1')
      await wrapper.find('form').trigger('submit')
      await flushPromises()
      const post = mocks.apiFetch.mock.calls.find((c) => String(c[1]?.method || '').toUpperCase() === 'POST')
      expect(post).toBeTruthy()
      expect(String(post[0])).toContain('admin_grant_points')
      expect(post[1].body).toContain('"resource_type":"gitlab_disk"')
      expect(post[1].body).toContain('"region":"tencent-sh-1"')
      expect(post[1].body).not.toContain('membership_tier')
      expect(wrapper.text()).toContain('赠送成功')
      expect(wrapper.text()).toContain('tencent-sh-1')
      wrapper.unmount()
    })

    it('任务帖不展示区域且 body 不含 region', async () => {
      const wrapper = mount(SystemAdminGrantPoints)
      await flushPromises()
      await wrapper.find('input[type="text"]').trigger('focus')
      await flushPromises()
      await wrapper.find('li').trigger('mousedown')
      expect(wrapper.find('[data-testid="grant-gitlab-region"]').exists()).toBe(false)
      // 数量须显式填写（默认留空）
      await wrapper.find('[data-testid="grant-quantity-input"]').setValue('1')
      await wrapper.find('form').trigger('submit')
      await flushPromises()
      const post = mocks.apiFetch.mock.calls.find((c) => String(c[1]?.method || '').toUpperCase() === 'POST')
      expect(post).toBeTruthy()
      expect(post[1].body).not.toContain('"region"')
      wrapper.unmount()
    })
  })
}

// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] SystemAdminUsers.tenants-tab.test.js requires vitest runtime')
} else {
  const { describe, it, expect, vi, beforeEach } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')

  const { apiFetch } = vi.hoisted(() => ({
    apiFetch: vi.fn(),
  }))

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch,
  }))
  vi.mock('../utils/requestErrorDisplay.js', () => ({
    showRequestError: vi.fn(),
  }))
  vi.mock('../utils/safeResponseJson.js', () => ({
    safeResponseJson: async () => ({ data: { invitations: [] }, traceId: '' }),
  }))
  vi.mock('../utils/traceId.js', () => ({
    extractTraceId: () => '',
  }))
  vi.mock('vue-router', () => ({
    useRoute: () => ({ query: {} }),
    useRouter: () => ({ replace: vi.fn() }),
  }))

  vi.mock('../components/UserListRow.vue', () => ({
    default: { name: 'UserListRow', template: '<tr />', props: ['user'] },
  }))
  vi.mock('../components/SystemAdminUserRechargeDrawer.vue', () => ({
    default: { name: 'SystemAdminUserRechargeDrawer', template: '<div />' },
  }))
  vi.mock('../components/SystemAdminUserKycDrawer.vue', () => ({
    default: { name: 'SystemAdminUserKycDrawer', template: '<div />' },
  }))
  vi.mock('../components/SystemAdminReferralPerformanceDrawer.vue', () => ({
    default: { name: 'SystemAdminReferralPerformanceDrawer', template: '<div />' },
  }))

  const { default: SystemAdminUsers } = await import('../views/SystemAdminUsers.vue')

  describe('SystemAdminUsers tenants tab', () => {
    beforeEach(() => {
      apiFetch.mockReset()
      apiFetch.mockImplementation(async (url) => {
        if (String(url).includes('/accounts/admin/tenants/')) {
          return {
            ok: true,
            json: async () => ({
              items: [{ id: 'c1', name: 'Test Co', creator_id: 'admin1', email: 'a@x.com', phone: '', created_at: '' }],
              total: 1,
            }),
            headers: { get: () => null },
          }
        }
        if (String(url).includes('/email-invitations/')) {
          return {
            ok: true,
            json: async () => ({ invitations: [] }),
            headers: { get: () => null },
          }
        }
        return {
          ok: true,
          json: async () => ({ users: [], total: 0 }),
          headers: { get: () => null },
        }
      })
    })

    it('shows 租户 tab between 已归档 and 推荐码申请, and mounts panel when selected', async () => {
      const wrapper = mount(SystemAdminUsers, {
        global: { stubs: { Teleport: true } },
      })
      await flushPromises()

      const tabs = wrapper.get('[data-testid="system-admin-users-tabs"]')
      expect(tabs.text()).toContain('活跃用户')
      expect(tabs.text()).toContain('已归档')
      expect(tabs.text()).toContain('租户')
      expect(tabs.text()).toContain('推荐码申请')
      const text = tabs.text().replace(/\s+/g, '')
      expect(text.indexOf('已归档')).toBeLessThan(text.indexOf('租户'))
      expect(text.indexOf('租户')).toBeLessThan(text.indexOf('推荐码申请'))

      await wrapper.get('[data-testid="system-admin-users-tab-tenants"]').trigger('click')
      await flushPromises()

      expect(wrapper.find('[data-testid="system-admin-tenants-panel"]').exists()).toBe(true)
      expect(wrapper.text()).toContain('租户列表')
      expect(wrapper.get('[data-testid="system-admin-tenants-table"]').text()).toContain('Test Co')
    })
  })
}

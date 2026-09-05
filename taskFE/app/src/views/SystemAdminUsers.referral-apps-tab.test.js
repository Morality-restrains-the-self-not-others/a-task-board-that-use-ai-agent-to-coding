// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] SystemAdminUsers.referral-apps-tab.test.js requires vitest runtime')
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

  describe('SystemAdminUsers referral-apps tab', () => {
    beforeEach(() => {
      apiFetch.mockReset()
      apiFetch.mockImplementation(async (url) => {
        if (String(url).includes('/referral/applications/')) {
          return {
            ok: true,
            json: async () => ({ items: [], total: 0 }),
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

    it('shows 推荐码申请 tab and mounts applications panel when selected', async () => {
      const wrapper = mount(SystemAdminUsers, {
        global: {
          stubs: { Teleport: true },
        },
      })
      await flushPromises()

      const tabs = wrapper.get('[data-testid="system-admin-users-tabs"]')
      expect(tabs.text()).toContain('活跃用户')
      expect(tabs.text()).toContain('已归档')
      expect(tabs.text()).toContain('租户')
      expect(tabs.text()).toContain('推荐码申请')

      await wrapper.get('[data-testid="system-admin-users-tab-referral-apps"]').trigger('click')
      await flushPromises()

      expect(wrapper.find('[data-testid="system-admin-referral-applications-panel"]').exists()).toBe(true)
      expect(wrapper.text()).toContain('推荐码申请列表')
    })
  })
}

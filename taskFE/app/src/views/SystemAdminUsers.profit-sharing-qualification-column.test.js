// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] SystemAdminUsers.profit-sharing-qualification-column.test.js requires vitest runtime')
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

  vi.mock('../components/SystemAdminUserRechargeDrawer.vue', () => ({
    default: { name: 'SystemAdminUserRechargeDrawer', template: '<div />' },
  }))
  vi.mock('../components/SystemAdminUserKycDrawer.vue', () => ({
    default: { name: 'SystemAdminUserKycDrawer', template: '<div />' },
  }))
  vi.mock('../components/SystemAdminReferralPerformanceDrawer.vue', () => ({
    default: { name: 'SystemAdminReferralPerformanceDrawer', template: '<div />' },
  }))
  vi.mock('../components/SystemAdminReferralApplicationsPanel.vue', () => ({
    default: { name: 'SystemAdminReferralApplicationsPanel', template: '<div />' },
  }))
  vi.mock('../components/SystemAdminEmailInvitationsSection.vue', () => ({
    default: { name: 'SystemAdminEmailInvitationsSection', template: '<div />' },
  }))

  const { default: SystemAdminUsers } = await import('../views/SystemAdminUsers.vue')

  describe('SystemAdminUsers profit sharing qualification column', () => {
    beforeEach(() => {
      apiFetch.mockReset()
      apiFetch.mockResolvedValue({
        ok: true,
        json: async () => ({
          users: [{
            id: 'u1',
            email: 'a@example.com',
            phone: '',
            login_methods: [],
            has_profit_sharing_qualification: true,
            is_active: true,
            is_archived: false,
          }],
          total: 1,
        }),
        headers: { get: () => null },
      })
    })

    it('places 是否获得分账资格 after 角色 and before 操作', async () => {
      const wrapper = mount(SystemAdminUsers, {
        global: { stubs: { Teleport: true } },
      })
      await flushPromises()

      const headers = wrapper.findAll('thead th').map((th) => th.text())
      const roleIdx = headers.indexOf('角色')
      const qualIdx = headers.indexOf('是否获得分账资格')
      const actionIdx = headers.indexOf('操作')
      expect(roleIdx).toBeGreaterThanOrEqual(0)
      expect(qualIdx).toBe(roleIdx + 1)
      expect(actionIdx).toBe(qualIdx + 1)
    })
  })
}

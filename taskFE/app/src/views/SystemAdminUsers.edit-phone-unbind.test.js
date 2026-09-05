// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] SystemAdminUsers.edit-phone-unbind.test.js requires vitest runtime')
} else {
  const { describe, it, expect, vi, beforeEach } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')

  const hoisted = vi.hoisted(() => ({
    apiFetch: vi.fn(),
  }))

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch: (...args) => hoisted.apiFetch(...args),
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

  const sampleUser = {
    id: '877397583960502272',
    email: 'contact@daydaymoney.com',
    phone: '18959264502',
    username: '软刀',
    login_methods: [{ method_type: 'phone', identifier: '18959264502' }],
    date_joined: '2026-08-01T00:00:00Z',
    last_login: '2026-08-21T10:30:00Z',
    is_active: true,
    is_archived: false,
    is_superuser: false,
    is_staff: false,
    is_tenant: true,
  }

  describe('SystemAdminUsers edit phone unbind', () => {
    beforeEach(() => {
      hoisted.apiFetch.mockReset()
      hoisted.apiFetch.mockImplementation((url, opts) => {
        if (opts?.method === 'PUT') {
          return Promise.resolve({
            ok: true,
            status: 200,
            json: async () => ({ ...sampleUser, phone: '' }),
            headers: { get: () => null },
          })
        }
        return Promise.resolve({
          ok: true,
          json: async () => ({ users: [sampleUser], total: 1 }),
          headers: { get: () => null },
        })
      })
    })

    it('clears the phone field when 解绑 is clicked', async () => {
      const wrapper = mount(SystemAdminUsers, {
        global: { stubs: { Teleport: true } },
      })
      await flushPromises()
      const editBtn = wrapper.findAll('button').find((b) => b.text() === '编辑')
      expect(editBtn).toBeTruthy()
      await editBtn.trigger('click')
      await flushPromises()

      expect(wrapper.get('#edit-phone').element.value).toBe('18959264502')
      await wrapper.get('[data-testid="unbind-phone-btn"]').trigger('click')
      expect(wrapper.get('#edit-phone').element.value).toBe('')
    })
  })
}

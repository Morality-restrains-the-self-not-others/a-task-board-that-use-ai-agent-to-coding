// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] SystemAdminUsers.superuser-checkbox.test.js requires vitest runtime')
} else {
  const { describe, it, expect, vi, beforeEach } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')

  const hoisted = vi.hoisted(() => ({
    apiFetch: vi.fn(),
    isPlatformRole: vi.fn(() => false),
    hasPlatformPerm: vi.fn(() => false),
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
  vi.mock('../composables/usePermissions.js', () => ({
    usePermissions: () => ({
      hasPlatformPerm: hoisted.hasPlatformPerm,
      isPlatformRole: hoisted.isPlatformRole,
      load: async () => {},
    }),
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
    id: 'u1',
    email: 'a@example.com',
    phone: '',
    username: 'alice',
    login_methods: [],
    date_joined: '2026-08-01T00:00:00Z',
    last_login: '2026-08-21T10:30:00Z',
    is_active: true,
    is_archived: false,
    is_superuser: false,
    is_staff: false,
    is_tenant: false,
  }

  describe('SystemAdminUsers superuser checkbox', () => {
    beforeEach(() => {
      hoisted.apiFetch.mockReset()
      hoisted.isPlatformRole.mockReset()
      hoisted.hasPlatformPerm.mockReset()
      hoisted.isPlatformRole.mockReturnValue(false)
      hoisted.hasPlatformPerm.mockReturnValue(false)
      hoisted.apiFetch.mockResolvedValue({
        ok: true,
        json: async () => ({ users: [sampleUser], total: 1 }),
        headers: { get: () => null },
      })
    })

    it('disables 超级用户 for non system admin', async () => {
      const wrapper = mount(SystemAdminUsers, {
        global: { stubs: { Teleport: true } },
      })
      await flushPromises()
      const editBtn = wrapper.findAll('button').find((b) => b.text() === '编辑')
      expect(editBtn).toBeTruthy()
      await editBtn.trigger('click')
      await flushPromises()
      const box = wrapper.get('#edit-is-superuser')
      expect(box.element.disabled).toBe(true)
    })

    it('enables 超级用户 for super_admin', async () => {
      hoisted.isPlatformRole.mockImplementation((r) => r === 'super_admin')
      const wrapper = mount(SystemAdminUsers, {
        global: { stubs: { Teleport: true } },
      })
      await flushPromises()
      const editBtn = wrapper.findAll('button').find((b) => b.text() === '编辑')
      await editBtn.trigger('click')
      await flushPromises()
      const box = wrapper.get('#edit-is-superuser')
      expect(box.element.disabled).toBe(false)
    })
  })
}

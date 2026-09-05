// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] SystemAdminUsers.add-phone-taken.test.js requires vitest runtime')
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
    extractTraceId: (source) => {
      if (source && typeof source === 'object' && source.trace_id) return source.trace_id
      return ''
    },
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
  const { showRequestError } = await import('../utils/requestErrorDisplay.js')

  const TRACE = 'a1b2c3d4-98e4-4eee-b954-64f0e049c38c'

  describe('SystemAdminUsers add phone taken', () => {
    beforeEach(() => {
      hoisted.apiFetch.mockReset()
      showRequestError.mockReset()
      hoisted.apiFetch.mockImplementation((url, opts) => {
        if (opts?.method === 'POST' && String(url).includes('/users/create/')) {
          return Promise.resolve({
            ok: false,
            status: 409,
            json: async () => ({
              status: 'error',
              error: '该手机号已被其他用户使用',
              detail: '该手机号已被其他用户使用',
              message: '该手机号已被其他用户使用',
              code: 'phone_taken',
              trace_id: TRACE,
            }),
            headers: {
              get: (name) => (String(name).toLowerCase() === 'x-trace-id' ? TRACE : null),
            },
          })
        }
        return Promise.resolve({
          ok: true,
          json: async () => ({ users: [], total: 0 }),
          headers: { get: () => null },
        })
      })
    })

    it('keeps the add modal open and shows a phone-field error with data-traceId', async () => {
      const wrapper = mount(SystemAdminUsers, {
        global: { stubs: { Teleport: true } },
      })
      await flushPromises()
      const addBtn = wrapper.findAll('button').find((b) => b.text() === '添加用户')
      expect(addBtn).toBeTruthy()
      await addBtn.trigger('click')
      await flushPromises()

      await wrapper.get('#add-username').setValue('新人')
      await wrapper.get('#add-email').setValue('new-user@test.com')
      await wrapper.get('#add-phone').setValue('13900001111')
      await wrapper.get('#add-password').setValue('secret12')
      await wrapper.get('form').trigger('submit')
      await flushPromises()

      expect(wrapper.find('#add-phone').exists()).toBe(true)
      const err = wrapper.get('[data-testid="add-phone-error"]')
      expect(err.text()).toContain('该手机号已被其他用户使用')
      expect(err.attributes('data-traceid') || err.attributes('data-traceId')).toBe(TRACE)
      expect(showRequestError).not.toHaveBeenCalled()
    })
  })
}

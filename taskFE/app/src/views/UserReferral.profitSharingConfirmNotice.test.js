// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] UserReferral.profitSharingConfirmNotice.test.js requires vitest runtime')
} else {
  const { beforeEach, describe, expect, it, vi } = await import('vitest')
  const { nextTick } = await import('vue')
  const { mount, flushPromises } = await import('@vue/test-utils')

  const hoisted = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
    getStoredUserIdMock: vi.fn(() => 'u1'),
    resolveAuthenticatedUserIdMock: vi.fn(async () => 'u1'),
    routeMock: { params: { tenant: '' }, query: {}, path: '/profile/referral/' },
    routerMock: { replace: vi.fn(() => Promise.resolve()) },
  }))

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch: (...args) => hoisted.apiFetchMock(...args),
  }))
  vi.mock('../utils/sessionUserIdUtils.js', () => ({
    getStoredUserId: () => hoisted.getStoredUserIdMock(),
    resolveAuthenticatedUserId: () => hoisted.resolveAuthenticatedUserIdMock(),
  }))
  vi.mock('vue-router', () => ({
    useRoute: () => hoisted.routeMock,
    useRouter: () => hoisted.routerMock,
  }))

  function jsonOk(body) {
    return Promise.resolve({
      ok: true,
      status: 200,
      json: async () => body,
      headers: { get: () => null },
    })
  }

  function stubStatus(extra) {
    hoisted.apiFetchMock.mockImplementation((url) => {
      if (String(url).includes('/referral-codes/status/')) {
        return jsonOk({
          access_code: 'abc123xyz',
          ...extra,
        })
      }
      if (String(url).includes('/referral/stats/')) {
        return jsonOk({ referral_count: 0, monthly_earnings: [] })
      }
      return jsonOk({})
    })
  }

  describe('UserReferral profit-sharing confirm notice T57', () => {
    beforeEach(() => {
      hoisted.apiFetchMock.mockReset()
    })

    it('shows the WeChat confirm-within-validity notice on the qualification card when unqualified', async () => {
      stubStatus({
        has_active_code: false,
        application_status: null,
        referral_rate_display: '5%',
      })
      const { default: UserReferral } = await import('./UserReferral.vue')
      const wrapper = mount(UserReferral, { global: { stubs: { UserCenterSidebar: true } } })
      await flushPromises()
      await nextTick()

      const notice = wrapper.get('[data-testid="referral-profit-sharing-confirm-notice"]')
      expect(notice.text()).toContain('有效期内')
      expect(notice.text()).toContain('点击确认')
      expect(notice.text()).toContain('无法补分')
    })

    it('keeps the notice visible after qualification is approved', async () => {
      stubStatus({
        has_active_code: true,
        application_status: 'approved',
        referral_rate_display: '5%',
        wechat_receiver_status: 'registered',
      })
      const { default: UserReferral } = await import('./UserReferral.vue')
      const wrapper = mount(UserReferral, { global: { stubs: { UserCenterSidebar: true } } })
      await flushPromises()
      await nextTick()

      expect(wrapper.get('[data-testid="referral-profit-sharing-confirm-notice"]').text()).toContain('无法补分')
    })
  })
}

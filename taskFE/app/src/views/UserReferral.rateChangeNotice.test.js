// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] UserReferral.rateChangeNotice.test.js requires vitest runtime')
} else {
  const { beforeEach, describe, expect, it, vi } = await import('vitest')
  const { nextTick } = await import('vue')
  const { mount, flushPromises } = await import('@vue/test-utils')

  const ACCESS_CODE = 'kN3pQ8xWm2'
  const NOTICE_SNIPPET = '固定为 5%'

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

  function stubStatus(status) {
    hoisted.apiFetchMock.mockImplementation((url) => {
      if (String(url).includes('/referral-codes/status/')) {
        return jsonOk({
          access_code: ACCESS_CODE,
          ...status,
        })
      }
      if (String(url).includes('/referral/stats/')) {
        return jsonOk({ referral_count: 0, monthly_earnings: [] })
      }
      return jsonOk({})
    })
  }

  async function mountReferral() {
    const { default: UserReferral } = await import('./UserReferral.vue')
    const wrapper = mount(UserReferral, { global: { stubs: { UserCenterSidebar: true } } })
    await flushPromises()
    await nextTick()
    return wrapper
  }

  describe('UserReferral rate change notice T56', () => {
    beforeEach(() => {
      hoisted.apiFetchMock.mockReset()
    })

    it('shows order-time backend rate notice when the user has no qualification', async () => {
      stubStatus({
        has_active_code: false,
        application_status: null,
        referral_rate_display: '12%',
      })
      const wrapper = await mountReferral()
      const notices = wrapper.findAll('[data-testid="referral-rate-change-notice"]')
      expect(notices.length).toBeGreaterThanOrEqual(1)
      expect(notices[0].text()).toContain('固定为 5%')
      expect(notices[0].text()).toContain(NOTICE_SNIPPET)
      expect(wrapper.get('[data-testid="referral-rate-apply-copy"]').text()).toBe('12%')
    })

    it('shows the same notice next to the qualified rate copy', async () => {
      stubStatus({
        has_active_code: true,
        application_status: 'approved',
        referral_rate_display: '8%',
        wechat_receiver_status: 'registered',
      })
      const wrapper = await mountReferral()
      expect(wrapper.get('[data-testid="referral-rate-copy"]').text()).toBe('8%')
      const notices = wrapper.findAll('[data-testid="referral-rate-change-notice"]')
      expect(notices.length).toBeGreaterThanOrEqual(2)
      expect(notices.every((n) => n.text().includes(NOTICE_SNIPPET))).toBe(true)
    })
  })
}

// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] UserReferral.applyIntro.test.js requires vitest runtime')
} else {
  const { beforeEach, describe, expect, it, vi } = await import('vitest')
  const { nextTick } = await import('vue')
  const { mount, flushPromises } = await import('@vue/test-utils')

  const VALID_INTRO = '我是平台活跃用户，日常使用任务与云主机，希望通过推荐帮助同事上手本平台。'

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

  describe('UserReferral apply personal intro', () => {
    beforeEach(() => {
      hoisted.apiFetchMock.mockReset()
      vi.stubGlobal('crypto', { randomUUID: () => 'ik-apply-1' })
      hoisted.apiFetchMock.mockImplementation((url) => {
        if (String(url).includes('/referral-codes/status/')) {
          return jsonOk({
            application_status: null,
            has_active_code: false,
            access_code: 'abc123xyz',
            service_account_bound: true,
          })
        }
        if (String(url).includes('/referral/stats/')) {
          return jsonOk({ referral_count: 0, monthly_earnings: [] })
        }
        if (String(url).includes('/referral-codes/apply/')) {
          return jsonOk({ status: 'pending', message: '申请已提交' })
        }
        return jsonOk({})
      })
    })

    it('POSTs personal_intro and Idempotency-Key then shows pending copy', async () => {
      const { default: UserReferral } = await import('./UserReferral.vue')
      const wrapper = mount(UserReferral, { global: { stubs: { UserCenterSidebar: true } } })
      await flushPromises()
      await nextTick()

      expect(wrapper.text()).toContain('分成名额有限')
      expect(wrapper.text()).toContain('账号标识')
      expect(wrapper.text()).toContain('用户身份信息')
      await wrapper.get('[data-testid="referral-personal-intro"]').setValue(VALID_INTRO)
      await wrapper.get('[data-testid="referral-legal-name"]').setValue('张三')
      await wrapper.get('[data-testid="referral-identity-bind-consent"]').setValue(true)
      await wrapper.get('[data-testid="referral-apply-submit"]').trigger('click')
      await flushPromises()

      const applyCall = hoisted.apiFetchMock.mock.calls.find((c) => String(c[0]).includes('/referral-codes/apply/'))
      expect(applyCall).toBeTruthy()
      expect(applyCall[1].method).toBe('POST')
      expect(applyCall[1].headers['Idempotency-Key']).toBe('ik-apply-1')
      expect(JSON.parse(applyCall[1].body)).toEqual({
        personal_intro: VALID_INTRO,
        legal_name: '张三',
        identity_bind_consent: true,
      })
      expect(wrapper.text()).toContain('资格审批中')
    })
  })
}

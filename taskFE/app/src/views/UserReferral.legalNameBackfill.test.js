// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] UserReferral.legalNameBackfill.test.js requires vitest runtime')
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
    statusLegalName: '', // 先空，补填成功后回显
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

  describe('UserReferral legal name backfill (OPT-20260823-048)', () => {
    beforeEach(() => {
      hoisted.apiFetchMock.mockReset()
      hoisted.statusLegalName = ''
      vi.stubGlobal('crypto', { randomUUID: () => 'ik-legal-1' })
      hoisted.apiFetchMock.mockImplementation((url) => {
        if (String(url).includes('/referral-codes/status/')) {
          return jsonOk({
            application_status: 'approved',
            has_active_code: true,
            access_code: 'abc123xyz',
            expires_in_days: 120,
            legal_name: hoisted.statusLegalName,
            wechat_receiver_status: hoisted.statusLegalName ? 'registered' : 'pending_openid',
          })
        }
        if (String(url).includes('/referral/stats/')) {
          return jsonOk({ referral_count: 0, monthly_earnings: [] })
        }
        if (String(url).includes('/referral/legal-name/')) {
          hoisted.statusLegalName = '王小明'
          return jsonOk({
            status: 'ok',
            legal_name: '王小明',
            wechat_receiver_status: 'registered',
          })
        }
        return jsonOk({})
      })
    })

    it('shows backfill form for active code with empty legal_name', async () => {
      const { default: UserReferral } = await import('./UserReferral.vue')
      const wrapper = mount(UserReferral, { global: { stubs: { UserCenterSidebar: true } } })
      await flushPromises()
      await nextTick()

      expect(wrapper.get('[data-testid="referral-legal-name-backfill"]').exists()).toBe(true)
      expect(wrapper.text()).toContain('请补填与微信实名认证完全一致的姓名')
      // 未输入时不提交
      expect(wrapper.get('[data-testid="referral-legal-name-backfill-submit"]').attributes('disabled')).toBeDefined()
    })

    it('POSTs legal_name with Idempotency-Key, hides form after success and shows receiver registered', async () => {
      const { default: UserReferral } = await import('./UserReferral.vue')
      const wrapper = mount(UserReferral, { global: { stubs: { UserCenterSidebar: true } } })
      await flushPromises()
      await nextTick()

      await wrapper.get('[data-testid="referral-legal-name-backfill-input"]').setValue(' 王小明 ')
      await wrapper.get('[data-testid="referral-legal-name-backfill-submit"]').trigger('click')
      await flushPromises()

      const updateCall = hoisted.apiFetchMock.mock.calls.find((c) => String(c[0]).includes('/referral/legal-name/'))
      expect(updateCall).toBeTruthy()
      expect(updateCall[1].method).toBe('POST')
      expect(updateCall[1].headers['Idempotency-Key']).toBe('ik-legal-1')
      expect(JSON.parse(updateCall[1].body)).toEqual({ legal_name: '王小明' })
      expect(wrapper.text()).toContain('个人名称已保存')

      // 回显新名称后补填表单消失，接收方状态更新为已登记
      await flushPromises()
      await nextTick()
      expect(wrapper.find('[data-testid="referral-legal-name-backfill"]').exists()).toBe(false)
      expect(wrapper.get('[data-testid="wechat-receiver-registered"]').exists()).toBe(true)
    })
  })
}

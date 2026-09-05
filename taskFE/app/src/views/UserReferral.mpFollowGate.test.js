// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] UserReferral.mpFollowGate.test.js requires vitest runtime')
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

  describe('UserReferral MP follow gate', () => {
    beforeEach(() => {
      hoisted.apiFetchMock.mockReset()
      vi.stubGlobal('crypto', { randomUUID: () => 'ik-mp-1' })
      hoisted.apiFetchMock.mockImplementation((url) => {
        if (String(url).includes('/referral-codes/status/')) {
          return jsonOk({
            application_status: null,
            has_active_code: false,
            access_code: 'abc',
            service_account_bound: false,
          })
        }
        if (String(url).includes('/wechat/mp/follow-qr/')) {
          return jsonOk({
            temp_id: '8803',
            qr_src: 'https://mp.weixin.qq.com/cgi-bin/showqrcode?ticket=TICKET',
            expires_at: '2026-08-27T15:50:00+08:00',
          })
        }
        return jsonOk({})
      })
    })

    it('hides apply form until service account is bound', async () => {
      const { default: UserReferral } = await import('./UserReferral.vue')
      const wrapper = mount(UserReferral, { global: { stubs: { UserCenterSidebar: true } } })
      await flushPromises()
      await nextTick()
      expect(wrapper.find('[data-testid="referral-mp-follow-gate"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="referral-apply-submit"]').exists()).toBe(false)
      expect(wrapper.get('[data-testid="referral-mp-qr"]').attributes('src')).toContain('showqrcode')
    })

    it('shows apply form after 我已关注 returns bound', async () => {
      hoisted.apiFetchMock.mockImplementation((url) => {
        if (String(url).includes('/referral-codes/status/')) {
          return jsonOk({
            application_status: null,
            has_active_code: false,
            access_code: 'abc',
            service_account_bound: false,
          })
        }
        if (String(url).includes('/wechat/mp/follow-qr/')) {
          return jsonOk({
            temp_id: '8803',
            qr_src: 'https://mp.weixin.qq.com/cgi-bin/showqrcode?ticket=TICKET',
            expires_at: '2026-08-27T15:50:00+08:00',
          })
        }
        if (String(url).includes('/wechat/mp/follow-status/')) {
          return jsonOk({ bound: true, has_unionid: true })
        }
        return jsonOk({})
      })
      const { default: UserReferral } = await import('./UserReferral.vue')
      const wrapper = mount(UserReferral, { global: { stubs: { UserCenterSidebar: true } } })
      await flushPromises()
      await nextTick()
      await wrapper.get('[data-testid="referral-mp-followed-btn"]').trigger('click')
      await flushPromises()
      await nextTick()
      expect(wrapper.find('[data-testid="referral-apply-submit"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="referral-mp-follow-gate"]').exists()).toBe(false)
    })

    it('tells phone-only users to bind wechat when has_unionid is false', async () => {
      hoisted.apiFetchMock.mockImplementation((url) => {
        if (String(url).includes('/referral-codes/status/')) {
          return jsonOk({
            application_status: null,
            has_active_code: false,
            access_code: 'abc',
            service_account_bound: false,
          })
        }
        if (String(url).includes('/wechat/mp/follow-qr/')) {
          return jsonOk({
            temp_id: '8803',
            qr_src: 'https://mp.weixin.qq.com/cgi-bin/showqrcode?ticket=TICKET',
            expires_at: '2026-08-27T15:50:00+08:00',
          })
        }
        if (String(url).includes('/wechat/mp/follow-status/')) {
          return jsonOk({ bound: false, has_unionid: false })
        }
        return jsonOk({})
      })
      const { default: UserReferral } = await import('./UserReferral.vue')
      const wrapper = mount(UserReferral, { global: { stubs: { UserCenterSidebar: true } } })
      await flushPromises()
      await nextTick()
      await wrapper.get('[data-testid="referral-mp-followed-btn"]').trigger('click')
      await flushPromises()
      await nextTick()
      expect(wrapper.find('[data-testid="referral-apply-submit"]').exists()).toBe(false)
      expect(wrapper.get('[data-testid="referral-mp-follow-error"]').text()).toContain('尚未绑定微信登录')
    })

    it('shows server conflict message without leaking other user id', async () => {
      hoisted.apiFetchMock.mockImplementation((url) => {
        if (String(url).includes('/referral-codes/status/')) {
          return jsonOk({
            application_status: null,
            has_active_code: false,
            access_code: 'abc',
            service_account_bound: false,
          })
        }
        if (String(url).includes('/wechat/mp/follow-qr/')) {
          return jsonOk({
            temp_id: '8803',
            qr_src: 'https://mp.weixin.qq.com/cgi-bin/showqrcode?ticket=TICKET',
            expires_at: '2026-08-27T15:50:00+08:00',
          })
        }
        if (String(url).includes('/wechat/mp/follow-status/')) {
          return jsonOk({
            bound: false,
            has_unionid: true,
            ticket_status: 'conflict',
            conflict_code: 'unionid_bound_other',
            message: '该微信已绑定其他账号。请用已绑定该微信的账号登录后再扫码。',
          })
        }
        return jsonOk({})
      })
      const { default: UserReferral } = await import('./UserReferral.vue')
      const wrapper = mount(UserReferral, { global: { stubs: { UserCenterSidebar: true } } })
      await flushPromises()
      await nextTick()
      await wrapper.get('[data-testid="referral-mp-followed-btn"]').trigger('click')
      await flushPromises()
      await nextTick()
      const text = wrapper.get('[data-testid="referral-mp-follow-error"]').text()
      expect(text).toContain('已绑定其他账号')
      expect(text).not.toMatch(/\d{15,}/)
      expect(wrapper.find('[data-testid="referral-apply-submit"]').exists()).toBe(false)
    })

    it('asks already-followed users to scan this page QR again', async () => {
      hoisted.apiFetchMock.mockImplementation((url) => {
        if (String(url).includes('/referral-codes/status/')) {
          return jsonOk({
            application_status: null,
            has_active_code: false,
            access_code: 'abc',
            service_account_bound: false,
          })
        }
        if (String(url).includes('/wechat/mp/follow-qr/')) {
          return jsonOk({
            temp_id: '8803',
            qr_src: 'https://mp.weixin.qq.com/cgi-bin/showqrcode?ticket=TICKET',
            expires_at: '2026-08-27T15:50:00+08:00',
          })
        }
        if (String(url).includes('/wechat/mp/follow-status/')) {
          return jsonOk({ bound: false, has_unionid: true, ticket_status: 'none' })
        }
        return jsonOk({})
      })
      const { default: UserReferral } = await import('./UserReferral.vue')
      const wrapper = mount(UserReferral, { global: { stubs: { UserCenterSidebar: true } } })
      await flushPromises()
      await nextTick()
      await wrapper.get('[data-testid="referral-mp-followed-btn"]').trigger('click')
      await flushPromises()
      await nextTick()
      expect(wrapper.get('[data-testid="referral-mp-follow-error"]').text()).toContain('再扫一次本页服务号二维码')
    })
  })
}

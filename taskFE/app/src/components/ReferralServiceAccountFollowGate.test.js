// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] ReferralServiceAccountFollowGate.test.js requires vitest runtime')
} else {
  const { describe, it, expect, vi, beforeEach } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')

  const hoisted = vi.hoisted(() => ({
    getActiveTokenMock: vi.fn(async () => 'tok-bind-1'),
    apiFetchMock: vi.fn(),
  }))

  vi.mock('../domain/auth/services/saved_accounts_store.js', () => ({
    getActiveToken: (...args) => hoisted.getActiveTokenMock(...args),
  }))
  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch: (...args) => hoisted.apiFetchMock(...args),
  }))

  const { default: Gate } = await import('./ReferralServiceAccountFollowGate.vue')

  function jsonOk(body) {
    return Promise.resolve({
      ok: true,
      status: 200,
      json: async () => body,
      headers: { get: () => null },
    })
  }

  describe('ReferralServiceAccountFollowGate', () => {
    beforeEach(() => {
      vi.stubGlobal('crypto', { randomUUID: () => 'ik-follow-1' })
      hoisted.getActiveTokenMock.mockReset()
      hoisted.getActiveTokenMock.mockResolvedValue('tok-bind-1')
      hoisted.apiFetchMock.mockReset()
      hoisted.apiFetchMock.mockImplementation((url) => {
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

    it('shows dynamic QR and hides slot until bound', async () => {
      const wrapper = mount(Gate, {
        props: { bound: false },
        slots: { default: '<form data-testid="apply-slot">申请表</form>' },
      })
      await flushPromises()
      expect(wrapper.get('[data-testid="referral-mp-follow-gate"]').exists()).toBe(true)
      expect(wrapper.get('[data-testid="referral-mp-qr"]').attributes('src')).toContain('showqrcode')
      expect(wrapper.find('[data-testid="apply-slot"]').exists()).toBe(false)
      expect(wrapper.text()).toContain('请先使用微信扫描')
      expect(wrapper.text()).toContain('再扫一次本页二维码')
      expect(wrapper.get('[data-testid="referral-mp-bind-wechat-btn"]').exists()).toBe(true)
      const qrCall = hoisted.apiFetchMock.mock.calls.find((c) => String(c[0]).includes('/follow-qr/'))
      expect(qrCall).toBeTruthy()
      expect(qrCall[1].method).toBe('POST')
      expect(qrCall[1].headers['Idempotency-Key']).toBe('ik-follow-1')
    })

    it('renders slot when bound', () => {
      const wrapper = mount(Gate, {
        props: { bound: true },
        slots: { default: '<form data-testid="apply-slot">申请表</form>' },
      })
      expect(wrapper.find('[data-testid="referral-mp-follow-gate"]').exists()).toBe(false)
      expect(wrapper.get('[data-testid="apply-slot"]').exists()).toBe(true)
      expect(hoisted.apiFetchMock).not.toHaveBeenCalled()
    })

    it('emits confirm from 我已关注 with click guard', async () => {
      const wrapper = mount(Gate, {
        props: { bound: false },
      })
      await flushPromises()
      await wrapper.get('[data-testid="referral-mp-followed-btn"]').trigger('click')
      await flushPromises()
      expect(wrapper.emitted('confirm')).toHaveLength(1)
    })

    it('navigates to wechat bind after writing token cookie', async () => {
      const wrapper = mount(Gate, {
        props: { bound: false },
      })
      await flushPromises()
      const loc = { href: 'http://localhost/profile/referral/' }
      vi.stubGlobal('location', loc)
      await wrapper.get('[data-testid="referral-mp-bind-wechat-btn"]').trigger('click')
      await flushPromises()
      expect(hoisted.getActiveTokenMock).toHaveBeenCalled()
      expect(wrapper.find('[data-testid="referral-mp-bind-error"]').exists()).toBe(false)
      expect(loc.href).toBe('/api/auth/wechat/bind/?app=web&next=%2Fprofile%2Freferral%2F')
    })

    it('binds data-traceId on error', async () => {
      const wrapper = mount(Gate, {
        props: {
          bound: false,
          error: '该微信已绑定其他账号。请用已绑定该微信的账号登录后再扫码。',
          errorTraceId: 'tid-mp-1',
        },
      })
      await flushPromises()
      const el = wrapper.get('[data-testid="referral-mp-follow-error"]')
      expect(el.text()).toContain('已绑定其他账号')
      expect(el.attributes('data-traceid') || el.attributes('data-traceId')).toBe('tid-mp-1')
    })
  })
}

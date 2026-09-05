// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] UnsubscribeConfirm.test.js requires vitest runtime')
} else {
  const { describe, expect, it, vi, beforeEach } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')

  const hoisted = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
    routeMock: { query: { ok: '1', token: 'tok-abc' } },
  }))

  vi.mock('../utils/apiUtils', () => ({
    apiFetch: (...args) => hoisted.apiFetchMock(...args),
  }))
  vi.mock('vue-router', () => ({
    useRoute: () => hoisted.routeMock,
  }))
  vi.mock('../utils/clickGuard.js', () => ({
    createClickGuard: () => ({
      run: async (fn) => fn({ idempotencyKey: 'ik-test' }),
      isBusy: () => false,
    }),
    mergeIdempotencyHeaders: (headers, key) => ({ ...(headers || {}), 'Idempotency-Key': key }),
  }))

  const { default: UnsubscribeConfirm } = await import('./UnsubscribeConfirm.vue')

  function okResponse(body) {
    return { ok: true, status: 200, json: async () => body }
  }

  beforeEach(() => {
    vi.clearAllMocks()
    window.apiFetch = hoisted.apiFetchMock
    hoisted.apiFetchMock.mockReset()
  })

  describe('UnsubscribeConfirm', () => {
    it('ok=1 展示已退订确认文案', () => {
      hoisted.routeMock.query = { ok: '1' }
      const wrapper = mount(UnsubscribeConfirm)
      expect(wrapper.text()).toContain('您已退订邮件邀请')
      expect(wrapper.find('a[href="/auth/login/"]').exists()).toBe(true)
    })

    it('ok=0 展示链接无效文案且无重新订阅按钮', () => {
      hoisted.routeMock.query = { ok: '0', token: 'tok-abc' }
      const wrapper = mount(UnsubscribeConfirm)
      expect(wrapper.text()).toContain('退订链接无效或已过期')
      expect(wrapper.find('[data-testid="email-resubscribe-btn"]').exists()).toBe(false)
    })

    it('ok=1 无 token 时不展示重新订阅按钮', () => {
      hoisted.routeMock.query = { ok: '1' }
      const wrapper = mount(UnsubscribeConfirm)
      expect(wrapper.find('[data-testid="email-resubscribe-btn"]').exists()).toBe(false)
    })

    it('点击重新订阅发送 POST 并携带 Idempotency-Key，成功后展示恢复文案', async () => {
      hoisted.routeMock.query = { ok: '1', token: 'tok-abc' }
      hoisted.apiFetchMock.mockResolvedValue(okResponse({ ok: true, resubscribed: true }))
      const wrapper = mount(UnsubscribeConfirm)
      const btn = wrapper.find('[data-testid="email-resubscribe-btn"]')
      expect(btn.exists()).toBe(true)
      await btn.trigger('click')
      await flushPromises()

      expect(hoisted.apiFetchMock).toHaveBeenCalledTimes(1)
      const [url, opts] = hoisted.apiFetchMock.mock.calls[0]
      expect(url).toContain('/api/public/email-resubscribe/?token=tok-abc')
      expect(opts.method).toBe('POST')
      expect(opts.headers['Idempotency-Key']).toBe('ik-test')
      expect(wrapper.text()).toContain('您已恢复接收邀请邮件')
    })

    it('重新订阅失败展示错误信息', async () => {
      hoisted.routeMock.query = { ok: '1', token: 'tok-abc' }
      hoisted.apiFetchMock.mockResolvedValue({
        ok: false,
        status: 400,
        json: async () => ({ message: '链接无效' }),
      })
      const wrapper = mount(UnsubscribeConfirm)
      await wrapper.find('[data-testid="email-resubscribe-btn"]').trigger('click')
      await flushPromises()
      expect(wrapper.text()).toContain('链接无效')
    })
  })
}

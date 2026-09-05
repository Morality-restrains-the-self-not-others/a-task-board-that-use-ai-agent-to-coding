// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] ResetPasswordRequest.click-guard.test.js requires vitest runtime')
} else {
  const { describe, it, expect, vi, beforeEach } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')

  const { apiFetch } = vi.hoisted(() => ({
    apiFetch: vi.fn(),
  }))
  const { push } = vi.hoisted(() => ({ push: vi.fn() }))

  vi.mock('../utils/apiUtils.js', () => ({ apiFetch }))
  vi.mock('vue-router', () => ({ useRouter: () => ({ push }) }))
  vi.mock('../utils/cookieUtils.js', () => ({ getCookie: vi.fn(() => '') }))
  // OPT-20260819-038: 让 guard 直接放行并把幂等键透传给 mergeIdempotencyHeaders，
  // 断言重置链接/验证码 POST 携带 Idempotency-Key 头。
  vi.mock('../utils/clickGuard.js', () => ({
    createClickGuard: () => ({
      run: async (fn) => fn({ idempotencyKey: 'ik-reset-1' }),
    }),
    mergeIdempotencyHeaders: (headers, key) => ({ ...(headers || {}), 'Idempotency-Key': key }),
  }))
  vi.mock('../utils/safeResponseJson.js', () => ({
    safeResponseJson: async () => ({ data: {}, traceId: '' }),
  }))

  const { default: ResetPasswordRequest } = await import('./ResetPasswordRequest.vue')

  function jsonOk() {
    return {
      ok: true,
      json: async () => ({}),
      headers: { get: () => 'application/json' },
    }
  }

  describe('ResetPasswordRequest 发送重置链接/验证码实施 clickGuard 接线', () => {
    beforeEach(() => {
      apiFetch.mockReset()
      push.mockReset()
      vi.stubGlobal('alert', vi.fn())
    })

    it('邮箱重置提交发起一次 POST 且携带 Idempotency-Key', async () => {
      apiFetch.mockImplementation(async () => jsonOk())
      const wrapper = mount(ResetPasswordRequest)

      await wrapper.find('input#email').setValue('test@example.com')
      await wrapper.find('form').trigger('submit')
      await flushPromises()

      const postCall = apiFetch.mock.calls.find(([url]) => url.includes('/send_password_reset_link/'))
      expect(postCall).toBeTruthy()
      expect(postCall[1].method).toBe('POST')
      expect(postCall[1].headers['Idempotency-Key']).toBe('ik-reset-1')
      expect(postCall[1].body).toContain('"email":"test@example.com"')
      wrapper.unmount()
    })

    it('手机验证码点击发起一次 POST 且携带 Idempotency-Key', async () => {
      apiFetch.mockImplementation(async () => jsonOk())
      const wrapper = mount(ResetPasswordRequest)

      await wrapper.find('input#phone').setValue('13800138000')
      await flushPromises()
      const codeBtn = wrapper.findAll('button').find((b) => b.text().includes('获取验证码'))
      expect(codeBtn).toBeTruthy()
      await codeBtn.trigger('click')
      await flushPromises()

      const postCall = apiFetch.mock.calls.find(([url]) => url.includes('/send_password_reset_code/'))
      expect(postCall).toBeTruthy()
      expect(postCall[1].method).toBe('POST')
      expect(postCall[1].headers['Idempotency-Key']).toBe('ik-reset-1')
      expect(postCall[1].body).toContain('"phone":"13800138000"')
      wrapper.unmount()
    })
  })
}

// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] UserProfileEmailBindingPanel.click-guard.test.js requires vitest runtime')
} else {
  const { describe, it, expect, vi, beforeEach } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')

  const { apiFetch } = vi.hoisted(() => ({
    apiFetch: vi.fn(),
  }))

  vi.mock('../utils/apiUtils', () => ({ apiFetch }))
  // OPT-20260819-038: 让 guard 直接放行并把幂等键透传给 mergeIdempotencyHeaders，
  // 断言发送验证码/绑定邮箱 POST 携带 Idempotency-Key 头。
  vi.mock('../utils/clickGuard.js', () => ({
    createClickGuard: () => ({
      run: async (fn) => fn({ idempotencyKey: 'ik-email-1' }),
    }),
    mergeIdempotencyHeaders: (headers, key) => ({ ...(headers || {}), 'Idempotency-Key': key }),
  }))

  const { default: Panel } = await import('./UserProfileEmailBindingPanel.vue')

  function jsonOk(body) {
    return {
      ok: true,
      json: async () => body,
      headers: { get: () => 'application/json' },
    }
  }

  describe('UserProfileEmailBindingPanel 发送验证码/绑定邮箱实施 clickGuard 接线', () => {
    beforeEach(() => {
      apiFetch.mockReset()
    })

    it('发送验证码 POST 携带 Idempotency-Key', async () => {
      apiFetch.mockImplementation(async () => jsonOk({ message: 'ok' }))
      const wrapper = mount(Panel, { props: { hasEmail: false, email: '' } })
      await wrapper.find('input[type="email"]').setValue('vendor@example.com')
      await wrapper.findAll('button').find((b) => b.text().includes('发送验证码')).trigger('click')
      await flushPromises()
      const call = apiFetch.mock.calls.find(([url]) => url.includes('/send_verification_code/'))
      expect(call).toBeTruthy()
      expect(call[1].headers['Idempotency-Key']).toBe('ik-email-1')
      expect(call[1].body).toContain('"email":"vendor@example.com"')
      wrapper.unmount()
    })

    it('绑定邮箱 POST 携带 Idempotency-Key', async () => {
      apiFetch.mockImplementation(async () => jsonOk({ bound: true, email: 'vendor@example.com' }))
      const wrapper = mount(Panel, { props: { hasEmail: false, email: '' } })
      await wrapper.find('input[type="email"]').setValue('vendor@example.com')
      await wrapper.find('input[autocomplete="one-time-code"]').setValue('123456')
      await wrapper.findAll('button').find((b) => b.text().includes('验证并绑定')).trigger('click')
      await flushPromises()
      const call = apiFetch.mock.calls.find(([url]) => url.includes('/bind_email/'))
      expect(call).toBeTruthy()
      expect(call[1].headers['Idempotency-Key']).toBe('ik-email-1')
      expect(call[1].body).toContain('"code":"123456"')
      wrapper.unmount()
    })
  })
}

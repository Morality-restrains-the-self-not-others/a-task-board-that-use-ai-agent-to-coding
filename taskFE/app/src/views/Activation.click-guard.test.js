// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] Activation.click-guard.test.js requires vitest runtime')
} else {
  const { describe, it, expect, vi, beforeEach } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')

  // Activation.vue 未 import apiFetch，靠 main.js window.apiFetch 全局（既有契约）
  const { apiFetch } = vi.hoisted(() => ({
    apiFetch: vi.fn(),
  }))
  window.apiFetch = apiFetch
  // OPT-20260819-038: 让 guard 直接放行并把幂等键透传给 mergeIdempotencyHeaders，
  // 断言 POST 携带 Idempotency-Key 头。
  vi.mock('../utils/clickGuard.js', () => ({
    createClickGuard: () => ({
      run: async (fn) => fn({ idempotencyKey: 'ik-test-activate' }),
    }),
    mergeIdempotencyHeaders: (headers, key) => ({ ...(headers || {}), 'Idempotency-Key': key }),
  }))

  const { default: View } = await import('./Activation.vue')

  function jsonOk(body) {
    return {
      ok: true,
      json: async () => body,
      headers: { get: () => 'application/json' },
    }
  }

  describe('Activation 写操作 clickGuard 接线', () => {
    beforeEach(() => {
      apiFetch.mockReset()
      apiFetch.mockImplementation(async () => jsonOk({ status: 'success', message: '激活成功' }))
    })

    it('激活账号 POST 携带 Idempotency-Key', async () => {
      const wrapper = mount(View, {
        global: {
          mocks: {
            $route: { params: { token: 'tok-1' } },
          },
        },
      })
      await flushPromises()

      const btn = wrapper.find('button.activate-button')
      expect(btn.exists()).toBe(true)
      await btn.trigger('click')
      await flushPromises()

      const postCall = apiFetch.mock.calls.find(([, opts]) => opts?.method === 'POST')
      expect(postCall).toBeTruthy()
      expect(postCall[0]).toBe('/api/accounts/users/confirm_activation/tok-1/')
      expect(postCall[1].headers['Idempotency-Key']).toBe('ik-test-activate')
    })
  })
}

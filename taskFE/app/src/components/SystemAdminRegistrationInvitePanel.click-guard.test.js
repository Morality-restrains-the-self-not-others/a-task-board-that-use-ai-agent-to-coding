// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] SystemAdminRegistrationInvitePanel.click-guard.test.js requires vitest runtime')
} else {
  const { describe, it, expect, vi, beforeEach } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')

  const { apiFetch } = vi.hoisted(() => ({
    apiFetch: vi.fn(),
  }))

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch,
  }))
  // OPT-20260819-038: 让 guard 直接放行并把幂等键透传给 mergeIdempotencyHeaders，
  // 断言 PUT 携带 Idempotency-Key 头。
  vi.mock('../utils/clickGuard.js', () => ({
    createClickGuard: () => ({
      run: async (fn) => fn({ idempotencyKey: 'ik-test-invite' }),
    }),
    mergeIdempotencyHeaders: (headers, key) => ({ ...(headers || {}), 'Idempotency-Key': key }),
  }))

  const { default: Component } = await import('./SystemAdminRegistrationInvitePanel.vue')

  function jsonOk(body) {
    return {
      ok: true,
      json: async () => body,
      headers: { get: () => 'application/json' },
    }
  }

  describe('SystemAdminRegistrationInvitePanel 写操作 clickGuard 接线', () => {
    beforeEach(() => {
      apiFetch.mockReset()
      apiFetch.mockImplementation(async (url, opts) => {
        if (opts?.method === 'GET' || !opts?.method) {
          return jsonOk({ enabled: false, daily_quota: 10, remaining_today: 10 })
        }
        return jsonOk({ enabled: false, daily_quota: 10, remaining_today: 10 })
      })
    })

    it('保存注册邀请策略 PUT 携带 Idempotency-Key', async () => {
      const wrapper = mount(Component, {})
      await flushPromises()

      const saveBtn = wrapper.find('[data-testid="registration-invite-policy-save"]')
      expect(saveBtn.exists()).toBe(true)
      await saveBtn.trigger('click')
      await flushPromises()

      const putCall = apiFetch.mock.calls.find(([, opts]) => opts?.method === 'PUT')
      expect(putCall).toBeTruthy()
      expect(putCall[0]).toBe('/api/system-admin/registration-invite-policy/')
      expect(putCall[1].headers['Idempotency-Key']).toBe('ik-test-invite')
      expect(putCall[1].body).toContain('"daily_quota"')
    })
  })
}

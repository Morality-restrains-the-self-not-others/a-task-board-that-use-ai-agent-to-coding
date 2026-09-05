// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] SystemAdminUserKycDrawer.click-guard.test.js requires vitest runtime')
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
  // 断言 POST 携带 Idempotency-Key 头。
  vi.mock('../utils/clickGuard.js', () => ({
    createClickGuard: () => ({
      run: async (fn) => fn({ idempotencyKey: 'ik-test-1' }),
    }),
    mergeIdempotencyHeaders: (headers, key) => ({ ...(headers || {}), 'Idempotency-Key': key }),
  }))

  const { default: Comp } = await import('./SystemAdminUserKycDrawer.vue')

  function jsonOk(body) {
    return {
      ok: true,
      json: async () => body,
      headers: { get: () => 'application/json' },
    }
  }

  describe('SystemAdminUserKycDrawer 写操作 clickGuard 接线', () => {
    beforeEach(() => {
      apiFetch.mockReset()
      apiFetch.mockImplementation(async (url, opts) => {
        if (!opts?.method) return jsonOk({ profile: {}, audit: [], latest_aml: null, limit_policies: [] })
        return jsonOk({ ok: true })
      })
    })

    it('触发自动评估 POST 携带 Idempotency-Key', async () => {
      const wrapper = mount(Comp, { props: { visible: true, userId: 'u-1' } })
      await flushPromises()
      const evalBtn = wrapper.findAll('button').find((b) => b.text().includes('触发自动评估'))
      await evalBtn.trigger('click')
      await flushPromises()

      const postCall = apiFetch.mock.calls.find(([, opts]) => opts?.method === 'POST')
      expect(postCall).toBeTruthy()
      expect(postCall[0]).toBe('/api/kyc/admin/users/u-1/evaluate/')
      expect(postCall[1].headers['Idempotency-Key']).toBe('ik-test-1')
    })

    it('人工覆盖 POST 携带 Idempotency-Key', async () => {
      const wrapper = mount(Comp, { props: { visible: true, userId: 'u-1' } })
      await flushPromises()
      const overrideBtn = wrapper.findAll('button').find((b) => b.text().includes('提交覆盖'))
      await overrideBtn.trigger('click')
      await flushPromises()

      const postCall = apiFetch.mock.calls.find(([, opts]) => opts?.method === 'POST')
      expect(postCall).toBeTruthy()
      expect(postCall[0]).toBe('/api/kyc/admin/users/u-1/override/')
      expect(postCall[1].headers['Idempotency-Key']).toBe('ik-test-1')
    })
  })
}

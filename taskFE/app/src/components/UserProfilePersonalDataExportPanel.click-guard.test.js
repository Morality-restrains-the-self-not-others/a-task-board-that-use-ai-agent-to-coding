// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] UserProfilePersonalDataExportPanel.click-guard.test.js requires vitest runtime')
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
      run: async (fn) => fn({ idempotencyKey: 'ik-test-export' }),
    }),
    mergeIdempotencyHeaders: (headers, key) => ({ ...(headers || {}), 'Idempotency-Key': key }),
  }))

  const { default: Component } = await import('./UserProfilePersonalDataExportPanel.vue')

  function jsonOk(body) {
    return {
      ok: true,
      json: async () => body,
      headers: { get: () => 'application/json' },
    }
  }

  describe('UserProfilePersonalDataExportPanel 写操作 clickGuard 接线', () => {
    beforeEach(() => {
      apiFetch.mockReset()
      apiFetch.mockImplementation(async (url, opts) => {
        if (opts?.method === 'GET' || !opts?.method) {
          if (String(url).includes('/status/')) {
            return jsonOk({ status: 'none' })
          }
          return jsonOk({})
        }
        return jsonOk({ status: 'ready', export_id: 'exp-1' })
      })
    })

    it('请求生成导出文件 POST 携带 Idempotency-Key', async () => {
      const wrapper = mount(Component, {})
      await flushPromises()

      const btn = wrapper.findAll('button').find((b) => b.text().includes('生成导出文件'))
      expect(btn).toBeTruthy()
      await btn.trigger('click')
      await flushPromises()

      const postCall = apiFetch.mock.calls.find(([, opts]) => opts?.method === 'POST')
      expect(postCall).toBeTruthy()
      expect(postCall[0]).toBe('/api/accounts/users/me/personal-data-export/request/')
      expect(postCall[1].headers['Idempotency-Key']).toBe('ik-test-export')
    })
  })
}

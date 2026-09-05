// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] AccessTokenManagementPanel.click-guard.test.js requires vitest runtime')
} else {
  const { describe, it, expect, vi, beforeEach } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')

  const mocks = vi.hoisted(() => ({
    apiFetch: vi.fn(),
    humanizeRequestErrorMessage: vi.fn((msg) => msg),
  }))

  vi.mock('../utils/apiUtils.js', () => ({ apiFetch: (...a) => mocks.apiFetch(...a) }))
  vi.mock('../utils/requestErrorDisplay.js', () => ({
    humanizeRequestErrorMessage: (...a) => mocks.humanizeRequestErrorMessage(...a),
  }))
  vi.mock('../utils/clickGuard.js', () => ({
    createClickGuard: () => ({
      run: async (fn) => fn({ idempotencyKey: 'ik-at-1' }),
    }),
    mergeIdempotencyHeaders: (headers, key) => ({ ...(headers || {}), 'Idempotency-Key': key }),
  }))

  const { default: Panel } = await import('./AccessTokenManagementPanel.vue')

  function jsonOk(body) {
    return {
      ok: true,
      json: async () => body,
      headers: { get: () => 'application/json' },
    }
  }

  function mountPanel() {
    return mount(Panel)
  }

  describe('AccessTokenManagementPanel 创建/吊销访问令牌实施 clickGuard 接线', () => {
    beforeEach(() => {
      mocks.apiFetch.mockReset()
      mocks.apiFetch.mockImplementation(async (url, opts) => {
        if (opts?.method === 'POST') return jsonOk({ token: 'tok-abc' })
        if (opts?.method === 'DELETE') return jsonOk({})
        return jsonOk({
          tokens: [{ id: 'tok-1', name: 'T1', is_revoked: false, expires_at: null, last_used_at: null }],
        })
      })
    })

    it('点「创建令牌」的 POST 携带 Idempotency-Key 头', async () => {
      const wrapper = mountPanel()
      await flushPromises()
      await wrapper.findAll('button').find((b) => b.text().includes('创建令牌')).trigger('click')
      await flushPromises()

      const postCalls = mocks.apiFetch.mock.calls.filter(([, opts]) => opts?.method === 'POST')
      expect(postCalls.length).toBeGreaterThan(0)
      for (const [, opts] of postCalls) {
        expect(opts.headers['Idempotency-Key']).toBe('ik-at-1')
      }
      wrapper.unmount()
    })

    it('点「吊销」的 DELETE 携带 Idempotency-Key 头', async () => {
      const wrapper = mountPanel()
      await flushPromises()
      const row = wrapper.find('[data-testid="access-token-row-active"]')
      expect(row.exists()).toBe(true)
      await row.find('button').trigger('click')
      await flushPromises()

      const delCalls = mocks.apiFetch.mock.calls.filter(([, opts]) => opts?.method === 'DELETE')
      expect(delCalls.length).toBeGreaterThan(0)
      for (const [, opts] of delCalls) {
        expect(opts.headers['Idempotency-Key']).toBe('ik-at-1')
      }
      wrapper.unmount()
    })
  })
}

// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] PrivacyReconsentGate.click-guard.test.js requires vitest runtime')
} else {
  const { describe, it, expect, vi, beforeEach } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')

  const mocks = vi.hoisted(() => ({
    apiFetch: vi.fn(),
    getCookie: vi.fn(() => 'u-1'),
    alert: vi.fn(),
  }))

  vi.mock('../utils/apiUtils.js', () => ({ apiFetch: (...a) => mocks.apiFetch(...a) }))
  vi.mock('../utils/cookieUtils.js', () => ({ getCookie: (...a) => mocks.getCookie(...a) }))
  vi.mock('../utils/modalService.js', () => ({ default: { alert: (...a) => mocks.alert(...a) } }))
  vi.mock('vue-router', () => ({
    useRoute: () => ({ name: 'workspace', fullPath: '/workspace', path: '/workspace' }),
  }))
  vi.mock('../utils/clickGuard.js', () => ({
    createClickGuard: () => ({
      run: async (fn) => fn({ idempotencyKey: 'ik-privacy-1' }),
    }),
    mergeIdempotencyHeaders: (headers, key) => ({ ...(headers || {}), 'Idempotency-Key': key }),
  }))

  const { default: Gate } = await import('./PrivacyReconsentGate.vue')

  function jsonOk(body) {
    return {
      ok: true,
      json: async () => body,
      headers: { get: () => 'application/json' },
    }
  }

  describe('PrivacyReconsentGate 同意提交实施 clickGuard 接线', () => {
    beforeEach(() => {
      mocks.apiFetch.mockReset()
      sessionStorage.clear()
      mocks.apiFetch.mockImplementation(async (url, opts) => {
        if (opts?.method === 'POST') return jsonOk({ accepted_privacy_policy_id: 'pp-1' })
        if (url.includes('/users/me/')) {
          return jsonOk({
            pending_privacy_policy: { id: 'pp-1', version: 'v3', title: '隐私政策', content: '正文' },
          })
        }
        return jsonOk({})
      })
    })

    it('点「同意并继续」的 POST 携带 Idempotency-Key 头', async () => {
      const wrapper = mount(Gate, { global: { stubs: { Teleport: true } } })
      await flushPromises()
      await flushPromises()

      const btn = wrapper.findAll('button').find((b) => b.text().includes('同意并继续'))
      expect(btn).toBeTruthy()
      await btn.trigger('click')
      await flushPromises()

      const postCalls = mocks.apiFetch.mock.calls.filter(
        ([url, opts]) => opts?.method === 'POST' && url.includes('/privacy-policy/consent/')
      )
      expect(postCalls.length).toBeGreaterThan(0)
      for (const [, opts] of postCalls) {
        expect(opts.headers['Idempotency-Key']).toBe('ik-privacy-1')
      }
      wrapper.unmount()
    })
  })
}

// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] CodeLangOptionsSettingsModal.click-guard.test.js requires vitest runtime')
} else {
  const { describe, it, expect, vi, beforeEach } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')

  const { apiFetch } = vi.hoisted(() => ({ apiFetch: vi.fn() }))
  vi.mock('../utils/apiUtils.js', () => ({ apiFetch }))
  vi.mock('../utils/requestErrorDisplay.js', () => ({ showRequestError: vi.fn() }))
  // OPT-20260819-038: 让 guard 直接放行并把幂等键透传给 mergeIdempotencyHeaders，
  // 断言保存 PUT 携带 Idempotency-Key 头。
  vi.mock('../utils/clickGuard.js', () => ({
    createClickGuard: () => ({
      run: async (fn) => fn({ idempotencyKey: 'ik-codelang-1' }),
    }),
    mergeIdempotencyHeaders: (headers, key) => ({ ...(headers || {}), 'Idempotency-Key': key }),
  }))

  const { default: Modal } = await import('./CodeLangOptionsSettingsModal.vue')

  function jsonOk(body) {
    return {
      ok: true,
      json: async () => body,
      headers: { get: () => 'application/json' },
    }
  }

  describe('CodeLangOptionsSettingsModal 保存实施 clickGuard 接线', () => {
    beforeEach(() => {
      apiFetch.mockReset()
    })

    it('点保存发起一次 PUT 且携带 Idempotency-Key', async () => {
      apiFetch.mockImplementation(async (url, opts) => {
        if (opts?.method === 'PUT') return jsonOk({ ok: true })
        return jsonOk({ options: ['go', 'rust', 'js'] })
      })
      const wrapper = mount(Modal, {
        props: { show: true, tenantId: 't-1', workspaceId: 'w-1' },
      })
      await flushPromises()

      await wrapper.find('[data-testid="code-lang-options-save"]').trigger('click')
      await flushPromises()

      const putCall = apiFetch.mock.calls.find(([, opts]) => opts?.method === 'PUT')
      expect(putCall).toBeTruthy()
      expect(putCall[0]).toContain('/code-lang-options/')
      expect(putCall[1].headers['Idempotency-Key']).toBe('ik-codelang-1')
      expect(putCall[1].body).toContain('"options":["go","rust","js"]')
      wrapper.unmount()
    })
  })
}

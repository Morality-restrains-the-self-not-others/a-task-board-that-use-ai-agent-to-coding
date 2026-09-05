// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] WorkspaceSettingsLlmBudgetModal.click-guard.test.js requires vitest runtime')
} else {
  const { describe, it, expect, vi, beforeEach } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')

  const { apiFetch } = vi.hoisted(() => ({ apiFetch: vi.fn() }))
  vi.mock('../utils/apiUtils.js', () => ({ apiFetch }))
  vi.mock('../utils/clickGuard.js', () => ({
    createClickGuard: () => ({
      run: async (fn) => fn({ idempotencyKey: 'ik-llm-budget-1' }),
    }),
    mergeIdempotencyHeaders: (headers, key) => ({ ...(headers || {}), 'Idempotency-Key': key }),
  }))

  const { default: Modal } = await import('./WorkspaceSettingsLlmBudgetModal.vue')

  function jsonOk(body) {
    return {
      ok: true,
      json: async () => body,
      headers: { get: () => 'application/json' },
    }
  }

  describe('WorkspaceSettingsLlmBudgetModal 保存实施 clickGuard 接线', () => {
    beforeEach(() => {
      apiFetch.mockReset()
      apiFetch.mockImplementation(async (url, opts) => {
        if (opts?.method === 'PATCH') return jsonOk({ ok: true })
        return jsonOk({ items: [] })
      })
    })

    it('点保存的 PATCH 携带 Idempotency-Key 头', async () => {
      const wrapper = mount(Modal, {
        props: { visible: true, tenantId: 't-1', workspaceId: 'w-1', workspaceLabel: '默认空间' },
      })
      await flushPromises()

      await wrapper.findAll('button').find((b) => b.text().includes('保存'))?.trigger('click')
      await flushPromises()

      const patchCalls = apiFetch.mock.calls.filter(([, opts]) => opts?.method === 'PATCH')
      expect(patchCalls.length).toBeGreaterThan(0)
      for (const [, opts] of patchCalls) {
        expect(opts.headers['Idempotency-Key']).toBe('ik-llm-budget-1')
      }
      wrapper.unmount()
    })
  })
}

// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] WorkspaceContainerImageAtModeToggle.click-guard.test.js requires vitest runtime')
} else {
  const { describe, it, expect, vi, beforeEach } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')

  const { apiFetch } = vi.hoisted(() => ({ apiFetch: vi.fn() }))
  vi.mock('../utils/apiUtils.js', () => ({ apiFetch }))
  vi.mock('../utils/clickGuard.js', () => ({
    createClickGuard: () => ({
      isBusy: () => false,
      run: async (fn) => fn({ idempotencyKey: 'ik-atmode-1' }),
    }),
    mergeIdempotencyHeaders: (headers, key) => ({ ...(headers || {}), 'Idempotency-Key': key }),
  }))

  const { default: Toggle } = await import('./WorkspaceContainerImageAtModeToggle.vue')

  function jsonOk(body) {
    return {
      ok: true,
      json: async () => body,
      headers: { get: () => 'application/json' },
    }
  }

  describe('WorkspaceContainerImageAtModeToggle 切换实施 clickGuard 接线', () => {
    beforeEach(() => {
      apiFetch.mockReset()
    })

    it('点切换的 PATCH 携带 Idempotency-Key 头', async () => {
      apiFetch.mockResolvedValue(jsonOk({}))
      const wrapper = mount(Toggle, {
        props: { tenantId: 't-1', workspaceId: 'w-1', initialEnabled: true },
      })

      await wrapper.find('[data-testid="workspace-container-image-at-mode-toggle"]').trigger('click')
      await flushPromises()

      const patchCalls = apiFetch.mock.calls.filter(([, opts]) => opts?.method === 'PATCH')
      expect(patchCalls.length).toBeGreaterThan(0)
      for (const [, opts] of patchCalls) {
        expect(opts.headers['Idempotency-Key']).toBe('ik-atmode-1')
      }
      wrapper.unmount()
    })
  })
}

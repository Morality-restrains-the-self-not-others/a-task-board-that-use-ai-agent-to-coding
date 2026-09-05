// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] DeliverableSystemSettings.click-guard.test.js requires vitest runtime')
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

  const { default: Comp } = await import('./DeliverableSystemSettings.vue')

  function jsonOk(body) {
    return {
      ok: true,
      json: async () => body,
      headers: { get: () => 'application/json' },
    }
  }

  describe('DeliverableSystemSettings 保存交付物体系 clickGuard 接线', () => {
    beforeEach(() => {
      apiFetch.mockReset()
      apiFetch.mockImplementation(async (url, opts) => {
        if (!opts?.method) return jsonOk({ status: 'success', deliverable_systems: [], deliverable_types: [] })
        return jsonOk({ status: 'success' })
      })
    })

    it('保存交付物体系 POST 携带 Idempotency-Key', async () => {
      const wrapper = mount(Comp, {
        props: { tenantId: 'tenant-test', workspaceId: 'ws-1' },
      })
      await flushPromises()
      await wrapper.find('#save-deliverable-system-btn').trigger('click')
      await flushPromises()

      const postCall = apiFetch.mock.calls.find(([, opts]) => opts?.method === 'POST')
      expect(postCall).toBeTruthy()
      expect(postCall[0]).toBe('/api/projects/manage-deliverable-system/tenant_id/tenant-test')
      expect(postCall[1].headers['Idempotency-Key']).toBe('ik-test-1')
    })
  })
}

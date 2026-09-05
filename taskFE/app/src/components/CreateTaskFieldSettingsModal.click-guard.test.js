// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] CreateTaskFieldSettingsModal.click-guard.test.js requires vitest runtime')
} else {
  const { describe, it, expect, vi, beforeEach } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')

  const { apiFetch } = vi.hoisted(() => ({ apiFetch: vi.fn() }))
  vi.mock('../utils/apiUtils.js', () => ({ apiFetch }))
  vi.mock('../utils/requestErrorDisplay.js', () => ({ showRequestError: vi.fn() }))
  // OPT-20260819-038: 让 guard 直接放行并把幂等键透传给 mergeIdempotencyHeaders，
  // 断言同批 3 个保存 PUT 携带同一 Idempotency-Key 头。
  vi.mock('../utils/clickGuard.js', () => ({
    createClickGuard: () => ({
      run: async (fn) => fn({ idempotencyKey: 'ik-fields-1' }),
    }),
    mergeIdempotencyHeaders: (headers, key) => ({ ...(headers || {}), 'Idempotency-Key': key }),
  }))

  const { default: Modal } = await import('./CreateTaskFieldSettingsModal.vue')

  function jsonOk(body) {
    return {
      ok: true,
      json: async () => body,
      headers: { get: () => 'application/json' },
    }
  }

  describe('CreateTaskFieldSettingsModal 保存实施 clickGuard 接线', () => {
    beforeEach(() => {
      apiFetch.mockReset()
    })

    it('点保存同批 3 个 PUT 均携带同一 Idempotency-Key', async () => {
      apiFetch.mockImplementation(async (url, opts) => {
        if (opts?.method === 'PUT') return jsonOk({ ok: true })
        return jsonOk({})
      })
      const wrapper = mount(Modal, {
        props: { show: true, tenantId: 't-1', workspaceId: 'w-1' },
      })
      await flushPromises()

      await wrapper.find('[data-testid="create-task-field-settings-save"]').trigger('click')
      await flushPromises()

      const putCalls = apiFetch.mock.calls.filter(([, opts]) => opts?.method === 'PUT')
      expect(putCalls.length).toBe(3)
      for (const [, opts] of putCalls) {
        expect(opts.headers['Idempotency-Key']).toBe('ik-fields-1')
      }
      wrapper.unmount()
    })
  })
}

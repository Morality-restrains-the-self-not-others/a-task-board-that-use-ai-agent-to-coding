// @vitest-environment jsdom
// OPT-20260819-038: 保存进度体系是写操作（POST），防连点双发。
if (!process.env.VITEST) {
  console.log('[skip] ProgressSystemSettings.click-guard.test.js requires vitest runtime')
} else {
  const { describe, it, expect, vi, beforeEach } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')

  const { apiFetch } = vi.hoisted(() => ({
    apiFetch: vi.fn(),
  }))

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch,
    extractErrorMessage: (data, resp, fallback) => data?.message || fallback,
  }))
  // OPT-20260819-038: 让 guard 直接放行并把幂等键透传给 mergeIdempotencyHeaders，
  // 断言 POST 携带 Idempotency-Key 头。
  vi.mock('../utils/clickGuard.js', () => ({
    createClickGuard: () => ({
      run: async (fn) => fn({ idempotencyKey: 'ik-test-progress' }),
      isBusy: () => false,
    }),
    mergeIdempotencyHeaders: (headers, key) => ({ ...(headers || {}), 'Idempotency-Key': key }),
  }))

  const { default: Component } = await import('./ProgressSystemSettings.vue')

  function jsonOk(body) {
    return {
      ok: true,
      json: async () => body,
      headers: { get: () => 'application/json' },
    }
  }

  describe('ProgressSystemSettings 写操作 clickGuard 接线', () => {
    beforeEach(() => {
      apiFetch.mockReset()
      apiFetch.mockImplementation(async (url, opts) => {
        if (opts?.method === 'POST') {
          return jsonOk({ status: 'success' })
        }
        return jsonOk({ status: 'success', project_progress_systems: [{ id: 'ps-1', name: '体系1', columns: [] }] })
      })
    })

    it('保存进度体系 POST 携带 Idempotency-Key', async () => {
      const wrapper = mount(Component, { props: { tenantId: 'ten1', workspaceId: 'ws1' } })
      await flushPromises()

      const saveBtn = wrapper.find('#save-progress-system-btn')
      expect(saveBtn.exists()).toBe(true)
      await saveBtn.trigger('click')
      await flushPromises()

      const postCall = apiFetch.mock.calls.find(([, opts]) => opts?.method === 'POST')
      expect(postCall).toBeTruthy()
      expect(postCall[0]).toBe('/api/projects/manage-progress-column/tenant_id/ten1')
      expect(postCall[1].headers['Idempotency-Key']).toBe('ik-test-progress')
      wrapper.unmount()
    })
  })
}

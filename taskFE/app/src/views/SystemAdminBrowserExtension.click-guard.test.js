// @vitest-environment jsdom
// OPT-20260819-038: 保存浏览器插件白名单是写操作（PUT），防连点双发。
if (!process.env.VITEST) {
  console.log('[skip] SystemAdminBrowserExtension.click-guard.test.js requires vitest runtime')
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
  // 断言 PUT 携带 Idempotency-Key 头。
  vi.mock('../utils/clickGuard.js', () => ({
    createClickGuard: () => ({
      run: async (fn) => fn({ idempotencyKey: 'ik-test-oidc-ext' }),
      isBusy: () => false,
    }),
    mergeIdempotencyHeaders: (headers, key) => ({ ...(headers || {}), 'Idempotency-Key': key }),
  }))

  const { default: View } = await import('./SystemAdminBrowserExtension.vue')

  function jsonOk(body) {
    return {
      ok: true,
      json: async () => body,
      headers: { get: () => 'application/json' },
    }
  }

  describe('SystemAdminBrowserExtension 写操作 clickGuard 接线', () => {
    beforeEach(() => {
      apiFetch.mockReset()
      apiFetch.mockImplementation(async (url, opts) => {
        if (opts?.method === 'PUT') {
          return jsonOk({ extension_ids: ['aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa'] })
        }
        return jsonOk({ extension_ids: [], client_id: 'client-1', managed_by: 'admin' })
      })
    })

    it('保存插件白名单 PUT 携带 Idempotency-Key', async () => {
      const wrapper = mount(View, {})
      await flushPromises()

      await wrapper.find('textarea').setValue('aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa')
      const saveBtn = wrapper.findAll('button').find((b) => b.text().trim() === '保存')
      expect(saveBtn).toBeTruthy()
      await saveBtn.trigger('click')
      await flushPromises()

      const putCall = apiFetch.mock.calls.find(([, opts]) => opts?.method === 'PUT')
      expect(putCall).toBeTruthy()
      expect(putCall[0]).toBe('/api/system-admin/oidc-extension/')
      expect(putCall[1].headers['Idempotency-Key']).toBe('ik-test-oidc-ext')
      expect(putCall[1].body).toContain('aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa')
      wrapper.unmount()
    })
  })
}

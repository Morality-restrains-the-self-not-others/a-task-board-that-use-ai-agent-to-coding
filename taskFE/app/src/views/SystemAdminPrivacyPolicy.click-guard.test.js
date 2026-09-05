// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] SystemAdminPrivacyPolicy.click-guard.test.js requires vitest runtime')
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
  // 断言 POST/PUT/DELETE 携带 Idempotency-Key 头。
  vi.mock('../utils/clickGuard.js', () => ({
    createClickGuard: () => ({
      run: async (fn) => fn({ idempotencyKey: 'ik-test-privacy' }),
    }),
    mergeIdempotencyHeaders: (headers, key) => ({ ...(headers || {}), 'Idempotency-Key': key }),
  }))

  const { default: View } = await import('./SystemAdminPrivacyPolicy.vue')

  function jsonOk(body) {
    return {
      ok: true,
      json: async () => body,
      headers: { get: () => 'application/json' },
    }
  }

  const STUBS = {
    SystemAdminPrivacyPolicyConsentQuery: { template: '<div />' },
  }

  describe('SystemAdminPrivacyPolicy 写操作 clickGuard 接线', () => {
    beforeEach(() => {
      apiFetch.mockReset()
      apiFetch.mockImplementation(async (url, opts) => {
        if (opts?.method === 'GET' || !opts?.method) {
          return jsonOk([
            { id: 'p-1', title: '隐私条款 v1', version: '1.0.0', content: '内容', is_active: true, is_material_change: false },
          ])
        }
        return jsonOk({})
      })
    })

    it('创建隐私条款 POST 携带 Idempotency-Key', async () => {
      const wrapper = mount(View, { global: { stubs: STUBS } })
      await flushPromises()

      const createBtn = wrapper.findAll('button').find((b) => b.text().trim() === '创建条款')
      expect(createBtn).toBeTruthy()
      await createBtn.trigger('click')
      await flushPromises()

      await wrapper.find('#create-title').setValue('新隐私条款')
      await wrapper.find('#create-version').setValue('2.0.0')
      await wrapper.find('#create-content').setValue('新的条款内容')
      await wrapper.find('form').trigger('submit')
      await flushPromises()

      const postCall = apiFetch.mock.calls.find(([, opts]) => opts?.method === 'POST')
      expect(postCall).toBeTruthy()
      expect(postCall[0]).toBe('/api/system-admin/privacy-policy/')
      expect(postCall[1].headers['Idempotency-Key']).toBe('ik-test-privacy')
      expect(postCall[1].body).toContain('新隐私条款')
    })

    it('删除隐私条款 DELETE 携带 Idempotency-Key', async () => {
      const wrapper = mount(View, { global: { stubs: STUBS } })
      await flushPromises()

      const delBtn = wrapper.findAll('button').find((b) => b.text().trim() === '删除')
      expect(delBtn).toBeTruthy()
      await delBtn.trigger('click')
      await flushPromises()

      const confirmBtn = wrapper.findAll('button').find((b) => b.text().includes('确认删除'))
      expect(confirmBtn).toBeTruthy()
      await confirmBtn.trigger('click')
      await flushPromises()

      const delCall = apiFetch.mock.calls.find(([, opts]) => opts?.method === 'DELETE')
      expect(delCall).toBeTruthy()
      expect(delCall[0]).toBe('/api/system-admin/privacy-policy/p-1/')
      expect(delCall[1].headers['Idempotency-Key']).toBe('ik-test-privacy')
    })
  })
}

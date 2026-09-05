// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] SystemAdminRecommendedLLMProviders.click-guard.test.js requires vitest runtime')
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

  const { default: View } = await import('./SystemAdminRecommendedLLMProviders.vue')

  function jsonOk(body) {
    return {
      ok: true,
      json: async () => body,
      headers: { get: () => 'application/json' },
    }
  }

  describe('SystemAdminRecommendedLLMProviders 写操作 clickGuard 接线', () => {
    beforeEach(() => {
      apiFetch.mockReset()
      apiFetch.mockImplementation(async (url, opts) => {
        if (!opts?.method) return jsonOk({ items: [] })
        return jsonOk({ items: [] })
      })
    })

    it('新增供应商 POST 携带 Idempotency-Key', async () => {
      const wrapper = mount(View)
      await flushPromises()
      await wrapper.find('button').trigger('click') // 新增供应商
      await flushPromises()
      const nameInput = wrapper.findAll('input')[0]
      await nameInput.setValue('DeepSeek')
      await flushPromises()
      const saveBtn = wrapper.findAll('button').find((b) => b.text().trim() === '保存')
      await saveBtn.trigger('click')
      await flushPromises()

      const postCall = apiFetch.mock.calls.find(([, opts]) => opts?.method === 'POST')
      expect(postCall).toBeTruthy()
      expect(postCall[0]).toBe('/api/system-admin/recommended-llm-providers/')
      expect(postCall[1].headers['Idempotency-Key']).toBe('ik-test-1')
      expect(postCall[1].body).toContain('DeepSeek')
    })

    it('删除供应商 DELETE 携带 Idempotency-Key', async () => {
      apiFetch.mockImplementation(async (url, opts) => {
        if (!opts?.method) {
          return jsonOk({ items: [{ id: 'llm-1', name: 'DeepSeek', docs_url: '', description: '' }] })
        }
        return jsonOk({ items: [] })
      })
      const wrapper = mount(View)
      await flushPromises()
      const delBtn = wrapper.findAll('button').find((b) => b.text().includes('删除'))
      await delBtn.trigger('click')
      await flushPromises()

      const delCall = apiFetch.mock.calls.find(([, opts]) => opts?.method === 'DELETE')
      expect(delCall).toBeTruthy()
      expect(delCall[0]).toBe('/api/system-admin/recommended-llm-providers/')
      expect(delCall[1].headers['Idempotency-Key']).toBe('ik-test-1')
    })
  })
}

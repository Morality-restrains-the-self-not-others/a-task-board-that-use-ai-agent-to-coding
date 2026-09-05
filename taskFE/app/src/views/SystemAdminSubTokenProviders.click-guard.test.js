// @vitest-environment jsdom
// OPT-20260819-038: sub-token 供应商配置（资源/安全路径）接入 createClickGuard + Idempotency-Key。
// 回归断言：保存 POST/PUT 与删除 DELETE 均携带 Idempotency-Key 头。
if (!process.env.VITEST) {
  console.log('[skip] SystemAdminSubTokenProviders.click-guard.test.js requires vitest runtime')
} else {
  const { describe, it, expect, vi, beforeEach } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')

  const { apiFetchMock } = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
  }))

  vi.mock('../utils/apiUtils', () => ({
    apiFetch: (...args) => apiFetchMock(...args),
  }))
  vi.mock('../utils/clickGuard.js', () => ({
    createClickGuard: () => ({
      run: async (fn) => fn({ idempotencyKey: 'ik-subtok-test' }),
    }),
    mergeIdempotencyHeaders: (headers, key) => ({ ...(headers || {}), 'Idempotency-Key': key }),
  }))
  vi.mock('../utils/requestErrorDisplay.js', () => ({
    showRequestError: vi.fn(),
  }))

  const { default: View } = await import('./SystemAdminSubTokenProviders.vue')

  function jsonOk(body = {}) {
    return {
      ok: true,
      status: 200,
      json: async () => body,
      traceId: '',
      headers: { get: () => 'application/json' },
    }
  }

  describe('SystemAdminSubTokenProviders 写操作 clickGuard 接线', () => {
    beforeEach(() => {
      apiFetchMock.mockReset()
      apiFetchMock.mockImplementation((url, options) => {
        const method = options?.method || 'GET'
        if (method === 'GET') {
          return Promise.resolve(jsonOk({
            items: [
              { id: 'p-1', provider_name: 'openai', base_url: 'https://api.openai.com', derive_endpoint: '/api/token/derive' },
            ],
          }))
        }
        return Promise.resolve(jsonOk({}))
      })
    })

    it('保存供应商 POST/PUT 携带 Idempotency-Key', async () => {
      const wrapper = mount(View)
      await flushPromises()

      // 打开新增表单
      const addBtn = wrapper.findAll('button').find((b) => b.text().includes('新增供应商'))
      await addBtn.trigger('click')
      await flushPromises()

      const saveBtn = wrapper.findAll('button').find((b) => b.text().includes('保存'))
      expect(saveBtn).toBeTruthy()
      await saveBtn.trigger('click')
      await flushPromises()

      const writes = apiFetchMock.mock.calls.filter(([, o]) => o?.method === 'POST' || o?.method === 'PUT')
      expect(writes).toHaveLength(1)
      const [, options] = writes[0]
      expect(options.method).toBe('POST')
      expect(options.headers['Idempotency-Key']).toBe('ik-subtok-test')
    })

    it('删除供应商 DELETE 携带 Idempotency-Key', async () => {
      const wrapper = mount(View)
      await flushPromises()

      const delBtn = wrapper.findAll('button').find((b) => b.text().includes('删除'))
      expect(delBtn).toBeTruthy()
      await delBtn.trigger('click')
      await flushPromises()

      const deletes = apiFetchMock.mock.calls.filter(([, o]) => o?.method === 'DELETE')
      expect(deletes).toHaveLength(1)
      const [, options] = deletes[0]
      expect(options.headers['Idempotency-Key']).toBe('ik-subtok-test')
    })
  })
}

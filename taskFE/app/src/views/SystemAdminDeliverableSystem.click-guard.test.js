// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] SystemAdminDeliverableSystem.click-guard.test.js requires vitest runtime')
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
  // 断言 POST/DELETE 携带 Idempotency-Key 头。
  vi.mock('../utils/clickGuard.js', () => ({
    createClickGuard: () => ({
      run: async (fn) => fn({ idempotencyKey: 'ik-test-deliverable' }),
    }),
    mergeIdempotencyHeaders: (headers, key) => ({ ...(headers || {}), 'Idempotency-Key': key }),
  }))

  const { default: View } = await import('./SystemAdminDeliverableSystem.vue')

  function jsonOk(body) {
    return {
      ok: true,
      json: async () => body,
      headers: { get: () => 'application/json' },
    }
  }

  describe('SystemAdminDeliverableSystem 写操作 clickGuard 接线', () => {
    beforeEach(() => {
      apiFetch.mockReset()
      apiFetch.mockImplementation(async (url, opts) => {
        if (opts?.method === 'GET' || !opts?.method) {
          return jsonOk([])
        }
        return jsonOk({})
      })
    })

    it('创建交付物体系 POST 携带 Idempotency-Key', async () => {
      const wrapper = mount(View, {})
      await flushPromises()

      const createBtn = wrapper.findAll('button').find((b) => b.text().trim() === '创建交付物体系')
      expect(createBtn).toBeTruthy()
      await createBtn.trigger('click')
      await flushPromises()

      await wrapper.find('#deliverable-system-name').setValue('新体系')
      await wrapper.find('#deliverable-system-description').setValue('描述')
      await wrapper.find('form').trigger('submit')
      await flushPromises()

      const postCall = apiFetch.mock.calls.find(([, opts]) => opts?.method === 'POST')
      expect(postCall).toBeTruthy()
      expect(postCall[0]).toBe('/api/system-admin/deliverable-systems/')
      expect(postCall[1].headers['Idempotency-Key']).toBe('ik-test-deliverable')
      expect(postCall[1].body).toContain('新体系')
    })

    it('删除交付物体系 DELETE 携带 Idempotency-Key', async () => {
      apiFetch.mockImplementation(async (url, opts) => {
        if (opts?.method === 'GET' || !opts?.method) {
          return jsonOk([{ id: 'ds-1', name: '体系1', level_names: ['层1'] }])
        }
        return jsonOk({})
      })
      const wrapper = mount(View, {})
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
      expect(delCall[0]).toBe('/api/system-admin/deliverable-systems/ds-1/')
      expect(delCall[1].headers['Idempotency-Key']).toBe('ik-test-deliverable')
    })
  })
}

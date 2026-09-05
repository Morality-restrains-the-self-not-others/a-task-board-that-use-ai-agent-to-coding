// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] SystemAdminDefaultColumnManagement.click-guard.test.js requires vitest runtime')
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
      run: async (fn) => fn({ idempotencyKey: 'ik-test-1' }),
    }),
    mergeIdempotencyHeaders: (headers, key) => ({ ...(headers || {}), 'Idempotency-Key': key }),
  }))

  const { default: View } = await import('./SystemAdminDefaultColumnManagement.vue')

  function jsonOk(body) {
    return {
      ok: true,
      json: async () => body,
      headers: { get: () => 'application/json' },
    }
  }

  const systemsPayload = () => ({
    status: 'success',
    project_progress_systems: [
      { id: 'sys-1', name: '标准进度', is_default: true, columns: [{ name: '待办' }] },
      { id: 'sys-2', name: '研发进度', is_default: false, columns: [{ name: '待办' }] },
    ],
  })

  describe('SystemAdminDefaultColumnManagement 写操作 clickGuard 接线', () => {
    beforeEach(() => {
      apiFetch.mockReset()
      apiFetch.mockImplementation(async (url, opts) => {
        if (!opts?.method) return jsonOk(systemsPayload())
        return jsonOk({ status: 'success', project_progress_systems: systemsPayload().project_progress_systems })
      })
    })

    async function mountView() {
      const wrapper = mount(View, {
        global: {
          stubs: {
            'router-link': { template: '<a><slot /></a>' },
            'router-view': true,
          },
        },
      })
      await flushPromises()
      return wrapper
    }

    it('保存进度体系 POST/PUT 携带 Idempotency-Key', async () => {
      const wrapper = await mountView()
      await wrapper.find('button.btn-primary').trigger('click') // 打开添加弹窗
      await flushPromises()
      const nameInput = wrapper.find('#system-name')
      await nameInput.setValue('测试进度体系')
      await flushPromises()
      const saveBtn = wrapper.findAll('button').find((b) => b.text().trim() === '保存')
      await saveBtn.trigger('click')
      await flushPromises()

      const postCall = apiFetch.mock.calls.find(([, opts]) => opts?.method === 'POST')
      expect(postCall).toBeTruthy()
      expect(postCall[0]).toBe('/api/system-admin/progress-systems/')
      expect(postCall[1].headers['Idempotency-Key']).toBe('ik-test-1')
    })

    it('设为默认 POST 携带 Idempotency-Key', async () => {
      const wrapper = await mountView()
      const defaultBtn = wrapper.findAll('button').find((b) => b.text().trim() === '设为默认')
      await defaultBtn.trigger('click')
      await flushPromises()

      const postCall = apiFetch.mock.calls.find(([, opts]) => opts?.method === 'POST')
      expect(postCall).toBeTruthy()
      expect(postCall[0]).toBe('/api/system-admin/progress-systems/sys-2/set-default/')
      expect(postCall[1].headers['Idempotency-Key']).toBe('ik-test-1')
    })

    it('删除进度体系 DELETE 携带 Idempotency-Key', async () => {
      vi.spyOn(window, 'confirm').mockReturnValue(true)
      const wrapper = await mountView()
      const sys2Row = wrapper.findAll('.border.border-gray-200.rounded-lg').find((r) => r.text().includes('研发进度'))
      const sys2Buttons = sys2Row.findAll('button') // [编辑, 设为默认, 删除]
      await sys2Buttons[2].trigger('click')
      await flushPromises()

      const delCall = apiFetch.mock.calls.find(([, opts]) => opts?.method === 'DELETE')
      expect(delCall).toBeTruthy()
      expect(delCall[0]).toBe('/api/system-admin/progress-systems/sys-2/')
      expect(delCall[1].headers['Idempotency-Key']).toBe('ik-test-1')
      vi.restoreAllMocks()
    })
  })
}

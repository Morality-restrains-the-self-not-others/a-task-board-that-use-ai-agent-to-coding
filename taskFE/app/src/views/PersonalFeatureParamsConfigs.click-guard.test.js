// @vitest-environment jsdom
// OPT-20260819-038: 个人 feature-params 配置保存/删除是写操作，防连点双发。
if (!process.env.VITEST) {
  console.log('[skip] PersonalFeatureParamsConfigs.click-guard.test.js requires vitest runtime')
} else {
  const { describe, it, expect, vi, beforeEach } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')

  const { apiFetch } = vi.hoisted(() => ({
    apiFetch: vi.fn(),
  }))

  vi.mock('vue-router', () => ({
    useRoute: () => ({ params: { id: 'ten1' } }),
  }))
  vi.mock('../utils/apiUtils', () => ({
    apiFetch,
  }))
  // OPT-20260819-038: 让 guard 直接放行并把幂等键透传给 mergeIdempotencyHeaders，
  // 断言 POST/DELETE 携带 Idempotency-Key 头。
  vi.mock('../utils/clickGuard.js', () => ({
    createClickGuard: () => ({
      run: async (fn) => fn({ idempotencyKey: 'ik-test-pfp' }),
      isBusy: () => false,
    }),
    mergeIdempotencyHeaders: (headers, key) => ({ ...(headers || {}), 'Idempotency-Key': key }),
  }))

  const { default: View } = await import('./PersonalFeatureParamsConfigs.vue')

  function jsonOk(body) {
    return {
      ok: true,
      json: async () => body,
      headers: { get: () => 'application/json' },
    }
  }

  describe('PersonalFeatureParamsConfigs 写操作 clickGuard 接线', () => {
    beforeEach(() => {
      window.confirm = vi.fn(() => true)
      apiFetch.mockReset()
      apiFetch.mockImplementation(async (url, opts) => {
        if (String(url).includes('/profile/')) {
          return jsonOk({ company_nicknames: [{ company_id: 'c1', company_name: '公司1' }] })
        }
        if (opts?.method === 'DELETE') {
          return jsonOk({})
        }
        if (opts?.method === 'POST' || opts?.method === 'PUT') {
          return jsonOk({ id: 'cfg-new' })
        }
        return jsonOk({ configs: [] })
      })
    })

    it('新建配置保存 POST 携带 Idempotency-Key', async () => {
      const wrapper = mount(View, {
        global: {
          stubs: {
            UserCenterSidebar: { template: '<div />' },
            EnvVarTableEditor: { template: '<div />' },
            MergedEnvPreview: { template: '<div />' },
            CollapsibleLLMConfigPanel: { template: '<div />' },
          },
        },
      })
      await flushPromises()

      const createBtn = wrapper.findAll('button').find((b) => b.text().includes('新建配置'))
      expect(createBtn).toBeTruthy()
      await createBtn.trigger('click')

      await wrapper.find('input[placeholder="例如：低成本日常任务"]').setValue('我的配置')
      const saveBtn = wrapper.findAll('button').find((b) => b.text().trim() === '保存')
      expect(saveBtn).toBeTruthy()
      await saveBtn.trigger('click')
      await flushPromises()

      const postCall = apiFetch.mock.calls.find(([, opts]) => opts?.method === 'POST')
      expect(postCall).toBeTruthy()
      expect(postCall[0]).toBe('/api/personal/feature-params-configs/')
      expect(postCall[1].headers['Idempotency-Key']).toBe('ik-test-pfp')
      wrapper.unmount()
    })

    it('删除配置 DELETE 携带 Idempotency-Key', async () => {
      apiFetch.mockImplementation(async (url, opts) => {
        if (String(url).includes('/profile/')) {
          return jsonOk({ company_nicknames: [] })
        }
        if (opts?.method === 'DELETE') {
          return jsonOk({})
        }
        return jsonOk({ configs: [{ id: 'cfg-1', name: '配置1', company_id: 'c1' }] })
      })
      const wrapper = mount(View, {
        global: {
          stubs: {
            UserCenterSidebar: { template: '<div />' },
            EnvVarTableEditor: { template: '<div />' },
            MergedEnvPreview: { template: '<div />' },
            CollapsibleLLMConfigPanel: { template: '<div />' },
          },
        },
      })
      await flushPromises()

      const delBtn = wrapper.findAll('button').find((b) => b.text().trim() === '删除')
      expect(delBtn).toBeTruthy()
      await delBtn.trigger('click')
      await flushPromises()

      const delCall = apiFetch.mock.calls.find(([, opts]) => opts?.method === 'DELETE')
      expect(delCall).toBeTruthy()
      expect(delCall[0]).toBe('/api/personal/feature-params-configs/cfg-1/')
      expect(delCall[1].headers['Idempotency-Key']).toBe('ik-test-pfp')
      wrapper.unmount()
    })
  })
}

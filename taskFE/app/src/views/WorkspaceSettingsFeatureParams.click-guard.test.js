// @vitest-environment jsdom
// OPT-20260819-038: 保存环境变量是写操作（POST），防连点双发。
if (!process.env.VITEST) {
  console.log('[skip] WorkspaceSettingsFeatureParams.click-guard.test.js requires vitest runtime')
} else {
  const { describe, it, expect, vi, beforeEach } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { createRouter, createMemoryHistory } = await import('vue-router')

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
      run: async (fn) => fn({ idempotencyKey: 'ik-test-feature-params' }),
      isBusy: () => false,
    }),
    mergeIdempotencyHeaders: (headers, key) => ({ ...(headers || {}), 'Idempotency-Key': key }),
  }))

  const { default: View } = await import('./WorkspaceSettingsFeatureParams.vue')

  const makeRouter = () => createRouter({
    history: createMemoryHistory(),
    routes: [{ path: '/tenant/:tenant/settings/feature-params/', component: { template: '<div />' } }],
  })

  function jsonOk(body) {
    return {
      ok: true,
      json: async () => body,
      headers: { get: () => 'application/json' },
    }
  }

  describe('WorkspaceSettingsFeatureParams 写操作 clickGuard 接线', () => {
    beforeEach(() => {
      apiFetch.mockReset()
      apiFetch.mockImplementation(async (url, opts) => {
        if (opts?.method === 'POST') {
          return jsonOk({ id: 'cfg-1' })
        }
        return jsonOk({ data: {} })
      })
    })

    it('保存环境变量 POST 携带 Idempotency-Key', async () => {
      const router = makeRouter()
      await router.push('/tenant/ten1/settings/feature-params/')
      await router.isReady()
      const wrapper = mount(View, {
        global: {
          plugins: [router],
          stubs: {
            EnvVarTableEditor: { template: '<div />' },
            MergedEnvPreview: { template: '<div />' },
            CollapsibleLLMConfigPanel: { template: '<div />' },
          },
        },
      })
      await flushPromises()

      const saveBtn = wrapper.findAll('button').find((b) => b.text().includes('保存'))
      expect(saveBtn).toBeTruthy()
      await saveBtn.trigger('click')
      await flushPromises()

      const postCall = apiFetch.mock.calls.find(([, opts]) => opts?.method === 'POST')
      expect(postCall).toBeTruthy()
      expect(postCall[0]).toBe('/api/cloud/feature-params/tenant_id/ten1')
      expect(postCall[1].headers['Idempotency-Key']).toBe('ik-test-feature-params')
      wrapper.unmount()
    })
  })
}

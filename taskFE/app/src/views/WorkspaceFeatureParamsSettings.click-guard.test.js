// @vitest-environment jsdom
// OPT-20260819-038 回归：工作空间功能参数 保存 POST / 治理开关 POST 写请求携带 Idempotency-Key 头。
if (!process.env.VITEST) {
  console.log('[skip] WorkspaceFeatureParamsSettings.click-guard.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { describe, expect, it, vi, beforeEach } = await import('vitest')

  const mocks = vi.hoisted(() => ({
    apiFetch: vi.fn(),
  }))

  vi.mock('../utils/apiUtils', () => ({
    apiFetch: (...args) => mocks.apiFetch(...args),
  }))
  vi.mock('vue-router', () => ({
    useRoute: () => ({ params: { tenant: 't1', workspace: 'ws1' }, query: {} }),
    useRouter: () => ({ push: vi.fn(), replace: vi.fn() }),
  }))
  vi.mock('../utils/featureParamsModelValidation.js', () => ({
    parseSupportedModels: () => [],
    validateModelsAgainstProviders: () => '',
  }))
  vi.mock('../utils/featureParamsSubTokenUi.js', () => ({
    resolveUseSubToken: () => false,
    applySubTokenUiPolicy: (x) => x,
  }))
  // OPT-20260819-038: 让 guard 直接放行并把幂等键透传给 mergeIdempotencyHeaders，
  // 断言 POST 携带 Idempotency-Key 头。
  vi.mock('../utils/clickGuard.js', () => ({
    createClickGuard: () => ({
      run: async (fn) => fn({ idempotencyKey: 'ik-test-fp' }),
      isBusy: () => false,
    }),
    mergeIdempotencyHeaders: (headers, key) => ({ ...(headers || {}), 'Idempotency-Key': key }),
  }))

  beforeEach(() => {
    vi.clearAllMocks()
    mocks.apiFetch.mockImplementation(async (url, opts) => {
      if (opts?.method === 'POST') {
        return { ok: true, json: async () => ({ success: true }) }
      }
      return {
        ok: true,
        json: async () => ({
          data: { use_company_default: false, extra_env_vars: [], providers: [], agent_model: '' },
          workspace: { allow_personal_feature_params: true },
        }),
      }
    })
  })

  const { default: WorkspaceFeatureParamsSettings } = await import('./WorkspaceFeatureParamsSettings.vue')

  const STUBS = {
    EnvVarTableEditor: { template: '<div />' },
    MergedEnvPreview: { template: '<div />' },
    CollapsibleLLMConfigPanel: { template: '<div />' },
  }

  describe('WorkspaceFeatureParamsSettings 写操作 clickGuard 接线', () => {
    it('保存配置 POST 携带 Idempotency-Key', async () => {
      const wrapper = mount(WorkspaceFeatureParamsSettings, { global: { stubs: STUBS } })
      await flushPromises()

      const saveBtn = wrapper.findAll('button').find((b) => b.text().includes('保存'))
      expect(saveBtn).toBeTruthy()
      await saveBtn.trigger('click')
      await flushPromises()

      const postCall = mocks.apiFetch.mock.calls.find(([, o]) => o?.method === 'POST')
      expect(postCall).toBeTruthy()
      expect(postCall[0]).toBe('/api/cloud/feature-params/tenant_id/t1/workspace_id/ws1')
      expect(postCall[1].headers['Idempotency-Key']).toBe('ik-test-fp')
    })

    it('治理开关切换 POST 携带 Idempotency-Key', async () => {
      const wrapper = mount(WorkspaceFeatureParamsSettings, { global: { stubs: STUBS } })
      await flushPromises()

      const checkbox = wrapper.find('input[type="checkbox"]')
      expect(checkbox.exists()).toBe(true)
      await checkbox.setValue(false)
      await flushPromises()

      const posts = mocks.apiFetch.mock.calls.filter(([, o]) => o?.method === 'POST')
      const govCall = posts[posts.length - 1]
      expect(govCall).toBeTruthy()
      expect(govCall[0]).toBe('/api/cloud/feature-params/tenant_id/t1/workspace_id/ws1')
      expect(govCall[1].headers['Idempotency-Key']).toBe('ik-test-fp')
    })
  })
}

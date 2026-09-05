// @vitest-environment jsdom
if (!process.env.VITEST) {
  // pre-commit 会以 node 直跑已暂存 *.test.js，非 Vitest 环境下跳过。
  console.log('[skip] PersonalFeatureParamsConfigs.error.test.js requires vitest runtime')
} else {
  const { mount } = await import('@vue/test-utils')
  const { nextTick } = await import('vue')
  const { beforeEach, describe, expect, it, vi } = await import('vitest')

  const hoistedMocks = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
    routeMock: { params: { id: '873093522473906176' }, query: {}, path: '/user/873093522473906176/profile/feature-params/' },
  }))

  vi.mock('../utils/apiUtils', () => ({
    apiFetch: hoistedMocks.apiFetchMock,
  }))

  vi.mock('vue-router', () => ({
    useRoute: () => hoistedMocks.routeMock,
  }))

  async function flushRender() {
    for (let i = 0; i < 12; i += 1) {
      await Promise.resolve()
      await nextTick()
    }
  }

  function failedResponse(body) {
    return {
      ok: false,
      status: 502,
      headers: {},
      json: async () => body,
    }
  }

  function okResponse(body) {
    return {
      ok: true,
      status: 200,
      headers: {},
      json: async () => body,
    }
  }

  describe('PersonalFeatureParamsConfigs 错误展示', () => {
    beforeEach(() => {
      hoistedMocks.apiFetchMock.mockReset()
      // loadCompanies 成功返回空公司列表
      hoistedMocks.apiFetchMock.mockImplementation(async (url) => {
        if (url === '/api/accounts/users/profile/') {
          return okResponse({ company_nicknames: [] })
        }
        if (url === '/api/personal/feature-params-configs/') {
          return failedResponse({ message: '读取个人配置失败', trace_id: 'trace-feature-params-001' })
        }
        throw new Error('unexpected url: ' + url)
      })
    })

    it('非 200 时透传服务端 message 而非硬编码「加载失败」，且错误元素带 data-traceId', async () => {
      const Comp = (await import('./PersonalFeatureParamsConfigs.vue')).default
      const wrapper = mount(Comp, {
        global: {
          stubs: {
            UserCenterSidebar: { template: '<aside class="stub-sidebar" />' },
            EnvVarTableEditor: { template: '<div />' },
            MergedEnvPreview: { template: '<div />' },
            CollapsibleLLMConfigPanel: { template: '<div />' },
          },
        },
      })
      await flushRender()

      const errorEl = wrapper.find('.bg-red-50.border-red-200')
      expect(errorEl.exists()).toBe(true)
      // 回归：透传服务端真实 message，不再只显示「加载失败」
      expect(errorEl.text()).toContain('加载个人配置失败: 读取个人配置失败')
      expect(errorEl.text()).not.toContain('加载个人配置失败: 加载失败')
      // 元规则：错误元素绑定 data-traceId（body.trace_id 回退源）
      // jsdom 中属性名被小写化，与既有用例约定一致地双写法断言
      expect(errorEl.attributes('data-traceid') || errorEl.attributes('data-traceId')).toBe('trace-feature-params-001')
    })

    it('服务端未携带 trace_id 且响应头无 x-trace-id 时，错误元素不带 data-traceId 属性', async () => {
      hoistedMocks.apiFetchMock.mockImplementation(async (url) => {
        if (url === '/api/accounts/users/profile/') {
          return okResponse({ company_nicknames: [] })
        }
        return failedResponse({ message: '未认证' })
      })
      const Comp = (await import('./PersonalFeatureParamsConfigs.vue')).default
      const wrapper = mount(Comp, {
        global: {
          stubs: {
            UserCenterSidebar: { template: '<aside class="stub-sidebar" />' },
            EnvVarTableEditor: { template: '<div />' },
            MergedEnvPreview: { template: '<div />' },
            CollapsibleLLMConfigPanel: { template: '<div />' },
          },
        },
      })
      await flushRender()

      const errorEl = wrapper.find('.bg-red-50.border-red-200')
      expect(errorEl.exists()).toBe(true)
      expect(errorEl.text()).toContain('加载个人配置失败: 未认证')
      expect(errorEl.attributes('data-traceid') || errorEl.attributes('data-traceId')).toBeUndefined()
    })
  })
}

// @vitest-environment jsdom
/**
 * 回归测试：个人「功能参数配置」页用户可见文案统一为「智能体资源配置」
 * （OPT-20260809-021）。
 *
 * 背景：入口称谓（侧边栏 UserCenterSidebar、工作面板功能参数入口等）已用
 * 「智能体资源配置」指代个人配置页；本页 h2 残留「我的功能参数配置」导致
 * 页面标题与入口称谓不一致。此用例断言 h2 使用统一称谓，防止回归为旧措辞。
 */
if (!process.env.VITEST) {
  console.log('[skip] PersonalFeatureParamsConfigs.wording.test.js requires vitest runtime')
} else {
  const { mount } = await import('@vue/test-utils')
  const { nextTick } = await import('vue')
  const { describe, expect, it, vi } = await import('vitest')

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

  function okResponse(body) {
    return { ok: true, status: 200, headers: {}, json: async () => body }
  }

  describe('PersonalFeatureParamsConfigs.vue 用户可见文案（OPT-20260809-021）', () => {
    it('h2 使用「我的智能体资源配置」而非「我的功能参数配置」', async () => {
      hoistedMocks.apiFetchMock.mockReset()
      hoistedMocks.apiFetchMock.mockImplementation(async (url) => {
        if (url === '/api/accounts/users/profile/') {
          return okResponse({ company_nicknames: [] })
        }
        return okResponse({ configs: [] })
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

      const h2 = wrapper.find('h2')
      expect(h2.text()).toContain('我的智能体资源配置')
      expect(wrapper.text()).not.toContain('我的功能参数配置')
    })
  })
}

// @vitest-environment jsdom
/**
 * 回归测试：工作空间级功能参数页用户可见「功能参数」文案统一为「智能体资源配置」
 * （OPT-20260809-026）。
 *
 * 背景：入口称谓（侧边栏 Sidebar.vue、人员管理页 MemberList.vue 链接）已用
 * 「智能体资源配置」指代 /settings/feature-params/ 页面；目标页正文残留「功能参数」
 * 措辞导致点击进入后称谓不一致。此用例断言正文两处用户可见文案使用统一称谓，
 * 防止回归为旧措辞。
 */
if (!process.env.VITEST) {
  console.log('[skip] WorkspaceFeatureParamsSettings.wording.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { describe, expect, it, vi } = await import('vitest')
  const { createRouter, createMemoryHistory } = await import('vue-router')
  const { apiFetchMock } = vi.hoisted(() => ({ apiFetchMock: vi.fn() }))
  vi.mock('../utils/apiUtils', () => ({
    apiFetch: (...args) => apiFetchMock(...args),
  }))

  const WorkspaceFeatureParamsSettings = (await import('./WorkspaceFeatureParamsSettings.vue')).default

  const makeRouter = () => createRouter({
    history: createMemoryHistory(),
    routes: [{
      path: '/tenant/:tenant/settings/workspace/:workspace/feature-params/',
      component: { template: '<div />' },
    }],
  })

  const mountPage = async (payload) => {
    apiFetchMock.mockResolvedValue({
      ok: true,
      json: async () => payload,
    })
    const router = makeRouter()
    await router.push('/tenant/874176608758427648/settings/workspace/ws-123/feature-params/')
    await router.isReady()
    const wrapper = mount(WorkspaceFeatureParamsSettings, {
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
    return wrapper
  }

  describe('WorkspaceFeatureParamsSettings.vue 用户可见文案（OPT-20260809-026）', () => {
    it('治理开关标签使用「个人智能体资源配置」而非「个人功能参数配置」', async () => {
      const wrapper = await mountPage({
        data: { use_company_default: true, company_config: null, id: '' },
        workspace: { allow_personal_feature_params: false },
      })
      expect(wrapper.text()).toContain('允许成员在此工作空间中使用个人智能体资源配置')
      expect(wrapper.text()).not.toContain('个人功能参数配置')
    })

    it('公司未配置时提示使用「智能体资源配置」而非「功能参数」', async () => {
      const wrapper = await mountPage({
        data: { use_company_default: true, company_config: null, id: '' },
        workspace: { allow_personal_feature_params: false },
      })
      expect(wrapper.text()).toContain('公司尚未配置智能体资源配置')
      expect(wrapper.text()).not.toContain('公司尚未配置功能参数')
    })
  })
}

// @vitest-environment jsdom
/**
 * 回归测试：工作空间级功能参数配置页内容区必须是独立滚动容器
 * （max-h 视口高度减导航栏 + overflow-y-auto）。OPT-20260809-022。
 *
 * 背景：与公司级 WorkspaceSettingsFeatureParams.vue 同构。此前内容区
 * overflow:visible，只有文档级滚动可用；对元素本身的滚动操作
 * （el.scrollTop / 元素内滚轮 / 自动化任务直接滚动该区域）全部无效，
 * 表现为「内容区域无法滚动」。修复为 max-h-[calc(100dvh-var(--app-navbar-h))] overflow-y-auto。
 *
 * 74px = sticky 导航栏实际高度（Navbar.logic.vue 实测；单一事实源 styles.css :root --app-navbar-h，OPT-20260809-023）。
 */
if (!process.env.VITEST) {
  console.log('[skip] WorkspaceFeatureParamsSettings.scroll.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { describe, expect, it, vi } = await import('vitest')
  const { createRouter, createMemoryHistory } = await import('vue-router')
  const { apiFetchMock } = vi.hoisted(() => ({ apiFetchMock: vi.fn() }))
  vi.mock('../utils/apiUtils', () => ({
    apiFetch: (...args) => apiFetchMock(...args),
  }))

  const WorkspaceFeatureParamsSettings = (await import('./WorkspaceFeatureParamsSettings.vue')).default

  // 组件 onMounted 调用 load() 依赖 useRoute().params.tenant / .params.workspace
  const makeRouter = () => createRouter({
    history: createMemoryHistory(),
    routes: [{
      path: '/tenant/:tenant/settings/workspace/:workspace/feature-params/',
      component: { template: '<div />' },
    }],
  })

  describe('WorkspaceFeatureParamsSettings.vue 滚动容器（OPT-20260809-022）', () => {
    it('T1: 根元素带独立滚动容器样式（max-h 视口减导航栏 + overflow-y-auto）', async () => {
      apiFetchMock.mockResolvedValue({
        ok: true,
        json: async () => ({
          data: {
            use_company_default: true,
            company_config: null,
            id: '',
            extra_env_vars: [],
            providers: [],
          },
          workspace: { allow_personal_feature_params: false },
        }),
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

      const root = wrapper.element
      const classes = root.className.split(' ')
      // 内容区自身作为滚动容器
      expect(classes).toContain('overflow-y-auto')
      // 高度上限 = 视口高度 − var(--app-navbar-h)（74px sticky 导航栏），内容超出时内部滚动而非撑破页面
      expect(classes.some((c) => c.includes('max-h-[calc(100dvh-var(--app-navbar-h))]'))).toBe(true)
      // 保持 flex 布局契约（与 App.vue router-view 传参一致）
      expect(classes).toContain('flex-1')
      expect(classes).toContain('min-w-0')
    })
  })
}

// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] App.layout.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { nextTick } = await import('vue')
  const { describe, expect, it, vi, afterEach } = await import('vitest')
  const { createRouter, createMemoryHistory } = await import('vue-router')

  const NavbarStub = { template: '<div data-view="Navbar" />' }
  const SidebarStub = { template: '<div data-view="Sidebar" />' }
  const ContentStub = { template: '<div data-view="Content" />' }

  const makeRouter = () =>
    createRouter({
      history: createMemoryHistory(),
      routes: [
        {
          path: '/tenant/:tenant/work-panel',
          name: 'work_panel',
          components: { Navbar: NavbarStub, Sidebar: SidebarStub, default: ContentStub },
        },
        {
          path: '/auth/login/',
          name: 'auth_login',
          components: { Navbar: NavbarStub, default: ContentStub },
        },
      ],
    })

  const mountApp = async (path) => {
    const { default: App } = await import('./App.vue')
    const router = makeRouter()
    await router.push(path)
    const wrapper = mount(App, {
      global: {
        plugins: [router],
        stubs: {
          ModalUi: { template: '<div />' },
          PrivacyReconsentGate: { template: '<div />' },
          PhoneVerifyAccessGate: { template: '<div />' },
        },
      },
    })
    await router.isReady()
    await flushPromises()
    await nextTick()
    return wrapper
  }

  describe('App — 整页布局（侧栏占满全高，导航栏位于其右侧）', () => {
    afterEach(() => {
      vi.restoreAllMocks()
    })

    it('租户页：DOM 顺序为 侧栏 → 导航栏 → 内容（导航栏在侧栏右侧的列内）', async () => {
      const wrapper = await mountApp('/tenant/t1/work-panel')

      // 整页 flex 行：h-full + min-h-0 撑开侧栏全高（OPT-20260810-048 h-screen 贴底）
      const row = wrapper.find('div.flex.h-full.min-h-0.min-w-0')
      expect(row.exists()).toBe(true)

      // 侧栏槽位在 flex 行内（位于导航栏左侧；具体顺序由下方全局 DOM 顺序断言保证）
      expect(row.find('div[data-view="Sidebar"]').exists()).toBe(true)

      // 右侧列：导航栏 + 内容，纵向排列（min-h-0 供内容区 overflow-y-auto 滚动）
      const column = wrapper.find('div.flex-1.min-w-0.min-h-0.flex.flex-col')
      expect(column.exists()).toBe(true)
      expect(column.find('div[data-view="Navbar"]').exists()).toBe(true)
      expect(column.find('div[data-view="Content"]').exists()).toBe(true)

      // 全局 DOM 顺序：Sidebar 必须先于 Navbar
      const views = wrapper.findAll('div[data-view]').map((v) => v.attributes('data-view'))
      expect(views).toEqual(['Sidebar', 'Navbar', 'Content'])
    })

    it('登录页（无侧栏槽位）：仅渲染 导航栏 → 内容，无侧栏', async () => {
      const wrapper = await mountApp('/auth/login/')

      const column = wrapper.find('div.flex-1.min-w-0.min-h-0.flex.flex-col')
      expect(column.exists()).toBe(true)
      const views = wrapper.findAll('div[data-view]').map((v) => v.attributes('data-view'))
      expect(views).toEqual(['Navbar', 'Content'])
    })

    it('长页滚动契约：壳层 overflow-hidden，内容区 overflow-y-auto（OPT-20260810-048）', async () => {
      const wrapper = await mountApp('/tenant/t1/work-panel')

      // 壳层 h-screen + overflow-hidden：窗口贴底，禁止 document/body 滚动
      const shell = wrapper.find('div.h-screen.min-h-0.overflow-hidden')
      expect(shell.exists()).toBe(true)
      expect(shell.classes()).toContain('overflow-hidden')

      // 主内容区是唯一纵向滚动容器：flex-1 + min-h-0 提供有界高度 + overflow-y-auto
      const content = wrapper.find('div.flex-1.min-w-0.min-h-0.overflow-y-auto.flex.flex-col')
      expect(content.exists()).toBe(true)
      expect(content.classes()).toContain('overflow-y-auto')
    })
  })
}

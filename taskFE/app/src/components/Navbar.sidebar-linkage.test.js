// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] Navbar.sidebar-linkage.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { nextTick, h } = await import('vue')
  const { beforeEach, describe, expect, it, vi } = await import('vitest')
  const { createRouter, createWebHistory } = await import('vue-router')

  const hoistedMocks = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
    getStoredUserIdMock: vi.fn(() => ''),
  }))

  vi.mock('../utils/apiUtils', () => ({
    apiFetch: hoistedMocks.apiFetchMock,
  }))

  vi.mock('../utils/sessionUserIdUtils', () => ({
    getStoredUserId: hoistedMocks.getStoredUserIdMock,
  }))

  // router-link 桩：to 解析为 href，透传 $attrs
  const RouterLinkStub = {
    props: { to: [String, Object] },
    setup(props, { slots, attrs }) {
      const href = typeof props.to === 'string' ? props.to : props.to?.path || '/'
      return () => h('a', { ...attrs, href }, slots.default?.())
    },
  }

  const DummyComponent = { template: '<div />' }
  const COLLAPSED_KEY = 'tenant-console-sidebar-collapsed'

  const makeRouter = () => createRouter({
    history: createWebHistory(),
    routes: [
      { path: '/tenant/:tenant/projects', component: DummyComponent },
      { path: '/tenant/:tenant/work-panel', component: DummyComponent },
      { path: '/tenant/:tenant/image-market', component: DummyComponent },
      { path: '/tenant/:tenant/settings/:section', component: DummyComponent },
      { path: '/tenant/:tenant/deliverable-systems/', component: DummyComponent },
      { path: '/tenant/:tenant/billing/', component: DummyComponent },
      { path: '/tenant/:tenant/billing/:section', component: DummyComponent },
      { path: '/', component: DummyComponent },
    ],
  })

  // 同页挂载 Sidebar + NavbarUI，验证「点击控制台导航 ↔ 顶部导航栏收起」双向联动
  const mountPair = async () => {
    const { default: Sidebar } = await import('./Sidebar.vue')
    const { default: NavbarUI } = await import('./Navbar.ui.vue')

    const router = makeRouter()
    await router.push('/tenant/t1/work-panel')

    const sidebar = mount(Sidebar, {
      global: { plugins: [router], stubs: { 'router-link': RouterLinkStub } },
    })
    const navbar = mount(NavbarUI, {
      props: {
        navbarType: 'user',
        user: { isAuthenticated: false, isSuperuser: false, username: '' },
        currentUser: { isAuthenticated: true, isSuperuser: false, username: 'testuser', avatarUrl: null, userId: '880000000000000001' },
        isUserAuthenticated: true,
        tenantPath: '/tenant/t1',
        route: { params: { tenant: 't1' }, query: {}, fullPath: '/tenant/t1/work-panel', path: '/tenant/t1/work-panel' },
        userCompanies: [{ id: 't1', name: 'TestCo' }],
        currentTenant: 't1',
      },
      global: {
        stubs: {
          'router-link': RouterLinkStub,
          'router-view': true,
          AccountSwitcherDropdown: true,
        },
      },
    })
    await flushPromises()
    await nextTick()
    return { sidebar, navbar }
  }

  describe('Sidebar ↔ Navbar 收起联动（点击「控制台导航」↔ 顶部导航栏完全收起）', () => {
    beforeEach(() => {
      vi.clearAllMocks()
      localStorage.removeItem('lastActiveTenantId')
      localStorage.removeItem(COLLAPSED_KEY)
      hoistedMocks.getStoredUserIdMock.mockReturnValue('')
    })

    it('点击侧栏「控制台导航」→ 侧栏缩窄 + 顶部导航栏完全隐藏；再点 → 两侧同步恢复', async () => {
      const { sidebar, navbar } = await mountPair()

      const aside = sidebar.find('aside.tenant-console-sidebar')
      // 初始展开态：侧栏标题可见，顶部导航栏完整
      expect(aside.find('h3').text()).toContain('控制台导航')
      expect(navbar.find('nav[data-alias="cmp-navbar-main"]').exists()).toBe(true)
      expect(navbar.find('a[data-testid="nav-work-panel"]').exists()).toBe(true)

      // 点击「控制台导航」→ 收起
      await aside.find('h3').trigger('click')
      await nextTick()
      expect(aside.classes()).toContain('tenant-console-sidebar-collapsed')
      expect(navbar.find('nav[data-alias="cmp-navbar-main"]').exists()).toBe(false)
      expect(navbar.find('a[data-testid="nav-work-panel"]').exists()).toBe(false)
      expect(localStorage.getItem(COLLAPSED_KEY)).toBe('1')

      // 再次点击「控制台导航」→ 展开
      await aside.find('h3').trigger('click')
      await nextTick()
      expect(aside.classes()).not.toContain('tenant-console-sidebar-collapsed')
      expect(aside.find('h3').text()).toContain('控制台导航')
      expect(navbar.find('nav[data-alias="cmp-navbar-main"]').exists()).toBe(true)
      expect(navbar.find('a[data-testid="nav-work-panel"]').exists()).toBe(true)
      expect(localStorage.getItem(COLLAPSED_KEY)).toBe('0')
    })
  })
}

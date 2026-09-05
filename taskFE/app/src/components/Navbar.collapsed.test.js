// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] Navbar.collapsed.test.js requires vitest runtime')
} else {
  const { mount } = await import('@vue/test-utils')
  const { describe, expect, it, vi, afterEach } = await import('vitest')
  const { nextTick, h } = await import('vue')

  const COLLAPSED_KEY = 'tenant-console-sidebar-collapsed'

  // 模拟「上次会话已收起」：必须在导入 Navbar.ui.vue 之前写入 localStorage，
  // 收起状态为模块级单例，首次导入时从 localStorage 初始化（持久化恢复路径）。
  localStorage.setItem(COLLAPSED_KEY, '1')

  // useRouter mock：Navbar.ui 内部仅作跳转用，本测试不涉及
  const mocks = vi.hoisted(() => ({ routerPush: vi.fn() }))
  vi.mock('vue-router', () => ({
    useRouter: () => ({ push: (...args) => mocks.routerPush(...args) }),
  }))

  // router-link 桩：to 解析为 href，透传 $attrs
  const RouterLinkStub = {
    props: { to: [String, Object] },
    setup(props, { slots, attrs }) {
      const href = typeof props.to === 'string' ? props.to : props.to?.path || '/'
      return () => h('a', { ...attrs, href }, slots.default?.())
    },
  }

  const { default: NavbarUI } = await import('./Navbar.ui.vue')
  const { useTenantConsoleSidebarCollapse } = await import('../composables/useTenantConsoleSidebarCollapse.js')

  const mountOptions = (props) => ({
    props,
    global: {
      stubs: {
        'router-link': RouterLinkStub,
        'router-view': true,
        AccountSwitcherDropdown: true,
      },
    },
  })

  const baseProps = () => ({
    navbarType: 'user',
    user: { isAuthenticated: false, isSuperuser: false, username: '' },
    currentUser: { isAuthenticated: true, isSuperuser: false, username: 'testuser', avatarUrl: null, userId: '880000000000000001' },
    isUserAuthenticated: true,
    tenantPath: '/tenant/880000000000000002',
    route: {
      params: { tenant: '880000000000000002' },
      query: {},
      fullPath: '/tenant/880000000000000002/work-panel/',
      path: '/tenant/880000000000000002/work-panel/',
    },
    userCompanies: [{ id: '880000000000000002', name: 'TestCo' }],
    currentTenant: '880000000000000002',
  })

  describe('Navbar — 顶部导航栏完全收起（与侧栏「控制台导航」共享状态，无独立切换按钮）', () => {
    afterEach(() => {
      // 复位共享单例状态，避免污染同文件后续用例
      const { expandSidebar } = useTenantConsoleSidebarCollapse()
      expandSidebar()
      localStorage.removeItem(COLLAPSED_KEY)
    })

    it('localStorage 持久化收起态（1）→ 导航栏完全隐藏（无 nav 元素、无切换按钮）', () => {
      // 单例从 localStorage 初始化即为收起（持久化恢复路径）
      const { collapsed } = useTenantConsoleSidebarCollapse()
      expect(collapsed.value).toBe(true)

      const wrapper = mount(NavbarUI, mountOptions(baseProps()))
      // 完全收起：整个 nav 不渲染，导航栏元素与内容均不存在
      expect(wrapper.find('nav[data-alias="cmp-navbar-main"]').exists()).toBe(false)
      expect(wrapper.find('a[href="/"]').exists()).toBe(false)
      expect(wrapper.find('a[data-testid="nav-work-panel"]').exists()).toBe(false)
    })

    it('共享状态展开（单例 expandSidebar）→ 导航栏完整渲染', () => {
      const { expandSidebar } = useTenantConsoleSidebarCollapse()
      expandSidebar()

      const wrapper = mount(NavbarUI, mountOptions(baseProps()))
      const nav = wrapper.find('nav[data-alias="cmp-navbar-main"]')
      expect(nav.exists()).toBe(true)
      expect(wrapper.find('a[href="/"]').text()).toContain('云端开发')
      expect(wrapper.find('a[data-testid="nav-work-panel"]').exists()).toBe(true)
      expect(wrapper.find('a[data-testid="nav-pricing"]').exists()).toBe(true)
    })

    it('共享状态收起（单例 toggleCollapsed）→ 导航栏随即隐藏；再展开 → 随即恢复', async () => {
      const { expandSidebar, toggleCollapsed } = useTenantConsoleSidebarCollapse()
      expandSidebar()
      await nextTick()

      const wrapper = mount(NavbarUI, mountOptions(baseProps()))
      expect(wrapper.find('nav[data-alias="cmp-navbar-main"]').exists()).toBe(true)

      // 收起（模拟侧栏「控制台导航」点击）→ 导航栏完全隐藏，localStorage 写 1
      toggleCollapsed()
      await nextTick()
      expect(wrapper.find('nav[data-alias="cmp-navbar-main"]').exists()).toBe(false)
      expect(localStorage.getItem(COLLAPSED_KEY)).toBe('1')

      // 再展开 → 导航栏恢复
      toggleCollapsed()
      await nextTick()
      expect(wrapper.find('nav[data-alias="cmp-navbar-main"]').exists()).toBe(true)
      expect(wrapper.find('a[data-testid="nav-work-panel"]').exists()).toBe(true)
      expect(localStorage.getItem(COLLAPSED_KEY)).toBe('0')
    })

    it('system_admin 通栏导航不受收起状态影响（始终完整渲染）', async () => {
      const { toggleCollapsed } = useTenantConsoleSidebarCollapse()
      toggleCollapsed() // 置为收起，验证 system_admin 不受影响
      await nextTick()

      const props = baseProps()
      props.navbarType = 'system_admin'
      const wrapper = mount(NavbarUI, mountOptions(props))
      const nav = wrapper.find('nav[data-alias="cmp-navbar-main"]')
      expect(nav.exists()).toBe(true)
      expect(wrapper.find('a[data-testid="nav-work-panel"]').exists()).toBe(true)
      expect(wrapper.find('a[data-testid="nav-pricing"]').exists()).toBe(true)
    })

    it('生产路径：navbarType 默认 user + /system-admin/ + 收起 → 顶栏仍渲染（无恢复入口）', async () => {
      const { toggleCollapsed, collapsed } = useTenantConsoleSidebarCollapse()
      toggleCollapsed()
      await nextTick()
      expect(collapsed.value).toBe(true)

      const props = baseProps()
      props.navbarType = 'user'
      props.route = { params: {}, query: { accessCode: 'DR2AKvP9J9' }, fullPath: '/system-admin/?accessCode=DR2AKvP9J9', path: '/system-admin/' }
      const wrapper = mount(NavbarUI, mountOptions(props))
      const nav = wrapper.find('nav[data-alias="cmp-navbar-main"]')
      expect(nav.exists()).toBe(true)
      expect(wrapper.find('a[href="/"]').text()).toContain('云端开发')
    })

    it('生产路径：收起态下首页 / 与 /pricing/ 仍渲染顶栏', async () => {
      const { toggleCollapsed, collapsed } = useTenantConsoleSidebarCollapse()
      toggleCollapsed()
      await nextTick()
      expect(collapsed.value).toBe(true)

      for (const path of ['/', '/pricing/']) {
        const props = baseProps()
        props.route = { params: {}, query: {}, fullPath: path, path }
        const wrapper = mount(NavbarUI, mountOptions(props))
        expect(wrapper.find('nav[data-alias="cmp-navbar-main"]').exists()).toBe(true)
        wrapper.unmount()
      }
    })
  })
}

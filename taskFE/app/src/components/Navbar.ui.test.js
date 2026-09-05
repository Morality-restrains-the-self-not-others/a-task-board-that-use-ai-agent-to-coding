// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] Navbar.ui.test.js requires vitest runtime')
} else {
  const { mount } = await import('@vue/test-utils')
  const { describe, expect, it, vi } = await import('vitest')
  const { h } = await import('vue')

  // 导航桩：本组件已改为真实 <a href>，不再依赖 useRouter.push
  vi.mock('vue-router', () => ({
    useRouter: () => ({ push: vi.fn() }),
  }))

  // router-link 桩：遗留子组件若仍用 router-link 时兼容
  const RouterLinkStub = {
    props: { to: [String, Object] },
    setup(props, { slots, attrs }) {
      const href = typeof props.to === 'string' ? props.to : props.to?.path || '/'
      return () => h('a', { ...attrs, href }, slots.default?.())
    },
  }

  const { default: NavbarUI } = await import('./Navbar.ui.vue')

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
    route: { params: {}, query: {}, fullPath: '/', path: '/' },
	userCompanies: [],
    currentTenant: '880000000000000002',
    gitResources: [],
  })

  describe('Navbar.ui — 工作面板链接 workPanelPath', () => {
    it('导航栏不再挂载任务搜索框（搜索已迁到工作面板标题行）', () => {
      const wrapper = mount(NavbarUI, mountOptions(baseProps()))
      expect(wrapper.find('[data-testid="work-panel-task-search"]').exists()).toBe(false)
      expect(wrapper.find('#work-panel-task-search-input').exists()).toBe(false)
    })

    it('有 tenantPath 时生成正确的 /tenant/:id/work-panel/ 路径', () => {
      const props = baseProps()
      props.tenantPath = '/tenant/111111'
      props.userCompanies = [{ id: '111111', name: 'TestCo' }]
      const wrapper = mount(NavbarUI, mountOptions(props))
      const link = wrapper.find('a[data-testid="nav-work-panel"]')
      expect(link.exists()).toBe(true)
      expect(link.attributes('href')).toBe('/tenant/111111/work-panel/')
    })

    it('没有 tenantPath 但有 currentTenant 时，应回退到 currentTenant 构造工作面板路径', () => {
      const props = baseProps()
      props.tenantPath = ''
      props.currentTenant = '222222'
      props.userCompanies = [{ id: '222222', name: 'TestCo' }]
      const wrapper = mount(NavbarUI, mountOptions(props))
      const link = wrapper.find('a[data-testid="nav-work-panel"]')
      expect(link.exists()).toBe(true)
      expect(link.attributes('href')).toBe('/tenant/222222/work-panel/')
    })

    it('同时没有 tenantPath 和 currentTenant 且无公司且无 localStorage 时，渲染「开始使用」引导链接', () => {
      const props = baseProps()
      props.tenantPath = ''
      props.currentTenant = ''
      props.userCompanies = []
      localStorage.removeItem('lastActiveTenantId')
      const wrapper = mount(NavbarUI, mountOptions(props))
      // 无公司且无 localStorage 回退时不应渲染「工作面板」，应渲染「开始使用」
      expect(wrapper.find('a[data-testid="nav-work-panel"]').exists()).toBe(false)
      const onboardingLink = wrapper.find('a[data-testid="nav-onboarding"]')
      expect(onboardingLink.exists()).toBe(true)
      expect(onboardingLink.attributes('href')).toBe('/onboarding/')
    })

    it('API 返回空公司列表但 localStorage 有回退租户时，仍渲染「工作面板」链接', () => {
      localStorage.setItem('lastActiveTenantId', '777777')
      const props = baseProps()
      props.tenantPath = ''
      props.currentTenant = ''
      props.userCompanies = []  // API returns no companies
      const wrapper = mount(NavbarUI, mountOptions(props))
      // hasTenantContext: localStorage fallback → 仍显示「工作面板」
      const link = wrapper.find('a[data-testid="nav-work-panel"]')
      expect(link.exists()).toBe(true)
      expect(link.attributes('href')).toBe('/tenant/777777/work-panel/')
      localStorage.removeItem('lastActiveTenantId')
    })

    it('同时没有 tenantPath 和 currentTenant 但有公司时，回退到 /onboarding/ 安全网', () => {
      const props = baseProps()
      props.tenantPath = ''
      props.currentTenant = ''
      props.userCompanies = [{ id: '880000000000000002', name: 'TestCo' }]
      localStorage.removeItem('lastActiveTenantId')
      const wrapper = mount(NavbarUI, mountOptions(props))
      const link = wrapper.find('a[data-testid="nav-work-panel"]')
      expect(link.exists()).toBe(true)
      // 无 tenantPath、无 currentTenant、无 localStorage → 回退到 onboarding
      expect(link.attributes('href')).toBe('/onboarding/')
    })

    it('有 tenantPath + accessCode query 时保留 accessCode 参数', () => {
      const props = baseProps()
      props.tenantPath = '/tenant/333333'
      props.userCompanies = [{ id: '333333', name: 'TestCo' }]
      props.route = { params: {}, query: { accessCode: 'abc123' }, fullPath: '/', path: '/' }
      const wrapper = mount(NavbarUI, mountOptions(props))
      const link = wrapper.find('a[data-testid="nav-work-panel"]')
      expect(link.exists()).toBe(true)
      expect(link.attributes('href')).toBe('/tenant/333333/work-panel/?accessCode=abc123')
    })

    it('currentTenant 回退时也保留 accessCode query', () => {
      const props = baseProps()
      props.tenantPath = ''
      props.currentTenant = '444444'
      props.userCompanies = [{ id: '444444', name: 'TestCo' }]
      props.route = { params: {}, query: { accessCode: 'xyz789' }, fullPath: '/', path: '/' }
      const wrapper = mount(NavbarUI, mountOptions(props))
      const link = wrapper.find('a[data-testid="nav-work-panel"]')
      expect(link.exists()).toBe(true)
      expect(link.attributes('href')).toBe('/tenant/444444/work-panel/?accessCode=xyz789')
    })

    it('超管无公司时渲染「系统管理」而不渲染「工作面板」', () => {
      const props = baseProps()
      // v63 RBAC: 平台角色（super_admin/employee）驱动系统管理入口
      props.currentUser = { ...props.currentUser, isPlatformStaff: true }
      props.userCompanies = []
      props.tenantPath = ''
      props.currentTenant = ''
      localStorage.removeItem('lastActiveTenantId')
      const wrapper = mount(NavbarUI, mountOptions(props))
      expect(wrapper.find('a[data-testid="nav-work-panel"]').exists()).toBe(false)
      expect(wrapper.find('[data-testid="nav-company-switcher"]').exists()).toBe(false)
      const adminLink = wrapper.find('a[href="/system-admin/"]')
      expect(adminLink.exists()).toBe(true)
    })

    it('平台角色被邀请进单公司时并列「系统管理」与「工作面板」', () => {
      const props = baseProps()
      props.currentUser = { ...props.currentUser, isPlatformStaff: true }
      props.userCompanies = [{ id: '111', name: '受邀公司' }]
      props.tenantPath = '/tenant/111'
      props.currentTenant = '111'
      const wrapper = mount(NavbarUI, mountOptions(props))
      expect(wrapper.find('a[href="/system-admin/"]').exists()).toBe(true)
      expect(wrapper.find('a[data-testid="nav-work-panel"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="nav-onboarding"]').exists()).toBe(false)
    })

    it('平台角色加入多公司时并列「系统管理」与公司切换下拉', async () => {
      const props = baseProps()
      props.currentUser = { ...props.currentUser, isPlatformStaff: true }
      props.tenantPath = '/tenant/111'
      props.currentTenant = '111'
      props.userCompanies = [
        { id: '111', name: '公司A' },
        { id: '222', name: '公司B' },
      ]
      const wrapper = mount(NavbarUI, mountOptions(props))
      expect(wrapper.find('a[href="/system-admin/"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="nav-company-switcher"]').exists()).toBe(true)
      await wrapper.find('button[data-testid="nav-work-panel"]').trigger('click')
      const items = wrapper.findAll('[data-testid="nav-company-menu"] [role="menuitem"]')
      expect(items.length).toBe(2)
      expect(items.map((i) => i.text()).join('|')).toContain('公司B')
    })

    it('遗留 isSuperuser 兜底：/me/ 平台角色字段缺失时仍渲染「系统管理」', () => {
      const props = baseProps()
      props.currentUser = { ...props.currentUser, isSuperuser: true, isPlatformStaff: false }
      props.tenantPath = ''
      props.currentTenant = ''
      props.userCompanies = []
      localStorage.removeItem('lastActiveTenantId')
      const wrapper = mount(NavbarUI, mountOptions(props))
      expect(wrapper.find('a[data-testid="nav-onboarding"]').exists()).toBe(false)
      expect(wrapper.find('a[href="/system-admin/"]').exists()).toBe(true)
    })

    it('平台角色 isPlatformStaff（v63 RBAC）渲染「系统管理」，即使无公司也绝不显示「开始使用」', () => {
      const props = baseProps()
      props.currentUser = { ...props.currentUser, isSuperuser: false, isPlatformStaff: true }
      props.tenantPath = ''
      props.currentTenant = ''
      props.userCompanies = [] // 平台员工无公司（不应自动创建）
      localStorage.removeItem('lastActiveTenantId')
      const wrapper = mount(NavbarUI, mountOptions(props))
      expect(wrapper.find('a[data-testid="nav-work-panel"]').exists()).toBe(false)
      expect(wrapper.find('a[data-testid="nav-onboarding"]').exists()).toBe(false)
      const adminLink = wrapper.find('a[href="/system-admin/"]')
      expect(adminLink.exists()).toBe(true)
    })

    it('localStorage 有 lastActiveTenantId 时，tenantPath/currentTenant 均为空也应回退到该租户', () => {
      localStorage.setItem('lastActiveTenantId', '555555')
      const props = baseProps()
      props.tenantPath = ''
      props.currentTenant = ''
      props.userCompanies = [{ id: '555555', name: 'TestCo' }]
      const wrapper = mount(NavbarUI, mountOptions(props))
      const link = wrapper.find('a[data-testid="nav-work-panel"]')
      expect(link.exists()).toBe(true)
      expect(link.attributes('href')).toBe('/tenant/555555/work-panel/')
      localStorage.removeItem('lastActiveTenantId')
    })

    it('localStorage 回退也保留 accessCode query', () => {
      localStorage.setItem('lastActiveTenantId', '666666')
      const props = baseProps()
      props.tenantPath = ''
      props.currentTenant = ''
      props.userCompanies = [{ id: '666666', name: 'TestCo' }]
      props.route = { params: {}, query: { accessCode: 'fallbackAc' }, fullPath: '/', path: '/' }
      const wrapper = mount(NavbarUI, mountOptions(props))
      const link = wrapper.find('a[data-testid="nav-work-panel"]')
      expect(link.exists()).toBe(true)
      expect(link.attributes('href')).toBe('/tenant/666666/work-panel/?accessCode=fallbackAc')
      localStorage.removeItem('lastActiveTenantId')
    })
  })

  describe('Navbar.ui — 公司切换下拉（多公司，工作面板链接改造）', () => {
    const multiCompanyProps = () => {
      const props = baseProps()
      props.tenantPath = '/tenant/111'
      props.currentTenant = '111'
      props.userCompanies = [
        { id: '111', name: '公司A' },
        { id: '222', name: '公司B' },
      ]
      return props
    }

    it('多公司时渲染下拉触发按钮而非普通链接，按钮显示当前公司名', () => {
      const wrapper = mount(NavbarUI, mountOptions(multiCompanyProps()))
      expect(wrapper.find('a[data-testid="nav-work-panel"]').exists()).toBe(false)
      const btn = wrapper.find('button[data-testid="nav-work-panel"]')
      expect(btn.exists()).toBe(true)
      expect(btn.text()).toContain('公司A')
      expect(btn.attributes('aria-haspopup')).toBe('menu')
      expect(btn.attributes('aria-expanded')).toBe('false')
    })

    it('当前公司名优先解析路由租户（tenantPath），其次 currentTenant', () => {
      const props = multiCompanyProps()
      props.tenantPath = '/tenant/222'
      const wrapper = mount(NavbarUI, mountOptions(props))
      expect(wrapper.find('button[data-testid="nav-work-panel"]').text()).toContain('公司B')
    })

    it('点击触发按钮展开菜单，列出全部公司名', async () => {
      const wrapper = mount(NavbarUI, mountOptions(multiCompanyProps()))
      const btn = wrapper.find('button[data-testid="nav-work-panel"]')
      await btn.trigger('click')
      expect(btn.attributes('aria-expanded')).toBe('true')
      const menu = wrapper.find('[data-testid="nav-company-menu"]')
      expect(menu.exists()).toBe(true)
      const items = wrapper.findAll('[data-testid="nav-company-menu"] [role="menuitem"]')
      expect(items.length).toBe(2)
      expect(items.map((i) => i.text()).join('|')).toContain('公司A')
      expect(items.map((i) => i.text()).join('|')).toContain('公司B')
    })

    it('当前公司菜单项标记 data-current=true', async () => {
      const wrapper = mount(NavbarUI, mountOptions(multiCompanyProps()))
      await wrapper.find('button[data-testid="nav-work-panel"]').trigger('click')
      const items = wrapper.findAll('[data-testid="nav-company-menu"] [role="menuitem"]')
      expect(items[0].attributes('data-current')).toBe('true')
      expect(items[1].attributes('data-current')).toBe('false')
    })

    it('公司菜单项是真实 a[href] 指向该公司工作面板（含当前公司，禁止 button 冒充链接）', async () => {
      const wrapper = mount(NavbarUI, mountOptions(multiCompanyProps()))
      await wrapper.find('button[data-testid="nav-work-panel"]').trigger('click')
      const items = wrapper.findAll('[data-testid="nav-company-menu"] [role="menuitem"]')
      expect(items[0].element.tagName).toBe('A')
      expect(items[0].attributes('href')).toBe('/tenant/111/work-panel/')
      expect(items[1].element.tagName).toBe('A')
      expect(items[1].attributes('href')).toBe('/tenant/222/work-panel/')
    })

    it('任务详情页当前公司行仍指向工作面板并保留 accessCode', async () => {
      const props = multiCompanyProps()
      props.route = {
        params: { tenant: '111' },
        query: { accessCode: 'DR2AKvP9J9' },
        fullPath: '/tenant/111/workspace/ws_-1/task-detail/task_1/?accessCode=DR2AKvP9J9',
        path: '/tenant/111/workspace/ws_-1/task-detail/task_1/',
      }
      const wrapper = mount(NavbarUI, mountOptions(props))
      await wrapper.find('button[data-testid="nav-work-panel"]').trigger('click')
      const current = wrapper.find('[data-testid="nav-company-menu"] [role="menuitem"][data-current="true"]')
      expect(current.exists()).toBe(true)
      expect(current.element.tagName).toBe('A')
      expect(current.attributes('href')).toBe('/tenant/111/work-panel/?accessCode=DR2AKvP9J9')
    })

    it('选择非当前公司 → emit company-switched 携带公司 id，且菜单关闭', async () => {
      const wrapper = mount(NavbarUI, mountOptions(multiCompanyProps()))
      await wrapper.find('button[data-testid="nav-work-panel"]').trigger('click')
      const items = wrapper.findAll('[data-testid="nav-company-menu"] [role="menuitem"]')
      await items[1].trigger('click')
      expect(wrapper.emitted('company-switched')).toBeTruthy()
      expect(wrapper.emitted('company-switched')[0]).toEqual(['222'])
      expect(wrapper.find('[data-testid="nav-company-menu"]').exists()).toBe(false)
    })

    it('选择当前公司 → 仍可导航（emit company-switched），菜单关闭', async () => {
      const wrapper = mount(NavbarUI, mountOptions(multiCompanyProps()))
      await wrapper.find('button[data-testid="nav-work-panel"]').trigger('click')
      const items = wrapper.findAll('[data-testid="nav-company-menu"] [role="menuitem"]')
      await items[0].trigger('click')
      expect(wrapper.emitted('company-switched')).toBeTruthy()
      expect(wrapper.emitted('company-switched')[0]).toEqual(['111'])
      expect(wrapper.find('[data-testid="nav-company-menu"]').exists()).toBe(false)
    })

    it('点击遮罩关闭菜单', async () => {
      const wrapper = mount(NavbarUI, mountOptions(multiCompanyProps()))
      await wrapper.find('button[data-testid="nav-work-panel"]').trigger('click')
      const overlay = wrapper.find('[data-testid="nav-company-menu-overlay"]')
      expect(overlay.exists()).toBe(true)
      await overlay.trigger('click')
      expect(wrapper.find('[data-testid="nav-company-menu"]').exists()).toBe(false)
    })

    it('公司菜单遮罩须带独立 dropdown-click-outside-overlay，避免继承全局模态灰罩', async () => {
      // 回归：模态样式已收窄为显式 .app-modal-overlay；下拉点击外部关闭使用独立
      // .dropdown-click-outside-overlay（自带 fixed+inset-0 定位 + 透明背景），不再依赖
      // 全局 .fixed.inset-0 规则。
      const wrapper = mount(NavbarUI, mountOptions(multiCompanyProps()))
      await wrapper.find('button[data-testid="nav-work-panel"]').trigger('click')
      const overlay = wrapper.find('[data-testid="nav-company-menu-overlay"]')
      expect(overlay.exists()).toBe(true)
      expect(overlay.classes()).toContain('dropdown-click-outside-overlay')
      // 不得自带半透明黑底 utility（灰罩应由 modal 专用样式提供）
      expect(overlay.classes().some((c) => /^bg-black/.test(c) || /^bg-gray/.test(c))).toBe(false)
    })

    it('Escape 键关闭菜单', async () => {
      const wrapper = mount(NavbarUI, mountOptions(multiCompanyProps()))
      const btn = wrapper.find('button[data-testid="nav-work-panel"]')
      await btn.trigger('click')
      expect(wrapper.find('[data-testid="nav-company-menu"]').exists()).toBe(true)
      await btn.trigger('keydown', { key: 'Escape' })
      expect(wrapper.find('[data-testid="nav-company-menu"]').exists()).toBe(false)
    })

    it('单公司时仍渲染普通「工作面板」链接，不渲染下拉', () => {
      const props = baseProps()
      props.tenantPath = '/tenant/111'
      props.currentTenant = '111'
      props.userCompanies = [{ id: '111', name: '公司A' }]
      const wrapper = mount(NavbarUI, mountOptions(props))
      expect(wrapper.find('a[data-testid="nav-work-panel"]').exists()).toBe(true)
      expect(wrapper.find('button[data-testid="nav-work-panel"]').exists()).toBe(false)
    })

    it('多公司时不再渲染旧独立 select 公司切换器', () => {
      const wrapper = mount(NavbarUI, mountOptions(multiCompanyProps()))
      expect(wrapper.find('select').exists()).toBe(false)
    })
  })

  describe('Navbar.ui — 代码仓库按仓库资源跳转', () => {
    const GIT_URL = 'https://gitlab-tencent-sh-1.daydaymoney.com/'
    const GIT_URL_SH5 = 'https://gitlab.daydaymoney.com/'
    const CURRENT = '/tenant/877397588196749312/settings/gitlab-connection/'

    const pageProps = () => {
      const props = baseProps()
      props.route = { params: { tenant: '877397588196749312' }, query: {}, fullPath: CURRENT, path: CURRENT }
      props.tenantPath = '/tenant/877397588196749312'
      props.currentTenant = '877397588196749312'
      props.userCompanies = [{ id: '877397588196749312', name: 'Co' }]
      return props
    }

    it('无仓库资源且列表已就绪时 href 为 /pricing/，并保留 accessCode（普通会员亦同）', () => {
      const props = pageProps()
      props.membershipTier = 'normal'
      props.gitResources = []
      props.gitResourcesStatus = 'ready'
      props.route = {
        ...props.route,
        query: { accessCode: 'cGSjehe9Ed' },
        fullPath: `${CURRENT}?accessCode=cGSjehe9Ed`,
      }
      const wrapper = mount(NavbarUI, mountOptions(props))
      const link = wrapper.find('a[data-testid="nav-git-service"]')
      expect(link.exists()).toBe(true)
      expect(link.element.tagName).toBe('A')
      expect(link.attributes('href')).toBe('/pricing/?accessCode=cGSjehe9Ed')
      expect(wrapper.find('[data-testid="nav-git-service-vip-badge"]').exists()).toBe(false)
    })

    it('gitlab-resources 拉取失败时 href 仍为当前页（fail-open）', () => {
      const props = pageProps()
      props.gitResources = []
      props.gitResourcesStatus = 'error'
      const wrapper = mount(NavbarUI, mountOptions(props))
      const link = wrapper.find('a[data-testid="nav-git-service"]')
      expect(link.attributes('href')).toBe(CURRENT)
      expect(link.attributes('href')).not.toContain('/pricing/')
    })

    it('仓库列表尚未就绪时 href 为当前页', () => {
      const props = pageProps()
      props.gitResources = []
      props.gitResourcesStatus = 'unknown'
      const wrapper = mount(NavbarUI, mountOptions(props))
      const link = wrapper.find('a[data-testid="nav-git-service"]')
      expect(link.attributes('href')).toBe(CURRENT)
      expect(link.attributes('href')).not.toContain('/pricing/')
    })

    it('普通会员获赠单区资源时渲染下拉，菜单项直链 gitlab_web_url', async () => {
      const props = pageProps()
      props.membershipTier = 'normal'
      props.gitResources = [{
        region: 'tencent-sh-1',
        region_name: '腾讯上海一区',
        gitlab_web_url: GIT_URL,
        provisioning_status: 'active',
      }]
      const wrapper = mount(NavbarUI, mountOptions(props))
      const trigger = wrapper.find('[data-testid="nav-git-service"]')
      expect(trigger.exists()).toBe(true)
      expect(trigger.element.tagName).toBe('BUTTON')
      expect(wrapper.find('[data-testid="nav-git-service-menu"]').exists()).toBe(false)
      await wrapper.find('[data-testid="nav-git-service-wrap"]').trigger('mouseenter')
      const menu = wrapper.find('[data-testid="nav-git-service-menu"]')
      expect(menu.exists()).toBe(true)
      const items = wrapper.findAll('a[data-testid="nav-git-service-region"]')
      expect(items).toHaveLength(1)
      expect(items[0].attributes('href')).toBe(GIT_URL)
      expect(items[0].attributes('target')).toBe('_blank')
      expect(items[0].text()).toContain('腾讯上海一区')
    })

    it('多区资源时渲染下拉，菜单项为各区域真实外链', async () => {
      const props = pageProps()
      props.membershipTier = 'normal'
      props.gitResources = [
        { region: 'tencent-sh-1', region_name: '腾讯上海一区', gitlab_web_url: GIT_URL, provisioning_status: 'active' },
        { region: 'tencent-shanghai-5', region_name: '腾讯上海五区', gitlab_web_url: GIT_URL_SH5, provisioning_status: 'active' },
      ]
      const wrapper = mount(NavbarUI, mountOptions(props))
      const trigger = wrapper.find('[data-testid="nav-git-service"]')
      expect(trigger.exists()).toBe(true)
      expect(wrapper.find('[data-testid="nav-git-service-menu"]').exists()).toBe(false)
      await trigger.trigger('click')
      const menu = wrapper.find('[data-testid="nav-git-service-menu"]')
      expect(menu.exists()).toBe(true)
      const items = wrapper.findAll('a[data-testid="nav-git-service-region"]')
      expect(items).toHaveLength(2)
      expect(items[0].attributes('href')).toBe(GIT_URL)
      expect(items[1].attributes('href')).toBe(GIT_URL_SH5)
      expect(items[0].attributes('target')).toBe('_blank')
    })

    it('悬停打开下拉，鼠标离开收起', async () => {
      const props = pageProps()
      props.gitResources = [
        { region: 'tencent-sh-1', region_name: '腾讯上海一区', gitlab_web_url: GIT_URL },
        { region: 'tencent-shanghai-5', region_name: '腾讯上海五区', gitlab_web_url: GIT_URL_SH5 },
      ]
      const wrapper = mount(NavbarUI, mountOptions(props))
      const wrap = wrapper.find('[data-testid="nav-git-service-wrap"]')
      await wrap.trigger('mouseenter')
      expect(wrapper.find('[data-testid="nav-git-service-menu"]').exists()).toBe(true)
      await wrap.trigger('mouseleave')
      expect(wrapper.find('[data-testid="nav-git-service-menu"]').exists()).toBe(false)
    })

    it('membershipTier=vip1 且无资源已就绪时显示 VIP1 角标，href 为 /pricing/', () => {
      const props = pageProps()
      props.membershipTier = 'vip1'
      props.gitResources = []
      props.gitResourcesStatus = 'ready'
      const wrapper = mount(NavbarUI, mountOptions(props))
      const link = wrapper.find('a[data-testid="nav-git-service"]')
      expect(link.attributes('href')).toBe('/pricing/')
      const badge = wrapper.find('[data-testid="nav-git-service-vip-badge"]')
      expect(badge.exists()).toBe(true)
      expect(badge.text()).toContain('VIP1')
    })

    it('未登录不渲染代码仓库入口', () => {
      const props = pageProps()
      props.isUserAuthenticated = false
      props.gitResources = [{ region: 'tencent-sh-1', gitlab_web_url: GIT_URL }]
      const wrapper = mount(NavbarUI, mountOptions(props))
      expect(wrapper.find('[data-testid="nav-git-service"]').exists()).toBe(false)
    })
  })

  describe('Navbar.ui — 品牌区 logo + 产品名（OPT-20260824-012）', () => {
    it('品牌区展示 favicon logo 与「云端开发」，与登录页视觉一致', () => {
      const wrapper = mount(NavbarUI, mountOptions(baseProps()))
      const brand = wrapper.get('[data-testid="navbar-brand"]')
      const img = brand.find('img')
      expect(img.exists()).toBe(true)
      expect(img.attributes('src')).toBe('/img/icon128.png')
      expect(img.attributes('alt')).toBe('云端开发')
      expect(brand.text()).toContain('云端开发')
    })
  })
}

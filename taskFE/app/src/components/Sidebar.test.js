// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] Sidebar.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { nextTick, h } = await import('vue')
  const { beforeEach, describe, expect, it, vi } = await import('vitest')
  const { createRouter, createWebHistory } = await import('vue-router')

  const hoistedMocks = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
    getStoredUserIdMock: vi.fn(() => ''),
    // 默认授予租户管理员全套权限，兼容既有「公司租户可见人员管理」用例
    permsCodes: [
      'project:view', 'project:manage', 'task:view', 'task:manage',
      'cloud:view', 'cloud:manage', 'member:manage', 'group:manage',
      'group-members:manage', 'company:view', 'company:manage',
      'workspace:manage', 'billing:view', 'billing:manage',
    ],
  }))

  vi.mock('../utils/apiUtils', () => ({
    apiFetch: hoistedMocks.apiFetchMock,
  }))

  // OPT-20260807-004 后 userId 判定走 getStoredUserId（localStorage 主 + cookie 回退）
  vi.mock('../utils/sessionUserIdUtils', () => ({
    getStoredUserId: hoistedMocks.getStoredUserIdMock,
  }))

  vi.mock('../composables/usePermissions.js', () => ({
    usePermissions: () => ({
      load: vi.fn(async () => {}),
      reload: vi.fn(async () => {}),
      invalidate: vi.fn(),
      hasPerm: (cid, code) => Array.isArray(hoistedMocks.permsCodes) && hoistedMocks.permsCodes.includes(code),
      hasPlatformPerm: () => false,
      isPlatformRole: () => false,
    }),
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
  const LAST_TENANT_STORAGE_KEY = 'lastActiveTenantId'

  const makeRouter = () => createRouter({
    history: createWebHistory(),
    routes: [
      { path: '/tenant/:tenant/projects', component: DummyComponent },
      { path: '/tenant/:tenant/work-panel', component: DummyComponent },
      { path: '/tenant/:tenant/image-market', component: DummyComponent },
      { path: '/tenant/:tenant/people/invite/', component: DummyComponent },
      { path: '/tenant/:tenant/people/manage/', component: DummyComponent },
      { path: '/tenant/:tenant/people/groups/', component: DummyComponent },
      { path: '/tenant/:tenant/people/access/', component: DummyComponent },
      { path: '/tenant/:tenant/settings/:section', component: DummyComponent },
      { path: '/tenant/:tenant/deliverable-systems/', component: DummyComponent },
      { path: '/tenant/:tenant/billing/', component: DummyComponent },
      { path: '/tenant/:tenant/billing/:section', component: DummyComponent },
      { path: '/', component: DummyComponent },
    ],
  })

  describe('Sidebar — tenantPath 计算属性 localStorage 回退', () => {
    beforeEach(() => {
      vi.clearAllMocks()
      localStorage.removeItem(LAST_TENANT_STORAGE_KEY)
      hoistedMocks.getStoredUserIdMock.mockReturnValue('')
    })

    it('route 有 tenant 参数时，初始化时直接使用路由租户', async () => {
      const router = makeRouter()
      await router.push('/tenant/tenant-from-route/projects')

      const { default: Sidebar } = await import('./Sidebar.vue')
      const wrapper = mount(Sidebar, {
        global: { plugins: [router], stubs: { 'router-link': RouterLinkStub } },
      })
      await flushPromises()
      await nextTick()

      const projectLink = wrapper.findAll('a').find(
        link => link.attributes('href') === '/tenant/tenant-from-route/projects'
      )
      expect(projectLink).toBeTruthy()
    })

    it('非租户路由下，localStorage 有 lastActiveTenantId 时应回退到 localStorage 租户', async () => {
      const router = makeRouter()
      await router.push('/')

      // getStoredUserId returns '' → initData skips API → currentTenant stays ''
      hoistedMocks.getStoredUserIdMock.mockReturnValue('')
      localStorage.setItem(LAST_TENANT_STORAGE_KEY, 'tenant-from-storage')

      const { default: Sidebar } = await import('./Sidebar.vue')
      const wrapper = mount(Sidebar, {
        global: { plugins: [router], stubs: { 'router-link': RouterLinkStub } },
      })
      await flushPromises()
      await nextTick()

      const projectLink = wrapper.findAll('a').find(
        link => link.attributes('href') === '/tenant/tenant-from-storage/projects'
      )
      expect(projectLink).toBeTruthy()
    })

    it('所有回退均失败时返回空字符串（链接仅为 /projects）', async () => {
      const router = makeRouter()
      await router.push('/')

      hoistedMocks.getStoredUserIdMock.mockReturnValue('')
      // 不设置 localStorage

      const { default: Sidebar } = await import('./Sidebar.vue')
      const wrapper = mount(Sidebar, {
        global: { plugins: [router], stubs: { 'router-link': RouterLinkStub } },
      })
      await flushPromises()
      await nextTick()

      const projectLink = wrapper.findAll('a').find(
        link => link.attributes('href') === '/projects'
      )
      expect(projectLink).toBeTruthy()
    })
  })

  describe('Sidebar — 菜单状态管理', () => {
    beforeEach(() => {
      vi.clearAllMocks()
      localStorage.removeItem(LAST_TENANT_STORAGE_KEY)
      hoistedMocks.getStoredUserIdMock.mockReturnValue('')
    })

    it('初始状态下所有菜单默认折叠', async () => {
      const router = makeRouter()
      await router.push('/')

      const { default: Sidebar } = await import('./Sidebar.vue')
      const wrapper = mount(Sidebar, {
        global: { plugins: [router], stubs: { 'router-link': RouterLinkStub } },
      })
      await flushPromises()
      await nextTick()

      // 子菜单项（邀请人）默认不可见 — 被 isCompanyTenant 控制且菜单未展开
      const inviteLink = wrapper.findAll('a').find(link =>
        link.attributes('href')?.includes('/people/invite/')
      )
      expect(inviteLink).toBeFalsy()
    })

    it('当前路由在 /settings/ 下时设置菜单自动展开', async () => {
      const router = makeRouter()
      await router.push('/tenant/t1/settings/company/')

      const { default: Sidebar } = await import('./Sidebar.vue')
      const wrapper = mount(Sidebar, {
        global: { plugins: [router], stubs: { 'router-link': RouterLinkStub } },
      })
      await flushPromises()
      await nextTick()

      // 设置子菜单中的"公司设置"链接应可见
      const companySettingsLink = wrapper.findAll('a').find(link =>
        link.attributes('href') === '/tenant/t1/settings/company/'
      )
      expect(companySettingsLink).toBeTruthy()
    })

    it('当前路由在 /billing/ 下时计费菜单自动展开', async () => {
      const router = makeRouter()
      await router.push('/tenant/t1/billing/')

      const { default: Sidebar } = await import('./Sidebar.vue')
      const wrapper = mount(Sidebar, {
        global: { plugins: [router], stubs: { 'router-link': RouterLinkStub } },
      })
      await flushPromises()
      await nextTick()

      // 计费子菜单中的"概览"链接应可见
      const overviewLink = wrapper.findAll('a').find(link =>
        link.attributes('href') === '/tenant/t1/billing/'
      )
      expect(overviewLink).toBeTruthy()
    })
  })

  describe('Sidebar — 工作面板保留 + 人员管理菜单按公司租户状态条件渲染', () => {
    const mountWithTenantApi = async (router, memberFlags) => {
      hoistedMocks.getStoredUserIdMock.mockReturnValue('userId-1')
      hoistedMocks.apiFetchMock.mockImplementation(async (url) => ({
        ok: true,
        json: async () => url.includes('/accounts/users/me/')
          ? { companies: [{ id: 't1' }], current_company: { id: 't1' } }
          : memberFlags,
      }))
      const { default: Sidebar } = await import('./Sidebar.vue')
      const wrapper = mount(Sidebar, {
        global: { plugins: [router], stubs: { 'router-link': RouterLinkStub } },
      })
      await flushPromises()
      await nextTick()
      return wrapper
    }

    beforeEach(() => {
      vi.clearAllMocks()
      localStorage.removeItem(LAST_TENANT_STORAGE_KEY)
      hoistedMocks.getStoredUserIdMock.mockReturnValue('')
      hoistedMocks.permsCodes = [
        'project:view', 'project:manage', 'task:view', 'task:manage',
        'cloud:view', 'cloud:manage', 'member:manage', 'group:manage',
        'group-members:manage', 'company:view', 'company:manage',
        'workspace:manage', 'billing:view', 'billing:manage',
      ]
    })

    it('公司租户（member_is_active=true）→ 工作面板保留（位置 2）+ 人员管理菜单独立可见', async () => {
      const router = makeRouter()
      await router.push('/tenant/t1/projects')
      const wrapper = await mountWithTenantApi(router, {
        member_is_active: true,
        member_is_admin: false,
        member_is_creator: false,
      })

      // 工作面板链接保留（位置 2，所有租户可见）
      const workPanelLink = wrapper.findAll('a').find(
        link => link.attributes('href') === '/tenant/t1/work-panel'
      )
      expect(workPanelLink).toBeTruthy()
      // 人员管理入口独立可见（镜像市场之后）
      const peopleEntry = wrapper.findAll('div').find(
        div => div.classes().includes('cursor-pointer') && div.text().includes('人员管理')
      )
      expect(peopleEntry).toBeTruthy()
    })

    it('公司管理员（member_is_admin=true）→ 同样工作面板保留 + 人员管理菜单可见', async () => {
      const router = makeRouter()
      await router.push('/tenant/t1/projects')
      const wrapper = await mountWithTenantApi(router, {
        member_is_active: true,
        member_is_admin: true,
        member_is_creator: false,
      })

      const workPanelLink = wrapper.findAll('a').find(
        link => link.attributes('href') === '/tenant/t1/work-panel'
      )
      expect(workPanelLink).toBeTruthy()
      const peopleEntry = wrapper.findAll('div').find(
        div => div.classes().includes('cursor-pointer') && div.text().includes('人员管理')
      )
      expect(peopleEntry).toBeTruthy()
    })

    it('公司租户点击人员管理 → 子菜单展开（邀请人/管理人员/管理分组/访问管理）', async () => {
      const router = makeRouter()
      await router.push('/tenant/t1/projects')
      const wrapper = await mountWithTenantApi(router, {
        member_is_active: true,
        member_is_admin: false,
        member_is_creator: false,
      })

      const peopleEntry = wrapper.findAll('div').find(
        div => div.classes().includes('cursor-pointer') && div.text().includes('人员管理')
      )
      await peopleEntry.trigger('click')
      await nextTick()

      const inviteLink = wrapper.findAll('a').find(
        link => link.attributes('href')?.includes('/people/invite/')
      )
      const manageLink = wrapper.findAll('a').find(
        link => link.attributes('href')?.includes('/people/manage/')
      )
      const groupsLink = wrapper.findAll('a').find(
        link => link.attributes('href')?.includes('/people/groups/')
      )
      const accessLink = wrapper.findAll('a').find(
        link => link.attributes('href')?.includes('/people/access/')
      )
      expect(inviteLink).toBeTruthy()
      expect(manageLink).toBeTruthy()
      expect(groupsLink).toBeTruthy()
      expect(accessLink).toBeTruthy()
    })

    it('公司租户但无 people 权限 → 人员管理隐藏', async () => {
      hoistedMocks.permsCodes = ['project:view', 'task:view', 'cloud:view']
      const router = makeRouter()
      await router.push('/tenant/t1/projects')
      const wrapper = await mountWithTenantApi(router, {
        member_is_active: true,
        member_is_admin: false,
        member_is_creator: false,
      })
      const peopleEntry = wrapper.findAll('div').find(
        div => div.classes().includes('cursor-pointer') && div.text().includes('人员管理')
      )
      expect(peopleEntry).toBeFalsy()
    })

    it('非公司租户（member_is_active=false）→ 工作面板可见，人员管理隐藏', async () => {
      const router = makeRouter()
      await router.push('/tenant/t1/projects')
      const wrapper = await mountWithTenantApi(router, {
        member_is_active: false,
        member_is_admin: false,
        member_is_creator: false,
      })

      const workPanelLink = wrapper.findAll('a').find(
        link => link.attributes('href') === '/tenant/t1/work-panel'
      )
      expect(workPanelLink).toBeTruthy()
      const peopleEntry = wrapper.findAll('div').find(
        div => div.classes().includes('cursor-pointer') && div.text().includes('人员管理')
      )
      expect(peopleEntry).toBeFalsy()
    })

    it('API 失败（无 cookie 跳过请求）→ 工作面板可见，人员管理隐藏', async () => {
      const router = makeRouter()
      await router.push('/tenant/t1/projects')
      // getStoredUserId 返回空（localStorage 与 cookie 均无）→ initData 跳过 API → 非公司租户态
      const { default: Sidebar } = await import('./Sidebar.vue')
      const wrapper = mount(Sidebar, {
        global: { plugins: [router], stubs: { 'router-link': RouterLinkStub } },
      })
      await flushPromises()
      await nextTick()

      expect(hoistedMocks.apiFetchMock).not.toHaveBeenCalled()
      const workPanelLink = wrapper.findAll('a').find(
        link => link.attributes('href') === '/tenant/t1/work-panel'
      )
      expect(workPanelLink).toBeTruthy()
      const peopleEntry = wrapper.findAll('div').find(
        div => div.classes().includes('cursor-pointer') && div.text().includes('人员管理')
      )
      expect(peopleEntry).toBeFalsy()
    })
  })

  describe('Sidebar — 控制台导航标题点击缩窄/展开', () => {
    const COLLAPSED_KEY = 'tenant-console-sidebar-collapsed'

    beforeEach(() => {
      vi.clearAllMocks()
      localStorage.removeItem(LAST_TENANT_STORAGE_KEY)
      localStorage.removeItem(COLLAPSED_KEY)
      hoistedMocks.getStoredUserIdMock.mockReturnValue('')
    })

    const mountSidebar = async (router) => {
      const { default: Sidebar } = await import('./Sidebar.vue')
      const wrapper = mount(Sidebar, {
        global: { plugins: [router], stubs: { 'router-link': RouterLinkStub } },
      })
      await flushPromises()
      await nextTick()
      return wrapper
    }

    it('默认展开态：无 collapsed class，标题文字可见', async () => {
      const router = makeRouter()
      await router.push('/tenant/t1/projects')
      const wrapper = await mountSidebar(router)

      const aside = wrapper.find('aside.tenant-console-sidebar')
      expect(aside.classes()).not.toContain('tenant-console-sidebar-collapsed')
      expect(aside.find('h3').text()).toContain('控制台导航')
    })

    it('aside 具备 min-h-0 + overflow-y-auto，菜单超高时可滚', async () => {
      const router = makeRouter()
      await router.push('/tenant/t1/projects')
      const wrapper = await mountSidebar(router)

      const aside = wrapper.find('aside.tenant-console-sidebar')
      expect(aside.classes()).toContain('h-full')
      expect(aside.classes()).toContain('min-h-0')
      expect(aside.classes()).toContain('overflow-y-auto')
    })

    it('点击标题 → 缩窄：class 切换、localStorage 写入 1、标签隐藏、title 悬停提示', async () => {
      const router = makeRouter()
      await router.push('/tenant/t1/projects')
      const wrapper = await mountSidebar(router)

      const aside = wrapper.find('aside.tenant-console-sidebar')
      await aside.find('h3').trigger('click')
      await nextTick()

      expect(aside.classes()).toContain('tenant-console-sidebar-collapsed')
      expect(localStorage.getItem(COLLAPSED_KEY)).toBe('1')

      const projectLink = wrapper.findAll('a').find(link =>
        link.attributes('href') === '/tenant/t1/projects'
      )
      expect(projectLink.attributes('title')).toBe('项目列表')
      // v-show 隐藏文字标签
      expect(projectLink.find('span').element.style.display).toBe('none')
    })

    it('缩窄态下子菜单不渲染（设置→公司设置不可见）', async () => {
      const router = makeRouter()
      await router.push('/tenant/t1/projects')
      const wrapper = await mountSidebar(router)

      await wrapper.find('aside.tenant-console-sidebar h3').trigger('click')
      await nextTick()

      const companyLink = wrapper.findAll('a').find(link =>
        link.attributes('href') === '/tenant/t1/settings/company/'
      )
      expect(companyLink).toBeFalsy()
    })

    it('缩窄态点击带子菜单入口（设置）→ 一步展开侧栏 + 展开子菜单（OPT-20260807-055）', async () => {
      const router = makeRouter()
      await router.push('/tenant/t1/projects')
      const wrapper = await mountSidebar(router)

      const aside = wrapper.find('aside.tenant-console-sidebar')
      await aside.find('h3').trigger('click')
      await nextTick()
      expect(aside.classes()).toContain('tenant-console-sidebar-collapsed')

      // 设置入口：带 cursor-pointer 的展开行
      const settingsEntry = wrapper.findAll('div').find(
        div => div.classes().includes('cursor-pointer') && div.text().includes('设置')
      )
      expect(settingsEntry).toBeTruthy()
      await settingsEntry.trigger('click')
      await nextTick()

      expect(aside.classes()).not.toContain('tenant-console-sidebar-collapsed')
      expect(localStorage.getItem(COLLAPSED_KEY)).toBe('0')

      // 一步到位：子菜单随即展开，无需二次点击
      const companyLink = wrapper.findAll('a').find(link =>
        link.attributes('href') === '/tenant/t1/settings/company/'
      )
      expect(companyLink).toBeTruthy()
    })

    // 收起状态为模块级单例（OPT-20260809-024 联动改造）：localStorage 在模块导入时读取一次
    // （= 页面加载时恢复），「localStorage=1 挂载即恢复缩窄」的持久化路径由
    // Navbar.collapsed.test.js 首个用例覆盖（导入前写 localStorage）。
    it('缩窄后同会话重新挂载 → 仍保持缩窄态（共享单例状态）', async () => {
      const router = makeRouter()
      await router.push('/tenant/t1/projects')
      const wrapper = await mountSidebar(router)
      await wrapper.find('aside.tenant-console-sidebar h3').trigger('click')
      await nextTick()
      expect(localStorage.getItem(COLLAPSED_KEY)).toBe('1')

      // 同会话内重新挂载组件实例：单例状态保持缩窄
      const wrapper2 = await mountSidebar(router)
      const aside = wrapper2.find('aside.tenant-console-sidebar')
      expect(aside.classes()).toContain('tenant-console-sidebar-collapsed')

      // 恢复展开，避免单例状态泄漏影响后续用例（共享单例下用例间状态不隔离）
      await wrapper2.find('aside.tenant-console-sidebar h3').trigger('click')
      await nextTick()
    })

    it('缩窄态一级入口保持可导航且不触发展开', async () => {
      const router = makeRouter()
      await router.push('/tenant/t1/projects')
      const wrapper = await mountSidebar(router)

      const aside = wrapper.find('aside.tenant-console-sidebar')
      await aside.find('h3').trigger('click')
      await nextTick()

      const workPanelLink = wrapper.findAll('a').find(link =>
        link.attributes('href') === '/tenant/t1/work-panel'
      )
      expect(workPanelLink).toBeTruthy()
      expect(workPanelLink.attributes('title')).toBe('工作面板')

      await workPanelLink.trigger('click')
      await nextTick()
      expect(aside.classes()).toContain('tenant-console-sidebar-collapsed')
    })
  })

  describe('Sidebar — 清库后陈旧 lastActiveTenantId', () => {
    beforeEach(() => {
      vi.clearAllMocks()
      localStorage.clear()
      hoistedMocks.getStoredUserIdMock.mockReturnValue('userId-1')
    })

    it('/me/ companies=[] 时清除 localStorage lastActiveTenantId', async () => {
      localStorage.setItem(LAST_TENANT_STORAGE_KEY, '874599492341493760')
      hoistedMocks.apiFetchMock.mockImplementation(async (url) => {
        if (String(url).includes('/accounts/users/me/')) {
          return {
            ok: true,
            json: async () => ({ companies: [], current_company: null }),
          }
        }
        return { ok: false, status: 404, json: async () => ({}) }
      })

      const router = makeRouter()
      await router.push('/tenant/874599492341493760/settings/company/')
      const { default: Sidebar } = await import('./Sidebar.vue')
      mount(Sidebar, {
        global: { plugins: [router], stubs: { 'router-link': RouterLinkStub } },
      })
      await flushPromises()
      await nextTick()

      expect(localStorage.getItem(LAST_TENANT_STORAGE_KEY)).toBeNull()
    })

    it('companies/current 404 时清除匹配的 lastActiveTenantId', async () => {
      localStorage.setItem(LAST_TENANT_STORAGE_KEY, 't1')
      hoistedMocks.apiFetchMock.mockImplementation(async (url) => {
        if (String(url).includes('/accounts/users/me/')) {
          return {
            ok: true,
            json: async () => ({
              companies: [{ id: 't1' }],
              current_company: { id: 't1' },
            }),
          }
        }
        if (String(url).includes('/accounts/companies/current/')) {
          return { ok: false, status: 404, json: async () => ({ detail: 'not found' }) }
        }
        return { ok: false, status: 500, json: async () => ({}) }
      })

      const router = makeRouter()
      await router.push('/tenant/t1/projects')
      const { default: Sidebar } = await import('./Sidebar.vue')
      mount(Sidebar, {
        global: { plugins: [router], stubs: { 'router-link': RouterLinkStub } },
      })
      await flushPromises()
      await nextTick()

      expect(localStorage.getItem(LAST_TENANT_STORAGE_KEY)).toBeNull()
    })
  })

  describe('Sidebar — 意见与建议', () => {
    it('F4 无 feedback 权限不渲染意见与建议', async () => {
      const router = makeRouter()
      await router.push('/tenant/t1/projects')
      const { default: Sidebar } = await import('./Sidebar.vue')
      const wrapper = mount(Sidebar, {
        global: { plugins: [router], stubs: { 'router-link': RouterLinkStub } },
      })
      await flushPromises()
      expect(wrapper.find('[data-testid=tenant-console-feedback-nav]').exists()).toBe(false)
    })

    it('有 page:nav.feedback 时渲染意见与建议', async () => {
      hoistedMocks.permsCodes.push('page:nav.feedback')
      hoistedMocks.apiFetchMock.mockResolvedValue({ ok: true, json: async () => ({ groups: [] }) })
      const router = makeRouter()
      await router.push('/tenant/t1/projects')
      const { default: Sidebar } = await import('./Sidebar.vue')
      const wrapper = mount(Sidebar, {
        global: { plugins: [router], stubs: { 'router-link': RouterLinkStub } },
      })
      await flushPromises()
      expect(wrapper.find('[data-testid=tenant-console-feedback-nav]').exists()).toBe(true)
      hoistedMocks.permsCodes.pop()
    })
  })
}

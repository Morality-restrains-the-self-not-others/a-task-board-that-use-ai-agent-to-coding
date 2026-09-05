// @vitest-environment jsdom
if (!process.env.VITEST) {
  // pre-commit 会以 node 直跑已暂存 *.test.js，非 Vitest 环境下跳过。
  console.log('[skip] PeoplePageAccess.test.js requires vitest runtime')
} else {
  const { mount } = await import('@vue/test-utils')
  const { nextTick } = await import('vue')
  const { beforeEach, describe, expect, it, vi } = await import('vitest')

  const hoistedMocks = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
    // v63 RBAC 权限码池：{ tenantId: [codes] }，测试按场景改写
    permsState: { t1: [] },
    routeMock: { params: { tenant: 't1' } },
    routerMock: { push: vi.fn() },
  }))

  // apiUtils 仅覆盖 apiFetch，其余导出（parseCompanyMembersResponse 等）保留真实实现
  vi.mock('../utils/apiUtils', async (importOriginal) => {
    const actual = await importOriginal()
    return { ...actual, apiFetch: hoistedMocks.apiFetchMock }
  })

  vi.mock('vue-router', () => ({
    useRoute: () => hoistedMocks.routeMock,
    useRouter: () => hoistedMocks.routerMock,
  }))

  // usePermissions 为模块级单例，直接 mock 掉以便逐测试注入权限码，避免跨用例污染
  vi.mock('../composables/usePermissions.js', () => ({
    usePermissions: () => ({
      load: vi.fn(async () => {}),
      reload: vi.fn(async () => {}),
      hasPerm: (cid, code) =>
        Array.isArray(hoistedMocks.permsState[cid]) && hoistedMocks.permsState[cid].includes(code),
      hasRegion: (cid, key) =>
        Array.isArray(hoistedMocks.permsState[cid]) && hoistedMocks.permsState[cid].includes(`region:${key}`),
      hasPage: (cid, key) =>
        Array.isArray(hoistedMocks.permsState[cid]) && hoistedMocks.permsState[cid].includes(`page:${key}`),
      hasPlatformPerm: () => false,
      isPlatformRole: () => false,
    }),
  }))

  async function flushRender() {
    for (let i = 0; i < 12; i += 1) {
      await Promise.resolve()
      await nextTick()
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

  function failedResponse(body) {
    return {
      ok: false,
      status: 403,
      headers: {},
      json: async () => body,
    }
  }

  const calledUrls = () => hoistedMocks.apiFetchMock.mock.calls.map((c) => c[0])

  function defaultApiImpl() {
    hoistedMocks.apiFetchMock.mockImplementation(async (url) => {
      if (url === '/api/accounts/users/me/') {
        return okResponse({
          companies: [{ id: 't1', name: 'ACME', is_admin: true }],
          current_company: { id: 't1' },
        })
      }
      if (url === '/api/projects/workspaces/tenant_id/t1') {
        return okResponse({ data: [] })
      }
      if (url === '/api/tenant/t1/accounts/groups/') {
        return okResponse([])
      }
      if (url === '/api/tenant/t1/accounts/members/company_members/') {
        return okResponse({ members: [] })
      }
      return failedResponse({ message: 'unexpected url: ' + url })
    })
  }

  describe('PeopleManage 权限兜底（OPT-20260809-025）', () => {
    beforeEach(() => {
      hoistedMocks.apiFetchMock.mockReset()
      hoistedMocks.permsState = { t1: [] }
      document.cookie = 'userId=test-user'
      defaultApiImpl()
    })

    it('拥有 member:manage 时渲染「管理人员」并拉取公司列表', async () => {
      hoistedMocks.permsState = { t1: ['member:manage'] }
      const Comp = (await import('./PeopleManage.vue')).default
      const wrapper = mount(Comp, {
        global: {
          stubs: {
            MemberList: { template: '<div class="stub-member-list" />' },
            PendingInvitations: { template: '<div class="stub-pending-invitations" />' },
          },
        },
      })
      await flushRender()

      expect(wrapper.text()).toContain('管理人员')
      expect(wrapper.text()).not.toContain('无权限访问')
      // 有权限时拉取 /me/ 以构建公司切换下拉框
      expect(calledUrls()).toContain('/api/accounts/users/me/')
    })

    it('无 member:manage 时展示「无权限访问」且不拉取 /me/', async () => {
      const Comp = (await import('./PeopleManage.vue')).default
      const wrapper = mount(Comp, {
        global: {
          stubs: {
            MemberList: { template: '<div class="stub-member-list" />' },
            PendingInvitations: { template: '<div class="stub-pending-invitations" />' },
          },
        },
      })
      await flushRender()

      expect(wrapper.text()).toContain('无权限访问')
      expect(wrapper.text()).not.toContain('管理人员')
      expect(calledUrls()).not.toContain('/api/accounts/users/me/')
    })
  })

  describe('PeopleGroups 权限兜底（OPT-20260809-025）', () => {
    beforeEach(() => {
      hoistedMocks.apiFetchMock.mockReset()
      hoistedMocks.permsState = { t1: [] }
      document.cookie = 'userId=test-user'
      defaultApiImpl()
    })

    it('拥有 group-members:manage 时渲染「管理分组」并拉取分组/成员接口', async () => {
      hoistedMocks.permsState = { t1: ['group-members:manage'] }
      const Comp = (await import('./PeopleGroups.vue')).default
      const wrapper = mount(Comp)
      await flushRender()

      expect(wrapper.text()).toContain('管理分组')
      expect(wrapper.text()).not.toContain('无权限访问')
      expect(calledUrls()).toContain('/api/tenant/t1/accounts/groups/')
      expect(calledUrls()).toContain('/api/tenant/t1/accounts/members/company_members/')
    })

    it('无 group:manage / group-members:manage 时展示「无权限访问」且跳过分组/成员拉取', async () => {
      const Comp = (await import('./PeopleGroups.vue')).default
      const wrapper = mount(Comp)
      await flushRender()

      expect(wrapper.text()).toContain('无权限访问')
      // 空态消息文本本身含「管理分组」，改断言仅授权态存在的 UI（创建分组按钮）
      expect(wrapper.text()).not.toContain('创建分组')
      // /me/ 属于公司下拉框通用接口仍可调用；分组/成员接口须被权限门拦住
      expect(calledUrls()).not.toContain('/api/tenant/t1/accounts/groups/')
      expect(calledUrls()).not.toContain('/api/tenant/t1/accounts/members/company_members/')
    })
  })

  describe('PeopleAccess 权限兜底', () => {
    beforeEach(() => {
      hoistedMocks.apiFetchMock.mockReset()
      hoistedMocks.permsState = { t1: [] }
      document.cookie = 'userId=test-user'
      defaultApiImpl()
    })

    it('拥有 member:manage 时渲染「访问管理」标题', async () => {
      hoistedMocks.permsState = { t1: ['member:manage'] }
      hoistedMocks.apiFetchMock.mockImplementation(async (url) => {
        if (url === '/api/accounts/users/me/') {
          return okResponse({ companies: [{ id: 't1', name: 'ACME', is_admin: true }], current_company: { id: 't1' } })
        }
        if (String(url).includes('/resource-groups/')) {
          return okResponse({ pages: [] })
        }
        if (url.includes('/company_members/') || url.includes('/groups/') || url.includes('member-role') || url.includes('group-role') || url.includes('/roles/')) {
          return okResponse([])
        }
        return failedResponse({ message: 'unexpected: ' + url })
      })
      const Comp = (await import('./PeopleAccess.vue')).default
      const wrapper = mount(Comp)
      await flushRender()
      expect(wrapper.text()).toContain('访问管理')
      expect(wrapper.text()).not.toContain('无权限访问')
    })

    it('拥有 page:people.access 时亦可进入访问管理', async () => {
      hoistedMocks.permsState = { t1: ['page:people.access'] }
      hoistedMocks.apiFetchMock.mockImplementation(async (url) => {
        if (String(url).includes('/resource-groups/')) return okResponse({ pages: [] })
        if (url.includes('/company_members/') || url.includes('/groups/') || url.includes('member-role') || url.includes('group-role') || url.includes('/roles/')) {
          return okResponse([])
        }
        return okResponse({})
      })
      const Comp = (await import('./PeopleAccess.vue')).default
      const wrapper = mount(Comp)
      await flushRender()
      expect(wrapper.text()).toContain('访问管理')
    })

    it('无 member:manage 时展示无权限空态', async () => {
      const Comp = (await import('./PeopleAccess.vue')).default
      const wrapper = mount(Comp)
      await flushRender()
      expect(wrapper.text()).toContain('无权限访问')
    })

    it('资源组目录 403 时展示权限文案并挂 data-traceId（非误报 dataMigrate 032）', async () => {
      hoistedMocks.permsState = { t1: ['member:manage'] }
      hoistedMocks.apiFetchMock.mockImplementation(async (url) => {
        if (String(url).includes('/api/auth/resource-groups/')) {
          return {
            ok: false,
            status: 403,
            headers: { get: () => 'trace-catalog-403' },
            traceId: 'trace-catalog-403',
            json: async () => ({
              detail: '权限不足，需要访问管理区域或 member:manage',
              trace_id: 'trace-catalog-403',
            }),
          }
        }
        if (url.includes('/company_members/') || url.includes('/groups/') || url.includes('member-role') || url.includes('group-role') || url.includes('/roles/')) {
          return okResponse([])
        }
        return okResponse({})
      })
      const Comp = (await import('./PeopleAccess.vue')).default
      const wrapper = mount(Comp)
      await flushRender()
      const errEl = wrapper.find('[data-testid="people-access-catalog-error"]')
      expect(errEl.exists()).toBe(true)
      expect(errEl.text()).toContain('权限不足')
      expect(errEl.text()).not.toContain('dataMigrate 032')
      expect(errEl.attributes('data-traceid') || errEl.attributes('data-traceId')).toBe('trace-catalog-403')
    })

    it('选择成员后展示可分配角色多选（v75）', async () => {
      hoistedMocks.permsState = { t1: ['member:manage'] }
      hoistedMocks.apiFetchMock.mockImplementation(async (url) => {
        if (String(url).includes('/api/auth/resource-groups/?company_id=')) {
          return okResponse({ pages: [] })
        }
        if (url.includes('/company_members/')) {
          return okResponse({ members: [{ id: 'm1', member_name: 'Alice' }] })
        }
        if (url.includes('/roles/company_id/')) {
          return okResponse([
            { id: 'sys', name: 'member', display_name: '成员', is_system: true, permissions: [] },
            { id: 'c1', name: 'custom_fin', display_name: '财务只读', is_system: false, permissions: [] },
          ])
        }
        if (url.includes('/groups/') || url.includes('member-role') || url.includes('group-role')) {
          return okResponse([])
        }
        return okResponse({})
      })
      const Comp = (await import('./PeopleAccess.vue')).default
      const wrapper = mount(Comp, {
        global: { stubs: { RouterLink: { template: '<a><slot /></a>' } } },
      })
      await flushRender()

      const subjectBtn = wrapper.findAll('button').find((b) => b.text().includes('Alice'))
      expect(subjectBtn).toBeTruthy()
      await subjectBtn.trigger('click')
      await flushRender()

      const picker = wrapper.find('[data-testid="people-access-role-picker"]')
      expect(picker.exists()).toBe(true)
      expect(wrapper.text()).toContain('财务只读')
      expect(wrapper.text()).toContain('有效权限预览')
      expect(wrapper.find('[data-testid="people-access-catalog-filter"]').exists()).toBe(false)
    })

    it('保存时 PUT role_names 而非隐式创建访问角色', async () => {
      hoistedMocks.permsState = { t1: ['member:manage'] }
      const puts = []
      hoistedMocks.apiFetchMock.mockImplementation(async (url, init) => {
        if (String(url).includes('/api/auth/resource-groups/?company_id=')) {
          return okResponse({ pages: [] })
        }
        if (url.includes('/company_members/')) {
          return okResponse({ members: [{ id: 'm1', member_name: 'Alice' }] })
        }
        if (url.includes('/roles/company_id/')) {
          return okResponse([
            { id: 'c1', name: 'custom_fin', display_name: '财务只读', is_system: false, permissions: [] },
          ])
        }
        if (url.includes('/member-role/') && init?.method === 'PUT') {
          puts.push(JSON.parse(init.body))
          return okResponse({ role_names: ['custom_fin'] })
        }
        if (url.includes('/groups/') || url.includes('member-role') || url.includes('group-role') || url.includes('/roles/')) {
          return okResponse([])
        }
        return okResponse({})
      })
      const Comp = (await import('./PeopleAccess.vue')).default
      const wrapper = mount(Comp, {
        global: { stubs: { RouterLink: { template: '<a><slot /></a>' } } },
      })
      await flushRender()
      await wrapper.findAll('button').find((b) => b.text().includes('Alice')).trigger('click')
      await flushRender()
      const checkbox = wrapper.find('[data-testid="people-access-role-picker"] input[type="checkbox"]')
      await checkbox.setValue(true)
      await wrapper.find('[data-testid="people-access-save"]').trigger('click')
      await flushRender()
      expect(puts.length).toBe(1)
      expect(puts[0].role_names).toEqual(['custom_fin'])
      expect(puts[0].role_name).toBeUndefined()
    })
  })

  describe('PeopleInvite 权限兜底（OPT-20260809-025）', () => {
    beforeEach(() => {
      hoistedMocks.apiFetchMock.mockReset()
      hoistedMocks.permsState = { t1: [] }
      globalThis.apiFetch = hoistedMocks.apiFetchMock // PeopleInvite 走 window.apiFetch 全局
      defaultApiImpl()
    })

    it('拥有 member:manage 时渲染「邀请人」并拉取工作空间', async () => {
      hoistedMocks.permsState = { t1: ['member:manage'] }
      const Comp = (await import('./PeopleInvite.vue')).default
      const wrapper = mount(Comp)
      await flushRender()

      expect(wrapper.text()).toContain('邀请人')
      expect(wrapper.text()).not.toContain('无权限访问')
      expect(calledUrls()).toContain('/api/projects/workspaces/tenant_id/t1')
    })

    it('无 member:manage 时展示「无权限访问」且不拉取工作空间', async () => {
      const Comp = (await import('./PeopleInvite.vue')).default
      const wrapper = mount(Comp)
      await flushRender()

      expect(wrapper.text()).toContain('无权限访问')
      expect(wrapper.text()).not.toContain('邀请人')
      expect(calledUrls()).not.toContain('/api/projects/workspaces/tenant_id/t1')
    })
  })
}

// @vitest-environment jsdom
// 回归测试：Navbar.logic 拉取租户会员等级（/api/tenant/{tid}/billing/membership/）
// 并透传给 NavbarUI 的 membershipTier prop（驱动「代码仓库」VIP1 角标与跳转）。
// 覆盖：成功透传 / 失败静默（fail-open：不显示角标不拦截）/ 无租户不请求。
if (!process.env.VITEST) {
  console.log('[skip] Navbar.logic.membershipTier.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { describe, expect, it, vi, beforeEach } = await import('vitest')

  const mocks = vi.hoisted(() => ({
    apiFetch: vi.fn(),
    getCookie: vi.fn(),
    useRoute: () => ({ params: {}, query: {}, name: 'home', fullPath: '/', path: '/' }),
    useRouter: () => ({ push: vi.fn() }),
  }))

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch: (...args) => mocks.apiFetch(...args),
    clearCachedAuthToken: () => {},
  }))
  vi.mock('../utils/cookieUtils.js', () => ({
    getCookie: (...args) => mocks.getCookie(...args),
    clearCookie: () => {},
  }))
  vi.mock('vue-router', () => ({
    useRoute: () => mocks.useRoute(),
    useRouter: () => mocks.useRouter(),
  }))
  // 账号存储等与本次回归无关的依赖置空
  vi.mock('../domain/auth/services/saved_accounts_store.js', () => ({
    listSavedAccounts: () => Promise.resolve([]),
    getActiveToken: () => Promise.resolve('test-token'),
    upsertSavedAccount: () => Promise.resolve(),
    onAccountStateChanged: () => () => {},
  }))

  const { default: NavbarLogic } = await import('./Navbar.logic.vue')

  const mountNavbar = () =>
    mount(NavbarLogic, {
      props: { user: { isAuthenticated: false, isSuperuser: false, username: '' } },
      global: {
        stubs: {
          NavbarUI: {
            template: '<div data-testid="stub-navbar" :data-git-count="(gitResources || []).length" :data-git-status="gitResourcesStatus">{{ membershipTier }}</div>',
            props: ['membershipTier', 'gitResources', 'gitResourcesStatus', 'currentUser', 'userCompanies', 'currentTenant', 'route'],
          },
        },
      },
    })

  describe('Navbar.logic — membershipTier 拉取与透传', () => {
    beforeEach(() => {
      mocks.apiFetch.mockReset()
      mocks.getCookie.mockReset()
      mocks.getCookie.mockReturnValue('user-1')
      try { localStorage.removeItem('lastActiveTenantId') } catch (_) {}
    })

    const meOk = (companies) => ({
      ok: true,
      json: async () => ({
        id: 'user-1',
        is_superuser: false,
        is_active: true,
        email: 'user@test.com',
        companies,
        current_company: companies[0] || null,
        login_methods: [],
        platform_roles: [],
      }),
    })

    it('me/ 200 且有租户时，拉取 membership 并透传 vip1', async () => {
      mocks.apiFetch.mockImplementation(async (url) => {
        const u = String(url)
        if (u.includes('/api/accounts/users/me/')) {
          return meOk([{ id: 't1', name: 'TestCo' }])
        }
        if (u.includes('/billing/membership/')) {
          return { ok: true, json: async () => ({ membership: { tier: 'vip1' } }) }
        }
        if (u.includes('/billing/gitlab-resources/')) {
          return { ok: true, json: async () => ({ resources: [] }) }
        }
        return { ok: false, json: async () => ({}) }
      })

      const wrapper = mountNavbar()
      await flushPromises()
      await new Promise((r) => setTimeout(r, 0))
      await flushPromises()

      const stub = wrapper.find('[data-testid="stub-navbar"]')
      expect(stub.text()).toBe('vip1')
      expect(mocks.apiFetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/tenant/t1/billing/membership/'),
        expect.anything(),
      )
    })

    it('membership API 失败时透传空串（fail-open：不显示角标不拦截）', async () => {
      mocks.apiFetch.mockImplementation(async (url) => {
        const u = String(url)
        if (u.includes('/api/accounts/users/me/')) {
          return meOk([{ id: 't1', name: 'TestCo' }])
        }
        if (u.includes('/billing/membership/')) {
          return { ok: false, json: async () => ({}) }
        }
        if (u.includes('/billing/gitlab-resources/')) {
          return { ok: true, json: async () => ({ resources: [] }) }
        }
        return { ok: false, json: async () => ({}) }
      })

      const wrapper = mountNavbar()
      await flushPromises()
      await new Promise((r) => setTimeout(r, 0))
      await flushPromises()

      const stub = wrapper.find('[data-testid="stub-navbar"]')
      expect(stub.text()).toBe('')
    })

    it('无租户（companies 为空）时不请求 membership API', async () => {
      mocks.apiFetch.mockImplementation(async (url) => {
        const u = String(url)
        if (u.includes('/api/accounts/users/me/')) {
          return meOk([])
        }
        return { ok: false, json: async () => ({}) }
      })

      const wrapper = mountNavbar()
      await flushPromises()
      await new Promise((r) => setTimeout(r, 0))
      await flushPromises()

      const membershipCalls = mocks.apiFetch.mock.calls.filter(([url]) =>
        String(url).includes('/billing/membership/'),
      )
      expect(membershipCalls).toHaveLength(0)
      const gitCalls = mocks.apiFetch.mock.calls.filter(([url]) =>
        String(url).includes('/billing/gitlab-resources/'),
      )
      expect(gitCalls).toHaveLength(0)
      expect(wrapper.find('[data-testid="stub-navbar"]').text()).toBe('')
    })

    it('me/ 200 且有租户时，拉取 gitlab-resources 并透传列表长度', async () => {
      mocks.apiFetch.mockImplementation(async (url) => {
        const u = String(url)
        if (u.includes('/api/accounts/users/me/')) {
          return meOk([{ id: 't1', name: 'TestCo' }])
        }
        if (u.includes('/billing/membership/')) {
          return { ok: true, json: async () => ({ membership: { tier: 'normal' } }) }
        }
        if (u.includes('/billing/gitlab-resources/')) {
          return {
            ok: true,
            json: async () => ({
              resources: [{ region: 'tencent-sh-1', region_name: '腾讯上海一区', gitlab_web_url: 'https://g.example' }],
            }),
          }
        }
        return { ok: false, json: async () => ({}) }
      })

      const wrapper = mountNavbar()
      await flushPromises()
      await new Promise((r) => setTimeout(r, 0))
      await flushPromises()

      const stub = wrapper.find('[data-testid="stub-navbar"]')
      expect(stub.text()).toBe('normal')
      expect(stub.attributes('data-git-count')).toBe('1')
      expect(stub.attributes('data-git-status')).toBe('ready')
      expect(mocks.apiFetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/tenant/t1/billing/gitlab-resources/'),
        expect.anything(),
      )
    })

    it('gitlab-resources API 失败时透传 gitResourcesStatus=error（fail-open）', async () => {
      mocks.apiFetch.mockImplementation(async (url) => {
        const u = String(url)
        if (u.includes('/api/accounts/users/me/')) {
          return meOk([{ id: 't1', name: 'TestCo' }])
        }
        if (u.includes('/billing/membership/')) {
          return { ok: true, json: async () => ({ membership: { tier: 'normal' } }) }
        }
        if (u.includes('/billing/gitlab-resources/')) {
          return { ok: false, json: async () => ({}) }
        }
        return { ok: false, json: async () => ({}) }
      })

      const warn = vi.spyOn(console, 'warn').mockImplementation(() => {})
      const wrapper = mountNavbar()
      await flushPromises()
      await new Promise((r) => setTimeout(r, 0))
      await flushPromises()
      warn.mockRestore()

      const stub = wrapper.find('[data-testid="stub-navbar"]')
      expect(stub.attributes('data-git-count')).toBe('0')
      expect(stub.attributes('data-git-status')).toBe('error')
    })

    it('无租户时 gitResourcesStatus 为 ready（空列表视为已知）', async () => {
      mocks.apiFetch.mockImplementation(async (url) => {
        const u = String(url)
        if (u.includes('/api/accounts/users/me/')) {
          return meOk([])
        }
        return { ok: false, json: async () => ({}) }
      })

      const wrapper = mountNavbar()
      await flushPromises()
      await new Promise((r) => setTimeout(r, 0))
      await flushPromises()

      expect(wrapper.find('[data-testid="stub-navbar"]').attributes('data-git-status')).toBe('ready')
    })
  })
}

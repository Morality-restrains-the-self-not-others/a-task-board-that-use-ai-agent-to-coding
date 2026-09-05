// @vitest-environment jsdom
// 回归：已登录用户访问登录/注册页时，Navbar 仍须拉取会话并展示昵称等已登录控件，
// 不得因 AUTH_ROUTE_NAMES 提前 setLoggedOutUser 而强制显示「登录/注册」。
if (!process.env.VITEST) {
  console.log('[skip] Navbar.logic.authRouteSession.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { describe, expect, it, vi, beforeEach } = await import('vitest')

  const mocks = vi.hoisted(() => ({
    apiFetch: vi.fn(),
    getCookie: vi.fn(),
    routeState: {
      params: {},
      query: {},
      name: 'auth_login',
      fullPath: '/auth/login/',
      path: '/auth/login/',
    },
    useRoute: () => mocks.routeState,
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
            template:
              '<div data-testid="stub-navbar"' +
              ' :data-authenticated="isUserAuthenticated ? \'true\' : \'false\'"' +
              ' :data-username="(currentUser && currentUser.username) || \'\'">' +
              '{{ isUserAuthenticated ? ((currentUser && currentUser.username) || \'user\') : \'guest\' }}' +
              '</div>',
            props: [
              'currentUser',
              'isUserAuthenticated',
              'userCompanies',
              'currentTenant',
              'route',
            ],
          },
        },
      },
    })

  describe('Navbar.logic — 登录/注册页保留已登录导航', () => {
    beforeEach(() => {
      mocks.apiFetch.mockReset()
      mocks.getCookie.mockReset()
      mocks.routeState.name = 'auth_login'
      mocks.routeState.fullPath = '/auth/login/'
      mocks.routeState.path = '/auth/login/'
      mocks.routeState.params = {}
      try {
        localStorage.removeItem('lastActiveTenantId')
      } catch (_) {}
      if (typeof window !== 'undefined') {
        delete window.currentUser
      }
    })

    it('auth_login + 有效会话：拉取 /me/ 后展示昵称（非 guest）', async () => {
      mocks.getCookie.mockReturnValue('880000000000000001')
      mocks.apiFetch.mockImplementation(async (url) => {
        const u = String(url)
        if (u.includes('/api/accounts/users/me/')) {
          return {
            ok: true,
            json: async () => ({
              id: '880000000000000001',
              username: '已登录昵称',
              is_superuser: false,
              is_active: true,
              email: 'user@test.com',
              companies: [{ id: '874599492341493760', name: 'TestCo' }],
              current_company: { id: '874599492341493760', name: 'TestCo' },
              login_methods: [],
              platform_roles: [],
            }),
          }
        }
        if (u.includes('/api/auth/user-roles/')) {
          return { ok: true, json: async () => ({ roles: [] }) }
        }
        if (u.includes('/billing/membership/')) {
          return { ok: true, json: async () => ({ membership: { tier: 'normal' } }) }
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
      expect(stub.exists()).toBe(true)
      expect(stub.attributes('data-authenticated')).toBe('true')
      expect(stub.attributes('data-username')).toBe('已登录昵称')
      expect(stub.text()).toContain('已登录昵称')
      expect(mocks.apiFetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/accounts/users/me/'),
        expect.anything(),
      )
    })

    it('auth_register + 无会话：保持未登录（guest）', async () => {
      mocks.routeState.name = 'auth_register'
      mocks.routeState.fullPath = '/auth/register/'
      mocks.routeState.path = '/auth/register/'
      mocks.getCookie.mockReturnValue('')
      mocks.apiFetch.mockImplementation(async (url) => {
        const u = String(url)
        if (u.includes('/api/accounts/users/profile/')) {
          return { ok: false, status: 401, json: async () => ({}) }
        }
        return { ok: false, json: async () => ({}) }
      })

      const wrapper = mountNavbar()
      await flushPromises()
      await new Promise((r) => setTimeout(r, 0))
      await flushPromises()

      const stub = wrapper.find('[data-testid="stub-navbar"]')
      expect(stub.attributes('data-authenticated')).toBe('false')
      expect(stub.text()).toContain('guest')
    })
  })
}

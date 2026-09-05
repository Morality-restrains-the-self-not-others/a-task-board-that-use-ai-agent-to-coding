// @vitest-environment jsdom
// OPT-20260810-015：Navbar.fetchCurrentUser 统一走 getStoredUserId。
// 覆盖「localStorage 有 id、cookie 为空(HttpOnly) 时直接走 /me/」，
// 以及「两处都空时回退 /profile/」两个分支。
if (!process.env.VITEST) {
  console.log('[skip] Navbar.logic.fetchCurrentUser.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { describe, expect, it, vi, beforeEach } = await import('vitest')

  const mocks = vi.hoisted(() => ({
    apiFetch: vi.fn(),
    getCookie: vi.fn(),
    clearCachedAuthToken: vi.fn(),
    useRoute: () => ({ params: {}, query: {}, name: 'home', fullPath: '/', path: '/' }),
    useRouter: () => ({ push: vi.fn() }),
  }))

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch: (...args) => mocks.apiFetch(...args),
    clearCachedAuthToken: (...args) => mocks.clearCachedAuthToken(...args),
  }))
  vi.mock('../utils/cookieUtils.js', () => ({
    getCookie: (...args) => mocks.getCookie(...args),
    setCookie: () => {},
    clearCookie: () => {},
  }))
  vi.mock('vue-router', () => ({
    useRoute: () => mocks.useRoute(),
    useRouter: () => mocks.useRouter(),
  }))
  // sessionUserIdUtils 不 mock：getStoredUserId 走真实实现（localStorage → cookie 回退）
  vi.mock('../domain/auth/services/saved_accounts_store.js', () => ({
    listSavedAccounts: () => Promise.resolve([]),
    upsertSavedAccount: () => Promise.resolve(),
    removeSavedAccount: () => Promise.resolve(),
    getActiveToken: () => Promise.resolve('test-token'),
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
              '>{{ (currentUser && currentUser.userId) || \'\' }}</div>',
            props: ['currentUser', 'isUserAuthenticated', 'userCompanies', 'currentTenant', 'route'],
          },
        },
      },
    })

  const settle = async () => {
    await flushPromises()
    await new Promise((r) => setTimeout(r, 0))
    await flushPromises()
  }

  const isMeCall = (url) => String(url).includes('/api/accounts/users/me/')
  const isProfileCall = (url) => String(url).includes('/api/accounts/users/profile/')

  describe('Navbar.logic — fetchCurrentUser 走 getStoredUserId', () => {
    beforeEach(() => {
      mocks.apiFetch.mockReset()
      mocks.getCookie.mockReset()
      mocks.clearCachedAuthToken.mockReset()
      mocks.getCookie.mockReturnValue('') // 模拟 HttpOnly：JS 读不到 userId cookie
      try { localStorage.removeItem('currentUserId') } catch (_) {}
      try { localStorage.removeItem('lastActiveTenantId') } catch (_) {}
      if (typeof window !== 'undefined') {
        delete window.currentUser
      }
    })

    it('localStorage 有 id、cookie 为空时直接走 /me/（不再多一次 /profile/ 往返）', async () => {
      localStorage.setItem('currentUserId', 'local-user-1')
      mocks.apiFetch.mockImplementation(async (url) => {
        if (isMeCall(url)) {
          return {
            ok: true,
            json: async () => ({
              id: 'local-user-1',
              is_superuser: false,
              is_active: true,
              email: 'local@test.com',
              companies: [],
              current_company: null,
              login_methods: [],
            }),
          }
        }
        return { ok: false, json: async () => ({}) }
      })

      const wrapper = mountNavbar()
      await settle()

      expect(mocks.apiFetch).toHaveBeenCalledWith(expect.stringContaining('/api/accounts/users/me/'), expect.anything())
      expect(mocks.apiFetch.mock.calls.some(([url]) => isProfileCall(url))).toBe(false)
      // applyMePayload 写入 userData.userId
      expect(wrapper.text()).toContain('local-user-1')
      expect(wrapper.find('[data-testid="stub-navbar"]').attributes('data-authenticated')).toBe('true')
    })

    it('localStorage 与 cookie 均为空时回退 /profile/ 拉取用户信息', async () => {
      mocks.apiFetch.mockImplementation(async (url) => {
        if (isProfileCall(url)) {
          return {
            ok: true,
            json: async () => ({
              user_id: 'profile-user-2',
              personal_nickname: 'profile-user',
              email: 'p@test.com',
            }),
          }
        }
        return { ok: false, json: async () => ({}) }
      })

      const wrapper = mountNavbar()
      await settle()

      expect(mocks.apiFetch.mock.calls.some(([url]) => isProfileCall(url))).toBe(true)
      expect(mocks.apiFetch).toHaveBeenCalledWith(expect.stringContaining('/api/accounts/users/profile/'), expect.anything())
    })

    // 清库重建回归：localStorage 残留 currentUserId，/me/ 401 不得假装已登录
    // （否则登录页仍渲染 account-switcher「未设置昵称」）。
    it('localStorage 有旧 id 且 /me/ 401：guest + 清本地态 + 再请求 profile 清 HttpOnly cookie', async () => {
      localStorage.setItem('currentUserId', '874599469415428096')
      mocks.apiFetch.mockImplementation(async (url) => {
        if (isMeCall(url)) {
          return { ok: false, status: 401, json: async () => ({ detail: '请先登录' }) }
        }
        if (isProfileCall(url)) {
          return { ok: false, status: 401, json: async () => ({ detail: 'authentication required' }) }
        }
        return { ok: false, status: 500, json: async () => ({}) }
      })

      const wrapper = mountNavbar()
      await settle()

      const stub = wrapper.find('[data-testid="stub-navbar"]')
      expect(stub.attributes('data-authenticated')).toBe('false')
      expect(localStorage.getItem('currentUserId')).toBeNull()
      expect(mocks.clearCachedAuthToken).toHaveBeenCalled()
      expect(mocks.apiFetch.mock.calls.some(([url]) => isProfileCall(url))).toBe(true)
    })

    it('localStorage 有旧 id 且 /me/ 404：同样登出（用户已被清库删除）', async () => {
      localStorage.setItem('currentUserId', '874599469415428096')
      mocks.apiFetch.mockImplementation(async (url) => {
        if (isMeCall(url)) {
          return { ok: false, status: 404, json: async () => ({ detail: 'user not found' }) }
        }
        if (isProfileCall(url)) {
          return { ok: false, status: 401, json: async () => ({}) }
        }
        return { ok: false, status: 500, json: async () => ({}) }
      })

      const wrapper = mountNavbar()
      await settle()

      expect(wrapper.find('[data-testid="stub-navbar"]').attributes('data-authenticated')).toBe('false')
      expect(localStorage.getItem('currentUserId')).toBeNull()
    })

    it('/me/ 200 且 companies=[]：清除陈旧 lastActiveTenantId（清库后无公司）', async () => {
      localStorage.setItem('currentUserId', 'local-user-1')
      localStorage.setItem('lastActiveTenantId', '874599492341493760')
      mocks.apiFetch.mockImplementation(async (url) => {
        if (isMeCall(url)) {
          return {
            ok: true,
            json: async () => ({
              id: 'local-user-1',
              is_superuser: false,
              companies: [],
              current_company: null,
              login_methods: [],
            }),
          }
        }
        return { ok: false, json: async () => ({}) }
      })

      mountNavbar()
      await settle()

      expect(localStorage.getItem('lastActiveTenantId')).toBeNull()
    })
  })
}

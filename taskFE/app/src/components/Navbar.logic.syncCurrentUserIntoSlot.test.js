// @vitest-environment jsdom
// OPT-20260823-010：syncCurrentUserIntoSlot 从运行时 await import() 改为静态导入
// getActiveToken（该模块已被 Navbar 自身静态导入，动态导入无代码分割收益且会 stale-404）。
// 回归：登录态写入槽位仍走 getActiveToken + upsertSavedAccount。
if (!process.env.VITEST) {
  console.log('[skip] Navbar.logic.syncCurrentUserIntoSlot.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { describe, expect, it, vi, beforeEach } = await import('vitest')

  const mocks = vi.hoisted(() => ({
    apiFetch: vi.fn(),
    getActiveToken: vi.fn(),
    upsertSavedAccount: vi.fn(),
    removeSavedAccount: vi.fn(),
    useRoute: () => ({ params: {}, query: {}, name: 'home', fullPath: '/', path: '/' }),
    useRouter: () => ({ push: vi.fn() }),
  }))

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch: (...args) => mocks.apiFetch(...args),
    clearCachedAuthToken: () => {},
  }))
  vi.mock('../utils/cookieUtils.js', () => ({
    getCookie: () => '',
    setCookie: () => {},
    clearCookie: () => {},
  }))
  vi.mock('vue-router', () => ({
    useRoute: () => mocks.useRoute(),
    useRouter: () => mocks.useRouter(),
  }))
  vi.mock('../domain/auth/services/saved_accounts_store.js', () => ({
    getActiveToken: (...args) => mocks.getActiveToken(...args),
    upsertSavedAccount: (...args) => mocks.upsertSavedAccount(...args),
    removeSavedAccount: (...args) => mocks.removeSavedAccount(...args),
    onAccountStateChanged: () => () => {},
  }))

  const { default: NavbarLogic } = await import('./Navbar.logic.vue')

  const mountNavbar = () =>
    mount(NavbarLogic, {
      props: { user: { isAuthenticated: false, isSuperuser: false, username: '' } },
      global: {
        stubs: {
          NavbarUI: {
            template: '<div data-testid="stub-navbar" />',
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

  describe('Navbar.logic — syncCurrentUserIntoSlot 走静态导入 getActiveToken', () => {
    beforeEach(() => {
      mocks.apiFetch.mockReset()
      mocks.getActiveToken.mockReset()
      mocks.upsertSavedAccount.mockReset()
      mocks.getActiveToken.mockResolvedValue('slot-token')
      try { localStorage.removeItem('currentUserId') } catch (_) {}
      try { localStorage.removeItem('lastActiveTenantId') } catch (_) {}
    })

    it('登录态时经 getActiveToken 把当前账号写入槽位', async () => {
      localStorage.setItem('currentUserId', 'local-user-1')
      mocks.apiFetch.mockImplementation(async (url) => {
        if (String(url).includes('/api/accounts/users/me/')) {
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

      mountNavbar()
      await settle()

      expect(mocks.getActiveToken).toHaveBeenCalled()
      expect(mocks.upsertSavedAccount).toHaveBeenCalledWith(
        expect.objectContaining({ userId: 'local-user-1', token: 'slot-token' })
      )
    })
  })
}

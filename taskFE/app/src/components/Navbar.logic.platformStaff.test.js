// @vitest-environment jsdom
// 回归测试：平台管理员菜单显示「系统管理」而非「开始使用」。
// 覆盖完整链路：/me/ 200（无 platform_roles 字段）→ /api/auth/user-roles/ 兜底
// → isPlatformStaff=true → 渲染「系统管理」。
// 同时验证 refreshPlatformStaffFlag 定义在 <script setup> 内（SFC 编译后
// 可调用；此前声明在 </script> 之后被编译器丢弃，调用抛 ReferenceError）。
if (!process.env.VITEST) {
  console.log('[skip] Navbar.logic.platformStaff.test.js requires vitest runtime')
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
  const { default: NavbarUI } = await import('./Navbar.ui.vue')

  const mountNavbar = () =>
    mount(NavbarLogic, {
      props: { user: { isAuthenticated: false, isSuperuser: false, username: '' } },
      global: {
        stubs: {
          NavbarUI: {
            template: '<div data-testid="stub-navbar"><a v-for="l in links" :key="l" :href="l" data-testid="nav-link">{{ l }}</a></div>',
            props: ['currentUser', 'userCompanies', 'currentTenant', 'route'],
            computed: {
              links() {
                // 模拟 Navbar.ui 判定：平台角色显示系统管理
                const u = this.currentUser || {}
                const isStaff = u.isPlatformStaff || u.isSuperuser
                if (isStaff) return ['/system-admin/']
                return ['/onboarding/']
              },
            },
          },
        },
      },
    })

  describe('Navbar.logic — 平台管理员菜单', () => {
    beforeEach(() => {
      mocks.apiFetch.mockReset()
      mocks.getCookie.mockReset()
      mocks.getCookie.mockReturnValue('bootstrap-admin')
      try { localStorage.removeItem('lastActiveTenantId') } catch (_) {}
    })

    it('me/ 200 且无 platform_roles 时，经 user-roles 兜底 → isPlatformStaff=true 显示系统管理', async () => {
      mocks.apiFetch.mockImplementation(async (url) => {
        const u = String(url)
        if (u.includes('/api/accounts/users/me/')) {
          return {
            ok: true,
            json: async () => ({
              id: 'bootstrap-admin',
              is_superuser: true,
              is_active: true,
              email: 'author@example.com',
              companies: [],
              current_company: null,
              login_methods: [],
              // 注意：故意不带 platform_roles —— 老后端 /me/ 载荷无此字段
            }),
          }
        }
        if (u.includes('/api/auth/user-roles/')) {
          return {
            ok: true,
            json: async () => ({ roles: [{ role: 'super_admin', level: 'platform', company_id: '' }] }),
          }
        }
        return { ok: false, json: async () => ({}) }
      })

      const wrapper = mountNavbar()
      await flushPromises()
      await new Promise((r) => setTimeout(r, 0))
      await flushPromises()

      const stub = wrapper.find('[data-testid="stub-navbar"]')
      expect(stub.exists()).toBe(true)
      const text = stub.text()
      expect(text).toContain('/system-admin/')
      expect(text).not.toContain('/onboarding/')
      // 关键：refreshPlatformStaffFlag 必须被调用且未抛 ReferenceError
      expect(mocks.apiFetch).toHaveBeenCalledWith(expect.stringContaining('/api/auth/user-roles/'), expect.anything())
    })

    // 清库/会话失效：/me/ 401·404 不得再经 user-roles 兜底假装平台管理员
    // （旧行为会让登录页仍显示已登录导航；见 2026-08-11-db-reset-stale-login-ui）。
    it('me/ 失败（401/404）时登出，不经 user-roles 兜底显示系统管理', async () => {
      mocks.apiFetch.mockImplementation(async (url) => {
        const u = String(url)
        if (u.includes('/api/accounts/users/me/')) {
          return { ok: false, status: 404, json: async () => ({ detail: 'user not found' }) }
        }
        if (u.includes('/api/accounts/users/profile/')) {
          return { ok: false, status: 401, json: async () => ({}) }
        }
        if (u.includes('/api/auth/user-roles/')) {
          return {
            ok: true,
            json: async () => ({ roles: [{ role: 'super_admin', level: 'platform', company_id: '' }] }),
          }
        }
        return { ok: false, status: 500, json: async () => ({}) }
      })

      const wrapper = mountNavbar()
      await flushPromises()
      await new Promise((r) => setTimeout(r, 0))
      await flushPromises()

      const text = wrapper.find('[data-testid="stub-navbar"]').text()
      expect(text).not.toContain('/system-admin/')
      expect(mocks.apiFetch.mock.calls.some(([url]) => String(url).includes('/api/auth/user-roles/'))).toBe(false)
    })

    it('普通用户（无角色无公司）显示 onboarding 引导', async () => {
      mocks.getCookie.mockReturnValue('normal-user-1')
      mocks.apiFetch.mockImplementation(async (url) => {
        const u = String(url)
        if (u.includes('/api/accounts/users/me/')) {
          return {
            ok: true,
            json: async () => ({
              id: 'normal-user-1',
              is_superuser: false,
              is_active: true,
              email: 'user@test.com',
              companies: [],
              current_company: null,
              login_methods: [],
            }),
          }
        }
        if (u.includes('/api/auth/user-roles/')) {
          return { ok: true, json: async () => ({ roles: [] }) }
        }
        return { ok: false, json: async () => ({}) }
      })

      const wrapper = mountNavbar()
      await flushPromises()
      await new Promise((r) => setTimeout(r, 0))
      await flushPromises()

      const text = wrapper.find('[data-testid="stub-navbar"]').text()
      expect(text).toContain('/onboarding/')
      expect(text).not.toContain('/system-admin/')
    })
  })
}

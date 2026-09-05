// @vitest-environment jsdom
/**
 * PeopleJoin CTA / 错误展示回归
 *
 * 1) 未登录邀请页不得只剩文案——须有「登录/注册后确认加入」与「拒绝邀请」
 * 2) join 失败时错误文案须写入 UI（catch 参数不得遮蔽 error ref）
 */
if (!process.env.VITEST) {
  console.log('[skip] PeopleJoin.unauth-actions.test.js requires vitest runtime')
} else {
  const { beforeEach, describe, expect, it, vi } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { POST_LOGIN_REDIRECT_STORAGE_KEY } = await import('../utils/authReturnUrl.js')

  const { apiFetchMock, getCookieMock, getStoredUserIdMock, routeMock, routerMock } = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
    getCookieMock: vi.fn(() => ''),
    getStoredUserIdMock: vi.fn(() => ''),
    routeMock: {
      params: { tenant: '874599492341493760' },
      query: { token: 'pqq6L3ES2DrGy6-p4m0XMnfujv_fNCgdvJxZ9do1IUw' },
    },
    routerMock: { push: vi.fn() },
  }))

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch: (...args) => apiFetchMock(...args),
    clearCachedAuthToken: vi.fn(),
  }))

  vi.mock('../utils/cookieUtils.js', () => ({
    getCookie: (...args) => getCookieMock(...args),
  }))

  vi.mock('../utils/sessionUserIdUtils.js', async (importOriginal) => {
    const actual = await importOriginal()
    return {
      ...actual,
      getStoredUserId: (...args) => getStoredUserIdMock(...args),
      clearStoredUserId: vi.fn(),
    }
  })

  vi.mock('vue-router', () => ({
    useRoute: () => routeMock,
    useRouter: () => routerMock,
  }))

  function okJson(body) {
    return {
      ok: true,
      status: 200,
      json: async () => body,
    }
  }

  describe('PeopleJoin 未登录态 CTA', () => {
    beforeEach(() => {
      vi.clearAllMocks()
      localStorage.clear()
      getCookieMock.mockReturnValue('')
      getStoredUserIdMock.mockReturnValue('')
      routeMock.params = { tenant: '874599492341493760' }
      routeMock.query = { token: 'pqq6L3ES2DrGy6-p4m0XMnfujv_fNCgdvJxZ9do1IUw' }
      Object.defineProperty(window, 'location', {
        value: {
          href: 'https://www.daydaymoney.com/tenant/874599492341493760/people/join/?token=pqq6L3ES2DrGy6-p4m0XMnfujv_fNCgdvJxZ9do1IUw',
          pathname: '/tenant/874599492341493760/people/join/',
          search: '?token=pqq6L3ES2DrGy6-p4m0XMnfujv_fNCgdvJxZ9do1IUw',
          origin: 'https://www.daydaymoney.com',
        },
        writable: true,
        configurable: true,
      })
      window.apiFetch = (...args) => apiFetchMock(...args)
      apiFetchMock.mockImplementation(async (url) => {
        if (String(url).includes('/accounts/users/me/')) {
          return { ok: false, status: 401, json: async () => ({}) }
        }
        if (String(url).includes('/accounts/companies/current/')) {
          return okJson({ name: 'team1' })
        }
        if (String(url).includes('/members/validate-invite/')) {
          return okJson({ valid: true })
        }
        return { ok: false, status: 404, json: async () => ({}) }
      })
    })

    it('未登录且邀请有效时渲染登录确认与拒绝按钮', async () => {
      const { default: PeopleJoin } = await import('./PeopleJoin.vue')
      const wrapper = mount(PeopleJoin)
      await flushPromises()

      expect(wrapper.text()).toContain('请先注册或登录后确认加入')
      const loginBtn = wrapper.find('[data-testid="people-join-login-btn"]')
      const rejectBtn = wrapper.find('[data-testid="people-join-reject-btn"]')
      expect(loginBtn.exists(), '登录确认按钮不得丢失').toBe(true)
      expect(rejectBtn.exists(), '拒绝邀请按钮不得丢失').toBe(true)
      expect(loginBtn.text()).toMatch(/登录|注册/)
      expect(rejectBtn.text()).toContain('拒绝邀请')
    })

    it('点击登录确认写入回跳并跳转 /auth/login/?next=', async () => {
      const { default: PeopleJoin } = await import('./PeopleJoin.vue')
      const wrapper = mount(PeopleJoin)
      await flushPromises()

      await wrapper.find('[data-testid="people-join-login-btn"]').trigger('click')

      const expectedPath =
        '/tenant/874599492341493760/people/join/?token=pqq6L3ES2DrGy6-p4m0XMnfujv_fNCgdvJxZ9do1IUw'
      expect(localStorage.getItem(POST_LOGIN_REDIRECT_STORAGE_KEY)).toBe(expectedPath)
      expect(window.location.href).toBe(`/auth/login/?next=${encodeURIComponent(expectedPath)}`)
    })

    it('点击拒绝邀请展示成功提示', async () => {
      const { default: PeopleJoin } = await import('./PeopleJoin.vue')
      const wrapper = mount(PeopleJoin)
      await flushPromises()

      await wrapper.find('[data-testid="people-join-reject-btn"]').trigger('click')
      expect(wrapper.text()).toContain('邀请已拒绝')
    })
  })

  describe('PeopleJoin 已登录加入失败展示', () => {
    beforeEach(() => {
      vi.clearAllMocks()
      localStorage.clear()
      getCookieMock.mockReturnValue('u1')
      getStoredUserIdMock.mockReturnValue('u1')
      routeMock.params = { tenant: 't1' }
      routeMock.query = { token: 'tok' }
      Object.defineProperty(window, 'location', {
        value: {
          href: 'http://localhost/tenant/t1/people/join/?token=tok',
          pathname: '/tenant/t1/people/join/',
          search: '?token=tok',
          origin: 'http://localhost',
        },
        writable: true,
        configurable: true,
      })
      window.apiFetch = (...args) => apiFetchMock(...args)
      apiFetchMock.mockImplementation(async (url, opts = {}) => {
        if (String(url).includes('/accounts/users/me/')) {
          return okJson({ id: 'u1' })
        }
        if (String(url).includes('/accounts/companies/current/')) {
          return okJson({ name: 'team1' })
        }
        if (String(url).includes('/members/validate-invite/')) {
          return okJson({ valid: true })
        }
        if (String(url).includes('/members/join/') && opts.method === 'POST') {
          return {
            ok: false,
            status: 400,
            json: async () => ({ error: '邀请已失效' }),
            // 真实网关响应头 X-Trace-Id：浏览器合并多行后可能为 "id1, id2"，取首段
            headers: {
              get: (name) =>
                String(name).toLowerCase() === 'x-trace-id' ? 'web-abc123' : null,
            },
          }
        }
        return { ok: false, status: 404, json: async () => ({}) }
      })
    })

    it('join API 失败时错误文案写入 UI（不因 catch 遮蔽 ref 丢失）', async () => {
      const { default: PeopleJoin } = await import('./PeopleJoin.vue')
      const wrapper = mount(PeopleJoin)
      await flushPromises()

      expect(wrapper.find('[data-testid="people-join-confirm-btn"]').exists()).toBe(true)
      await wrapper.find('[data-testid="people-join-confirm-btn"]').trigger('click')
      await flushPromises()

      expect(wrapper.text()).toContain('邀请已失效')
    })

    it('join API 失败时错误容器携带 data-traceId（供 Loki 全链路检索）', async () => {
      const { default: PeopleJoin } = await import('./PeopleJoin.vue')
      const wrapper = mount(PeopleJoin)
      await flushPromises()

      await wrapper.find('[data-testid="people-join-confirm-btn"]').trigger('click')
      await flushPromises()

      const errDiv = wrapper.find('[data-traceId="web-abc123"]')
      expect(errDiv.exists(), '失败响应头 X-Trace-Id 应写入错误容器 data-traceId').toBe(true)
      expect(errDiv.text()).toContain('邀请已失效')
    })
  })

  /**
   * 登录后回跳邀请页：activate-session 落下的 userId 常为 HttpOnly，
   * document.cookie 读不到；不得仅凭 getCookie('userId') 空值判未登录。
   */
  describe('PeopleJoin 登录后回跳已认证态', () => {
    beforeEach(() => {
      vi.clearAllMocks()
      localStorage.clear()
      getCookieMock.mockReturnValue('')
      getStoredUserIdMock.mockReturnValue('')
      routeMock.params = { tenant: '874599492341493760' }
      routeMock.query = { token: 'pqq6L3ES2DrGy6-p4m0XMnfujv_fNCgdvJxZ9do1IUw' }
      Object.defineProperty(window, 'location', {
        value: {
          href: 'https://www.daydaymoney.com/tenant/874599492341493760/people/join/?token=pqq6L3ES2DrGy6-p4m0XMnfujv_fNCgdvJxZ9do1IUw',
          pathname: '/tenant/874599492341493760/people/join/',
          search: '?token=pqq6L3ES2DrGy6-p4m0XMnfujv_fNCgdvJxZ9do1IUw',
          origin: 'https://www.daydaymoney.com',
        },
        writable: true,
        configurable: true,
      })
      window.apiFetch = (...args) => apiFetchMock(...args)
      apiFetchMock.mockImplementation(async (url) => {
        if (String(url).includes('/accounts/users/me/')) {
          return { ok: false, status: 401, json: async () => ({}) }
        }
        if (String(url).includes('/accounts/users/profile/')) {
          return okJson({ user_id: '880000000000000001' })
        }
        if (String(url).includes('/accounts/companies/current/')) {
          return okJson({ name: 'team1' })
        }
        if (String(url).includes('/members/validate-invite/')) {
          return okJson({ valid: true })
        }
        return { ok: false, status: 404, json: async () => ({}) }
      })
    })

    it('无可读 userId cookie 但 session/profile 有效时展示确认加入', async () => {
      const { default: PeopleJoin } = await import('./PeopleJoin.vue')
      const wrapper = mount(PeopleJoin)
      await flushPromises()

      expect(wrapper.find('[data-testid="people-join-confirm-btn"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="people-join-login-btn"]').exists()).toBe(false)
      expect(wrapper.text()).toContain('点击下方按钮确认加入')
      expect(wrapper.text()).not.toContain('请先注册或登录后确认加入')
      expect(
        apiFetchMock.mock.calls.some((c) => String(c[0]).includes('/accounts/users/profile/')),
      ).toBe(true)
    })
  })
}

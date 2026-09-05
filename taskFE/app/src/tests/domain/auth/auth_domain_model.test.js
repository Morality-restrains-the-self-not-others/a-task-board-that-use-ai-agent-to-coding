// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] auth_domain_model.test.js requires vitest runtime')
} else {
  const { beforeEach, describe, expect, it, vi } = await import('vitest')
  const { PostLoginReturnUrl } = await import(
    '../../../domain/auth/value_objects/post_login_return_url_value_object.js'
  )
  const { AuthSessionGuardService } = await import(
    '../../../domain/auth/services/auth_session_guard_service.js'
  )
  const { savePostLoginRedirect, POST_LOGIN_REDIRECT_STORAGE_KEY } = await import(
    '../../../utils/authReturnUrl.js'
  )

  describe('PostLoginReturnUrl', () => {
    it('接受站内 task-detail 路径', () => {
      const url = new PostLoginReturnUrl(
        '/tenant/1/workspace/2/task-detail/3/?relayToTrae=true',
      )
      expect(url.value).toBe('/tenant/1/workspace/2/task-detail/3/?relayToTrae=true')
    })

    it('拒绝协议相对路径', () => {
      expect(() => new PostLoginReturnUrl('//evil.com/path')).toThrow(/无效的登录回跳地址/)
    })
  })

  describe('savePostLoginRedirect', () => {
    beforeEach(() => {
      localStorage.clear()
    })

    it('写入 postLoginRedirect', () => {
      const ok = savePostLoginRedirect('/tenant/1/workspace/2/task-detail/3/?relayToTrae=true')
      expect(ok).toBe(true)
      expect(localStorage.getItem(POST_LOGIN_REDIRECT_STORAGE_KEY)).toBe(
        '/tenant/1/workspace/2/task-detail/3/?relayToTrae=true',
      )
    })

    it('拒绝外部 URL', () => {
      const ok = savePostLoginRedirect('https://evil.com/')
      expect(ok).toBe(false)
      expect(localStorage.getItem(POST_LOGIN_REDIRECT_STORAGE_KEY)).toBeNull()
    })
  })

  describe('AuthSessionGuardService', () => {
    beforeEach(() => {
      document.cookie = 'userId=stale-user; path=/'
    })

    it('profile 200 时认为已登录', async () => {
      const apiFetch = vi.fn().mockResolvedValue({
        ok: true,
        json: async () => ({ user_id: '827923618451263488' }),
      })
      const clearUserIdCookie = vi.fn()
      const guard = new AuthSessionGuardService({
        apiFetch,
        getCookie: () => 'stale-user',
        clearUserIdCookie,
      })
      await expect(guard.isAuthenticated()).resolves.toBe(true)
      expect(clearUserIdCookie).not.toHaveBeenCalled()
    })

    it('profile 200 且 cookie 缺失时回填 userId', async () => {
      document.cookie = 'userId=; expires=Thu, 01 Jan 1970 00:00:00 GMT; path=/'
      const apiFetch = vi.fn().mockResolvedValue({
        ok: true,
        json: async () => ({ user_id: '827923618451263488' }),
      })
      const guard = new AuthSessionGuardService({
        apiFetch,
        getCookie: () => '',
        clearUserIdCookie: vi.fn(),
      })
      await expect(guard.isAuthenticated()).resolves.toBe(true)
      expect(document.cookie).toContain('userId=827923618451263488')
    })

    it('profile 403 时清除 stale userId', async () => {
      const apiFetch = vi.fn(async (url) => {
        if (String(url).includes('/accounts/users/me/')) {
          return { ok: false, status: 401 }
        }
        return { ok: false, status: 403 }
      })
      const clearUserIdCookie = vi.fn()
      const guard = new AuthSessionGuardService({
        apiFetch,
        getCookie: () => 'stale-user',
        clearUserIdCookie,
      })
      await expect(guard.isAuthenticated()).resolves.toBe(false)
      expect(clearUserIdCookie).toHaveBeenCalledTimes(1)
    })

    it('userId cookie 且 /me/ 200 时跳过 profile', async () => {
      const apiFetch = vi.fn(async (url) => {
        if (String(url).includes('/accounts/users/me/')) {
          return { ok: true, json: async () => ({ id: '827923618451263488' }) }
        }
        return { ok: false, status: 403 }
      })
      const guard = new AuthSessionGuardService({
        apiFetch,
        getCookie: () => '827923618451263488',
        clearUserIdCookie: vi.fn(),
      })
      await expect(guard.isAuthenticated()).resolves.toBe(true)
      expect(apiFetch).toHaveBeenCalledTimes(1)
      expect(String(apiFetch.mock.calls[0][0])).toContain('/accounts/users/me/')
    })

    it('profile 403 时清除 stale authToken', async () => {
      localStorage.setItem('authToken', 'stale-token')
      const apiFetch = vi.fn().mockResolvedValue({ ok: false, status: 403 })
      const clearAuthToken = vi.fn(() => localStorage.removeItem('authToken'))
      const guard = new AuthSessionGuardService({
        apiFetch,
        getCookie: () => '',
        clearUserIdCookie: vi.fn(),
        clearAuthToken,
      })
      await expect(guard.isAuthenticated()).resolves.toBe(false)
      expect(clearAuthToken).toHaveBeenCalledTimes(1)
      expect(localStorage.getItem('authToken')).toBeNull()
    })
  })
}

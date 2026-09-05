// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] sessionUserIdUtils.test.js requires vitest runtime')
} else {
  const { beforeEach, describe, expect, it, vi } = await import('vitest')
  const {
    clearStoredUserId, getStoredUserId, resolveAuthenticatedUserId, syncUserIdCookieFromProfile,
    validateStoredUserIdAgainstServer, resetUserIdValidationCache,
  } = await import('./sessionUserIdUtils.js')
  const { getCookie } = await import('./cookieUtils.js')

  const hoistedMocks = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
  }))

  vi.mock('./apiUtils.js', () => ({
    apiFetch: hoistedMocks.apiFetchMock,
  }))

  function profileOk(body) {
    return Promise.resolve({ ok: true, status: 200, json: async () => body })
  }

  const VALID_USER_ID = '827923618451263488'
  const OTHER_USER_ID = '314159265358979323'

  describe('resolveAuthenticatedUserId', () => {
    beforeEach(() => {
      vi.clearAllMocks()
      resetUserIdValidationCache()
      sessionStorage.removeItem('impersonatorAccountBackup')
      localStorage.clear()
      document.cookie = 'userId=; expires=Thu, 01 Jan 1970 00:00:00 GMT; path=/'
    })

    it('returns userId from cookie when present and validates against server in background', async () => {
      document.cookie = `userId=${VALID_USER_ID}; path=/`
      hoistedMocks.apiFetchMock.mockImplementation(() => profileOk({ user_id: VALID_USER_ID }))
      await expect(resolveAuthenticatedUserId()).resolves.toBe(VALID_USER_ID)
      // OPT-20260824-064: 命中本地缓存时后台校验（TTL 限频），会发 profile 请求；
      // 服务端一致时本地值不变。
      await new Promise((resolve) => setTimeout(resolve, 0))
      expect(hoistedMocks.apiFetchMock).toHaveBeenCalledWith(
        '/api/accounts/users/profile/',
        expect.objectContaining({ credentials: 'include' }),
      )
    })

    it('falls back to profile user_id when cookie is missing', async () => {
      hoistedMocks.apiFetchMock.mockResolvedValue({
        ok: true,
        json: async () => ({ user_id: VALID_USER_ID }),
      })
      await expect(resolveAuthenticatedUserId()).resolves.toBe(VALID_USER_ID)
      expect(hoistedMocks.apiFetchMock).toHaveBeenCalledWith(
        '/api/accounts/users/profile/',
        expect.objectContaining({ credentials: 'include' }),
      )
      expect(getCookie('userId')).toBe(VALID_USER_ID)
    })

    it('refills stale stored userId when server profile differs (OPT-20260824-064)', async () => {
      localStorage.setItem('currentUserId', OTHER_USER_ID)
      hoistedMocks.apiFetchMock.mockImplementation(() => profileOk({ user_id: VALID_USER_ID }))
      const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {})
      await expect(resolveAuthenticatedUserId()).resolves.toBe(OTHER_USER_ID)
      await new Promise((resolve) => setTimeout(resolve, 0))
      expect(localStorage.getItem('currentUserId')).toBe(VALID_USER_ID)
      expect(warnSpy).toHaveBeenCalled()
      warnSpy.mockRestore()
    })

    it('skips server validation during an impersonation session', async () => {
      sessionStorage.setItem('impersonatorAccountBackup', JSON.stringify({ userId: '1', token: 't' }))
      localStorage.setItem('currentUserId', OTHER_USER_ID)
      hoistedMocks.apiFetchMock.mockImplementation(() => profileOk({ user_id: VALID_USER_ID }))
      await expect(resolveAuthenticatedUserId()).resolves.toBe(OTHER_USER_ID)
      await new Promise((resolve) => setTimeout(resolve, 0))
      // 模拟会话由 status 端点自愈接管，此处不应发 profile 校验。
      expect(hoistedMocks.apiFetchMock).not.toHaveBeenCalled()
      expect(localStorage.getItem('currentUserId')).toBe(OTHER_USER_ID)
    })

    it('TTL 限频：TTL 内二次 resolve 不重复发校验请求', async () => {
      document.cookie = `userId=${VALID_USER_ID}; path=/`
      hoistedMocks.apiFetchMock.mockImplementation(() => profileOk({ user_id: VALID_USER_ID }))
      await resolveAuthenticatedUserId()
      await new Promise((resolve) => setTimeout(resolve, 0))
      const callsAfterFirst = hoistedMocks.apiFetchMock.mock.calls.length
      await resolveAuthenticatedUserId()
      await new Promise((resolve) => setTimeout(resolve, 0))
      expect(hoistedMocks.apiFetchMock.mock.calls.length).toBe(callsAfterFirst)
    })
  })

  describe('validateStoredUserIdAgainstServer', () => {
    beforeEach(() => {
      vi.clearAllMocks()
      resetUserIdValidationCache()
      localStorage.clear()
      document.cookie = 'userId=; expires=Thu, 01 Jan 1970 00:00:00 GMT; path=/'
    })

    it('refills and warns when server user_id differs', async () => {
      localStorage.setItem('currentUserId', OTHER_USER_ID)
      hoistedMocks.apiFetchMock.mockImplementation(() => profileOk({ user_id: VALID_USER_ID }))
      const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {})
      await expect(validateStoredUserIdAgainstServer(OTHER_USER_ID)).resolves.toBe(VALID_USER_ID)
      expect(localStorage.getItem('currentUserId')).toBe(VALID_USER_ID)
      expect(warnSpy).toHaveBeenCalled()
      warnSpy.mockRestore()
    })

    it('does not refill when server user_id matches', async () => {
      localStorage.setItem('currentUserId', VALID_USER_ID)
      hoistedMocks.apiFetchMock.mockImplementation(() => profileOk({ user_id: VALID_USER_ID }))
      const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {})
      await expect(validateStoredUserIdAgainstServer(VALID_USER_ID)).resolves.toBe(VALID_USER_ID)
      expect(localStorage.getItem('currentUserId')).toBe(VALID_USER_ID)
      expect(warnSpy).not.toHaveBeenCalled()
      warnSpy.mockRestore()
    })

    it('returns empty and keeps stored value when profile fails', async () => {
      localStorage.setItem('currentUserId', VALID_USER_ID)
      hoistedMocks.apiFetchMock.mockResolvedValue({ ok: false, status: 401, json: async () => ({}) })
      await expect(validateStoredUserIdAgainstServer(VALID_USER_ID)).resolves.toBe('')
      expect(localStorage.getItem('currentUserId')).toBe(VALID_USER_ID)
    })
  })

  describe('syncUserIdCookieFromProfile', () => {
    beforeEach(() => {
      document.cookie = 'userId=; expires=Thu, 01 Jan 1970 00:00:00 GMT; path=/'
    })

    it('sets cookie when profile has user_id and cookie is missing', () => {
      expect(syncUserIdCookieFromProfile({ user_id: '827923618451263488' })).toBe(true)
      expect(getCookie('userId')).toBe('827923618451263488')
    })

    it('does not overwrite matching cookie', () => {
      document.cookie = 'userId=827923618451263488; path=/'
      expect(syncUserIdCookieFromProfile({ user_id: '827923618451263488' })).toBe(true)
      expect(getCookie('userId')).toBe('827923618451263488')
    })
  })

  describe('getStoredUserId（OPT-20260810-032 WorkPanel 初始化同源语义）', () => {
    beforeEach(() => {
      localStorage.clear()
      document.cookie = 'userId=; expires=Thu, 01 Jan 1970 00:00:00 GMT; path=/'
    })

    it('localStorage currentUserId 优先（仅有 localStorage、cookie 为空也能识别登录态）', () => {
      localStorage.setItem('currentUserId', '827923618451263488')
      expect(getStoredUserId()).toBe('827923618451263488')
    })

    it('cookie 回退：localStorage 为空时读 userId cookie', () => {
      document.cookie = 'userId=314159265358979323; path=/'
      expect(getStoredUserId()).toBe('314159265358979323')
    })

    it('两者皆空返回空串，不抛错', () => {
      expect(getStoredUserId()).toBe('')
    })
  })

  describe('clearStoredUserId', () => {
    it('clears localStorage and the userId cookie（双变体：域 + host-only）', () => {
      localStorage.setItem('currentUserId', '827923618451263488')
      document.cookie = 'userId=827923618451263488; path=/'
      clearStoredUserId()
      expect(localStorage.getItem('currentUserId')).toBeNull()
      expect(getCookie('userId')).toBe('')
    })

    it('is safe when nothing is stored', () => {
      expect(() => clearStoredUserId()).not.toThrow()
    })
  })
}

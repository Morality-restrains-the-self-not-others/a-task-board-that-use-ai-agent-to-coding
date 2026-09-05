// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] activate_session_service.test.js requires vitest runtime')
} else {
  const { beforeEach, describe, expect, it, vi } = await import('vitest')
  const { persistLoginSuccessCredentials } = await import('./activate_session_service.js')

  const hoistedMocks = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
    setActiveAccountMock: vi.fn(),
    getActiveTokenMock: vi.fn(),
    removeSavedAccountMock: vi.fn(),
    storeUserIdMock: vi.fn(),
    setCachedAuthTokenMock: vi.fn(),
  }))

  vi.mock('../../../utils/apiUtils.js', () => ({
    apiFetch: hoistedMocks.apiFetchMock,
    setCachedAuthToken: hoistedMocks.setCachedAuthTokenMock,
    clearCachedAuthToken: vi.fn(),
  }))

  vi.mock('./saved_accounts_store.js', () => ({
    setActiveAccount: hoistedMocks.setActiveAccountMock,
    getActiveToken: hoistedMocks.getActiveTokenMock,
    removeSavedAccount: hoistedMocks.removeSavedAccountMock,
    getSavedAccount: vi.fn(),
  }))

  vi.mock('../../../utils/sessionUserIdUtils.js', () => ({
    storeUserId: hoistedMocks.storeUserIdMock,
    clearStoredUserId: vi.fn(),
  }))

  vi.mock('../../../utils/cookieUtils.js', () => ({
    clearCookie: vi.fn(),
  }))

  const LOGIN_DATA = {
    user: { id: '827923618451263488', username: 'author@example.com', email: 'author@example.com' },
    token: 'tok-abc-123',
  }

  beforeEach(() => {
    vi.clearAllMocks()
    hoistedMocks.getActiveTokenMock.mockResolvedValue('')
    hoistedMocks.apiFetchMock.mockResolvedValue({
      ok: true,
      json: async () => ({ user: { id: '827923618451263488', username: 'author@example.com' }, token: 'tok-abc-123' }),
    })
  })

  describe('persistLoginSuccessCredentials', () => {
    it('activates the session via activate-session so HttpOnly cookies are set after login', async () => {
      await persistLoginSuccessCredentials(LOGIN_DATA)

      // activate-session 请求：POST（真实 apiFetch 会基于 setCachedAuthToken
      // 的缓存自动附加 Authorization 头；服务端 Set-Cookie userId+token HttpOnly）
      expect(hoistedMocks.apiFetchMock).toHaveBeenCalledWith(
        '/api/accounts/users/activate-session/',
        expect.objectContaining({
          method: 'POST',
          body: expect.stringContaining('"user_id":"827923618451263488"'),
        }),
      )
      expect(hoistedMocks.setCachedAuthTokenMock).toHaveBeenCalledWith('tok-abc-123')
      // 账号槽 upsert 由 activate-session 成功路径完成
      expect(hoistedMocks.setActiveAccountMock).toHaveBeenCalledWith(
        expect.objectContaining({ userId: '827923618451263488', token: 'tok-abc-123' }),
      )
    })

    it('syncs the server-confirmed userId to local storage after activate-session succeeds', async () => {
      await persistLoginSuccessCredentials(LOGIN_DATA)

      // 服务端 activate-session 已确认身份 → 本地主存储 currentUserId 必须同步为
      // 服务端返回的权威 userId（activate 成功分支此前缺失该写入，模拟登录/账号
      // 切换后 localStorage 残留旧账号 ID，以旧 ID 构造的 API 路径被后端 403，
      // 如 git-identities「获取身份列表失败」）。
      expect(hoistedMocks.storeUserIdMock).toHaveBeenCalledWith('827923618451263488')
    })

    it('stores the impersonated target user id when persisting an impersonation login', async () => {
      // 模拟登录（/api/system-admin/users/{uid}/impersonate/ → persist(data)）：
      // data.user.id 为被模拟用户，activate 成功后 currentUserId 必须切换到
      // 目标用户，否则 git-identities 等页面仍以模拟者 ID 请求 → 403。
      const impersonationData = {
        user: { id: '877397583960502272', username: 'target-user', email: '' },
        token: 'imp-token-456',
      }
      hoistedMocks.apiFetchMock.mockResolvedValue({
        ok: true,
        json: async () => ({
          user: { id: '877397583960502272', username: 'target-user' },
          token: 'imp-token-456',
        }),
      })
      await persistLoginSuccessCredentials(impersonationData)

      expect(hoistedMocks.storeUserIdMock).toHaveBeenCalledWith('877397583960502272')
    })

    it('falls back to account slot write when activate-session fails, without throwing', async () => {
      hoistedMocks.apiFetchMock.mockResolvedValue({ ok: false, status: 500, json: async () => ({}) })
      await expect(persistLoginSuccessCredentials(LOGIN_DATA)).resolves.toBeUndefined()
      expect(hoistedMocks.setActiveAccountMock).toHaveBeenCalled()
    })

    it('stores userId cookie fallback when everything fails', async () => {
      hoistedMocks.apiFetchMock.mockResolvedValue({ ok: false, status: 500, json: async () => ({}) })
      hoistedMocks.setActiveAccountMock.mockRejectedValue(new Error('no plugin'))
      await expect(persistLoginSuccessCredentials(LOGIN_DATA)).resolves.toBeUndefined()
      expect(hoistedMocks.storeUserIdMock).toHaveBeenCalledWith('827923618451263488')
    })

    it('still activates the session when account store is unavailable (getActiveToken throws)', async () => {
      // getActiveToken 读取异常（如 localStorage 不可用）；activate-session 的
      // HttpOnly 会话 cookie 兜底路径不应被账号槽可用性门禁阻断（OPT-20260807-067 场景）。
      hoistedMocks.getActiveTokenMock.mockRejectedValue(new Error('store unavailable'))
      hoistedMocks.setActiveAccountMock.mockRejectedValue(new Error('store unavailable'))
      await persistLoginSuccessCredentials(LOGIN_DATA)

      expect(hoistedMocks.apiFetchMock).toHaveBeenCalledWith(
        '/api/accounts/users/activate-session/',
        expect.objectContaining({ method: 'POST' }),
      )
      // 账号槽 upsert 失败不阻断主流程
      expect(hoistedMocks.setCachedAuthTokenMock).toHaveBeenCalledWith('tok-abc-123')
    })
  })
}

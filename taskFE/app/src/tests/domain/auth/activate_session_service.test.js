// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] activate_session_service.test.js requires vitest runtime')
} else {
  const { beforeEach, describe, expect, it, vi } = await import('vitest')
  const {
    activateSavedAccountSession,
    persistLoginAccountSlot,
    persistLoginSuccessCredentials,
    resolveSwitchHref,
  } = await import('../../../domain/auth/services/activate_session_service.js')
  // 直接使用真实 saved_accounts_store（localStorage），listSavedAccounts 断言保持真实；
  // setActiveAccount 以 spy 包装以断言调用参数。
  const store = await import('../../../domain/auth/services/saved_accounts_store.js')
  const { listSavedAccounts, clearSavedAccounts } = store
  const setActiveAccountSpy = vi.spyOn(store, 'setActiveAccount')

  describe('activate_session_service', () => {
    beforeEach(() => {
      localStorage.clear()
      clearSavedAccounts()
      document.cookie = 'userId=; expires=Thu, 01 Jan 1970 00:00:00 GMT; path=/'
    })

    it('activate 成功写入 cookie/token/槽', async () => {
      const apiFetch = vi.fn().mockResolvedValue({
        ok: true,
        json: async () => ({
          user: { id: '864758149662404608', username: 'bob' },
          token: 'tok-bob',
        }),
      })
      const result = await activateSavedAccountSession({
        userId: '864758149662404608',
        token: 'tok-bob',
        apiFetch,
        username: 'bob',
      })
      expect(result.userId).toBe('864758149662404608')
      // token/槽位写入本机账号槽（localStorage）
      expect(setActiveAccountSpy).toHaveBeenCalledWith(
        expect.objectContaining({ userId: '864758149662404608', token: 'tok-bob', username: 'bob' }),
      )
      expect(listSavedAccounts()).toHaveLength(1)
      expect(apiFetch).toHaveBeenCalledWith(
        '/api/accounts/users/activate-session/',
        expect.objectContaining({
          method: 'POST',
        }),
      )
      expect(listSavedAccounts()[0].token).toBe('tok-bob')
    })

    it('401 时从槽移除并抛错', async () => {
      const { upsertSavedAccount } = await import(
        '../../../domain/auth/services/saved_accounts_store.js'
      )
      upsertSavedAccount({ userId: '1', username: 'a', token: 'bad' })
      const apiFetch = vi.fn().mockResolvedValue({
        ok: false,
        status: 401,
        json: async () => ({ error: '令牌无效或已过期' }),
      })
      await expect(
        activateSavedAccountSession({ userId: '1', token: 'bad', apiFetch }),
      ).rejects.toThrow(/令牌无效/)
      expect(listSavedAccounts()).toHaveLength(0)
    })

    it('persistLoginAccountSlot 写入槽', async () => {
      await persistLoginAccountSlot({
        userId: '42',
        token: 't42',
        username: 'carol',
      })
      expect(listSavedAccounts()[0].username).toBe('carol')
      expect(setActiveAccountSpy).toHaveBeenCalledWith(
        expect.objectContaining({ token: 't42', username: 'carol' }),
      )
    })

    it('persistLoginSuccessCredentials 写入槽与凭据', async () => {
      await persistLoginSuccessCredentials({
        user: { id: '99', username: 'dana', email: 'd@example.com' },
        token: 'tok-dana',
      })
      expect(listSavedAccounts()[0].userId).toBe('99')
      expect(listSavedAccounts()[0].username).toBe('dana')
      expect(setActiveAccountSpy).toHaveBeenCalledWith(
        expect.objectContaining({ token: 'tok-dana', userId: '99' }),
      )
    })

    it('persistLoginSuccessCredentials 成功后同步 localStorage currentUserId（模拟会话安全）', async () => {
      // 模拟登录（impersonate → persist(data)）后 localStorage currentUserId 必须
      // 切换为目标用户；残留模拟者 ID 会导致以 userId 构造的 API 路径被后端 403
      // （git-identities「获取身份列表失败」根因，修复见 activate 成功分支 storeUserId）。
      await persistLoginSuccessCredentials({
        user: { id: '99', username: 'dana', email: 'd@example.com' },
        token: 'tok-dana',
      })
      expect(localStorage.getItem('currentUserId')).toBe('99')
    })

    it('resolveSwitchHref 替换 user 路径段', () => {
      expect(
        resolveSwitchHref('/user/111/profile/', '222'),
      ).toBe('/user/222/profile/')
      expect(resolveSwitchHref('/pricing/', '222')).toBe('/pricing/')
    })

    it('T10 resolveSwitchHref 离开租户 URL 并去掉 workspace_id', () => {
      expect(
        resolveSwitchHref('/tenant/99/work-panel/?workspace_id=7&tab=tasks', '222'),
      ).toBe('/user/222/profile/')
      expect(
        resolveSwitchHref('/user/111/projects/?workspace_id=7&sort=name', '222'),
      ).toBe('/user/222/projects/?sort=name')
    })

    it('T12 activate 成功后清除 JS 可读的旧 sessionid', async () => {
      document.cookie = 'sessionid=old-session-a; path=/'
      const apiFetch = vi.fn().mockResolvedValue({
        ok: true,
        json: async () => ({
          user: { id: '864758149662404608', username: 'bob' },
          token: 'tok-bob',
        }),
      })
      await activateSavedAccountSession({
        userId: '864758149662404608',
        token: 'tok-bob',
        apiFetch,
      })
      // HttpOnly 新 cookie 由服务端 Set-Cookie 写入，此处仅断言非 HttpOnly 旧值被清
      expect(document.cookie).not.toMatch(/sessionid=old-session-a/)
    })
  })
}

// @vitest-environment jsdom
// 微信回调凭据补全（resolveWechatSessionIdentity）单元测试：
// 回归保护「微信扫码登录 → onboarding 设置公司名 401」事故 —
// 回调仅携带 token 时，须解析用户身份并落盘 userId cookie（网关 forward-auth 兜底认证），
// 同时同步 active 账号槽。
if (!process.env.VITEST) {
  console.log('[skip] sessionUserIdUtils.wechat.test.js requires vitest runtime')
} else {
  const { beforeEach, describe, expect, it, vi } = await import('vitest')

  vi.mock('./apiUtils.js', async (importOriginal) => {
    const actual = await importOriginal()
    return {
      ...actual,
      apiFetch: vi.fn(),
    }
  })

  // mock 账号槽模块（真实断言 setActiveAccount 调用；sessionUserIdUtils 仅动态
  // import 其 setActiveAccount，与模块内其余导出无关）
  vi.mock('../domain/auth/services/saved_accounts_store.js', () => ({
    setActiveAccount: vi.fn(),
    getActiveToken: vi.fn(),
    getCurrentUserId: vi.fn(),
  }))

  const { resolveWechatSessionIdentity } = await import('./sessionUserIdUtils.js')
  const { apiFetch } = await import('./apiUtils.js')
  const { setActiveAccount } = await import(
    '../domain/auth/services/saved_accounts_store.js'
  )

  const PROFILE_OK = {
    user_id: '864758149662404608',
    username: 'wechat-user',
    avatar_url: 'https://wx.qlogo.cn/avatar/1',
  }

  describe('resolveWechatSessionIdentity（微信回调凭据补全）', () => {
    beforeEach(() => {
      vi.clearAllMocks()
      localStorage.clear()
      document.cookie = 'userId=; expires=Thu, 01 Jan 1970 00:00:00 GMT; path=/'
    })

    it('profile 解析成功：写入 userId cookie 并同步 active 账号槽（携带回调 token）', async () => {
      apiFetch.mockResolvedValue({
        ok: true,
        json: async () => PROFILE_OK,
      })
      const result = await resolveWechatSessionIdentity('wechat-tok-abc')
      expect(result).toEqual(expect.objectContaining({ ok: true }))
      // userId cookie 落盘（网关 forward-auth 兜底认证所需）
      expect(document.cookie).toContain('userId=864758149662404608')
      // 账号槽同步：token 必须用回调携带的 wechat_token（内存 token）
      expect(setActiveAccount).toHaveBeenCalledWith(
        expect.objectContaining({
          userId: '864758149662404608',
          token: 'wechat-tok-abc',
          username: 'wechat-user',
        }),
      )
    })

    it('profile 请求显式携带回调 token（不依赖内存缓存时序）', async () => {
      apiFetch.mockResolvedValue({
        ok: true,
        json: async () => PROFILE_OK,
      })
      await resolveWechatSessionIdentity('wechat-tok-abc')
      expect(apiFetch).toHaveBeenCalledWith(
        '/api/accounts/users/profile/',
        expect.objectContaining({
          headers: expect.objectContaining({
            Authorization: 'Token wechat-tok-abc',
          }),
        }),
      )
    })

    it('profile 瞬时失败：重试 1 次后成功（网络抖动兜底）', async () => {
      apiFetch
        .mockResolvedValueOnce({
          ok: false,
          status: 502,
          json: async () => ({}),
        })
        .mockResolvedValueOnce({
          ok: true,
          json: async () => PROFILE_OK,
        })
      const result = await resolveWechatSessionIdentity('wechat-tok-abc')
      expect(result).toEqual(expect.objectContaining({ ok: true }))
      expect(document.cookie).toContain('userId=864758149662404608')
      // 2 次 profile（1 失败 + 重试成功）+ 1 次 activate-session（OPT-20260807-005
      // 引入的 HttpOnly 会话 cookie 落盘调用）
      expect(apiFetch).toHaveBeenCalledTimes(3)
      expect(apiFetch).toHaveBeenLastCalledWith(
        '/api/accounts/users/activate-session/',
        expect.objectContaining({ method: 'POST' }),
      )
    })

    it('profile 请求失败（401 等）：重试后仍失败 → 不抛错、返回 false（不阻断登录跳转）', async () => {
      apiFetch.mockResolvedValue({
        ok: false,
        status: 401,
        json: async () => ({}),
      })
      await expect(resolveWechatSessionIdentity('wechat-tok-abc')).resolves.toEqual({
        ok: false,
        profile: null,
      })
      expect(document.cookie).not.toContain('userId=')
      // 401 也重试 1 次，共 2 次调用
      expect(apiFetch).toHaveBeenCalledTimes(2)
    })

    it('账号槽写入失败（setActiveAccount 拒绝）：userId cookie 兜底仍生效', async () => {
      apiFetch.mockResolvedValue({
        ok: true,
        json: async () => PROFILE_OK,
      })
      setActiveAccount.mockRejectedValue(new Error('槽位写入失败'))
      const result = await resolveWechatSessionIdentity('wechat-tok-abc')
      expect(result).toEqual(expect.objectContaining({ ok: true }))
      expect(document.cookie).toContain('userId=864758149662404608')
    })

    it('profile 响应缺少 user_id：返回 false', async () => {
      apiFetch.mockResolvedValue({
        ok: true,
        json: async () => ({ username: 'no-id' }),
      })
      await expect(resolveWechatSessionIdentity('wechat-tok-abc')).resolves.toEqual({
        ok: false,
        profile: null,
      })
    })
  })
}

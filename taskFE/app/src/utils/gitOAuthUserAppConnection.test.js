// @vitest-environment node
/**
 * OPT-20260819-001: 共享 Git OAuth user-app 绑定查询。
 * 创建任务门禁与评论侧 linked-projects 共用同一超时/解析逻辑。
 */
if (!process.env.VITEST) {
  console.log('[skip] gitOAuthUserAppConnection.test.js requires vitest runtime')
} else {
  const { describe, expect, it, vi, beforeEach } = await import('vitest')

  const hoistedMocks = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
  }))

  vi.mock('./apiUtils.js', () => ({
    apiFetch: hoistedMocks.apiFetchMock,
  }))

  const { fetchGitOAuthUserAppConnected, classifyGitOAuthProbePayload } = await import('./gitOAuthUserAppConnection.js')

  function jsonRes(body, ok = true, status = ok ? 200 : 400) {
    return { ok, status, json: async () => body }
  }

  describe('fetchGitOAuthUserAppConnected', () => {
    beforeEach(() => {
      hoistedMocks.apiFetchMock.mockReset()
    })

    it('connected=true 返回 true', async () => {
      hoistedMocks.apiFetchMock.mockResolvedValue(jsonRes({ connected: true }))
      expect(await fetchGitOAuthUserAppConnected('https://github.com/a/b.git')).toBe(true)
      expect(hoistedMocks.apiFetchMock).toHaveBeenCalledWith(
        '/api/git-oauth/user-app-connection/?repo_url=https%3A%2F%2Fgithub.com%2Fa%2Fb.git',
        expect.objectContaining({ credentials: 'include', skipSessionExpiredRedirect: true }),
      )
    })

    it('connections 数组含 connected 项返回 true', async () => {
      hoistedMocks.apiFetchMock.mockResolvedValue(jsonRes({ connections: [{ connected: false }, { connected: true }] }))
      expect(await fetchGitOAuthUserAppConnected('https://gitlab.example/c.git')).toBe(true)
    })

    it('未绑定返回 false', async () => {
      hoistedMocks.apiFetchMock.mockResolvedValue(jsonRes({ connected: false }))
      expect(await fetchGitOAuthUserAppConnected('https://github.com/a/b.git')).toBe(false)
    })

    it('401 视为未绑定且请求带 skipSessionExpiredRedirect', async () => {
      hoistedMocks.apiFetchMock.mockResolvedValue(jsonRes(
        { detail: '无法解析登录凭据，请重新登录' },
        false,
        401,
      ))
      expect(await fetchGitOAuthUserAppConnected('https://github.com/a/b.git')).toBe(false)
      expect(hoistedMocks.apiFetchMock).toHaveBeenCalledWith(
        expect.any(String),
        expect.objectContaining({ skipSessionExpiredRedirect: true }),
      )
    })

    it('非 2xx 抛 Error 并挂 traceId', async () => {
      hoistedMocks.apiFetchMock.mockResolvedValue(jsonRes({ detail: '服务不可用' }, false, 503))
      const err = await fetchGitOAuthUserAppConnected('https://github.com/a/b.git').catch((e) => e)
      expect(err).toBeInstanceOf(Error)
      expect(err.message).toBe('服务不可用')
    })

    it('超时抛 timeout 文案', async () => {
      hoistedMocks.apiFetchMock.mockRejectedValue(Object.assign(new Error('aborted'), { name: 'AbortError' }))
      const err = await fetchGitOAuthUserAppConnected('https://github.com/a/b.git').catch((e) => e)
      expect(err.message).toBe('检查 OAuth 绑定状态超时，请确认服务可用后重试')
    })

    it('repoUrl 为空时不带 query', async () => {
      hoistedMocks.apiFetchMock.mockResolvedValue(jsonRes({ connected: false }))
      await fetchGitOAuthUserAppConnected('')
      expect(hoistedMocks.apiFetchMock).toHaveBeenCalledWith(
        '/api/git-oauth/user-app-connection/',
        expect.any(Object),
      )
    })

    it('probeAccessToken 带 probe_access_token=1 且仅当 access_token_valid=true 才算绑定', async () => {
      hoistedMocks.apiFetchMock.mockResolvedValue(jsonRes({
        connected: true,
        access_token_valid: false,
      }))
      expect(await fetchGitOAuthUserAppConnected('https://github.com/a/b.git', {
        probeAccessToken: true,
      })).toBe(false)
      expect(hoistedMocks.apiFetchMock).toHaveBeenCalledWith(
        '/api/git-oauth/user-app-connection/?repo_url=https%3A%2F%2Fgithub.com%2Fa%2Fb.git&probe_access_token=1',
        expect.objectContaining({ credentials: 'include' }),
      )

      hoistedMocks.apiFetchMock.mockResolvedValue(jsonRes({
        connected: true,
        access_token_valid: true,
      }))
      expect(await fetchGitOAuthUserAppConnected('https://github.com/a/b.git', {
        probeAccessToken: true,
      })).toBe(true)
    })

    it('probeAccessToken 缺少 access_token_valid 时回退 connected（生产 probe 可能省略该字段）', async () => {
      hoistedMocks.apiFetchMock.mockResolvedValue(jsonRes({ connected: true }))
      expect(await fetchGitOAuthUserAppConnected('https://github.com/a/b.git', {
        probeAccessToken: true,
      })).toBe(true)

      hoistedMocks.apiFetchMock.mockResolvedValue(jsonRes({ connected: false }))
      expect(await fetchGitOAuthUserAppConnected('https://github.com/a/b.git', {
        probeAccessToken: true,
      })).toBe(false)
    })

    it('probe network_status=unreachable 不算 token 失效', async () => {
      hoistedMocks.apiFetchMock.mockResolvedValue(jsonRes({
        connected: true,
        access_token_valid: false,
        network_status: 'unreachable',
      }))
      expect(await fetchGitOAuthUserAppConnected('https://gitlab.example/c.git', {
        probeAccessToken: true,
      })).toBe(true)
    })
  })

  describe('classifyGitOAuthProbePayload', () => {
    it('marks GitLab network unreachable separately from invalid tokens', () => {
      expect(classifyGitOAuthProbePayload({
        connected: true,
        access_token_valid: false,
        network_status: 'unreachable',
      })).toEqual({ networkUnreachable: true, tokenValid: false })
      expect(classifyGitOAuthProbePayload({
        connected: true,
        access_token_valid: false,
      })).toEqual({ networkUnreachable: false, tokenValid: false })
      expect(classifyGitOAuthProbePayload({
        connected: true,
        access_token_valid: true,
        network_status: 'skipped_intranet',
      })).toEqual({ networkUnreachable: false, tokenValid: true })
    })
  })
}

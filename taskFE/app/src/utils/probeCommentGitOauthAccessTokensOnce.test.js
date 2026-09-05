// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] probeCommentGitOauthAccessTokensOnce.test.js requires vitest runtime')
} else {
  const { describe, expect, it, vi } = await import('vitest')
  const { probeCommentGitOauthAccessTokensOnce } = await import('./probeCommentGitOauthAccessTokensOnce.js')

  const gitlabUrl = 'https://gitlab-tencent-sh-1.daydaymoney.com/example-user/somanyad.git'

  describe('probeCommentGitOauthAccessTokensOnce', () => {
    it('probes each appearing-bound url once and records invalid tokens', async () => {
      const fetchConnected = vi.fn()
        .mockResolvedValueOnce(false)
        .mockResolvedValueOnce(true)
      const probedUrls = new Set()
      const readiness = {
        loading: false,
        hasOAuthRepos: true,
        allBound: true,
        unboundRepoUrls: [],
      }

      const first = await probeCommentGitOauthAccessTokensOnce({
        urls: [gitlabUrl],
        readiness,
        probedUrls,
        fetchConnected,
      })
      expect(first.invalidRepoUrls).toEqual([gitlabUrl])
      expect(fetchConnected).toHaveBeenCalledTimes(1)
      expect(fetchConnected).toHaveBeenCalledWith(gitlabUrl, { probeAccessToken: true })

      const second = await probeCommentGitOauthAccessTokensOnce({
        urls: [gitlabUrl],
        readiness,
        probedUrls,
        fetchConnected,
      })
      expect(second.invalidRepoUrls).toEqual([])
      expect(fetchConnected).toHaveBeenCalledTimes(1)
    })

    it('does not probe when readiness says unbound', async () => {
      const fetchConnected = vi.fn()
      const result = await probeCommentGitOauthAccessTokensOnce({
        urls: [gitlabUrl],
        readiness: {
          loading: false,
          hasOAuthRepos: true,
          allBound: false,
          unboundRepoUrls: [gitlabUrl],
        },
        probedUrls: new Set(),
        fetchConnected,
      })
      expect(result.probed).toEqual([])
      expect(fetchConnected).not.toHaveBeenCalled()
    })

    it('records GitLab network unreachable separately from invalid tokens', async () => {
      const fetchOutcome = vi.fn().mockResolvedValue({ networkUnreachable: true, tokenValid: false })
      const result = await probeCommentGitOauthAccessTokensOnce({
        urls: [gitlabUrl],
        readiness: {
          loading: false,
          hasOAuthRepos: true,
          allBound: true,
          unboundRepoUrls: [],
        },
        probedUrls: new Set(),
        fetchOutcome,
      })
      expect(result.unreachableRepoUrls).toEqual([gitlabUrl])
      expect(result.invalidRepoUrls).toEqual([])
    })

    it('does not throw on probe timeout; records checkFailedRepoUrls and traceId', async () => {
      const timeout = Object.assign(new Error('检查 OAuth 绑定状态超时，请确认服务可用后重试'), {
        traceId: 'e0a7ec3d-133a-4ff9-aa9d-15780e06d1eb',
      })
      const fetchOutcome = vi.fn().mockRejectedValue(timeout)
      const result = await probeCommentGitOauthAccessTokensOnce({
        urls: [gitlabUrl],
        readiness: {
          loading: false,
          hasOAuthRepos: true,
          allBound: true,
          unboundRepoUrls: [],
        },
        probedUrls: new Set(),
        fetchOutcome,
      })
      expect(result.checkFailedRepoUrls).toEqual([gitlabUrl])
      expect(result.invalidRepoUrls).toEqual([])
      expect(result.probeError).toBe('检查 OAuth 绑定状态超时，请确认服务可用后重试')
      expect(result.probeTraceId).toBe('e0a7ec3d-133a-4ff9-aa9d-15780e06d1eb')
    })

    it('force=true probes a previously check_failed url even though readiness summary is no longer bound (OPT-20260902-011)', async () => {
      const fetchOutcome = vi.fn().mockResolvedValue({ networkUnreachable: false, tokenValid: true })
      const probedUrls = new Set([gitlabUrl])
      const readiness = {
        loading: false,
        hasOAuthRepos: true,
        allBound: true,
        unboundRepoUrls: [],
        checkFailedRepoUrls: [gitlabUrl],
        probeError: '检查 OAuth 绑定状态超时，请确认服务可用后重试',
      }
      // 未 force：摘要为 check_failed → 不再自动探测
      const auto = await probeCommentGitOauthAccessTokensOnce({
        urls: [gitlabUrl],
        readiness,
        probedUrls,
        fetchOutcome,
      })
      expect(auto.probed).toEqual([])
      expect(fetchOutcome).not.toHaveBeenCalled()
      // force：绕开门禁强制重探测（重试前调用方已把 url 移出 probedUrls）
      probedUrls.delete(gitlabUrl)
      const retried = await probeCommentGitOauthAccessTokensOnce({
        urls: [gitlabUrl],
        readiness,
        probedUrls,
        force: true,
        fetchOutcome,
      })
      expect(retried.probed).toEqual([gitlabUrl])
      expect(retried.checkFailedRepoUrls).toEqual([])
      expect(fetchOutcome).toHaveBeenCalledTimes(1)
    })
  })
}

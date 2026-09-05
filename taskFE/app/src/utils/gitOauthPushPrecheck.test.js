// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] gitOauthPushPrecheck.test.js requires vitest runtime')
} else {
  const { describe, expect, it, vi, beforeEach } = await import('vitest')

  const hoisted = vi.hoisted(() => ({
    fetchConnected: vi.fn(),
  }))

  vi.mock('./gitOAuthUserAppConnection.js', () => ({
    fetchGitOAuthUserAppConnected: hoisted.fetchConnected,
  }))
  vi.mock('./requestErrorDisplay.js', () => ({
    showRequestError: vi.fn(),
  }))

  const {
    assertGitOauthBoundForRepoUrls,
    blockCommentRunIfGitOauthUnbound,
    collectOAuthRepoUrls,
    gitOauthCommentGrantMissingMessage,
    gitOauthUnboundActionMessage,
    gitOauthUnboundReasonForRepoUrls,
    repoUrlsFromTaskRepoRows,
  } = await import('./gitOauthPushPrecheck.js')

  describe('gitOauthPushPrecheck', () => {
    beforeEach(() => {
      hoisted.fetchConnected.mockReset()
    })

    it('collects unique GitHub/GitLab urls and skips generic git', () => {
      expect(collectOAuthRepoUrls([
        'https://github.com/a/b.git',
        'https://github.com/a/b.git',
        'git://example.com/plain.git',
        'https://gitlab.example/c.git',
      ])).toEqual([
        'https://github.com/a/b.git',
        'https://gitlab.example/c.git',
      ])
    })

    it('maps task repo rows to urls', () => {
      expect(repoUrlsFromTaskRepoRows([
        { url: 'https://github.com/a/b.git' },
        { url: '  ' },
        { url: 'https://gitlab.example/c.git' },
      ])).toEqual([
        'https://github.com/a/b.git',
        'https://gitlab.example/c.git',
      ])
    })

    it('assert ok when no oauth repos', async () => {
      const result = await assertGitOauthBoundForRepoUrls(['git://example.com/plain.git'])
      expect(result).toEqual({ ok: true, unboundRepoUrls: [] })
      expect(hoisted.fetchConnected).not.toHaveBeenCalled()
    })

    it('assert lists unbound oauth repos', async () => {
      hoisted.fetchConnected.mockImplementation(async (url) => url.includes('github'))
      const result = await assertGitOauthBoundForRepoUrls([
        'https://github.com/a/b.git',
        'https://gitlab.example/c.git',
      ])
      expect(result.ok).toBe(false)
      expect(result.unboundRepoUrls).toEqual(['https://gitlab.example/c.git'])
    })

    it('reason is empty when allowBare', async () => {
      expect(await gitOauthUnboundReasonForRepoUrls(
        ['https://github.com/a/b.git'],
        { allowBare: true },
      )).toBe('')
      expect(hoisted.fetchConnected).not.toHaveBeenCalled()
    })

    it('reason uses action label when unbound', async () => {
      hoisted.fetchConnected.mockResolvedValue(false)
      const reason = await gitOauthUnboundReasonForRepoUrls(
        ['https://github.com/a/b.git'],
        { actionLabel: '提交并运行' },
      )
      expect(reason).toBe(gitOauthUnboundActionMessage('提交并运行'))
      expect(reason).toMatch(/提交并运行前/)
      expect(reason).toMatch(/OAuth/)
    })

    it('blocks when probe reports access_token_valid=false despite DB connected', async () => {
      // 推送预检路径传 probeAccessToken: true；后端返回 access_token_valid=false 时
      // fetchGitOAuthUserAppConnected 降级为未绑定，即使 DB connected 也要拦截。
      hoisted.fetchConnected.mockResolvedValue(false)
      const reason = await gitOauthUnboundReasonForRepoUrls(
        ['https://github.com/a/b.git'],
        { actionLabel: '推送', probeAccessToken: true },
      )
      expect(reason).toBe(gitOauthUnboundActionMessage('推送'))
      expect(hoisted.fetchConnected).toHaveBeenCalledWith(
        'https://github.com/a/b.git',
        expect.objectContaining({ probeAccessToken: true }),
      )
    })

    it('passes through when probe reports access_token_valid=true', async () => {
      hoisted.fetchConnected.mockResolvedValue(true)
      const reason = await gitOauthUnboundReasonForRepoUrls(
        ['https://github.com/a/b.git'],
        { actionLabel: '推送', probeAccessToken: true },
      )
      expect(reason).toBe('')
    })

    it('blockCommentRunIfGitOauthUnbound shows error and returns true when unbound', async () => {
      hoisted.fetchConnected.mockResolvedValue(false)
      const { showRequestError } = await import('./requestErrorDisplay.js')
      const blocked = await blockCommentRunIfGitOauthUnbound([
        { project: { git_repos: ['https://github.com/a/b.git'] } },
      ])
      expect(blocked).toBe(true)
      expect(showRequestError).toHaveBeenCalledWith(gitOauthUnboundActionMessage('提交并运行'))
    })

    it('blockCommentRunIfGitOauthUnbound blocks when L1 connected but session grant ticket missing', async () => {
      hoisted.fetchConnected.mockResolvedValue(true)
      const { showRequestError } = await import('./requestErrorDisplay.js')
      const blocked = await blockCommentRunIfGitOauthUnbound([
        { project: { git_repos: ['https://github.com/a/b.git'] } },
      ])
      expect(blocked).toBe(true)
      expect(showRequestError).toHaveBeenCalledWith(gitOauthCommentGrantMissingMessage('提交并运行'))
    })

    it('blockCommentRunIfGitOauthUnbound passes when L1 connected and session grant exists', async () => {
      const memory = {}
      globalThis.sessionStorage = {
        getItem: (k) => (k in memory ? memory[k] : null),
        setItem: (k, v) => { memory[k] = String(v) },
      }
      const { rememberGrantTicket } = await import('./grantTicketSession.js')
      rememberGrantTicket('tkt-1', 'https://github.com/a/b.git')
      hoisted.fetchConnected.mockResolvedValue(true)
      const blocked = await blockCommentRunIfGitOauthUnbound([
        { project: { git_repos: ['https://github.com/a/b.git'] } },
      ])
      expect(blocked).toBe(false)
    })
  })
}

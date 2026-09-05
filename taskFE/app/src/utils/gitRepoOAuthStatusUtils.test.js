// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] gitRepoOAuthStatusUtils.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const {
    GIT_REPO_OAUTH_STATUS,
    gitRepoOAuthNeedsAttentionRank,
  gitRepoOAuthStatusBadgeClass,
  gitRepoOAuthStatusHint,
  isPlaceholderGitRepoOAuthStatus,
    resolveGitRepoOAuthStatusLabel,
    sortReposByOAuthAttention,
  } = await import('./gitRepoOAuthStatusUtils.js')

  describe('gitRepoOAuthStatusUtils', () => {
    it('maps token_status to Chinese labels', () => {
      expect(resolveGitRepoOAuthStatusLabel('token_available')).toBe('已授权')
      expect(resolveGitRepoOAuthStatusLabel('not_bound')).toBe('需要授权')
      expect(resolveGitRepoOAuthStatusLabel('token_error')).toBe('授权异常')
      expect(resolveGitRepoOAuthStatusLabel('not_applicable')).toBe('无需 OAuth')
    })

    it('shows loading label while checking', () => {
      expect(resolveGitRepoOAuthStatusLabel('', { loading: true })).toBe('检查中…')
    })

    it('explains token_error so status+url+retry are not the only copy', () => {
      expect(gitRepoOAuthStatusHint('token_error')).toContain('无法访问该仓库')
      expect(gitRepoOAuthStatusHint('token_error')).toContain('重新绑定')
      expect(gitRepoOAuthStatusHint('not_bound')).toBe('')
      expect(gitRepoOAuthStatusHint('token_available')).toBe('')
    })

    it('uses semantic badge colors per status', () => {
      expect(gitRepoOAuthStatusBadgeClass('token_available')).toContain('green')
      expect(gitRepoOAuthStatusBadgeClass('not_bound')).toContain('amber')
      expect(gitRepoOAuthStatusBadgeClass('token_error')).toContain('red')
    })

    it('treats empty and not_applicable as placeholder', () => {
      expect(isPlaceholderGitRepoOAuthStatus('')).toBe(true)
      expect(isPlaceholderGitRepoOAuthStatus('not_applicable')).toBe(true)
      expect(isPlaceholderGitRepoOAuthStatus('not_bound')).toBe(false)
    })

    it('ranks unauthorized ahead of authorized', () => {
      expect(gitRepoOAuthNeedsAttentionRank(GIT_REPO_OAUTH_STATUS.NOT_BOUND)).toBeLessThan(
        gitRepoOAuthNeedsAttentionRank(GIT_REPO_OAUTH_STATUS.TOKEN_AVAILABLE),
      )
      expect(gitRepoOAuthNeedsAttentionRank(GIT_REPO_OAUTH_STATUS.TOKEN_ERROR)).toBeLessThan(
        gitRepoOAuthNeedsAttentionRank(GIT_REPO_OAUTH_STATUS.TOKEN_AVAILABLE),
      )
    })

    it('sorts repos so unauthorized rows stay on top while preserving relative order', () => {
      const rows = [
        { path: 'DaydaymoneyGrafana', url: 'https://gitlab.daydaymoney.com/g/DaydaymoneyGrafana.git' },
        { path: 'docs', url: 'https://gitlab.daydaymoney.com/g/docs.git' },
        { path: 'runAll', url: 'https://gitlab.daydaymoney.com/g/runAll.git' },
      ]
      const statusByUrl = {
        'https://gitlab.daydaymoney.com/g/DaydaymoneyGrafana.git': 'token_available',
        'https://gitlab.daydaymoney.com/g/docs.git': 'not_bound',
        'https://gitlab.daydaymoney.com/g/runAll.git': 'token_available',
      }
      const sorted = sortReposByOAuthAttention(rows, (row) => statusByUrl[row.url] || '')
      expect(sorted.map((r) => r.path)).toEqual(['docs', 'DaydaymoneyGrafana', 'runAll'])
    })
  })
}

// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] commentOAuthGrantCheck.test.js requires vitest runtime')
} else {
  const { beforeEach, describe, expect, it } = await import('vitest')
  const {
    commentOAuthGrantMissingMessage,
    commentOAuthGrantMissingReason,
    mergeSessionGrantIntoReadiness,
    missingCommentOAuthGrantSite,
    sessionGrantUnboundRepoUrls,
  } = await import('./commentOAuthGrantCheck.js')
  const { rememberGrantTicket } = await import('./grantTicketSession.js')

  describe('commentOAuthGrantCheck', () => {
    const memory = {}
    beforeEach(() => {
      for (const key of Object.keys(memory)) delete memory[key]
      globalThis.sessionStorage = {
        getItem: (k) => (k in memory ? memory[k] : null),
        setItem: (k, v) => { memory[k] = String(v) },
      }
    })

    it('treats L1-bound composer as unbound until session grant ticket exists', () => {
      const merged = mergeSessionGrantIntoReadiness({
        hasOAuthRepos: true,
        loading: false,
        allBound: true,
        unboundRepoUrls: [],
      }, ['https://github.com/acme/demo.git'])
      expect(merged.allBound).toBe(false)
      expect(merged.unboundRepoUrls).toEqual(['https://github.com/acme/demo.git'])
      expect(sessionGrantUnboundRepoUrls(['https://github.com/acme/demo.git'])).toEqual([
        'https://github.com/acme/demo.git',
      ])
    })

    it('keeps allBound when session grant ticket covers the gitsite', () => {
      rememberGrantTicket('tkt-1', 'https://github.com/acme/demo.git')
      const merged = mergeSessionGrantIntoReadiness({
        hasOAuthRepos: true,
        loading: false,
        allBound: true,
        unboundRepoUrls: [],
      }, ['https://github.com/acme/demo.git'])
      expect(merged.allBound).toBe(true)
      expect(sessionGrantUnboundRepoUrls(['https://github.com/acme/demo.git'])).toEqual([])
    })

    it('detects missing oauth_gitsite on the running comment', () => {
      expect(missingCommentOAuthGrantSite({
        id: 'cmt_1',
        repo_identities: [{ repo_url: 'https://github.com/acme/demo.git' }],
      }, ['https://github.com/acme/demo.git'])).toBe('github.com')
      expect(missingCommentOAuthGrantSite({
        id: 'cmt_1',
        repo_identities: [{ repo_url: 'https://github.com/acme/demo.git', oauth_gitsite: 'github.com' }],
      }, ['https://github.com/acme/demo.git'])).toBe('')
    })

    it('blocks layer push when the running comment has no L2 grant', () => {
      const reason = commentOAuthGrantMissingReason({
        displayComments: [
          {
            id: 'cmt_run',
            repo_identities: [{ repo_url: 'https://github.com/acme/demo.git' }],
          },
        ],
        commentId: 'cmt_run',
      }, ['https://github.com/acme/demo.git'], '提交并推送')
      expect(reason).toBe(commentOAuthGrantMissingMessage('github.com', '提交并推送'))
      expect(reason).toMatch(/提交并推送前/)
    })

    it('allows layer push when the running comment already has L2', () => {
      const reason = commentOAuthGrantMissingReason({
        displayComments: [
          {
            id: 'cmt_run',
            repo_identities: [{
              repo_url: 'https://github.com/acme/demo.git',
              oauth_gitsite: 'github.com',
            }],
          },
        ],
        commentId: 'cmt_run',
      }, ['https://github.com/acme/demo.git'], '提交并推送')
      expect(reason).toBe('')
    })
  })
}

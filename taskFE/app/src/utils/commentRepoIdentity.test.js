// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] commentRepoIdentity.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const {
    collectLinkedRepoUrls,
    commentComposerGitOauthHint,
    resolveCommentRunOauthBlockedReason,
    resolveLinkedRepoProject,
    taskProjectsAllowNestedClone,
    validateCommentRepoIdentities,
  } = await import('./commentRepoIdentity.js')

  describe('commentRepoIdentity', () => {
    it('collects unique repo urls from linked projects', () => {
      const urls = collectLinkedRepoUrls([
        { project: { git_repos: ['https://github.com/a/b.git', 'https://github.com/a/b.git'] } },
        { project: { git_repos: ['https://gitlab.example/c.git'] } },
      ])
      expect(urls).toEqual(['https://github.com/a/b.git', 'https://gitlab.example/c.git'])
    })

    it('resolves linked repo to project clone config', () => {
      const row = resolveLinkedRepoProject(
        [
          {
            project_id: 'p1',
            project: {
              git_repos: ['https://github.com/a/b.git'],
              auto_clone_nested_repos: false,
            },
          },
        ],
        'https://github.com/a/b.git',
      )
      expect(row).toEqual({ projectId: 'p1', autoCloneNestedRepos: false })
    })

    it('defaults autoCloneNestedRepos to true when project omits the field', () => {
      const row = resolveLinkedRepoProject(
        [{ id: 'p2', project: { git_repos: ['https://gitlab.example/c.git'] } }],
        'https://gitlab.example/c.git',
      )
      expect(row).toEqual({ projectId: 'p2', autoCloneNestedRepos: true })
    })

    it('taskProjectsAllowNestedClone is false when any linked project disabled nested clone', () => {
      expect(taskProjectsAllowNestedClone([
        { project: { auto_clone_nested_repos: false } },
      ])).toBe(false)
      expect(taskProjectsAllowNestedClone([
        { project: { auto_clone_nested_repos: true } },
      ])).toBe(true)
      expect(taskProjectsAllowNestedClone([])).toBe(true)
    })

    it('returns null when repo has no project_id', () => {
      expect(resolveLinkedRepoProject(
        [{ project: { git_repos: ['https://github.com/a/b.git'] } }],
        'https://github.com/a/b.git',
      )).toBeNull()
    })

    it('rejects missing git identity for a required repo', () => {
      const err = validateCommentRepoIdentities(
        [{ repo_url: 'https://gitlab.example/c.git', git_identity_id: '' }],
        ['https://gitlab.example/c.git'],
      )
      expect(err).toContain('Git 提交身份')
    })

    it('rejects missing github user for github repo', () => {
      const err = validateCommentRepoIdentities(
        [{ repo_url: 'https://github.com/a/b.git', git_identity_id: 'gid-1' }],
        ['https://github.com/a/b.git'],
      )
      expect(err).toContain('授权账号')
    })

    it('git oauth hint is unbound when repos need oauth and none are bound', () => {
      const hint = commentComposerGitOauthHint({
        hasOAuthRepos: true,
        loading: false,
        allBound: false,
        unboundRepoUrls: ['https://github.com/a/b.git'],
      })
      expect(hint.kind).toBe('unbound')
      expect(hint.text).toMatch(/Git OAuth 使用授权/)
      expect(hint.text).toMatch(/提交并运行/)
      expect(hint.text).not.toMatch(/仍可发送评论/)
    })

    it('blocks submit-and-run when oauth is unbound or still loading', () => {
      expect(resolveCommentRunOauthBlockedReason({ hasOAuthRepos: false })).toBe('')
      expect(resolveCommentRunOauthBlockedReason({
        hasOAuthRepos: true,
        loading: true,
      })).toMatch(/正在检查/)
      expect(resolveCommentRunOauthBlockedReason({
        hasOAuthRepos: true,
        loading: false,
        allBound: false,
      })).toMatch(/提交并运行前/)
      expect(resolveCommentRunOauthBlockedReason({
        hasOAuthRepos: true,
        loading: false,
        allBound: true,
      })).toBe('')
      expect(resolveCommentRunOauthBlockedReason({
        hasOAuthRepos: true,
        loading: false,
        allBound: true,
      }, ['https://github.com/a/b.git'])).toMatch(/使用授权/)
    })

    it('git oauth hint is bound when all oauth repos are connected', () => {
      const hint = commentComposerGitOauthHint({
        hasOAuthRepos: true,
        loading: false,
        allBound: true,
        unboundRepoUrls: [],
      })
      expect(hint.kind).toBe('bound')
      expect(hint.text).toMatch(/已绑定 Git OAuth/)
    })

    it('git oauth hint is loading while connection status is in flight', () => {
      const hint = commentComposerGitOauthHint({
        hasOAuthRepos: true,
        loading: true,
        allBound: false,
        unboundRepoUrls: [],
      })
      expect(hint.kind).toBe('loading')
      expect(hint.text).toMatch(/正在检查/)
    })

    it('accepts complete selections', () => {
      const err = validateCommentRepoIdentities(
        [
          {
            repo_url: 'https://github.com/a/b.git',
            git_identity_id: 'gid-1',
            github_user_id: '9',
          },
        ],
        ['https://github.com/a/b.git'],
      )
      expect(err).toBe('')
    })
  })
}

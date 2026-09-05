// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] commentExecutionGitOauth.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const {
    collectCommentOauthRepoUrls,
    collectDisplayedCommentOauthRepoUrls,
    commentGitOauthSummary,
    commentOauthUrlsNeedingAccessTokenProbe,
    gitPrOauthStatusErrorText,
    mergeReadinessAfterAccessTokenProbe,
    sharedUserGitOAuthIsBound,
    shouldShowGitPrOauthBind,
  } = await import('./commentExecutionGitOauth.js')

  const gitlabUrl = 'https://gitlab-tencent-sh-1.daydaymoney.com/example-user/somanyad.git'
  const githubUrl = 'https://github.com/acme/demo.git'
  const sshUrl = 'git@internal.example:group/repo.git'

  describe('collectCommentOauthRepoUrls', () => {
    it('returns empty when neither comment nor fallback has oauth-capable urls', () => {
      expect(collectCommentOauthRepoUrls([], [])).toEqual([])
      expect(collectCommentOauthRepoUrls(null, undefined)).toEqual([])
      expect(collectCommentOauthRepoUrls([{ repo_url: sshUrl, git_identity_id: 'gi_1' }])).toEqual([])
    })

    it('keeps unique oauth-capable urls from comment identities without requiring git_identity_id', () => {
      expect(
        collectCommentOauthRepoUrls([
          { repo_url: gitlabUrl },
          { repo_url: gitlabUrl, git_identity_id: 'gi_1' },
          { repo_url: githubUrl, git_identity_id: 'gi_2' },
          { repo_url: sshUrl },
          null,
        ]),
      ).toEqual([gitlabUrl, githubUrl])
    })

    it('falls back to task-level identities when comment has no oauth urls', () => {
      expect(
        collectCommentOauthRepoUrls(
          [{ repo_url: sshUrl }],
          [{ repo_url: gitlabUrl, git_identity_id: 'gi_work' }],
        ),
      ).toEqual([gitlabUrl])
    })

    it('prefers comment urls over fallback when comment has oauth-capable repos', () => {
      expect(
        collectCommentOauthRepoUrls(
          [{ repo_url: githubUrl }],
          [{ repo_url: gitlabUrl }],
        ),
      ).toEqual([githubUrl])
    })
  })

  describe('commentGitOauthSummary', () => {
    it('returns null when there are no oauth-capable repos', () => {
      expect(commentGitOauthSummary([], { loading: false, allBound: true, unboundRepoUrls: [] })).toBeNull()
      expect(commentGitOauthSummary([sshUrl], { loading: false, hasOAuthRepos: true, allBound: true })).toBeNull()
    })

    it('shows checking when readiness is missing or still loading', () => {
      expect(commentGitOauthSummary([gitlabUrl], null)).toMatchObject({
        kind: 'loading',
        text: 'Git OAuth · 检查中',
      })
      expect(commentGitOauthSummary([gitlabUrl], { loading: true, unboundRepoUrls: [] })).toMatchObject({
        kind: 'loading',
        text: 'Git OAuth · 检查中',
      })
    })

    it('shows bound when loading finished and comment urls are not unbound', () => {
      const item = commentGitOauthSummary([gitlabUrl], {
        loading: false,
        hasOAuthRepos: true,
        allBound: true,
        unboundRepoUrls: [],
      })
      expect(item).toMatchObject({
        kind: 'bound',
        text: 'Git OAuth · 已绑定',
      })
      expect(item.title).toContain('gitlab-tencent-sh-1.daydaymoney.com')
      expect(item.unboundRepoUrls).toEqual([])
    })

    it('shows unbound with the first unbound repo when any comment oauth url is unbound', () => {
      const item = commentGitOauthSummary([gitlabUrl, githubUrl], {
        loading: false,
        hasOAuthRepos: true,
        allBound: false,
        unboundRepoUrls: [githubUrl],
      })
      expect(item).toMatchObject({
        kind: 'unbound',
        text: 'Git OAuth · 未绑定',
        bindLabel: '去绑定 github.com',
      })
      expect(item.unboundRepoUrls).toEqual([githubUrl])
      expect(item.title).toContain(githubUrl)
    })

    it('counts multiple unbound oauth repos in the title', () => {
      const item = commentGitOauthSummary([gitlabUrl, githubUrl], {
        loading: false,
        hasOAuthRepos: true,
        allBound: false,
        unboundRepoUrls: [gitlabUrl, githubUrl],
      })
      expect(item.kind).toBe('unbound')
      expect(item.text).toBe('Git OAuth · 未绑定 2')
      expect(item.unboundRepoUrls).toEqual([gitlabUrl, githubUrl])
    })

    it('treats oauth-capable comment urls as unbound when task reports no oauth targets', () => {
      const item = commentGitOauthSummary([gitlabUrl], {
        loading: false,
        hasOAuthRepos: false,
        allBound: false,
        unboundRepoUrls: [],
      })
      expect(item.kind).toBe('unbound')
      expect(item.unboundRepoUrls).toEqual([gitlabUrl])
    })

    it('overlays bound_no_write when OAuth is bound but last push was permission denied', () => {
      const pushError =
        'remote: Permission to ruandao/helloworld.git denied to alice.'
      const item = commentGitOauthSummary(
        [githubUrl],
        {
          loading: false,
          hasOAuthRepos: true,
          allBound: true,
          unboundRepoUrls: [],
        },
        pushError,
      )
      expect(item).toMatchObject({
        kind: 'bound_no_write',
        text: 'Git OAuth · 无写权限',
        bindLabel: '换账号授权 github.com',
      })
      expect(item.unboundRepoUrls).toEqual([githubUrl])
      expect(item.title).toContain(githubUrl)
      expect(item.title).toContain('Permission to ruandao/helloworld.git denied')
    })

    it('keeps unbound when token is missing even if push error is permission denied', () => {
      const item = commentGitOauthSummary(
        [githubUrl],
        {
          loading: false,
          hasOAuthRepos: true,
          allBound: false,
          unboundRepoUrls: [githubUrl],
        },
        'remote: Permission to ruandao/helloworld.git denied to alice.',
      )
      expect(item.kind).toBe('unbound')
      expect(item.text).toBe('Git OAuth · 未绑定')
    })

    it('keeps bound for generic push errors that are not permission denied', () => {
      const item = commentGitOauthSummary(
        [gitlabUrl],
        {
          loading: false,
          hasOAuthRepos: true,
          allBound: true,
          unboundRepoUrls: [],
        },
        'network unreachable',
      )
      expect(item.kind).toBe('bound')
      expect(item.text).toBe('Git OAuth · 已绑定')
    })

    it('shows 网络不可达 when probe reports GitLab unreachable and does not treat it as unbound', () => {
      const item = commentGitOauthSummary([gitlabUrl], {
        loading: false,
        hasOAuthRepos: true,
        allBound: true,
        unboundRepoUrls: [],
        unreachableRepoUrls: [gitlabUrl],
      })
      expect(item).toMatchObject({
        kind: 'unreachable',
        text: 'Git OAuth · 网络不可达',
        bindLabel: '',
      })
      expect(item.unboundRepoUrls).toEqual([])
    })

    it('prefers unbound over unreachable when the token is missing', () => {
      const item = commentGitOauthSummary([gitlabUrl], {
        loading: false,
        hasOAuthRepos: true,
        allBound: false,
        unboundRepoUrls: [gitlabUrl],
        unreachableRepoUrls: [gitlabUrl],
      })
      expect(item.kind).toBe('unbound')
      expect(item.text).toBe('Git OAuth · 未绑定')
    })

    it('shows 检查超时 on probe timeout without claiming bound or unbound', () => {
      const item = commentGitOauthSummary([githubUrl], {
        loading: false,
        hasOAuthRepos: true,
        allBound: true,
        unboundRepoUrls: [],
        checkFailedRepoUrls: [githubUrl],
        probeError: '检查 OAuth 绑定状态超时，请确认服务可用后重试',
        probeTraceId: 'e0a7ec3d-133a-4ff9-aa9d-15780e06d1eb',
      })
      expect(item).toMatchObject({
        kind: 'check_failed',
        text: 'Git OAuth · 检查超时',
        bindLabel: '',
        traceId: 'e0a7ec3d-133a-4ff9-aa9d-15780e06d1eb',
      })
      expect(item.title).toContain('检查 OAuth 绑定状态超时')
      expect(item.unboundRepoUrls).toEqual([])
    })

    it('shows 检查失败 when probe error is not a timeout', () => {
      const item = commentGitOauthSummary([githubUrl], {
        loading: false,
        hasOAuthRepos: true,
        allBound: true,
        unboundRepoUrls: [],
        checkFailedRepoUrls: [githubUrl],
        probeError: '无法确认 Git OAuth AccessToken 是否有效',
      })
      expect(item.kind).toBe('check_failed')
      expect(item.text).toBe('Git OAuth · 检查失败')
    })

    it('prefers unbound over check_failed when the token is missing', () => {
      const item = commentGitOauthSummary([githubUrl], {
        loading: false,
        hasOAuthRepos: true,
        allBound: false,
        unboundRepoUrls: [githubUrl],
        checkFailedRepoUrls: [githubUrl],
        probeError: '检查 OAuth 绑定状态超时，请确认服务可用后重试',
      })
      expect(item.kind).toBe('unbound')
    })
  })

  describe('collectDisplayedCommentOauthRepoUrls', () => {
    it('returns empty when the comments feed is empty', () => {
      expect(collectDisplayedCommentOauthRepoUrls([], [{ repo_url: gitlabUrl }])).toEqual([])
    })

    it('dedupes oauth urls across displayed comments and falls back per comment', () => {
      expect(
        collectDisplayedCommentOauthRepoUrls(
          [
            { repo_identities: [{ repo_url: gitlabUrl }] },
            { repo_identities: [{ repo_url: gitlabUrl }, { repo_url: githubUrl }] },
            { repo_identities: [] },
          ],
          [{ repo_url: gitlabUrl }],
        ),
      ).toEqual([gitlabUrl, githubUrl])
    })
  })

  describe('commentOauthUrlsNeedingAccessTokenProbe', () => {
    const boundReadiness = {
      loading: false,
      hasOAuthRepos: true,
      allBound: true,
      unboundRepoUrls: [],
    }

    it('returns bound comment urls only after readiness finished as bound', () => {
      expect(commentOauthUrlsNeedingAccessTokenProbe([gitlabUrl], { loading: true })).toEqual([])
      expect(commentOauthUrlsNeedingAccessTokenProbe([gitlabUrl], {
        loading: false,
        hasOAuthRepos: true,
        allBound: false,
        unboundRepoUrls: [gitlabUrl],
      })).toEqual([])
      expect(commentOauthUrlsNeedingAccessTokenProbe([gitlabUrl], boundReadiness)).toEqual([gitlabUrl])
    })
  })

  describe('mergeReadinessAfterAccessTokenProbe', () => {
    it('marks probing as loading so the chip stays 检查中', () => {
      const next = mergeReadinessAfterAccessTokenProbe(
        { loading: false, hasOAuthRepos: true, allBound: true, unboundRepoUrls: [], startBlocked: false },
        { probing: true, invalidRepoUrls: [] },
      )
      expect(next.loading).toBe(true)
      expect(next.allBound).toBe(true)
    })

    it('adds probe-invalid urls to unbound and blocks start', () => {
      const next = mergeReadinessAfterAccessTokenProbe(
        { loading: false, hasOAuthRepos: true, allBound: true, unboundRepoUrls: [], startBlocked: false },
        { probing: false, invalidRepoUrls: [gitlabUrl] },
      )
      expect(next.loading).toBe(false)
      expect(next.allBound).toBe(false)
      expect(next.startBlocked).toBe(true)
      expect(next.unboundRepoUrls).toEqual([gitlabUrl])
    })

    it('keeps allBound when GitLab is only unreachable', () => {
      const next = mergeReadinessAfterAccessTokenProbe(
        { loading: false, hasOAuthRepos: true, allBound: true, unboundRepoUrls: [], startBlocked: false },
        { probing: false, invalidRepoUrls: [], unreachableRepoUrls: [gitlabUrl] },
      )
      expect(next.allBound).toBe(true)
      expect(next.startBlocked).toBe(false)
      expect(next.unboundRepoUrls).toEqual([])
      expect(next.unreachableRepoUrls).toEqual([gitlabUrl])
    })

    it('keeps allBound but blocks start when AccessToken probe times out', () => {
      const next = mergeReadinessAfterAccessTokenProbe(
        { loading: false, hasOAuthRepos: true, allBound: true, unboundRepoUrls: [], startBlocked: false },
        {
          probing: false,
          invalidRepoUrls: [],
          checkFailedRepoUrls: [githubUrl],
          probeError: '检查 OAuth 绑定状态超时，请确认服务可用后重试',
          probeTraceId: 'e0a7ec3d-133a-4ff9-aa9d-15780e06d1eb',
        },
      )
      expect(next.allBound).toBe(true)
      expect(next.startBlocked).toBe(true)
      expect(next.unboundRepoUrls).toEqual([])
      expect(next.checkFailedRepoUrls).toEqual([githubUrl])
      expect(next.probeError).toBe('检查 OAuth 绑定状态超时，请确认服务可用后重试')
      expect(next.probeTraceId).toBe('e0a7ec3d-133a-4ff9-aa9d-15780e06d1eb')
    })
  })

  describe('shared Git OAuth bind vs PR merge CTA', () => {
    const bound = { loading: false, allBound: true, unboundRepoUrls: [] }
    const unbound = { loading: false, allBound: false, unboundRepoUrls: [gitlabUrl] }

    it('treats finished allBound with empty unbound list as shared bound', () => {
      expect(sharedUserGitOAuthIsBound(bound)).toBe(true)
      expect(sharedUserGitOAuthIsBound(null)).toBe(false)
      expect(sharedUserGitOAuthIsBound({ loading: true, allBound: true, unboundRepoUrls: [] })).toBe(false)
      expect(sharedUserGitOAuthIsBound(unbound)).toBe(false)
    })

    it('hides PR 去绑定 and oauth-expired error when the shared token is bound', () => {
      const expired = 'Git OAuth 授权已失效，请重新绑定后再合并'
      expect(shouldShowGitPrOauthBind(expired, bound)).toBe(false)
      expect(gitPrOauthStatusErrorText(expired, bound)).toBe('')
      expect(shouldShowGitPrOauthBind(expired, unbound)).toBe(true)
      expect(gitPrOauthStatusErrorText(expired, unbound)).toBe(expired)
    })
  })
}

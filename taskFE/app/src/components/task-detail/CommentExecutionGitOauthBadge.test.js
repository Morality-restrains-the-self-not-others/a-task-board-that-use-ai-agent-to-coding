// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] CommentExecutionGitOauthBadge.test.js requires vitest runtime')
} else {
  const { mount } = await import('@vue/test-utils')
  const { describe, expect, it } = await import('vitest')
  const { default: CommentExecutionGitOauthBadge } = await import('./CommentExecutionGitOauthBadge.vue')

  const githubUrl = 'https://github.com/ruandao/helloworld.git'
  const gitlabUrl = 'http://gitlab.daydaymoney.com/ljy124/repo.git'
  const boundReadiness = {
    loading: false,
    hasOAuthRepos: true,
    allBound: true,
    unboundRepoUrls: [],
  }

  describe('CommentExecutionGitOauthBadge', () => {
    it('shows green 已绑定 when OAuth is bound and there is no push error', () => {
      const wrapper = mount(CommentExecutionGitOauthBadge, {
        props: {
          repoIdentities: [{ repo_url: githubUrl }],
          oauthReadiness: boundReadiness,
        },
      })
      const chip = wrapper.get('[data-testid="comment-execution-git-oauth"]')
      expect(chip.text()).toBe('Git OAuth · 已绑定')
      expect(chip.attributes('data-kind')).toBe('bound')
      expect(wrapper.find('[data-testid="comment-execution-git-oauth-bind"]').exists()).toBe(false)
    })

    it('shows 网络不可达 without a bind href when GitLab is unreachable', () => {
      const wrapper = mount(CommentExecutionGitOauthBadge, {
        props: {
          repoIdentities: [{ repo_url: gitlabUrl }],
          oauthReadiness: {
            ...boundReadiness,
            unreachableRepoUrls: [gitlabUrl],
          },
        },
      })
      const chip = wrapper.get('[data-testid="comment-execution-git-oauth"]')
      expect(chip.text()).toBe('Git OAuth · 网络不可达')
      expect(chip.attributes('data-kind')).toBe('unreachable')
      expect(wrapper.find('[data-testid="comment-execution-git-oauth-bind"]').exists()).toBe(false)
    })

    it('shows 检查超时 chip with data-traceId and no bind href when probe timed out', () => {
      const wrapper = mount(CommentExecutionGitOauthBadge, {
        props: {
          repoIdentities: [{ repo_url: githubUrl }],
          oauthReadiness: {
            ...boundReadiness,
            checkFailedRepoUrls: [githubUrl],
            probeError: '检查 OAuth 绑定状态超时，请确认服务可用后重试',
            probeTraceId: 'e0a7ec3d-133a-4ff9-aa9d-15780e06d1eb',
          },
        },
      })
      const chip = wrapper.get('[data-testid="comment-execution-git-oauth"]')
      expect(chip.text()).toBe('Git OAuth · 检查超时')
      expect(chip.attributes('data-kind')).toBe('check_failed')
      expect(chip.attributes('data-traceid') || chip.attributes('data-traceId')).toBe(
        'e0a7ec3d-133a-4ff9-aa9d-15780e06d1eb',
      )
      expect(wrapper.find('[data-testid="comment-execution-git-oauth-bind"]').exists()).toBe(false)
    })

    it('shows a real retry button for check_failed and emits retry-oauth-probe on click (OPT-20260902-011)', async () => {
      const wrapper = mount(CommentExecutionGitOauthBadge, {
        props: {
          repoIdentities: [{ repo_url: githubUrl }],
          oauthReadiness: {
            ...boundReadiness,
            checkFailedRepoUrls: [githubUrl],
            probeError: '检查 OAuth 绑定状态超时，请确认服务可用后重试',
            probeTraceId: 'e0a7ec3d-133a-4ff9-aa9d-15780e06d1eb',
          },
        },
      })
      const chip = wrapper.get('[data-testid="comment-execution-git-oauth"]')
      expect(chip.attributes('data-kind')).toBe('check_failed')
      const retry = wrapper.get('[data-testid="comment-execution-git-oauth-retry"]')
      expect(retry.element.tagName).toBe('BUTTON')
      expect(retry.element.type).toBe('button')
      expect(retry.text()).toContain('重试')
      await retry.trigger('click')
      const events = wrapper.emitted('retry-oauth-probe') || []
      expect(events.length).toBe(1)
      expect(events[0][0]).toEqual([githubUrl])
    })

    it('does not show a retry button when check_failed belongs to another repo (OPT-20260902-011)', () => {
      const wrapper = mount(CommentExecutionGitOauthBadge, {
        props: {
          repoIdentities: [{ repo_url: githubUrl }],
          oauthReadiness: {
            ...boundReadiness,
            checkFailedRepoUrls: [gitlabUrl],
            probeError: '检查 OAuth 绑定状态超时，请确认服务可用后重试',
          },
        },
      })
      const chip = wrapper.get('[data-testid="comment-execution-git-oauth"]')
      // 本评论的 url 未失败 → summary 仍 bound，不显示重试
      expect(chip.attributes('data-kind')).toBe('bound')
      expect(wrapper.find('[data-testid="comment-execution-git-oauth-retry"]').exists()).toBe(false)
    })

    it('shows 无写权限 and a real re-authorize href when last push was permission denied', () => {
      const wrapper = mount(CommentExecutionGitOauthBadge, {
        props: {
          repoIdentities: [{ repo_url: githubUrl }],
          oauthReadiness: boundReadiness,
          lastPushError: 'remote: Permission to ruandao/helloworld.git denied to alice.',
        },
      })
      const chip = wrapper.get('[data-testid="comment-execution-git-oauth"]')
      expect(chip.text()).toBe('Git OAuth · 无写权限')
      expect(chip.attributes('data-kind')).toBe('bound_no_write')
      expect(chip.attributes('title')).toContain('Permission to ruandao/helloworld.git denied')
      const bind = wrapper.get('[data-testid="comment-execution-git-oauth-bind"]')
      expect(bind.element.tagName).toBe('A')
      expect(bind.text()).toContain('换账号授权')
      expect(bind.attributes('href')).toContain('/api/git-oauth/github-start-from-gateway/')
      expect(bind.attributes('href')).toContain(encodeURIComponent(githubUrl))
    })
  })
}

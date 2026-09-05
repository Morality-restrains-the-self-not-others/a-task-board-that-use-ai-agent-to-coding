// @ts-check
// @vitest-environment jsdom
/**
 * pre-commit 会以 node 直跑已暂存 *.test.js，非 Vitest 环境下跳过。
 */
if (!process.env.VITEST) {
  console.log('[skip] ProjectDetailGitReposSection.oauth-row.test.js requires vitest runtime')
} else {
  const { describe, expect, it, vi } = await import('vitest')
  const Mock = vi.hoisted(() => {
    const { ref } = require('vue')
    const PATH_A_URL = 'http://115.29.110.74/example-user/somanyad.git'
    return {
      PATH_A_URL,
      statusByUrl: { [PATH_A_URL]: 'token_error' },
      oauthErrorByUrl: {},
      oauthErrorTraceIdByUrl: {},
      nestedReposRef: ref([]),
      nestedLoadingRef: ref(false),
      nestedErrorRef: ref(''),
      nestedErrorTraceIdRef: ref(''),
      refreshGitReposOAuthStatus: vi.fn(async () => {}),
    }
  })

  const { mount, flushPromises } = await import('@vue/test-utils')

  vi.mock('vue-router', () => ({
    useRoute: () => ({ params: { tenant: 't1', id: 'p1' }, query: {} }),
    useRouter: () => ({ push: vi.fn(), replace: vi.fn() }),
  }))

  vi.mock('../utils/gitSiteOAuthCallbackUtils.js', () => ({
    applyOAuthCallbackFromRoute: vi.fn(async () => null),
  }))

  vi.mock('../composables/useProjectDetailGitRepos.js', () => {
    const { ref } = require('vue')
    return {
      useProjectDetailGitRepos: () => ({
        router: { push: vi.fn(), replace: vi.fn() },
        projectGitRepoList: ref([Mock.PATH_A_URL]),
        projectGitRepoEntries: ref([{ url: Mock.PATH_A_URL, cloneAlias: '' }]),
        branchPreviewLoading: ref(false),
        branchPreviewError: ref(''),
        repoBranchPreviews: ref([]),
        fetchProjectRepoBranchesPreview: vi.fn(),
        bootstrapGitOAuthCatalog: vi.fn(async () => {}),
        applyGitReposStatusFromApi: vi.fn(),
        refreshGitReposOAuthStatus: Mock.refreshGitReposOAuthStatus,
        isRepoOAuthActionDisabled: () => false,
        repoOAuthButtonLabel: () => '重试',
        shouldShowRepoOAuthButton: () => true,
        repoOAuthErrorByUrl: (url) => Mock.oauthErrorByUrl[url] || '',
        repoOAuthErrorTraceIdByUrl: (url) => Mock.oauthErrorTraceIdByUrl[url] || '',
        repoOAuthTokenStatus: (url) => Mock.statusByUrl[url] || '',
        repoOAuthStatusLabel: () => '授权异常',
        repoOAuthStatusBadgeClass: () => 'bg-red-50 text-red-700 border-red-200',
        startRepoOAuthConnect: vi.fn(),
      }),
    }
  })

  vi.mock('../composables/useProjectNestedGitRepos.js', () => ({
    useProjectNestedGitRepos: () => ({
      nestedRepos: Mock.nestedReposRef,
      nestedLoading: Mock.nestedLoadingRef,
      nestedError: Mock.nestedErrorRef,
      nestedErrorTraceId: Mock.nestedErrorTraceIdRef,
      fetchNestedGitRepos: vi.fn(),
    }),
  }))

  const { default: ProjectDetailGitReposSection } = await import('./ProjectDetailGitReposSection.vue')
  const PATH_A_URL = Mock.PATH_A_URL

  describe('ProjectDetailGitReposSection oauth row layout', () => {
    it('不把授权状态、仓库 URL、重试挤在同一行 innerText 里', async () => {
      const wrapper = mount(ProjectDetailGitReposSection, {
        props: {
          project: {
            id: 'p1',
            company: '877397588196749312',
            git_repos: [PATH_A_URL],
            git_repo_entries: [{ url: PATH_A_URL }],
            git_repos_status: [{ repo_url: PATH_A_URL, token_status: 'token_error' }],
          },
        },
      })
      await flushPromises()

      const row = wrapper.get('[data-testid="project-detail-git-repo-row"]')
      const statusRow = row.get('.flex.flex-wrap')
      expect(statusRow.get('[data-testid="git-repo-oauth-status-0"]').text()).toBe('授权异常')
      expect(statusRow.get('[data-testid="git-repo-oauth-action-0"]').text()).toBe('重试')
      expect(statusRow.text()).not.toContain(PATH_A_URL)
      expect(row.get('[data-testid="git-repo-url-link"]').text()).toBe(PATH_A_URL)
      expect(row.get('[data-testid="git-repo-oauth-hint-0"]').text()).toContain('重新绑定')
      wrapper.unmount()
    })

    it('批量校验失败错误节点带 git-repo-validate-error 与 data-traceId', async () => {
      Mock.oauthErrorByUrl[PATH_A_URL] = 'urls 不能为空'
      Mock.oauthErrorTraceIdByUrl[PATH_A_URL] = 'trace-http-detail-batch'
      const wrapper = mount(ProjectDetailGitReposSection, {
        props: {
          project: {
            id: 'p1',
            company: '877397588196749312',
            git_repos: [PATH_A_URL],
            git_repo_entries: [{ url: PATH_A_URL }],
            git_repos_status: [{ repo_url: PATH_A_URL, token_status: 'not_bound' }],
          },
        },
      })
      await flushPromises()
      const err = wrapper.get('[data-testid="git-repo-validate-error"]')
      expect(err.text()).toBe('urls 不能为空')
      expect(err.attributes('data-traceid')).toBe('trace-http-detail-batch')
      expect(wrapper.find('[data-testid="git-repo-oauth-hint-0"]').exists()).toBe(false)
      wrapper.unmount()
      delete Mock.oauthErrorByUrl[PATH_A_URL]
      delete Mock.oauthErrorTraceIdByUrl[PATH_A_URL]
    })

    it('OAuth 回调用 severity=success 才回流刷新，不用 outcome', async () => {
      const { applyOAuthCallbackFromRoute } = await import('../utils/gitSiteOAuthCallbackUtils.js')
      const project = {
        id: 'p1',
        company: '877397588196749312',
        git_repos: [PATH_A_URL],
        git_repo_entries: [{ url: PATH_A_URL }],
        git_repos_status: [{ repo_url: PATH_A_URL, token_status: 'token_error' }],
      }

      applyOAuthCallbackFromRoute.mockResolvedValueOnce({ outcome: 'success', code: 'ok' })
      Mock.refreshGitReposOAuthStatus.mockClear()
      const missed = mount(ProjectDetailGitReposSection, { props: { project } })
      await flushPromises()
      expect(missed.emitted('oauth-callback-success')).toBeFalsy()
      expect(Mock.refreshGitReposOAuthStatus.mock.calls.some((c) => c[1]?.force === true)).toBe(false)
      missed.unmount()

      applyOAuthCallbackFromRoute.mockResolvedValueOnce({ severity: 'success', code: 'ok' })
      Mock.refreshGitReposOAuthStatus.mockClear()
      const hit = mount(ProjectDetailGitReposSection, { props: { project } })
      await flushPromises()
      expect(hit.emitted('oauth-callback-success')).toBeTruthy()
      expect(Mock.refreshGitReposOAuthStatus).toHaveBeenCalledWith(expect.anything(), { force: true })
      hit.unmount()
    })
  })
}

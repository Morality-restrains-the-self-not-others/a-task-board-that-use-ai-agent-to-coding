// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] ProjectDetailGitReposSection.nested-badge.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { describe, expect, it, vi } = await import('vitest')
  vi.mock('vue-router', () => ({
    useRoute: () => ({ params: { tenant: 't1', id: 'p1' }, query: {} }),
    useRouter: () => ({ push: vi.fn(), replace: vi.fn() }),
  }))

  vi.mock('../utils/gitSiteOAuthCallbackUtils.js', () => ({
    applyOAuthCallbackFromRoute: vi.fn(async () => null),
  }))

  // vi.hoisted so mock factories (also hoisted) can reference it
  const { makeRef } = vi.hoisted(() => ({
    makeRef: (val) => ({ __v_isRef: true, value: val }),
  }))

  vi.mock('../composables/useProjectDetailGitRepos.js', () => ({
    useProjectDetailGitRepos: () => ({
      router: { push: vi.fn(), replace: vi.fn() },
      projectGitRepoList: makeRef(['https://gitlab.daydaymoney.com/g/parent.git']),
      projectGitRepoEntries: makeRef([
        { url: 'https://gitlab.daydaymoney.com/g/parent.git', cloneAlias: '' },
      ]),
      branchPreviewLoading: makeRef(false),
      branchPreviewError: makeRef(''),
      repoBranchPreviews: makeRef([]),
      fetchProjectRepoBranchesPreview: vi.fn(),
      bootstrapGitOAuthCatalog: vi.fn(async () => {}),
      applyGitReposStatusFromApi: vi.fn(),
      refreshGitReposOAuthStatus: vi.fn(async () => {}),
      isRepoOAuthActionDisabled: () => true,
      repoOAuthButtonLabel: () => '',
      shouldShowRepoOAuthButton: () => false,
      repoOAuthErrorByUrl: () => '',
      repoOAuthErrorTraceIdByUrl: () => '',
      repoOAuthTokenStatus: () => 'token_available',
      repoOAuthStatusLabel: () => '已授权',
      repoOAuthStatusBadgeClass: () => 'border-gray-200 bg-gray-50 text-gray-700',
      startRepoOAuthConnect: vi.fn(),
    }),
  }))

  vi.mock('../composables/useProjectNestedGitRepos.js', () => ({
    useProjectNestedGitRepos: () => ({
      nestedRepos: makeRef([
        {
          path: 'DaydaymoneyGrafana',
          url: 'https://gitlab.daydaymoney.com/g/DaydaymoneyGrafana.git',
          source: 'gitmodules',
        },
        {
          path: 'vendor/lib',
          url: 'https://gitlab.daydaymoney.com/g/lib.git',
          source: 'gitmodules',
        },
      ]),
      nestedLoading: makeRef(false),
      nestedError: makeRef(''),
      nestedErrorTraceId: makeRef(''),
      fetchNestedGitRepos: vi.fn(),
    }),
  }))

  const { default: ProjectDetailGitReposSection } = await import('./ProjectDetailGitReposSection.vue')

  describe('ProjectDetailGitReposSection nested source badge', () => {
    it('对 gitmodules 来源显示 submodule 徽章，不逐行重复「嵌套仓」', async () => {
      const wrapper = mount(ProjectDetailGitReposSection, {
        props: {
          project: {
            id: 'p1',
            git_repos: ['https://gitlab.daydaymoney.com/g/parent.git'],
            git_repo_entries: [{ url: 'https://gitlab.daydaymoney.com/g/parent.git' }],
            git_repos_status: [],
          },
        },
      })
      await flushPromises()

      const nestedSection = wrapper.get('[data-testid="project-detail-nested-git-repos"]')
      const rows = nestedSection.findAll('[data-testid="project-detail-nested-git-repo-row"]')
      expect(rows).toHaveLength(2)

      const badges = nestedSection.findAll(
        '[data-testid="project-detail-nested-git-repo-submodule-badge"]',
      )
      expect(badges).toHaveLength(2)
      expect(badges[0].text()).toBe('submodule')
      expect(badges[1].text()).toBe('submodule')

      expect(nestedSection.text()).not.toContain('嵌套仓')
      expect(rows[0].text()).toContain('DaydaymoneyGrafana')
      expect(rows[0].find('[data-testid="project-detail-nested-git-repo-submodule-badge"]').exists()).toBe(
        true,
      )
      expect(rows[1].find('[data-testid="project-detail-nested-git-repo-submodule-badge"]').exists()).toBe(
        true,
      )

      wrapper.unmount()
    })
  })
}

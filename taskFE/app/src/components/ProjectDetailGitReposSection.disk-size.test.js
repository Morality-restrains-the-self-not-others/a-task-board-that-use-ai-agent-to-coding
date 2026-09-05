// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] ProjectDetailGitReposSection.disk-size.test.js requires vitest runtime')
} else {
const { describe, expect, it, vi } = await import('vitest')
const { mount, flushPromises } = await import('@vue/test-utils')
// vi.mock 工厂中引用的变量必须通过 vi.hoisted() 定义（Vitest 会把工厂 hoist 到模块顶部）。
const { ref } = vi.hoisted(() => ({ ref: require('vue').ref }))

vi.mock('vue-router', () => ({
  useRoute: () => ({ params: { tenant: 't1', id: 'p1' }, query: {} }),
  useRouter: () => ({ push: vi.fn(), replace: vi.fn() }),
}))

vi.mock('../utils/gitSiteOAuthCallbackUtils.js', () => ({
  applyOAuthCallbackFromRoute: vi.fn(async () => null),
}))

vi.mock('../composables/useProjectDetailGitRepos.js', () => ({
  useProjectDetailGitRepos: () => ({
    router: { push: vi.fn(), replace: vi.fn() },
    projectGitRepoList: ref([
      'https://gitlab.daydaymoney.com/g/internal.git',
      'https://github.com/o/external.git',
    ]),
    projectGitRepoEntries: ref([
      {
        url: 'https://gitlab.daydaymoney.com/g/internal.git',
        cloneAlias: '',
        isInternal: true,
        diskSizeBytes: 2048,
      },
      {
        url: 'https://github.com/o/external.git',
        cloneAlias: '',
        isInternal: false,
        diskSizeBytes: null,
      },
    ]),
    branchPreviewLoading: ref(false),
    branchPreviewError: ref(''),
    repoBranchPreviews: ref([]),
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
    nestedRepos: ref([]),
    nestedLoading: ref(false),
    nestedError: ref(''),
    nestedErrorTraceId: ref(''),
    fetchNestedGitRepos: vi.fn(),
  }),
}))

const { default: ProjectDetailGitReposSection } = await import('./ProjectDetailGitReposSection.vue')

describe('ProjectDetailGitReposSection disk size badge', () => {
  // covers useProjectDetailGitRepos diskSizeBytes mapping
  it('仅内部仓显示磁盘占用，外部仓不渲染节点', async () => {
    const wrapper = mount(ProjectDetailGitReposSection, {
      props: {
        project: {
          id: 'p1',
          git_repos: [
            'https://gitlab.daydaymoney.com/g/internal.git',
            'https://github.com/o/external.git',
          ],
          git_repo_entries: [
            {
              url: 'https://gitlab.daydaymoney.com/g/internal.git',
              is_internal: true,
              disk_size_bytes: 2048,
            },
            { url: 'https://github.com/o/external.git', is_internal: false },
          ],
          git_repos_status: [],
        },
      },
    })
    await flushPromises()

    const badges = wrapper.findAll('[data-testid="git-repo-disk-size"]')
    expect(badges).toHaveLength(1)
    expect(badges[0].text()).toContain('磁盘：')
    expect(badges[0].text()).toContain('2 KB')

    const rows = wrapper.findAll('[data-testid="project-detail-git-repo-row"]')
    expect(rows).toHaveLength(2)
    expect(rows[1].find('[data-testid="git-repo-disk-size"]').exists()).toBe(false)
  })
})

}

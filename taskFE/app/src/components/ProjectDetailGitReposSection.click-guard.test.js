// @vitest-environment jsdom
// OPT-20260819-038 回归：自动克隆开关切换 PUT 携带 Idempotency-Key 头。
if (!process.env.VITEST) {
  console.log('[skip] ProjectDetailGitReposSection.click-guard.test.js requires vitest runtime')
} else {
  const { describe, expect, it, vi, beforeEach } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')

  const mocks = vi.hoisted(() => ({
    apiFetch: vi.fn(),
  }))

  vi.mock('vue-router', () => ({
    useRoute: () => ({ params: { tenant: 't1', id: 'p1' }, query: {} }),
    useRouter: () => ({ push: vi.fn(), replace: vi.fn() }),
  }))
  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch: (...args) => mocks.apiFetch(...args),
    clearCachedAuthToken: () => {},
  }))
  vi.mock('../utils/gitSiteOAuthCallbackUtils.js', () => ({
    applyOAuthCallbackFromRoute: vi.fn(async () => null),
  }))
  vi.mock('../composables/useProjectDetailGitRepos.js', () => {
    const { ref } = require('vue')
    return {
      useProjectDetailGitRepos: () => ({
        router: { push: vi.fn(), replace: vi.fn() },
        projectGitRepoList: ref(['https://gitlab.daydaymoney.com/g/parent.git']),
        projectGitRepoEntries: ref([{ url: 'https://gitlab.daydaymoney.com/g/parent.git', cloneAlias: '' }]),
        branchPreviewLoading: ref(false),
        branchPreviewError: ref(''),
        repoBranchPreviews: ref([]),
        fetchProjectRepoBranchesPreview: vi.fn(),
        bootstrapGitOAuthCatalog: vi.fn(async () => {}),
        applyGitReposStatusFromApi: vi.fn(),
        refreshGitReposOAuthStatus: vi.fn(),
        isRepoOAuthActionDisabled: () => false,
        repoOAuthButtonLabel: () => 'OAuth 授权',
        shouldShowRepoOAuthButton: () => false,
        repoOAuthErrorByUrl: () => '',
        repoOAuthErrorTraceIdByUrl: () => '',
        repoOAuthTokenStatus: () => '',
        repoOAuthStatusLabel: () => '已授权',
        repoOAuthStatusBadgeClass: () => 'border-gray-200 bg-gray-50 text-gray-700',
        startRepoOAuthConnect: vi.fn(),
      }),
    }
  })
  vi.mock('../composables/useProjectNestedGitRepos.js', () => {
    const { ref } = require('vue')
    return {
      useProjectNestedGitRepos: () => ({
        nestedRepos: ref([]),
        nestedLoading: ref(false),
        nestedError: ref(''),
        nestedErrorTraceId: ref(''),
        fetchNestedGitRepos: vi.fn(),
      }),
    }
  })
  // OPT-20260819-038: 让 guard 直接放行并把幂等键透传给 mergeIdempotencyHeaders，
  // 断言 PUT 携带 Idempotency-Key 头。
  vi.mock('../utils/clickGuard.js', () => ({
    createClickGuard: () => ({
      run: async (fn) => fn({ idempotencyKey: 'ik-test-autoclone' }),
      isBusy: () => false,
    }),
    mergeIdempotencyHeaders: (headers, key) => ({ ...(headers || {}), 'Idempotency-Key': key }),
  }))

  const { default: ProjectDetailGitReposSection } = await import('./ProjectDetailGitReposSection.vue')

  beforeEach(() => {
    vi.clearAllMocks()
    mocks.apiFetch.mockImplementation(async (url, opts) => {
      if (opts?.method === 'PUT') {
        return { ok: true, json: async () => ({ auto_clone_nested_repos: false }) }
      }
      return { ok: true, json: async () => ({}) }
    })
  })

  describe('ProjectDetailGitReposSection 写操作 clickGuard 接线', () => {
    it('自动克隆切换 PUT 携带 Idempotency-Key', async () => {
      const wrapper = mount(ProjectDetailGitReposSection, {
        props: {
          project: {
            id: 'p1',
            auto_clone_nested_repos: true,
            git_repos: [],
            git_repo_entries: [],
            git_repos_status: [],
          },
        },
      })
      await flushPromises()

      const toggle = wrapper.find('[data-testid="project-detail-auto-clone-nested-repos"]')
      expect(toggle.exists()).toBe(true)
      await toggle.setValue(false)
      await flushPromises()

      const putCall = mocks.apiFetch.mock.calls.find(([, o]) => o?.method === 'PUT')
      expect(putCall).toBeTruthy()
      expect(putCall[0]).toBe('/api/projects/tenant_id/t1/p1/')
      expect(putCall[1].headers['Idempotency-Key']).toBe('ik-test-autoclone')
    })
  })
}

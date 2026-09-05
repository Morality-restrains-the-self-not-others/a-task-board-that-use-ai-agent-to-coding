// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] useProjectDetailGitRepos.oauth-traceId.test.js requires vitest runtime')
} else {
const { beforeEach, describe, expect, it, vi } = await import('vitest')
const { ref } = await import('vue')

// vi.mock 工厂引用的变量必须经 vi.hoisted() 定义（Vitest 会把工厂 hoist 到模块顶部）。
const hoisted = vi.hoisted(() => ({
  apiFetchMock: vi.fn(),
  ref: require('vue').ref,
}))
const apiFetchMock = hoisted.apiFetchMock

vi.mock('vue-router', () => ({
  useRoute: () => ({
    params: { tenant: 't1', id: 'p1' },
    path: '/tenant/t1/projects/p1/',
    query: {},
  }),
  useRouter: () => ({ push: vi.fn(), replace: vi.fn() }),
}))

vi.mock('../utils/apiUtils.js', () => ({
  apiFetch: (...args) => hoisted.apiFetchMock(...args),
}))

vi.mock('./useProjectGitOAuthCatalog.js', () => ({
  useProjectGitOAuthCatalog: () => ({
    touchRepoOAuthButtons: vi.fn(),
    bootstrapGitOAuthCatalog: vi.fn(async () => {}),
  }),
}))

vi.mock('./useProjectRepoBranchPreview.js', () => ({
  useProjectRepoBranchPreview: () => ({
    branchPreviewLoading: hoisted.ref(false),
    branchPreviewError: hoisted.ref(''),
    repoBranchPreviews: hoisted.ref([]),
    fetchProjectRepoBranchesPreview: vi.fn(),
  }),
}))

vi.mock('../utils/repoOAuthAuthorizeUtils.js', () => ({
  resolveRepoOAuthAuthorizeUrl: () => '/api/git-oauth/github-start-from-gateway/',
  resolveRepoOAuthProviderInfo: () => ({ provider: 'github', service_provider: 'default' }),
}))

vi.mock('../utils/githubAppReturnStorage.js', () => ({
  createGithubAppReturnKey: () => 'rk',
  setGithubAppReturnTarget: vi.fn(),
}))

const { useProjectDetailGitRepos } = await import('./useProjectDetailGitRepos.js')

describe('useProjectDetailGitRepos OAuth start error data-traceId', () => {
  beforeEach(() => {
    apiFetchMock.mockReset()
  })

  it('stores response.traceId when start-from-gateway fails without authorize_url', async () => {
    apiFetchMock.mockResolvedValue({
      ok: true,
      traceId: 'tid-oauth-start-1',
      json: async () => ({}),
    })

    const project = ref({ git_repos: [], git_repo_entries: [], git_repos_status: [] })
    const { startRepoOAuthConnect, repoOAuthErrorByUrl, repoOAuthErrorTraceIdByUrl } =
      useProjectDetailGitRepos({ project })

    const repoUrl = 'https://github.com/acme/demo.git'
    await startRepoOAuthConnect(repoUrl)

    expect(repoOAuthErrorByUrl(repoUrl)).toBe('无法启动 OAuth 授权')
    expect(repoOAuthErrorTraceIdByUrl(repoUrl)).toBe('tid-oauth-start-1')
  })

  it('stores X-Trace-Id from failed JSON body/response when detail present', async () => {
    apiFetchMock.mockResolvedValue({
      ok: false,
      status: 503,
      traceId: 'tid-oauth-start-503',
      json: async () => ({ detail: 'upstream unavailable', trace_id: 'tid-from-body' }),
    })

    const project = ref({ git_repos: [], git_repo_entries: [], git_repos_status: [] })
    const { startRepoOAuthConnect, repoOAuthErrorByUrl, repoOAuthErrorTraceIdByUrl } =
      useProjectDetailGitRepos({ project })

    const repoUrl = 'https://github.com/acme/demo.git'
    await startRepoOAuthConnect(repoUrl)

    expect(repoOAuthErrorByUrl(repoUrl)).toBe('upstream unavailable')
    // response.traceId 优先（apiFetch 已按头/体解析）
    expect(repoOAuthErrorTraceIdByUrl(repoUrl)).toBe('tid-oauth-start-503')
  })

  it('stores error.traceId on timeout/network failure', async () => {
    const err = new Error('network down')
    err.name = 'TypeError'
    err.traceId = 'tid-oauth-net'
    apiFetchMock.mockRejectedValue(err)

    const project = ref({ git_repos: [], git_repo_entries: [], git_repos_status: [] })
    const { startRepoOAuthConnect, repoOAuthErrorByUrl, repoOAuthErrorTraceIdByUrl } =
      useProjectDetailGitRepos({ project })

    const repoUrl = 'https://github.com/acme/demo.git'
    await startRepoOAuthConnect(repoUrl)

    expect(repoOAuthErrorByUrl(repoUrl)).toBe('network down')
    expect(repoOAuthErrorTraceIdByUrl(repoUrl)).toBe('tid-oauth-net')
  })

  it('clears traceId when starting a new authorize attempt', async () => {
    apiFetchMock
      .mockResolvedValueOnce({
        ok: false,
        status: 502,
        traceId: 'tid-old',
        json: async () => ({}),
      })
      .mockResolvedValueOnce({
        ok: false,
        status: 502,
        json: async () => ({}),
      })

    const project = ref({ git_repos: [], git_repo_entries: [], git_repos_status: [] })
    const { startRepoOAuthConnect, repoOAuthErrorTraceIdByUrl } = useProjectDetailGitRepos({
      project,
    })

    const repoUrl = 'https://github.com/acme/demo.git'
    await startRepoOAuthConnect(repoUrl)
    expect(repoOAuthErrorTraceIdByUrl(repoUrl)).toBe('tid-old')

    await startRepoOAuthConnect(repoUrl)
    expect(repoOAuthErrorTraceIdByUrl(repoUrl)).toBe('')
  })
})
}

// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] useProjectDetailGitRepos.batch.test.js requires vitest runtime')
} else {
const { beforeEach, describe, expect, it, vi } = await import('vitest')
const { ref } = await import('vue')
// vi.mock 工厂中引用的变量必须通过 vi.hoisted() 定义（Vitest 会把工厂 hoist 到模块顶部）。
const hoisted = vi.hoisted(() => ({
  apiFetchMock: vi.fn(),
  ref: require('vue').ref,
  catalogCompanyId: '',
}))
// 测试体直接以模块内别名引用同一批 mock
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
  useProjectGitOAuthCatalog: (_apiFetch, getCompanyId) => {
    hoisted.catalogCompanyId =
      typeof getCompanyId === 'function' ? String(getCompanyId() || '') : ''
    return {
      touchRepoOAuthButtons: vi.fn(),
      bootstrapGitOAuthCatalog: vi.fn(async () => {}),
    }
  },
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
  resolveRepoOAuthAuthorizeUrl: () => '/api/oauth/start/',
  resolveRepoOAuthProviderInfo: (url) => {
    if (String(url || '').includes('gitlab') || String(url || '').includes('github')) {
      return { provider: 'gitlab', service_provider: 'default' }
    }
    return null
  },
}))

vi.mock('../utils/githubAppReturnStorage.js', () => ({
  createGithubAppReturnKey: () => 'rk',
  setGithubAppReturnTarget: vi.fn(),
}))

const { useProjectDetailGitRepos } = await import('./useProjectDetailGitRepos.js')

describe('useProjectDetailGitRepos batch oauth status', () => {
  beforeEach(() => {
    apiFetchMock.mockReset()
  })

  it('refreshGitReposOAuthStatus uses validate-git-repos batch POST', async () => {
    apiFetchMock.mockResolvedValue({
      ok: true,
      json: async () => ({
        results: [
          {
            url: 'https://gitlab.daydaymoney.com/g/docs.git',
            token_status: 'not_bound',
          },
          {
            url: 'https://gitlab.daydaymoney.com/g/DaydaymoneyGrafana.git',
            token_status: 'token_available',
          },
        ],
      }),
    })

    const project = ref({
      git_repos: [],
      git_repo_entries: [],
      git_repos_status: [],
    })
    const { refreshGitReposOAuthStatus, repoOAuthTokenStatus, repoOAuthStatusLabel } =
      useProjectDetailGitRepos({ project })
    expect(hoisted.catalogCompanyId).toBe('t1')

    await refreshGitReposOAuthStatus([
      'https://gitlab.daydaymoney.com/g/DaydaymoneyGrafana.git',
      'https://gitlab.daydaymoney.com/g/docs.git',
    ])

    expect(apiFetchMock).toHaveBeenCalledTimes(1)
    const [url, opts] = apiFetchMock.mock.calls[0]
    expect(url).toContain('/projects/validate-git-repos/')
    expect(opts.method).toBe('POST')
    const body = JSON.parse(opts.body)
    expect(body.urls).toHaveLength(2)
    expect(body.probe_access).toBe(true)
    expect(opts.timeout).toBe(60000)
    expect(repoOAuthTokenStatus('https://gitlab.daydaymoney.com/g/docs.git')).toBe('not_bound')
    expect(repoOAuthStatusLabel('https://gitlab.daydaymoney.com/g/docs.git')).toBe('需要授权')
    expect(repoOAuthStatusLabel('https://gitlab.daydaymoney.com/g/DaydaymoneyGrafana.git')).toBe('已授权')
  })

  it('他人仓不可访问时 git_repos_status=token_error 不得显示已授权', () => {
    const repoUrl = 'https://github.com/test-ruandao/helloworld.git'
    const project = ref({
      git_repos: [repoUrl],
      git_repo_entries: [{ url: repoUrl }],
      git_repos_status: [],
    })
    const { applyGitReposStatusFromApi, repoOAuthStatusLabel, repoOAuthTokenStatus } =
      useProjectDetailGitRepos({ project })
    applyGitReposStatusFromApi([{ repo_url: repoUrl, token_status: 'token_error' }])
    expect(repoOAuthTokenStatus(repoUrl)).toBe('token_error')
    expect(repoOAuthStatusLabel(repoUrl)).toBe('授权异常')
    expect(repoOAuthStatusLabel(repoUrl)).not.toBe('已授权')
  })

  it('OAuth catalog company prefers project.company over route tenant', () => {
    const project = ref({
      company: '877397588196749312',
      git_repos: [],
      git_repo_entries: [],
      git_repos_status: [],
    })
    useProjectDetailGitRepos({ project })
    expect(hoisted.catalogCompanyId).toBe('877397588196749312')
  })

  it('force 刷新会重新校验已是 token_error 的仓', async () => {
    const repoUrl = 'https://gitlab.daydaymoney.com/g/docs.git'
    apiFetchMock.mockResolvedValue({
      ok: true,
      json: async () => ({ results: [{ url: repoUrl, token_status: 'token_available' }] }),
    })
    const project = ref({
      git_repos: [repoUrl],
      git_repo_entries: [{ url: repoUrl }],
    })
    const { applyGitReposStatusFromApi, refreshGitReposOAuthStatus, repoOAuthTokenStatus } =
      useProjectDetailGitRepos({ project })
    applyGitReposStatusFromApi([{ repo_url: repoUrl, token_status: 'token_error' }])
    expect(repoOAuthTokenStatus(repoUrl)).toBe('token_error')

    await refreshGitReposOAuthStatus([repoUrl])
    expect(apiFetchMock).not.toHaveBeenCalled()

    await refreshGitReposOAuthStatus([repoUrl], { force: true })
    expect(apiFetchMock).toHaveBeenCalledTimes(1)
    expect(repoOAuthTokenStatus(repoUrl)).toBe('token_available')
  })

  it('HTTP 失败展示服务端文案和 data-traceId，而不是一律 token_error', async () => {
    const { VALIDATE_GIT_REPO_NETWORK_MSG } = await import('../utils/gitRepoValidateError.js')
    const repoUrl = 'https://gitlab.daydaymoney.com/g/docs.git'
    apiFetchMock.mockResolvedValue({
      ok: false,
      status: 400,
      traceId: 'trace-http-detail-batch',
      json: async () => ({
        error: 'urls 不能为空',
        message: 'urls 不能为空',
        trace_id: 'trace-http-detail-batch',
      }),
    })
    const project = ref({ git_repos: [repoUrl], git_repo_entries: [{ url: repoUrl }] })
    const { refreshGitReposOAuthStatus, repoOAuthErrorByUrl, repoOAuthErrorTraceIdByUrl, repoOAuthTokenStatus } =
      useProjectDetailGitRepos({ project })

    await refreshGitReposOAuthStatus([repoUrl], { force: true })

    expect(apiFetchMock).toHaveBeenCalledTimes(1)
    expect(apiFetchMock.mock.calls[0][0]).toContain('/projects/validate-git-repos/')
    expect(repoOAuthErrorByUrl(repoUrl)).toBe('urls 不能为空')
    expect(repoOAuthErrorByUrl(repoUrl)).not.toBe(VALIDATE_GIT_REPO_NETWORK_MSG)
    expect(repoOAuthErrorTraceIdByUrl(repoUrl)).toBe('trace-http-detail-batch')
    expect(repoOAuthTokenStatus(repoUrl)).not.toBe('token_error')
    const [, opts] = apiFetchMock.mock.calls[0]
    expect(opts.timeout).toBe(60000)
  })

  it('请求超时展示超时文案并保留 traceId', async () => {
    const { VALIDATE_GIT_REPO_TIMEOUT_MSG } = await import('../utils/gitRepoValidateError.js')
    const repoUrl = 'https://gitlab.daydaymoney.com/g/docs.git'
    const err = new Error('请求超时（30 秒），请检查网络后重试')
    err.name = 'TimeoutError'
    err.traceId = 'trace-timeout-detail-batch'
    apiFetchMock.mockRejectedValue(err)
    const project = ref({ git_repos: [repoUrl], git_repo_entries: [{ url: repoUrl }] })
    const { refreshGitReposOAuthStatus, repoOAuthErrorByUrl, repoOAuthErrorTraceIdByUrl, repoOAuthTokenStatus } =
      useProjectDetailGitRepos({ project })

    await refreshGitReposOAuthStatus([repoUrl], { force: true })

    expect(apiFetchMock).toHaveBeenCalledTimes(1)
    expect(repoOAuthErrorByUrl(repoUrl)).toBe(VALIDATE_GIT_REPO_TIMEOUT_MSG)
    expect(repoOAuthErrorTraceIdByUrl(repoUrl)).toBe('trace-timeout-detail-batch')
    expect(repoOAuthTokenStatus(repoUrl)).not.toBe('token_error')
  })
})

}

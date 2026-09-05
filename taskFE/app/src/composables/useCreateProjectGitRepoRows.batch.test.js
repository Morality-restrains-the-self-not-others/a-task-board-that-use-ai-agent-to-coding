// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] useCreateProjectGitRepoRows.batch.test.js requires vitest runtime')
} else {
const { afterEach, beforeEach, describe, expect, it, vi } = await import('vitest')

// vi.mock 工厂引用的变量必须经 vi.hoisted() 定义（Vitest 会把工厂 hoist 到模块顶部）。
const hoisted = vi.hoisted(() => ({
  apiFetchMock: vi.fn(),
  catalogCompanyId: '',
}))
const apiFetchMock = hoisted.apiFetchMock

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

vi.mock('../utils/githubAppReturnStorage.js', () => ({
  createGithubAppReturnKey: vi.fn(() => 'key'),
  setGithubAppReturnTarget: vi.fn(),
}))

vi.mock('../utils/gitSiteOAuthCallbackUtils.js', () => ({
  applyOAuthCallbackFromRoute: vi.fn(async () => null),
}))

vi.mock('../utils/repoOAuthAuthorizeUtils.js', () => ({
  resolveRepoOAuthAuthorizeUrl: () => '/api/oauth/start/',
  resolveRepoOAuthProviderInfo: () => ({ provider: 'github', service_provider: 'default' }),
}))

const { useCreateProjectGitRepoRows } = await import('./useCreateProjectGitRepoRows.js')

describe('useCreateProjectGitRepoRows batch validate', () => {
  const route = { params: { tenant: 't1' } }
  const router = { replace: vi.fn() }

  beforeEach(() => {
    apiFetchMock.mockReset()
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('debounces multi-row changes into one validate-git-repos POST with probe_access', async () => {
    apiFetchMock.mockResolvedValue({
      ok: true,
      json: async () => ({
        results: [
          {
            url: 'https://github.com/org/a.git',
            is_accessible: false,
            token_status: 'not_bound',
            message: 'GitHub repository not found',
          },
          {
            url: 'https://github.com/org/b.git',
            is_accessible: true,
            token_status: 'token_available',
            message: '',
          },
        ],
      }),
    })

    const {
      gitRepoRows,
      gitRepoRowErrors,
      REPO_INACCESSIBLE_MSG,
      shouldShowRepoOAuthButton,
    } = useCreateProjectGitRepoRows(route, router)

    gitRepoRows.value = [
      { id: 1, url: 'https://github.com/org/a.git', cloneAlias: '' },
      { id: 2, url: 'https://github.com/org/b.git', cloneAlias: '' },
    ]

    await vi.advanceTimersByTimeAsync(600)
    await Promise.resolve()
    await Promise.resolve()

    expect(apiFetchMock).toHaveBeenCalledTimes(1)
    const [url, opts] = apiFetchMock.mock.calls[0]
    expect(url).toContain('/projects/validate-git-repos/')
    expect(opts.method).toBe('POST')
    const body = JSON.parse(opts.body)
    expect(body.probe_access).toBe(true)
    expect(body.urls).toEqual([
      'https://github.com/org/a.git',
      'https://github.com/org/b.git',
    ])

    expect(gitRepoRowErrors.value[1]).toBe(REPO_INACCESSIBLE_MSG)
    expect(gitRepoRowErrors.value[2]).toBeUndefined()
    expect(shouldShowRepoOAuthButton('https://github.com/org/a.git')).toBe(true)
    expect(shouldShowRepoOAuthButton('https://github.com/org/b.git')).toBe(false)
    expect(hoisted.catalogCompanyId).toBe('t1')
  })

  it('HTTP 失败展示服务端文案和 data-traceId，而不是笼统的网络失败', async () => {
    const { VALIDATE_GIT_REPO_NETWORK_MSG } = await import('../utils/gitRepoValidateError.js')
    apiFetchMock.mockResolvedValue({
      ok: false,
      status: 400,
      traceId: 'trace-http-batch',
      json: async () => ({
        error: 'urls 不能为空',
        message: 'urls 不能为空',
        trace_id: 'trace-http-batch',
      }),
    })

    const { gitRepoRows, gitRepoRowErrors, gitRepoRowErrorTraceId } = useCreateProjectGitRepoRows(
      route,
      router,
    )
    gitRepoRows.value = [{ id: 1, url: 'https://github.com/org/a.git', cloneAlias: '' }]
    await vi.advanceTimersByTimeAsync(600)
    await Promise.resolve()
    await Promise.resolve()

    expect(gitRepoRowErrors.value[1]).toBe('urls 不能为空')
    expect(gitRepoRowErrors.value[1]).not.toBe(VALIDATE_GIT_REPO_NETWORK_MSG)
    expect(gitRepoRowErrorTraceId.value[1]).toBe('trace-http-batch')
    const [, opts] = apiFetchMock.mock.calls[0]
    expect(opts.timeout).toBe(60000)
  })

  it('请求超时展示超时文案并保留 traceId', async () => {
    const { VALIDATE_GIT_REPO_TIMEOUT_MSG } = await import('../utils/gitRepoValidateError.js')
    const err = new Error('请求超时（30 秒），请检查网络后重试')
    err.name = 'TimeoutError'
    err.traceId = 'trace-timeout-batch'
    apiFetchMock.mockRejectedValue(err)

    const { gitRepoRows, gitRepoRowErrors, gitRepoRowErrorTraceId } = useCreateProjectGitRepoRows(
      route,
      router,
    )
    gitRepoRows.value = [{ id: 1, url: 'https://github.com/org/a.git', cloneAlias: '' }]
    await vi.advanceTimersByTimeAsync(600)
    await Promise.resolve()
    await Promise.resolve()

    expect(gitRepoRowErrors.value[1]).toBe(VALIDATE_GIT_REPO_TIMEOUT_MSG)
    expect(gitRepoRowErrorTraceId.value[1]).toBe('trace-timeout-batch')
  })

  it('200 但结果缺该 URL 时不误报网络失败', async () => {
    const {
      VALIDATE_GIT_REPO_MISSING_RESULT_MSG,
      VALIDATE_GIT_REPO_NETWORK_MSG,
    } = await import('../utils/gitRepoValidateError.js')
    apiFetchMock.mockResolvedValue({
      ok: true,
      status: 200,
      traceId: 'trace-miss-batch',
      json: async () => ({ results: [] }),
    })

    const { gitRepoRows, gitRepoRowErrors, gitRepoRowErrorTraceId } = useCreateProjectGitRepoRows(
      route,
      router,
    )
    gitRepoRows.value = [{ id: 1, url: 'https://github.com/org/a.git', cloneAlias: '' }]
    await vi.advanceTimersByTimeAsync(600)
    await Promise.resolve()
    await Promise.resolve()

    expect(gitRepoRowErrors.value[1]).toBe(VALIDATE_GIT_REPO_MISSING_RESULT_MSG)
    expect(gitRepoRowErrors.value[1]).not.toBe(VALIDATE_GIT_REPO_NETWORK_MSG)
    expect(gitRepoRowErrorTraceId.value[1]).toBe('trace-miss-batch')
  })

  it('retryValidateGitRepoRow 再次 POST 同一 URL', async () => {
    apiFetchMock.mockRejectedValue(Object.assign(new Error('Failed to fetch'), { name: 'TypeError' }))

    const {
      gitRepoRows,
      gitRepoRowErrors,
      retryValidateGitRepoRow,
    } = useCreateProjectGitRepoRows(route, router)
    gitRepoRows.value = [{ id: 1, url: 'https://github.com/org/a.git', cloneAlias: '' }]
    await vi.advanceTimersByTimeAsync(600)
    await Promise.resolve()
    await Promise.resolve()
    expect(gitRepoRowErrors.value[1]).toBeTruthy()

    apiFetchMock.mockReset()
    apiFetchMock.mockResolvedValue({
      ok: true,
      json: async () => ({
        results: [
          {
            url: 'https://github.com/org/a.git',
            is_accessible: true,
            token_status: 'token_available',
          },
        ],
      }),
    })
    await retryValidateGitRepoRow(1)
    await Promise.resolve()
    expect(apiFetchMock).toHaveBeenCalledTimes(1)
    expect(gitRepoRowErrors.value[1]).toBeUndefined()
  })

  it('startRepoOAuthConnect 把当前页 path 与 accessCode 写入 OAuth next', async () => {
    const storage = await import('../utils/githubAppReturnStorage.js')
    apiFetchMock.mockResolvedValue({
      ok: true,
      json: async () => ({ authorize_url: 'https://example.test/oauth' }),
    })
    Object.defineProperty(window, 'location', {
      configurable: true,
      value: {
        pathname: '/tenant/t/projects/p/',
        search: '?accessCode=9aaHjbryhL',
        href: '',
      },
    })
    const { startRepoOAuthConnect } = useCreateProjectGitRepoRows(route, router)
    await startRepoOAuthConnect('https://github.com/org/a.git')
    expect(storage.setGithubAppReturnTarget).toHaveBeenCalledWith(
      'key',
      '/tenant/t/projects/p/?accessCode=9aaHjbryhL',
    )
    expect(String(apiFetchMock.mock.calls[0][0])).toContain(
      encodeURIComponent('/tenant/t/projects/p/?accessCode=9aaHjbryhL'),
    )
  })

  it('T1 bootstrap 从 location.search 记住 grant_ticket', async () => {
    vi.useRealTimers()
    const repo = 'https://gitlab-tencent-sh-1.daydaymoney.com/example-user/ram-work.git'
    sessionStorage.clear()
    Object.defineProperty(window, 'location', {
      configurable: true,
      value: {
        pathname: '/tenant/t1/projects/create/',
        search: `?gitlab=ok&grant_ticket=tkt-create-1&repo_url=${encodeURIComponent(repo)}`,
      },
    })
    const { bootstrapGitReposOnMount } = useCreateProjectGitRepoRows(
      { params: { tenant: 't1' }, query: { grant_ticket: 'tkt-create-1', repo_url: repo } },
      router,
    )
    await bootstrapGitReposOnMount()
    const { sessionGrantTicketForRepo } = await import('../utils/grantTicketSession.js')
    expect(sessionGrantTicketForRepo(repo)).toBe('tkt-create-1')
    // OPT-20260902-006: bootstrap 会把 repo_url 填回首行并排 500ms 校验定时器；真实
    // 时钟下测试结束前等它落定，避免跨测试残留定时器。
    apiFetchMock.mockResolvedValue({
      ok: true,
      json: async () => ({ results: [] }),
    })
    await new Promise((resolve) => setTimeout(resolve, 600))
  })

  it('OPT-20260902-006: bootstrap 把 route.query.repo_url 写回首行空白输入框并触发校验', async () => {
    const repo = 'https://gitlab-tencent-sh-1.daydaymoney.com/example-user/ram-work.git'
    apiFetchMock.mockResolvedValue({
      ok: true,
      json: async () => ({ results: [] }),
    })
    const { gitRepoRows, bootstrapGitReposOnMount } = useCreateProjectGitRepoRows(
      { params: { tenant: 't1' }, query: { repo_url: repo } },
      router,
    )
    await bootstrapGitReposOnMount()
    expect(gitRepoRows.value[0].url).toBe(repo)
    await vi.advanceTimersByTimeAsync(600)
    await Promise.resolve()
    await Promise.resolve()
    expect(apiFetchMock).toHaveBeenCalled()
    const validateCall = apiFetchMock.mock.calls.find(([u]) => String(u).includes('/projects/validate-git-repos/'))
    expect(validateCall).toBeTruthy()
    const body = JSON.parse(validateCall[1].body)
    expect(body.urls).toContain(repo)
  })
})

}

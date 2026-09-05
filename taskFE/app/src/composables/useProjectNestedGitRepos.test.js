// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] useProjectNestedGitRepos.test.js requires vitest runtime')
} else {
const { ref } = await import('vue')
const { beforeEach, describe, expect, it, vi } = await import('vitest')
const { useProjectNestedGitRepos } = await import('./useProjectNestedGitRepos.js')

// vi.mock 工厂引用的变量必须经 vi.hoisted() 定义（Vitest 会把工厂 hoist 到模块顶部）。
const hoisted = vi.hoisted(() => ({
  apiFetchMock: vi.fn(),
}))
const apiFetchMock = hoisted.apiFetchMock

vi.mock('../utils/apiUtils.js', () => ({
  apiFetch: (...args) => hoisted.apiFetchMock(...args),
}))

function jsonResponse(body, ok = true, traceId = '') {
  const headers = {
    get: (name) => {
      if (String(name).toLowerCase() === 'x-trace-id') return traceId || null
      return null
    },
  }
  return {
    ok,
    status: ok ? 200 : 500,
    headers,
    traceId: traceId || undefined,
    json: async () => body,
  }
}

describe('useProjectNestedGitRepos', () => {
  beforeEach(() => {
    apiFetchMock.mockReset()
  })

  it('loads nested repos via fetchNestedGitRepos', async () => {
    apiFetchMock.mockResolvedValue(
      jsonResponse({
        parent_repo_url: 'https://gitlab.daydaymoney.com/g/ram-work.git',
        nested_repos: [
          { path: 'task2app', url: 'https://gitlab.daydaymoney.com/g/task2app.git', source: 'gitmodules' },
        ],
        error: '',
      }),
    )
    const { nestedRepos, nestedError, fetchNestedGitRepos } = useProjectNestedGitRepos({
      tenantId: 't1',
      projectId: 'p1',
      repoUrl: 'https://gitlab.daydaymoney.com/g/ram-work.git',
      auto: false,
    })
    await fetchNestedGitRepos()
    expect(apiFetchMock).toHaveBeenCalled()
    expect(nestedRepos.value).toHaveLength(1)
    expect(nestedRepos.value[0].path).toBe('task2app')
    expect(nestedError.value).toBe('')
  })

  it('surfaces API error string', async () => {
    apiFetchMock.mockResolvedValue(
      jsonResponse({
        parent_repo_url: 'https://gitlab.daydaymoney.com/g/ram-work.git',
        nested_repos: [],
        error: '未检测到可用授权',
      }),
    )
    const { nestedError, fetchNestedGitRepos } = useProjectNestedGitRepos({
      tenantId: 't1',
      projectId: 'p1',
      repoUrl: 'https://gitlab.daydaymoney.com/g/ram-work.git',
      auto: false,
    })
    await fetchNestedGitRepos()
    expect(nestedError.value).toContain('未检测到可用授权')
  })

  it('stores nestedErrorTraceId from response X-Trace-Id on business error', async () => {
    apiFetchMock.mockResolvedValue(
      jsonResponse(
        {
          parent_repo_url: 'https://github.com/ruandao/somanyad.git',
          nested_repos: [],
          error: '无法访问父仓库（远端返回不可见/无权限）',
        },
        true,
        'tid-nested-parent-1',
      ),
    )
    const { nestedError, nestedErrorTraceId, fetchNestedGitRepos } = useProjectNestedGitRepos({
      tenantId: 't1',
      projectId: 'p1',
      repoUrl: 'https://github.com/ruandao/somanyad.git',
      auto: false,
    })
    await fetchNestedGitRepos()
    expect(nestedError.value).toContain('无法访问父仓库')
    expect(nestedErrorTraceId.value).toBe('tid-nested-parent-1')
  })

  it('stores nestedErrorTraceId on HTTP error and clears on success', async () => {
    apiFetchMock.mockResolvedValueOnce(
      jsonResponse({ detail: 'boom' }, false, 'tid-nested-http-500'),
    )
    const { nestedError, nestedErrorTraceId, fetchNestedGitRepos } = useProjectNestedGitRepos({
      tenantId: 't1',
      projectId: 'p1',
      repoUrl: 'https://github.com/ruandao/somanyad.git',
      auto: false,
    })
    await fetchNestedGitRepos()
    expect(nestedError.value).toBeTruthy()
    expect(nestedErrorTraceId.value).toBe('tid-nested-http-500')

    apiFetchMock.mockResolvedValueOnce(
      jsonResponse({
        parent_repo_url: 'https://github.com/ruandao/somanyad.git',
        nested_repos: [],
        error: '',
      }, true, 'tid-nested-ok'),
    )
    await fetchNestedGitRepos()
    expect(nestedError.value).toBe('')
    expect(nestedErrorTraceId.value).toBe('')
  })

  it('does not forge nestedErrorTraceId for client-only gate', async () => {
    const { nestedError, nestedErrorTraceId, fetchNestedGitRepos } = useProjectNestedGitRepos({
      tenantId: '',
      projectId: '',
      repoUrl: 'https://github.com/ruandao/somanyad.git',
      auto: false,
    })
    await fetchNestedGitRepos()
    expect(apiFetchMock).not.toHaveBeenCalled()
    expect(nestedError.value).toContain('项目信息未加载')
    expect(nestedErrorTraceId.value).toBe('')
  })

  it('skips fetch when repoUrl empty', async () => {
    const { nestedRepos, fetchNestedGitRepos } = useProjectNestedGitRepos({
      tenantId: 't1',
      projectId: 'p1',
      repoUrl: '',
      auto: false,
    })
    await fetchNestedGitRepos()
    expect(apiFetchMock).not.toHaveBeenCalled()
    expect(nestedRepos.value).toEqual([])
  })

  it('auto watch triggers when repoUrl ref set', async () => {
    apiFetchMock.mockResolvedValue(
      jsonResponse({
        parent_repo_url: 'https://gitlab.daydaymoney.com/g/ram-work.git',
        nested_repos: [{ path: 'docs', url: 'https://gitlab.daydaymoney.com/g/docs.git', source: 'gitmodules' }],
        error: '',
      }),
    )
    const repoUrl = ref('https://gitlab.daydaymoney.com/g/ram-work.git')
    const { nestedRepos, nestedLoading } = useProjectNestedGitRepos({
      tenantId: ref('t1'),
      projectId: ref('p1'),
      repoUrl,
      auto: true,
    })
    await vi.waitFor(() => expect(nestedLoading.value).toBe(false))
    expect(apiFetchMock).toHaveBeenCalled()
    expect(nestedRepos.value[0].path).toBe('docs')
  })
})
}

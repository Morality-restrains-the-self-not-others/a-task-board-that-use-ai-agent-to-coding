// @vitest-environment node
if (!process.env.VITEST) {
  // pre-commit 会以 node 直跑已暂存 *.test.js，非 Vitest 环境下跳过。
  console.log('[skip] taskDetailFetchFns.repoCloneIdentity.test.js requires vitest runtime')
} else {
  const { describe, expect, it, vi, beforeEach } = await import('vitest')
  const { apiFetch } = await import('../../utils/apiUtils.js')
  const {
    maybeAutoApplyDefaultRepoCloneIdentities,
    onRepoReclone,
  } = await import('./taskDetailFetchFns.js')

vi.mock('../../utils/apiUtils.js', () => ({
  apiFetch: vi.fn(),
}))


function ref(value) {
  return { value }
}

function buildDeps(overrides = {}) {
  return {
    effectiveTenantId: ref('tenant-1'),
    effectiveWorkspaceId: ref('ws-1'),
    effectiveTaskId: ref('task-1'),
    layerGitIdentityOptions: ref([
      { id: 'default-id', is_default: true, display_name: 'Default' },
      { id: 'other-id', is_default: false, display_name: 'Other' },
    ]),
    layerGitIdentityLoading: ref(false),
    workspaceProjectsLoading: ref(false),
    taskRepoRows: ref([{ url: 'https://gitlab.com/group/repo.git', projectName: 'P', projectId: 'p1' }]),
    repoCloneIdentityByUrl: ref({}),
    repoCloneIdentitySaving: ref(false),
    repoCloneIdentitySaveError: ref(''),
    localTask: ref({ parameters: { repo_clone_git_identities: {} } }),
    repoCloneIdentityUserTouchedByUrl: ref({}),
    repoCloneIdentityAutoApplyInFlight: ref(false),
    repoCloneIdentityIdForUrl: (url) => {
      const u = String(url || '').trim()
      return String(overrides.repoCloneIdentityByUrl?.value?.[u] || '').trim()
    },
    ...overrides,
  }
}

describe('maybeAutoApplyDefaultRepoCloneIdentities', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('未保存克隆身份时自动 PATCH 公司默认身份', async () => {
    const deps = buildDeps()
    apiFetch.mockResolvedValue({
      ok: true,
      json: async () => ({
        ok: true,
        repo_clone_git_identities: {
          'https://gitlab.com/group/repo.git': 'default-id',
        },
      }),
    })

    await maybeAutoApplyDefaultRepoCloneIdentities(deps)

    expect(apiFetch).toHaveBeenCalledTimes(1)
    const [, options] = apiFetch.mock.calls[0]
    const body = JSON.parse(options.body)
    expect(body.repo_clone_git_identities['https://gitlab.com/group/repo.git']).toBe('default-id')
    expect(deps.repoCloneIdentityByUrl.value['https://gitlab.com/group/repo.git']).toBe('default-id')
    expect(deps.localTask.value.parameters.repo_clone_git_identities['https://gitlab.com/group/repo.git']).toBe(
      'default-id',
    )
  })

  it('已有克隆身份时不发起 PATCH', async () => {
    const deps = buildDeps({
      repoCloneIdentityByUrl: ref({ 'https://gitlab.com/group/repo.git': 'saved-id' }),
    })
    deps.repoCloneIdentityIdForUrl = (url) =>
      String(deps.repoCloneIdentityByUrl.value[String(url || '').trim()] || '')

    await maybeAutoApplyDefaultRepoCloneIdentities(deps)

    expect(apiFetch).not.toHaveBeenCalled()
  })

  it('用户手动改过后不再自动套用默认身份', async () => {
    const deps = buildDeps({
      repoCloneIdentityUserTouchedByUrl: ref({ 'https://gitlab.com/group/repo.git': true }),
    })

    await maybeAutoApplyDefaultRepoCloneIdentities(deps)

    expect(apiFetch).not.toHaveBeenCalled()
  })

  it('无 is_default 身份时不发起 PATCH', async () => {
    const deps = buildDeps({
      layerGitIdentityOptions: ref([{ id: 'other-id', is_default: false }]),
    })

    await maybeAutoApplyDefaultRepoCloneIdentities(deps)

    expect(apiFetch).not.toHaveBeenCalled()
  })
})

describe('onRepoReclone 失败时写入 traceId（OPT-20260816-003 回归）', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  function recloneDeps(overrides = {}) {
    return {
      effectiveTenantId: ref('tenant-1'),
      effectiveWorkspaceId: ref('ws-1'),
      effectiveTaskId: ref('task-1'),
      containerEndpointRegistered: ref(true),
      recloneLoadingByUrl: ref({}),
      recloneStatusByUrl: ref({}),
      recloneErrorTraceIdByUrl: ref({}),
      repoRecloneGlobalLoading: ref(false),
      repoCloneIdentityIdForUrl: () => '',
      containerForwardCommentId: ref('cmt-default'),
      ...overrides,
    }
  }

  it('请求失败时把响应 traceId 写入 recloneErrorTraceIdByUrl', async () => {
    const url = 'https://gitlab.com/group/repo.git'
    const deps = recloneDeps()
    apiFetch.mockResolvedValue({
      ok: false,
      headers: {
        get: (name) => (String(name).toLowerCase() === 'x-trace-id' ? 'trace-fail-001' : null),
      },
      json: async () => ({ detail: '容器内仓库忙，重试失败' }),
    })

    await onRepoReclone(url, deps)

    expect(deps.recloneStatusByUrl.value[url]).toBe('容器内仓库忙，重试失败')
    expect(deps.recloneErrorTraceIdByUrl.value[url]).toBe('trace-fail-001')
    expect(deps.recloneLoadingByUrl.value[url]).toBe(false)
  })

  it('成功态清空 traceId', async () => {
    const url = 'https://gitlab.com/group/repo.git'
    const deps = recloneDeps({
      recloneErrorTraceIdByUrl: ref({ [url]: 'old-trace' }),
      containerForwardCommentId: ref('cmt-fallback'),
    })
    apiFetch.mockResolvedValue({
      ok: true,
      json: async () => ({ async: true }),
    })

    await onRepoReclone(url, deps)

    expect(deps.recloneStatusByUrl.value[url]).toBe('')
    expect(deps.recloneErrorTraceIdByUrl.value[url]).toBe('')
  })

  it('payload 带 comment_id 时写入 path 与 body', async () => {
    const url = 'https://gitlab.com/group/repo.git'
    const deps = recloneDeps({ containerForwardCommentId: ref('cmt-fallback') })
    apiFetch.mockResolvedValue({
      ok: true,
      json: async () => ({ async: true }),
    })

    await onRepoReclone({ repoUrl: url, commentId: 'cmt-card' }, deps)

    expect(apiFetch).toHaveBeenCalledTimes(1)
    const [apiPath, options] = apiFetch.mock.calls[0]
    expect(apiPath).toContain('/comment_id/cmt-card/')
    expect(apiPath).not.toContain('cmt-fallback')
    const body = JSON.parse(options.body)
    expect(body.comment_id).toBe('cmt-card')
    expect(body.repo_url).toBe(url)
  })

  it('无评论 id 时不发请求并展示 缺少评论ID', async () => {
    const url = 'https://gitlab.com/group/repo.git'
    const deps = recloneDeps({ containerForwardCommentId: ref('') })
    await onRepoReclone(url, deps)
    expect(apiFetch).not.toHaveBeenCalled()
    expect(deps.recloneStatusByUrl.value[url]).toBe('缺少评论ID')
  })

  it('有 comment_id 时即使任务级 endpoint 未注册仍 POST，不展示「容器未启动」', async () => {
    const url = 'https://gitlab.com/group/ram-work.git'
    const deps = recloneDeps({
      containerEndpointRegistered: ref(false),
      containerForwardCommentId: ref('cmt-fallback'),
    })
    apiFetch.mockResolvedValue({
      ok: true,
      json: async () => ({ async: true }),
    })

    await onRepoReclone({ repoUrl: url, commentId: 'cmt-card' }, deps)

    expect(apiFetch).toHaveBeenCalledTimes(1)
    const [apiPath, options] = apiFetch.mock.calls[0]
    expect(apiPath).toContain('/comment_id/cmt-card/')
    expect(JSON.parse(options.body).comment_id).toBe('cmt-card')
    expect(deps.recloneStatusByUrl.value[url]).not.toBe('容器未启动')
    expect(deps.recloneStatusByUrl.value[url]).toBe('')
  })
})
}

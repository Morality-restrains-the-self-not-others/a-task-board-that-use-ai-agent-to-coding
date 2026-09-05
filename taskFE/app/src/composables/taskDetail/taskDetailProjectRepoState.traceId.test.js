// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] taskDetailProjectRepoState.traceId.test.js requires vitest runtime')
} else {
const { describe, expect, it, vi, beforeEach, afterEach } = await import('vitest')
const { ref } = await import('vue')

vi.mock('../../utils/apiUtils.js', () => ({
  apiFetch: vi.fn(),
}))

const { apiFetch } = await import('../../utils/apiUtils.js')
const { createTaskDetailProjectRepoState } = await import('./taskDetailProjectRepoState.js')

describe('getRepoBranchErrorTraceId', () => {
  beforeEach(() => {
    apiFetch.mockReset()
  })
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('stores TraceId for failed branch list fetch', async () => {
    apiFetch.mockResolvedValue({
      ok: false,
      status: 403,
      traceId: 'tid-branch-403',
      json: async () => ({ detail: '无权限拉取分支' }),
      headers: { get: () => null },
    })

    const state = createTaskDetailProjectRepoState({
      effectiveTenantId: ref('t1'),
      effectiveWorkspaceId: ref('ws1'),
      effectiveTaskId: ref('task1'),
      localTask: ref(null),
      editingTask: ref(null),
      isEditing: ref(true),
    })

    await state.fetchRepoBranches('proj-1', 'https://github.com/acme/repo.git', true)

    expect(state.getRepoBranchError('proj-1', 'https://github.com/acme/repo.git')).toContain('无权限')
    expect(state.getRepoBranchErrorTraceId('proj-1', 'https://github.com/acme/repo.git')).toBe('tid-branch-403')
    expect(state.commonMergeTargetBranchesErrorTraceId.value).toBe('')
  })

  it('exposes first failing repo TraceId on commonMergeTargetBranchesErrorTraceId', async () => {
    apiFetch.mockResolvedValue({
      ok: false,
      status: 502,
      traceId: 'tid-merge-first',
      json: async () => ({ error: 'upstream bad gateway' }),
      headers: { get: () => null },
    })

    const editingTask = ref({
      linkedProjects: [
        {
          project_id: 'proj-1',
          repo_branches: { 'https://github.com/acme/repo.git': 'main' },
        },
      ],
    })
    const state = createTaskDetailProjectRepoState({
      effectiveTenantId: ref('t1'),
      effectiveWorkspaceId: ref('ws1'),
      effectiveTaskId: ref('task1'),
      localTask: ref(null),
      editingTask,
      isEditing: ref(true),
    })
    state.workspaceProjects.value = [
      { id: 'proj-1', git_repos: ['https://github.com/acme/repo.git'] },
    ]

    await state.fetchRepoBranches('proj-1', 'https://github.com/acme/repo.git', true)

    expect(state.commonMergeTargetBranchesError.value).toBeTruthy()
    expect(state.commonMergeTargetBranchesErrorTraceId.value).toBe('tid-merge-first')
  })
})

}

// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] taskDetailProjectRepoState.test.js requires vitest runtime')
} else {
const { describe, expect, it, vi, beforeEach } = await import('vitest')
const { ref } = await import('vue')

vi.mock('../../utils/apiUtils.js', () => ({
  apiFetch: vi.fn(),
}))

const { apiFetch } = await import('../../utils/apiUtils.js')
const {
  repoBranchMapKey,
  createTaskDetailProjectRepoState,
} = await import('./taskDetailProjectRepoState.js')

function buildDeps(overrides = {}) {
  return {
    effectiveTenantId: ref('tenant-1'),
    effectiveWorkspaceId: ref('ws-1'),
    effectiveTaskId: ref('task-1'),
    localTask: ref({
      projects: [
        {
          project_id: 'proj-1',
          repo_index: 0,
          base_branch: 'main',
        },
      ],
      parameters: {
        repo_clone_git_identities: {
          'https://github.com/acme/repo.git': 'identity-1',
        },
      },
    }),
    editingTask: ref(null),
    isEditing: ref(false),
    repoCloneIdentityByUrl: ref({}),
    ...overrides,
  }
}

describe('repoBranchMapKey', () => {
  it('joins project id and repo url with hyphen', () => {
    expect(repoBranchMapKey('proj-1', 'https://github.com/a/b.git')).toBe(
      'proj-1-https://github.com/a/b.git',
    )
  })
})

describe('taskRepoRows', () => {
  it('returns rows only for projects linked on the task', () => {
    const deps = buildDeps({
      localTask: ref({
        projects: [{ project_id: 'proj-1', repo_index: 0, base_branch: 'main' }],
      }),
    })
    const state = createTaskDetailProjectRepoState(deps)
    state.workspaceProjects.value = [
      {
        id: 'proj-1',
        name: 'Alpha',
        git_repos: ['https://github.com/acme/repo.git', 'https://github.com/acme/other.git'],
      },
      {
        id: 'proj-2',
        name: 'Beta',
        git_repos: ['https://github.com/acme/unlinked.git'],
      },
    ]

    expect(state.taskRepoRows.value).toEqual([
      {
        url: 'https://github.com/acme/repo.git',
        projectName: 'Alpha',
        projectId: 'proj-1',
      },
      {
        url: 'https://github.com/acme/other.git',
        projectName: 'Alpha',
        projectId: 'proj-1',
      },
    ])
  })

  it('returns empty when task has no linked projects', () => {
    const deps = buildDeps({
      localTask: ref({ projects: [] }),
    })
    const state = createTaskDetailProjectRepoState(deps)
    state.workspaceProjects.value = [
      { id: 'proj-1', name: 'Alpha', git_repos: ['https://github.com/acme/repo.git'] },
    ]
    expect(state.taskRepoRows.value).toEqual([])
  })

  it('falls back to task API repo URL when workspace catalog is empty', () => {
    const deps = buildDeps({
      localTask: ref({
        projects: [{
          project_id: 'proj-1',
          repo_index: 0,
          base_branch: 'master',
          stored_repo_address: 'https://github.com/ruandao/helloworld',
          project_repo_url: 'https://github.com/test-ruandao/helloworld.git',
        }],
      }),
    })
    const state = createTaskDetailProjectRepoState(deps)
    state.workspaceProjects.value = []
    expect(state.taskRepoRows.value).toEqual([
      {
        url: 'https://github.com/test-ruandao/helloworld.git',
        projectName: 'proj-1',
        projectId: 'proj-1',
      },
    ])
  })

  it('reads projects when localTask was assigned a workspace list', () => {
    const deps = buildDeps({
      localTask: ref([
        { id: 'task-other', projects: [] },
        {
          id: 'task-1',
          projects: [{
            project_id: 'proj-1',
            repo_index: 0,
            base_branch: 'master',
            project_repo_url: 'https://github.com/test-ruandao/helloworld.git',
          }],
        },
      ]),
    })
    const state = createTaskDetailProjectRepoState(deps)
    state.workspaceProjects.value = []
    expect(state.taskRepoRows.value).toEqual([
      {
        url: 'https://github.com/test-ruandao/helloworld.git',
        projectName: 'proj-1',
        projectId: 'proj-1',
      },
    ])
    expect(state.taskProjectsWithDetails.value).toHaveLength(1)
  })
})

describe('taskProjectsWithDetails', () => {
  it('flags mismatch when stored task repo owner differs from current project git_repos', () => {
    const deps = buildDeps({
      localTask: ref({
        projects: [
          {
            project_id: 'proj-1',
            repo_index: 0,
            base_branch: 'master',
            stored_repo_address: 'https://github.com/ruandao/helloworld',
            repo_address_mismatch: false,
          },
        ],
      }),
    })
    const state = createTaskDetailProjectRepoState(deps)
    state.workspaceProjects.value = [
      {
        id: 'proj-1',
        name: 'helloworld',
        git_repos: ['https://github.com/test-ruandao/helloworld.git'],
      },
    ]
    const rows = state.taskProjectsWithDetails.value
    expect(rows).toHaveLength(1)
    expect(rows[0].repo_address_mismatch).toBe(true)
    expect(rows[0].stored_repo_address).toBe('https://github.com/ruandao/helloworld')
  })
})

describe('validateLinkedProjectReposBeforeSendToAi', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('returns ok when task has no linked repos', async () => {
    const deps = buildDeps({
      localTask: ref({ projects: [], parameters: {} }),
    })
    const state = createTaskDetailProjectRepoState(deps)

    const result = await state.validateLinkedProjectReposBeforeSendToAi()

    expect(result).toEqual({ ok: true, message: '' })
    expect(apiFetch).not.toHaveBeenCalled()
  })

  it('validates clone identity and github binding for linked repos', async () => {
    const deps = buildDeps()
    const state = createTaskDetailProjectRepoState(deps)
    state.workspaceProjects.value = [
      {
        id: 'proj-1',
        name: 'Alpha',
        git_repos: ['https://github.com/acme/repo.git'],
      },
    ]

    apiFetch.mockResolvedValue({
      ok: true,
      json: async () => ({
        repo_bindings: [
          {
            repo_slug: 'acme/repo',
            selected_github_user_id: 'gh-user-1',
          },
        ],
      }),
    })

    const result = await state.validateLinkedProjectReposBeforeSendToAi()

    expect(apiFetch).toHaveBeenCalledTimes(1)
    // 回归：funcName-first 路径契约（原 bug 拼成 task_id/${taskId}github-credential-status/ 缺斜杠）。
    expect(String(apiFetch.mock.calls[0][0])).toBe(
      '/api/cloud/compute/github-credential-status/tenant_id/tenant-1/workspace_id/ws-1/task_id/task-1/',
    )
    expect(result.ok).toBe(true)
  })
})

describe('syncStaleTaskRepoAddresses 触发重新克隆（OPT-20260829-025）', () => {
  beforeEach(() => {
    apiFetch.mockReset()
  })

  it('容器已注册时同步成功后对每个 git_repo 触发 onRepoReclone', async () => {
    const recloneCalls = []
    const repoRecloneBridge = {
      fn: (payload) => {
        recloneCalls.push(payload)
        return Promise.resolve()
      },
    }
    const containerEndpointRegistered = ref(true)
    const deps = buildDeps({
      repoRecloneBridge,
      containerEndpointRegistered,
      localTask: ref({
        projects: [{ project_id: 'proj-1', repo_index: 0, base_branch: 'main' }],
      }),
    })
    apiFetch.mockResolvedValue({
      ok: true,
      json: async () => ({ projects: [{ project_id: 'proj-1', repo_index: 0, base_branch: 'main' }] }),
    })
    const state = createTaskDetailProjectRepoState(deps)
    state.workspaceProjects.value = [
      {
        id: 'proj-1',
        name: 'Alpha',
        git_repos: ['https://github.com/acme/repo.git', 'https://github.com/acme/other.git'],
      },
    ]

    await state.syncStaleTaskRepoAddresses()

    expect(recloneCalls.map((c) => c.repoUrl)).toEqual([
      'https://github.com/acme/repo.git',
      'https://github.com/acme/other.git',
    ])
    expect(state.staleRepoSyncNeedsReclone.value).toBe(true)
  })

  it('容器未注册时同步成功不触发重新克隆', async () => {
    const recloneCalls = []
    const repoRecloneBridge = {
      fn: (payload) => {
        recloneCalls.push(payload)
        return Promise.resolve()
      },
    }
    const deps = buildDeps({
      repoRecloneBridge,
      containerEndpointRegistered: ref(false),
      localTask: ref({ projects: [{ project_id: 'proj-1', repo_index: 0, base_branch: 'main' }] }),
    })
    apiFetch.mockResolvedValue({ ok: true, json: async () => ({}) })
    const state = createTaskDetailProjectRepoState(deps)
    state.workspaceProjects.value = [
      {
        id: 'proj-1',
        name: 'Alpha',
        git_repos: ['https://github.com/acme/repo.git'],
      },
    ]

    await state.syncStaleTaskRepoAddresses()

    expect(recloneCalls.length).toBe(0)
    expect(state.staleRepoSyncNeedsReclone.value).toBe(false)
  })

  it('PATCH 返回空 projects 时不擦掉已加载关联项目', async () => {
    const deps = buildDeps({
      localTask: ref({
        id: 'task-1',
        projects: [{
          project_id: 'proj-1',
          stored_repo_address: 'https://github.com/ruandao/helloworld',
        }],
      }),
    })
    apiFetch.mockResolvedValue({
      ok: true,
      json: async () => ({ id: 'task-1', projects: [] }),
    })
    const state = createTaskDetailProjectRepoState(deps)
    await state.syncStaleTaskRepoAddresses()
    expect(deps.localTask.value.projects).toEqual([
      {
        project_id: 'proj-1',
        stored_repo_address: 'https://github.com/ruandao/helloworld',
      },
    ])
  })
})

}

// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] githubRepoBindingState.test.js requires vitest runtime')
} else {
const { describe, expect, it, vi, beforeEach } = await import('vitest')

vi.mock('../../utils/apiUtils.js', () => ({
  apiFetch: vi.fn(),
}))

const { createGithubRepoBindingState } = await import('./githubRepoBindingState.js')
const { apiFetch } = await import('../../utils/apiUtils.js')

function buildState() {
  return createGithubRepoBindingState({
    getRouteIds: () => ({ tenantId: 't', workspaceId: 'w', taskId: '1' }),
  })
}

describe('createGithubRepoBindingState', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('reads draft then bound github user id for a repo', () => {
    const state = buildState()
    expect(state.selectedGithubUserIdForRepo('https://github.com/acme/demo.git')).toBe('')
    state.repoBindingBySlug.value = {
      'acme/demo': { selected_github_user_id: '99' },
    }
    expect(state.selectedGithubUserIdForRepo('https://github.com/acme/demo.git')).toBe('99')
    expect(state.hasGithubRepoBindingSaved('https://github.com/acme/demo.git')).toBe(true)
    state.setSelectedGithubUserIdForRepo('https://github.com/acme/demo.git', '42')
    expect(state.selectedGithubUserIdForRepo('https://github.com/acme/demo.git')).toBe('42')
  })

  it('calls status with funcName-first path (no missing slash regression)', async () => {
    const state = buildState()
    apiFetch.mockResolvedValue({
      ok: true,
      json: async () => ({ repo_bindings: [], github_connections: [] }),
    })

    await state.fetchGithubRepoBindingStatus()

    expect(apiFetch).toHaveBeenCalledTimes(1)
    // 回归：原 bug 拼成 task_id/${taskId}github-credential-status/ 缺斜杠。
    expect(String(apiFetch.mock.calls[0][0])).toBe(
      '/api/cloud/compute/github-credential-status/tenant_id/t/workspace_id/w/task_id/1/',
    )
  })

  it('calls approve with funcName-first path and payload', async () => {
    const state = buildState()
    apiFetch.mockResolvedValue({
      ok: true,
      json: async () => ({ approved: true }),
    })
    state.setSelectedGithubUserIdForRepo('https://github.com/acme/demo.git', '42')

    await state.saveGithubRepoBinding('https://github.com/acme/demo.git')

    // save → refresh status（两次调用）：先 approve 后 status。
    expect(apiFetch).toHaveBeenCalledTimes(2)
    expect(String(apiFetch.mock.calls[0][0])).toBe(
      '/api/cloud/compute/github-credential-approve/tenant_id/t/workspace_id/w/task_id/1/',
    )
    expect(apiFetch.mock.calls[0][1].method).toBe('POST')
    expect(JSON.parse(apiFetch.mock.calls[0][1].body)).toEqual({
      repo_url: 'https://github.com/acme/demo.git',
      repo_slug: 'acme/demo',
      github_user_id: '42',
    })
    expect(String(apiFetch.mock.calls[1][0])).toBe(
      '/api/cloud/compute/github-credential-status/tenant_id/t/workspace_id/w/task_id/1/',
    )
  })
})
}

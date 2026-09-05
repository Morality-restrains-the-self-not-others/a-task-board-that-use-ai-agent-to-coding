// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] useProjectDetailGitRepos.disk-size.test.js requires vitest runtime')
} else {
const { beforeEach, describe, expect, it, vi } = await import('vitest')
const { ref } = await import('vue')
const { flushPromises } = await import('@vue/test-utils')
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
  resolveRepoOAuthAuthorizeUrl: () => '',
  resolveRepoOAuthProviderInfo: () => null,
}))

const { useProjectDetailGitRepos } = await import('./useProjectDetailGitRepos.js')

describe('useProjectDetailGitRepos phase-B disk size', () => {
  beforeEach(() => {
    apiFetchMock.mockReset()
  })

  it('loads disk_size_bytes from git-repo-disk-sizes after project id is set', async () => {
    const internal = 'https://gitlab.daydaymoney.com/g/internal.git'
    apiFetchMock.mockResolvedValue({
      ok: true,
      json: async () => ({
        git_repo_entries: [
          { url: internal, is_internal: true, disk_size_bytes: 2048 },
        ],
      }),
    })
    const project = ref({
      id: 'proj_disk',
      git_repos: [internal],
      git_repo_entries: [{ url: internal, clone_alias: '', is_internal: true }],
    })
    const { projectGitRepoEntries } = useProjectDetailGitRepos({ project })
    await flushPromises()
    expect(apiFetchMock).toHaveBeenCalled()
    expect(String(apiFetchMock.mock.calls[0][0])).toContain('/git-repo-disk-sizes/')
    expect(String(apiFetchMock.mock.calls[0][0])).toContain('proj_disk')
    expect(apiFetchMock.mock.calls[0][1].method).toBe('GET')
    expect(projectGitRepoEntries.value[0].diskSizeBytes).toBe(2048)
  })

  it('does not GET disk-sizes until project id exists', async () => {
    const project = ref({
      git_repos: ['https://gitlab.daydaymoney.com/g/internal.git'],
      git_repo_entries: [{ url: 'https://gitlab.daydaymoney.com/g/internal.git' }],
    })
    useProjectDetailGitRepos({ project })
    await flushPromises()
    expect(apiFetchMock).not.toHaveBeenCalled()
  })
})

}

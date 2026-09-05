// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] useProjectDetailGitRepos.oauth-return.test.js requires vitest runtime')
} else {
  const { beforeEach, describe, expect, it, vi } = await import('vitest')
  const { ref } = await import('vue')

  const hoisted = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
    setGithubAppReturnTargetMock: vi.fn(),
    routeMock: {
      params: { tenant: '877397588196749312', id: 'proj_880498883115905024' },
      path: '/tenant/877397588196749312/projects/proj_880498883115905024/',
      query: { accessCode: '9aaHjbryhL' },
    },
    ref: require('vue').ref,
  }))

  vi.mock('vue-router', () => ({
    useRoute: () => hoisted.routeMock,
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
    resolveRepoOAuthAuthorizeUrl: () => '/api/git-oauth/gitlab-start-from-gateway/',
    resolveRepoOAuthProviderInfo: () => ({ provider: 'gitlab', service_provider: 'default' }),
  }))

  vi.mock('../utils/githubAppReturnStorage.js', () => ({
    createGithubAppReturnKey: () => 'ab'.repeat(16),
    setGithubAppReturnTarget: (...args) => hoisted.setGithubAppReturnTargetMock(...args),
  }))

  const { useProjectDetailGitRepos } = await import('./useProjectDetailGitRepos.js')

  const REPO_URL = 'https://gitlab.com/acme/demo.git'
  const PROJECT_PATH = '/tenant/877397588196749312/projects/proj_880498883115905024/'
  const ACCESS = '9aaHjbryhL'

  function stubLocation({ pathname, search }) {
    const locationMock = {
      href: `http://localhost:4000${pathname}${search}`,
      pathname,
      search,
    }
    Object.defineProperty(window, 'location', {
      value: locationMock,
      writable: true,
      configurable: true,
    })
    return locationMock
  }

  describe('useProjectDetailGitRepos OAuth retry returns to current project page', () => {
    beforeEach(() => {
      hoisted.apiFetchMock.mockReset()
      hoisted.setGithubAppReturnTargetMock.mockReset()
      hoisted.routeMock.query = { accessCode: ACCESS }
      hoisted.routeMock.path = PROJECT_PATH
      hoisted.apiFetchMock.mockResolvedValue({
        ok: true,
        json: async () => ({ authorize_url: 'https://oauth.example/gitlab' }),
      })
    })

    it('writes next/return_url from location.search including accessCode', async () => {
      const locationMock = stubLocation({
        pathname: PROJECT_PATH,
        search: `?accessCode=${ACCESS}`,
      })
      const project = ref({ git_repos: [REPO_URL], git_repo_entries: [{ url: REPO_URL }] })
      const { startRepoOAuthConnect } = useProjectDetailGitRepos({ project })

      await startRepoOAuthConnect(REPO_URL)

      const expected = `${PROJECT_PATH}?accessCode=${ACCESS}`
      expect(hoisted.setGithubAppReturnTargetMock).toHaveBeenCalledWith('ab'.repeat(16), expected)
      const startUrl = String(hoisted.apiFetchMock.mock.calls[0][0])
      expect(startUrl).toContain(`next=${encodeURIComponent(expected)}`)
      expect(locationMock.href).toBe('https://oauth.example/gitlab')
    })

    it('falls back to route.query.accessCode when location.search is empty', async () => {
      stubLocation({ pathname: PROJECT_PATH, search: '' })
      const project = ref({ git_repos: [REPO_URL], git_repo_entries: [{ url: REPO_URL }] })
      const { startRepoOAuthConnect } = useProjectDetailGitRepos({ project })

      await startRepoOAuthConnect(REPO_URL)

      const expected = `${PROJECT_PATH}?accessCode=${ACCESS}`
      expect(hoisted.setGithubAppReturnTargetMock).toHaveBeenCalledWith('ab'.repeat(16), expected)
      const startUrl = String(hoisted.apiFetchMock.mock.calls[0][0])
      expect(startUrl).toContain(`next=${encodeURIComponent(expected)}`)
    })
  })
}

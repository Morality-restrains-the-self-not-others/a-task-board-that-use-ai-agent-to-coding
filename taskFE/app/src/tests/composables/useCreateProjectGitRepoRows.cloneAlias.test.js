// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] useCreateProjectGitRepoRows.cloneAlias.test.js requires vitest runtime')
} else {
const { describe, expect, it, vi } = await import('vitest')
const { useCreateProjectGitRepoRows } = await import('../../composables/useCreateProjectGitRepoRows.js')

vi.mock('../../utils/apiUtils.js', () => ({
  apiFetch: vi.fn(),
}))

vi.mock('../../composables/useProjectGitOAuthCatalog.js', () => ({
  useProjectGitOAuthCatalog: () => ({
    touchRepoOAuthButtons: vi.fn(),
    bootstrapGitOAuthCatalog: vi.fn(),
  }),
}))

vi.mock('../../utils/githubAppReturnStorage.js', () => ({
  createGithubAppReturnKey: vi.fn(() => 'key'),
  setGithubAppReturnTarget: vi.fn(),
}))

vi.mock('../../utils/gitSiteOAuthCallbackUtils.js', () => ({
  applyOAuthCallbackFromRoute: vi.fn(async () => null),
}))

describe('useCreateProjectGitRepoRows clone alias', () => {
  const route = { params: { tenant: '850256677331562496' } }
  const router = { replace: vi.fn() }

  it('trimmedGitRepoPayload emits plain strings when no alias', () => {
    const { gitRepoRows, trimmedGitRepoPayload } = useCreateProjectGitRepoRows(route, router)
    gitRepoRows.value = [
      { id: 1, url: 'https://github.com/org/a.git', cloneAlias: '' },
      { id: 2, url: '  ', cloneAlias: 'ignored' },
    ]
    expect(trimmedGitRepoPayload()).toEqual(['https://github.com/org/a.git'])
  })

  it('trimmedGitRepoPayload emits object entries when alias is set', () => {
    const { gitRepoRows, trimmedGitRepoPayload } = useCreateProjectGitRepoRows(route, router)
    gitRepoRows.value = [
      { id: 1, url: 'https://github.com/org/a.git', cloneAlias: 'frontend' },
      { id: 2, url: 'git@gitlab.com:group/backend.git', cloneAlias: ' api ' },
    ]
    expect(trimmedGitRepoPayload()).toEqual([
      { url: 'https://github.com/org/a.git', clone_alias: 'frontend' },
      { url: 'git@gitlab.com:group/backend.git', clone_alias: 'api' },
    ])
  })

  it('duplicateCloneAliasError detects case-insensitive duplicates', () => {
    const { gitRepoRows, duplicateCloneAliasError } = useCreateProjectGitRepoRows(route, router)
    gitRepoRows.value = [
      { id: 1, url: 'https://github.com/org/a.git', cloneAlias: 'Web' },
      { id: 2, url: 'https://github.com/org/b.git', cloneAlias: 'web' },
    ]
    expect(duplicateCloneAliasError()).toBe('同一项目内仓库别名不能重复')
  })

  it('duplicateCloneAliasError ignores empty aliases', () => {
    const { gitRepoRows, duplicateCloneAliasError } = useCreateProjectGitRepoRows(route, router)
    gitRepoRows.value = [
      { id: 1, url: 'https://github.com/org/a.git', cloneAlias: '' },
      { id: 2, url: 'https://github.com/org/b.git', cloneAlias: '   ' },
    ]
    expect(duplicateCloneAliasError()).toBe('')
  })
})
}

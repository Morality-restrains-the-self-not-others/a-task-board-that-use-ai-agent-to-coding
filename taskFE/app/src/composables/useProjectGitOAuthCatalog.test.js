// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] useProjectGitOAuthCatalog.test.js requires vitest runtime')
} else {
const { describe, expect, it } = await import('vitest')
const { setGitOAuthProviderCatalogForTests } = await import('../utils/repoOAuthAuthorizeUtils.js')
const { useProjectGitOAuthCatalog } = await import('./useProjectGitOAuthCatalog.js')

describe('useProjectGitOAuthCatalog', () => {
  it('bootstrap 把 getCompanyId 传给 providers catalog 请求', async () => {
    setGitOAuthProviderCatalogForTests(null)
    let seenPath = ''
    const apiFetch = async (path) => {
      seenPath = String(path || '')
      return { ok: true, json: async () => ({ providers: [] }) }
    }
    try {
      const { bootstrapGitOAuthCatalog } = useProjectGitOAuthCatalog(
        apiFetch,
        () => '877397588196749312',
      )
      await bootstrapGitOAuthCatalog()
      expect(seenPath).toContain('/api/git-oauth/providers/')
      expect(seenPath).toContain('company_id=877397588196749312')
    } finally {
      setGitOAuthProviderCatalogForTests(null)
    }
  })
})
}

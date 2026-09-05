// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] formatLayerGitPushMultiRepoDetail.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const {
    formatOauthMultiRepoPushDetail,
    formatLayerGitPushFailureMessage,
    labelLayerGitPushRepo,
  } = await import('./formatLayerGitPushMultiRepoDetail.js')

  describe('formatLayerGitPushMultiRepoDetail', () => {
    it('labels parent and nested child with path', () => {
      expect(labelLayerGitPushRepo({ github_slug: 'acme/parent', rel_prefix: '' })).toBe('acme/parent')
      expect(
        labelLayerGitPushRepo({ github_slug: 'acme/child', rel_prefix: 'vendor/child' }),
      ).toBe('acme/child（路径 vendor/child）')
    })

    it('lists success and failure for partial push', () => {
      const text = formatOauthMultiRepoPushDetail([
        { github_slug: 'acme/parent', rel_prefix: 'parent', push_ok: true },
        {
          github_slug: 'acme/child',
          rel_prefix: 'parent/child',
          push_ok: false,
          detail: 'permission denied',
        },
      ])
      expect(text).toContain('部分仓库推送未成功（成功 1，失败 1）')
      expect(text).toContain('成功：acme/parent（路径 parent）')
      expect(text).toContain('失败：acme/child（路径 parent/child） — permission denied')
    })

    it('prefers multirepo body over bare detail', () => {
      const msg = formatLayerGitPushFailureMessage({
        detail: 'old short',
        github_oauth_multirepo: {
          repos: [
            { github_slug: 'a/b', push_ok: true },
            { github_slug: 'c/d', push_ok: false, detail: 'x' },
          ],
        },
      })
      expect(msg).toContain('成功：a/b')
      expect(msg).toContain('失败：c/d')
      expect(msg).not.toBe('old short')
    })
  })
}

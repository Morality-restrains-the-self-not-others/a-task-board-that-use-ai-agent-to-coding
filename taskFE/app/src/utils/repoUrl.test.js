// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] repoUrl.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const { isHttpRepoUrl } = await import('./repoUrl.js')

  describe('isHttpRepoUrl', () => {
    it('识别 http/https 仓库地址', () => {
      expect(isHttpRepoUrl('https://gitlab.daydaymoney.com/a/b.git')).toBe(true)
      expect(isHttpRepoUrl('http://10.2.150.68:8012/a/b.git')).toBe(true)
    })

    it('非 http(s) 保持纯文本（ssh/git@/空）', () => {
      expect(isHttpRepoUrl('git@github.com:acme/demo.git')).toBe(false)
      expect(isHttpRepoUrl('ssh://git@host/repo.git')).toBe(false)
      expect(isHttpRepoUrl('')).toBe(false)
      expect(isHttpRepoUrl(null)).toBe(false)
      expect(isHttpRepoUrl('  ')).toBe(false)
    })
  })
}

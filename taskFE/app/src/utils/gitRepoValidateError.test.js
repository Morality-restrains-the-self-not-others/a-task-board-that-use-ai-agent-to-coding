// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] gitRepoValidateError.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const {
    indexValidateResultsByUrl,
    lookupValidateResultByUrl,
  } = await import('./gitRepoValidateError.js')

  describe('lookupValidateResultByUrl', () => {
    it('matches exact url then .git / trailing-slash variants', () => {
      const byUrl = indexValidateResultsByUrl([
        { url: 'https://gitlab-tencent-sh-1.daydaymoney.com/g/demo.git', token_status: 'token_available' },
      ])
      expect(lookupValidateResultByUrl(
        byUrl,
        'https://gitlab-tencent-sh-1.daydaymoney.com/g/demo.git',
      )?.token_status).toBe('token_available')
      expect(lookupValidateResultByUrl(
        byUrl,
        'https://gitlab-tencent-sh-1.daydaymoney.com/g/demo',
      )?.token_status).toBe('token_available')
    })

    it('returns null when no row matches', () => {
      const byUrl = indexValidateResultsByUrl([
        { url: 'https://gitlab.example/g/a.git', token_status: 'token_available' },
      ])
      expect(lookupValidateResultByUrl(byUrl, 'https://gitlab.example/g/other.git')).toBeNull()
    })
  })
}

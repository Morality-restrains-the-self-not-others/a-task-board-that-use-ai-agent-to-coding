/**
 * feature-params base_url：拼接脏数据修复（不按运营商改写端点）
 */
if (!process.env.VITEST) {
  console.log('[skip] featureParamsBaseUrl.test.js requires vitest runtime')
} else {
  const { describe, it, expect } = await import('vitest')
  const { repairConcatenatedAbsoluteUrl } = await import('./featureParamsBaseUrl.js')

  describe('repairConcatenatedAbsoluteUrl', () => {
    it('keeps a single absolute URL', () => {
      expect(repairConcatenatedAbsoluteUrl('https://api.deepseek.com/v1')).toBe(
        'https://api.deepseek.com/v1',
      )
    })

    it('keeps operator-typed anthropic paths', () => {
      expect(repairConcatenatedAbsoluteUrl('https://api.deepseek.com/anthropic')).toBe(
        'https://api.deepseek.com/anthropic',
      )
    })

    it('keeps the last URL when two absolute URLs were glued', () => {
      expect(
        repairConcatenatedAbsoluteUrl(
          'https://api.deepseek.com/v1https://api.deepseek.com',
        ),
      ).toBe('https://api.deepseek.com')
    })

    it('keeps query-embedded URLs intact', () => {
      expect(
        repairConcatenatedAbsoluteUrl('https://example.com/p?next=https://other.example/v1'),
      ).toBe('https://example.com/p?next=https://other.example/v1')
    })
  })
}

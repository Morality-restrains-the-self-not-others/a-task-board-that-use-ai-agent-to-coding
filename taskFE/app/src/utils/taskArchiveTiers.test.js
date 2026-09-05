if (!process.env.VITEST) {
  console.log('[skip] taskArchiveTiers.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const { archiveTierLabel, ARCHIVE_TIER_OPTIONS } = await import('./taskArchiveTiers.js')

  describe('archiveTierLabel', () => {
    it('maps 7d to 免费存放期(7天)', () => {
      expect(archiveTierLabel('7d')).toBe('免费存放期(7天)')
    })

    it('defaults missing tier to 7d label', () => {
      expect(archiveTierLabel('')).toBe('免费存放期(7天)')
    })

    it('returns unknown codes as-is', () => {
      expect(archiveTierLabel('99y')).toBe('99y')
    })

    it('includes 3m in the option list', () => {
      expect(ARCHIVE_TIER_OPTIONS.some((o) => o.value === '3m')).toBe(true)
    })
  })
}

// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] peopleInviteExpiration.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const {
    DEFAULT_INVITE_EXPIRATION_DAYS,
    INVITE_EXPIRATION_DAY_OPTIONS,
    MAX_INVITE_EXPIRATION_DAYS,
  } = await import('./peopleInviteExpiration.js')

  describe('peopleInviteExpiration', () => {
    it('选项覆盖季度/半年/一年且默认 90、上限 365', () => {
      expect(INVITE_EXPIRATION_DAY_OPTIONS).toEqual([1, 3, 7, 14, 30, 90, 180, 365])
      expect(DEFAULT_INVITE_EXPIRATION_DAYS).toBe(90)
      expect(MAX_INVITE_EXPIRATION_DAYS).toBe(365)
      expect(Math.max(...INVITE_EXPIRATION_DAY_OPTIONS)).toBe(MAX_INVITE_EXPIRATION_DAYS)
    })
  })
}

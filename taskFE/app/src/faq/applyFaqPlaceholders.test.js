// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] applyFaqPlaceholders.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const { applyFaqPlaceholders, CONTACT_EMAIL_PLACEHOLDER } = await import(
    './applyFaqPlaceholders.js'
  )

  describe('applyFaqPlaceholders', () => {
    it('replaces every {{contactEmail}} with the configured address', () => {
      const src = `请发送邮件至 ${CONTACT_EMAIL_PLACEHOLDER}，或再发至 ${CONTACT_EMAIL_PLACEHOLDER}。`
      expect(applyFaqPlaceholders(src, 'ops@example.com')).toBe(
        '请发送邮件至 ops@example.com，或再发至 ops@example.com。',
      )
    })

    it('returns content unchanged when no placeholder is present', () => {
      expect(applyFaqPlaceholders('没有邮箱', 'ops@example.com')).toBe('没有邮箱')
    })

    it('throws when a placeholder is present but contactEmail is missing', () => {
      expect(() => applyFaqPlaceholders(`联系 ${CONTACT_EMAIL_PLACEHOLDER}`, '')).toThrow(
        /contactEmail/,
      )
      expect(() => applyFaqPlaceholders(`联系 ${CONTACT_EMAIL_PLACEHOLDER}`, '   ')).toThrow(
        /contactEmail/,
      )
    })
  })
}

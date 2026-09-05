// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] adminUserSuperuserGate.test.js requires vitest runtime')
} else {
  const { describe, it, expect } = await import('vitest')
  const { canSetSuperuserFlag, omitUnauthorizedSuperuser } = await import('./adminUserSuperuserGate.js')

  describe('adminUserSuperuserGate', () => {
    it('allows super_admin or platform:manage', () => {
      expect(canSetSuperuserFlag((r) => r === 'super_admin', () => false)).toBe(true)
      expect(canSetSuperuserFlag(() => false, (c) => c === 'platform:manage')).toBe(true)
      expect(canSetSuperuserFlag(() => false, () => false)).toBe(false)
    })

    it('omits is_superuser when unauthorized', () => {
      const out = omitUnauthorizedSuperuser(
        { username: 'a', is_superuser: true, password: '' },
        false,
      )
      expect(out.username).toBe('a')
      expect(out).not.toHaveProperty('is_superuser')
      expect(out).not.toHaveProperty('password')
    })

    it('keeps is_superuser when authorized', () => {
      const out = omitUnauthorizedSuperuser(
        { username: 'a', is_superuser: true, password: 'x' },
        true,
      )
      expect(out.is_superuser).toBe(true)
      expect(out.password).toBe('x')
    })
  })
}

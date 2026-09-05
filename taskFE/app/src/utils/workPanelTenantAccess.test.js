// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] workPanelTenantAccess.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const { resolveUnauthorizedTenantRedirect } = await import('./workPanelTenantAccess.js')

  describe('resolveUnauthorizedTenantRedirect', () => {
    it('租户在 companies 内时不跳转', () => {
      expect(
        resolveUnauthorizedTenantRedirect({
          tenantIdFromUrl: '850',
          userData: { companies: [{ id: '850' }], id: 'u1' },
          userId: 'u1',
        }),
      ).toBeNull()
    })

    it('租户不在 companies 内时跳转个人中心', () => {
      const result = resolveUnauthorizedTenantRedirect({
        tenantIdFromUrl: '850',
        userData: { companies: [{ id: '999' }], id: 'u2' },
        userId: 'u2',
      })
      expect(result).toEqual({
        href: '/user/u2/profile/',
        reason: '当前账号无权访问该租户工作面板',
      })
    })

    it('无 companies 但有 userId 时回退到个人资料页', () => {
      const result = resolveUnauthorizedTenantRedirect({
        tenantIdFromUrl: '850',
        userData: { id: 'u1' },
        userId: 'u1',
      })
      expect(result).toMatchObject({
        href: '/user/u1/profile/',
      })
    })

    it('无 companies 且无 userId 时不跳转', () => {
      expect(
        resolveUnauthorizedTenantRedirect({
          tenantIdFromUrl: '850',
          userData: { id: '' },
          userId: '',
        }),
      ).toBeNull()
    })

    it('current_company 匹配时允许访问', () => {
      expect(
        resolveUnauthorizedTenantRedirect({
          tenantIdFromUrl: '850',
          userData: {
            companies: [{ id: 'other' }],
            current_company: { id: '850' },
            id: 'u1',
          },
          userId: 'u1',
        }),
      ).toBeNull()
    })

    it('userData 为 null 但有 userId 时回退到个人资料页', () => {
      const result = resolveUnauthorizedTenantRedirect({
        tenantIdFromUrl: '850',
        userData: null,
        userId: 'u3',
      })
      expect(result).toMatchObject({
        href: '/user/u3/profile/',
      })
    })

    it('userData 为 null 且无 userId 时不跳转', () => {
      expect(
        resolveUnauthorizedTenantRedirect({
          tenantIdFromUrl: '850',
          userData: null,
          userId: '',
        }),
      ).toBeNull()
    })

    it('无 tenantId 时不跳转', () => {
      expect(
        resolveUnauthorizedTenantRedirect({
          tenantIdFromUrl: '',
          userData: { companies: [{ id: '850' }], id: 'u1' },
          userId: 'u1',
        }),
      ).toBeNull()
    })

    it('userData 不可用时无 userId 不跳转', () => {
      expect(
        resolveUnauthorizedTenantRedirect({
          tenantIdFromUrl: '850',
          userData: undefined,
          userId: '',
        }),
      ).toBeNull()
    })
  })
}

// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] tenantAccessDeniedPrompt.test.js requires vitest runtime')
} else {
  const { describe, expect, it, vi, beforeEach } = await import('vitest')
  const {
    promptNavigateOnTenantAccessDenied,
    buildProfileHref,
  } = await import('./tenantAccessDeniedPrompt.js')

  describe('buildProfileHref', () => {
    it('有 userId → /user/:id/profile/', () => {
      expect(buildProfileHref('875216526544760832')).toBe('/user/875216526544760832/profile/')
    })

    it('无 userId → /profile/', () => {
      expect(buildProfileHref('')).toBe('/profile/')
      expect(buildProfileHref(null)).toBe('/profile/')
    })
  })

  describe('promptNavigateOnTenantAccessDenied', () => {
    let navigate

    beforeEach(() => {
      navigate = vi.fn()
    })

    it('用户确认 → navigated 且调用 navigate(targetHref)', async () => {
      const confirm = vi.fn(async () => true)
      const result = await promptNavigateOnTenantAccessDenied({
        targetHref: '/user/1/profile/',
        confirm,
        navigate,
      })
      expect(result).toBe('navigated')
      expect(confirm).toHaveBeenCalledTimes(1)
      expect(navigate).toHaveBeenCalledWith('/user/1/profile/')
    })

    it('用户取消 → stayed 且不 navigate（禁止静默回弹）', async () => {
      const confirm = vi.fn(async () => {
        throw false
      })
      const result = await promptNavigateOnTenantAccessDenied({
        targetHref: '/user/1/profile/',
        confirm,
        navigate,
      })
      expect(result).toBe('stayed')
      expect(navigate).not.toHaveBeenCalled()
    })

    it('空 targetHref → stayed 且不弹窗', async () => {
      const confirm = vi.fn()
      const result = await promptNavigateOnTenantAccessDenied({
        targetHref: '  ',
        confirm,
        navigate,
      })
      expect(result).toBe('stayed')
      expect(confirm).not.toHaveBeenCalled()
      expect(navigate).not.toHaveBeenCalled()
    })
  })
}

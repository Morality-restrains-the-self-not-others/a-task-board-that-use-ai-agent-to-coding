// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] navbarCollapsedVisibility.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const {
    hasTenantConsoleNavRestore,
    shouldHideNavbarWhenCollapsed,
  } = await import('./navbarCollapsedVisibility.js')

  describe('hasTenantConsoleNavRestore', () => {
    it('租户控制台路径（有「控制台导航」）为 true', () => {
      expect(hasTenantConsoleNavRestore('/tenant/874599492341493760/work-panel/')).toBe(true)
      expect(hasTenantConsoleNavRestore('/tenant/t1/projects/')).toBe(true)
      expect(hasTenantConsoleNavRestore('/projects/')).toBe(true)
    })

    it('系统管理 / 公网页 / 无恢复入口的租户页为 false', () => {
      expect(hasTenantConsoleNavRestore('/system-admin/')).toBe(false)
      expect(hasTenantConsoleNavRestore('/system-admin/?accessCode=DR2AKvP9J9')).toBe(false)
      expect(hasTenantConsoleNavRestore('/system-admin/users/')).toBe(false)
      expect(hasTenantConsoleNavRestore('/')).toBe(false)
      expect(hasTenantConsoleNavRestore('/pricing/')).toBe(false)
      expect(hasTenantConsoleNavRestore('/auth/login/')).toBe(false)
      expect(hasTenantConsoleNavRestore('/profile/')).toBe(false)
      expect(hasTenantConsoleNavRestore('/tenant/t1/profile/')).toBe(false)
      expect(hasTenantConsoleNavRestore('/tenant/t1/people/join/')).toBe(false)
    })
  })

  describe('shouldHideNavbarWhenCollapsed', () => {
    it('未收起时不隐藏', () => {
      expect(shouldHideNavbarWhenCollapsed({
        collapsed: false,
        navbarType: 'user',
        path: '/tenant/t1/work-panel/',
      })).toBe(false)
    })

    it('收起 + 租户控制台路径 → 隐藏（预期行为）', () => {
      expect(shouldHideNavbarWhenCollapsed({
        collapsed: true,
        navbarType: 'user',
        path: '/tenant/t1/work-panel/',
      })).toBe(true)
    })

    it('生产路径：navbarType 默认 user + /system-admin/ + 收起 → 不隐藏', () => {
      expect(shouldHideNavbarWhenCollapsed({
        collapsed: true,
        navbarType: 'user',
        path: '/system-admin/',
      })).toBe(false)
    })

    it('navbarType=system_admin 始终不隐藏（兼容旧测例；生产路由不传此值）', () => {
      expect(shouldHideNavbarWhenCollapsed({
        collapsed: true,
        navbarType: 'system_admin',
        path: '/',
      })).toBe(false)
    })
  })
}

// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] system_admin_route_guard_service.test.js requires vitest runtime')
} else {
  const { describe, expect, it, vi } = await import('vitest')
  const {
    SystemAdminRouteGuardService,
    isSystemSuperuser,
    resolveWorkPanelPathFromUser,
  } = await import(
    '../../../domain/auth/services/system_admin_route_guard_service.js'
  )

  describe('isSystemSuperuser', () => {
    it('T6: 字符串 True 视为超管', () => {
      expect(isSystemSuperuser({ is_superuser: 'True' })).toBe(true)
      expect(isSystemSuperuser({ is_superuser: true })).toBe(true)
      expect(isSystemSuperuser({ is_superuser: false })).toBe(false)
      expect(isSystemSuperuser({})).toBe(false)
    })
  })

  describe('resolveWorkPanelPathFromUser', () => {
    it('T2: 优先 current_company.id', () => {
      expect(
        resolveWorkPanelPathFromUser({
          current_company: { id: '850256677331562496' },
          companies: [{ id: '1' }],
        }),
      ).toBe('/tenant/850256677331562496/work-panel/')
    })

    it('T3: 无 current_company 时用 companies[0]', () => {
      expect(
        resolveWorkPanelPathFromUser({
          companies: [{ id: '850256677331562496' }],
        }),
      ).toBe('/tenant/850256677331562496/work-panel/')
    })

    it('T4: 无公司回退 onboarding 引导页', () => {
      expect(resolveWorkPanelPathFromUser({})).toBe('/onboarding/')
      expect(resolveWorkPanelPathFromUser({ companies: [] })).toBe('/onboarding/')
    })
  })

  describe('SystemAdminRouteGuardService.resolveAccess', () => {
    it('T1: 超管放行', async () => {
      const apiFetch = vi.fn(async (url) => {
        if (String(url).includes('/accounts/users/me/')) {
          return {
            ok: true,
            json: async () => ({
              is_superuser: true,
              current_company: { id: '850256677331562496' },
            }),
          }
        }
        return { ok: false, status: 404 }
      })
      const guard = new SystemAdminRouteGuardService({
        apiFetch,
        getCookie: () => '827923618451263488',
      })
      await expect(guard.resolveAccess()).resolves.toEqual({ allowed: true })
    })

    it('T2: 非超管跳转当前公司 work-panel', async () => {
      const apiFetch = vi.fn(async (url) => {
        if (String(url).includes('/accounts/users/me/')) {
          return {
            ok: true,
            json: async () => ({
              is_superuser: false,
              current_company: { id: '850256677331562496' },
            }),
          }
        }
        return { ok: false, status: 404 }
      })
      const guard = new SystemAdminRouteGuardService({
        apiFetch,
        getCookie: () => '827923618451263488',
      })
      await expect(guard.resolveAccess()).resolves.toEqual({
        allowed: false,
        redirectPath: '/tenant/850256677331562496/work-panel/',
      })
    })

    it('T5: me 失败 fail-closed 保留在系统管理页', async () => {
      const apiFetch = vi.fn(async () => ({ ok: false, status: 401 }))
      const guard = new SystemAdminRouteGuardService({
        apiFetch,
        getCookie: () => '827923618451263488',
      })
      await expect(guard.resolveAccess()).resolves.toEqual({
        allowed: false,
        redirectPath: '/system-admin/',
      })
    })
  })
}

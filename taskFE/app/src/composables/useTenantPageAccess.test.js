// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] useTenantPageAccess.test.js requires vitest runtime')
} else {
  const { describe, expect, it, vi, beforeEach } = await import('vitest')
  const { ref } = await import('vue')

  const hoisted = vi.hoisted(() => ({
    loaded: null,
    hasPage: vi.fn(),
    route: { params: { tenant: 't1' } },
  }))

  vi.mock('vue-router', () => ({
    useRoute: () => hoisted.route,
  }))

  vi.mock('./usePermissions.js', () => ({
    usePermissions: () => ({
      loaded: hoisted.loaded,
      hasPage: (...args) => hoisted.hasPage(...args),
    }),
  }))

  const { useTenantPageAccess } = await import('./useTenantPageAccess.js')

  describe('useTenantPageAccess page:* 访问判定', () => {
    beforeEach(() => {
      hoisted.hasPage.mockReset()
    })

    it('权限快照未加载时 fail-open（视为可访问），避免首帧误闪空态', () => {
      hoisted.loaded = ref(false)
      hoisted.hasPage.mockReturnValue(false)
      const { accessAllowed } = useTenantPageAccess('settings.status')
      expect(accessAllowed.value).toBe(true)
      expect(hoisted.hasPage).not.toHaveBeenCalled()
    })

    it('权限快照加载后按 page:* 判定', () => {
      hoisted.loaded = ref(true)
      hoisted.hasPage.mockReturnValue(true)
      const { accessAllowed } = useTenantPageAccess('settings.status')
      expect(accessAllowed.value).toBe(true)
      expect(hoisted.hasPage).toHaveBeenCalledWith('t1', 'settings.status')
    })

    it('无 page:* 权限时为 false（渲染 TenantPageAccessEmpty）', () => {
      hoisted.loaded = ref(true)
      hoisted.hasPage.mockReturnValue(false)
      const { accessAllowed } = useTenantPageAccess('settings.cloud')
      expect(accessAllowed.value).toBe(false)
    })

    it('无租户参数时一律不允许', () => {
      hoisted.loaded = ref(true)
      hoisted.route = { params: {} }
      const { accessAllowed } = useTenantPageAccess('settings.cloud')
      expect(accessAllowed.value).toBe(false)
    })
  })
}

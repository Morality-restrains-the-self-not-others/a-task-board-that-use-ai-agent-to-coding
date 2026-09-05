// @vitest-environment jsdom
if (!process.env.VITEST) {
  // pre-commit 会以 node 直跑已暂存 *.test.js，非 Vitest 环境下跳过。
  console.log('[skip] usePermissions.test.js requires vitest runtime')
} else {
  const { describe, expect, it, vi } = await import('vitest')

  const hoisted = vi.hoisted(() => ({
    apiFetch: vi.fn(),
  }))
  const apiFetch = hoisted.apiFetch

  // apiUtils 仅覆盖 apiFetch，其余导出保留真实实现
  vi.mock('../utils/apiUtils.js', async (importOriginal) => {
    const actual = await importOriginal()
    return { ...actual, apiFetch: (...args) => hoisted.apiFetch(...args) }
  })

  function jsonResponse(body, { ok = true } = {}) {
    return { ok, json: async () => body }
  }

  /** 每次重置模块拿到全新单例，注入指定 tenant_perms / platform roles。 */
  async function freshPermissions(tenantPerms, platformRoles = []) {
    vi.resetModules()
    apiFetch.mockReset()
    apiFetch.mockImplementation(async (url) => {
      const u = String(url)
      if (u.includes('/user-permissions/')) {
        return jsonResponse({ tenant_perms: tenantPerms })
      }
      if (u.includes('/user-roles/')) {
        return jsonResponse({ roles: platformRoles.map((r) => ({ role: r, level: 'platform' })) })
      }
      return jsonResponse({})
    })
    const { usePermissions } = await import('./usePermissions.js')
    const perms = usePermissions()
    await perms.load(apiFetch)
    return perms
  }

  describe('usePermissions v72/v73 region/page 判定', () => {
    it('hasPage 识别 legacy/view/operate 三种 page 码', async () => {
      const perms = await freshPermissions({ t1: ['page:people.access', 'page:billing:view'] })
      expect(perms.hasPage('t1', 'people.access')).toBe(true) // legacy bare
      expect(perms.hasPage('t1', 'billing')).toBe(true)       // view-qualified
      expect(perms.hasPage('t1', 'nonexistent')).toBe(false)
      expect(perms.hasPage('t2', 'people.access')).toBe(false) // 其它租户隔离
    })

    it('hasRegionView 含 legacy/view/operate；hasRegion 与其一致', async () => {
      const perms = await freshPermissions({ t1: ['region:people.access.save_actions:view'] })
      expect(perms.hasRegionView('t1', 'people.access.save_actions')).toBe(true)
      expect(perms.hasRegion('t1', 'people.access.save_actions')).toBe(true)
      expect(perms.hasRegion('t1', 'other.region')).toBe(false)
    })

    it('hasRegionOperate 只认 operate/legacy，不认 view-only', async () => {
      const permsView = await freshPermissions({ t1: ['region:r1:view'] })
      expect(permsView.hasRegionOperate('t1', 'r1')).toBe(false)

      const permsOperate = await freshPermissions({ t1: ['region:r1:operate'] })
      expect(permsOperate.hasRegionOperate('t1', 'r1')).toBe(true)

      const permsLegacy = await freshPermissions({ t1: ['region:r1'] })
      expect(permsLegacy.hasRegionOperate('t1', 'r1')).toBe(true)
    })

    it('reload 重新拉取并刷新 tenant_perms', async () => {
      const perms = await freshPermissions({ t1: ['page:people.access'] })
      expect(perms.hasPage('t1', 'people.access')).toBe(true)

      apiFetch.mockReset()
      apiFetch.mockImplementation(async (url) => {
        const u = String(url)
        if (u.includes('/user-permissions/')) {
          return jsonResponse({ tenant_perms: { t1: [] } })
        }
        if (u.includes('/user-roles/')) {
          return jsonResponse({ roles: [] })
        }
        return jsonResponse({})
      })
      await perms.reload(apiFetch)
      expect(perms.hasPage('t1', 'people.access')).toBe(false)
    })
  })
}

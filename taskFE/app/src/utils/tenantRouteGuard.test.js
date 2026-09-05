// @vitest-environment jsdom
/**
 * 路由级陈旧租户守卫（OPT-20260811-005）：
 * /tenant/:tenant/... 导航时 URL tenant 不在用户公司列表 → 清 lastActiveTenantId 并按现有公司跳转。
 */
if (!process.env.VITEST) {
  console.log('[skip] tenantRouteGuard.test.js requires vitest runtime')
} else {
  const { beforeEach, describe, expect, it, vi } = await import('vitest')
  const meResponse = (companies) => ({
    ok: true,
    status: 200,
    json: async () => ({ companies, current_company: companies[0] || null }),
  })

  const {
    isGuestTenantRoute,
    tenantSubPath,
    handleTenantRouteGuard,
    checkTenantMembership,
    resetTenantRouteGuardCache,
  } = await import('./tenantRouteGuard.js')

  beforeEach(() => {
    resetTenantRouteGuardCache()
    localStorage.clear()
  })

  describe('isGuestTenantRoute', () => {
    it('people/join 与 people/invite 跳过守卫', () => {
      expect(isGuestTenantRoute({ path: '/tenant/t1/people/join/' })).toBe(true)
      expect(isGuestTenantRoute({ path: '/tenant/t1/people/invite/' })).toBe(true)
    })

    it('带 accessCode / access_code query 的任务分享跳过守卫', () => {
      expect(isGuestTenantRoute({ path: '/tenant/t1/workspace/w1/task-detail/9/', query: { accessCode: 'x' } })).toBe(true)
      expect(isGuestTenantRoute({ path: '/tenant/t1/workspace/w1/task-detail/9/', query: { access_code: 'x' } })).toBe(true)
    })

    it('普通成员页不跳过', () => {
      expect(isGuestTenantRoute({ path: '/tenant/t1/work-panel/' })).toBe(false)
      expect(isGuestTenantRoute({ path: '/tenant/t1/billing/orders/' })).toBe(false)
    })

    it('accessCode 不能放行非任务分享页（projects / work-panel）', () => {
      expect(
        isGuestTenantRoute({ path: '/tenant/t1/projects/proj-1/', query: { accessCode: 'x' } }),
      ).toBe(false)
      expect(
        isGuestTenantRoute({ path: '/tenant/t1/work-panel/', query: { accessCode: 'x' } }),
      ).toBe(false)
      expect(
        isGuestTenantRoute({ path: '/tenant/t1/billing/orders/', query: { access_code: 'x' } }),
      ).toBe(false)
    })

    it('task-detail 无 accessCode 仍受守卫', () => {
      expect(isGuestTenantRoute({ path: '/tenant/t1/workspace/w1/task-detail/9/' })).toBe(false)
    })
  })

  describe('tenantSubPath', () => {
    it('保留子路径与尾斜杠', () => {
      expect(tenantSubPath('/tenant/123/billing/orders/')).toBe('/billing/orders/')
      expect(tenantSubPath('/tenant/123/work-panel/')).toBe('/work-panel/')
    })

    it('仅租户根路径 → /', () => {
      expect(tenantSubPath('/tenant/123/')).toBe('/')
    })

    it('非租户路径 → /', () => {
      expect(tenantSubPath('/auth/login/')).toBe('/')
    })
  })

  describe('checkTenantMembership', () => {
    it('tenant 在公司列表内 → valid=true', async () => {
      const fetchMe = vi.fn(async () => meResponse([{ id: 't1' }, { id: 't2' }]))
      const r = await checkTenantMembership({ tenantId: 't2', fetchMe })
      expect(r.known).toBe(true)
      expect(r.valid).toBe(true)
    })

    it('tenant 不在列表内 → valid=false', async () => {
      const fetchMe = vi.fn(async () => meResponse([{ id: 't1' }]))
      const r = await checkTenantMembership({ tenantId: 'stale-999', fetchMe })
      expect(r.known).toBe(true)
      expect(r.valid).toBe(false)
    })

    it('无公司 → valid=false 且 companies=[]', async () => {
      const fetchMe = vi.fn(async () => meResponse([]))
      const r = await checkTenantMembership({ tenantId: 't1', fetchMe })
      expect(r.known).toBe(true)
      expect(r.valid).toBe(false)
      expect(r.companies).toEqual([])
    })

    it('fetch 失败 → known=false（fail-open）', async () => {
      const fetchMe = vi.fn(async () => {
        throw new Error('network down')
      })
      const r = await checkTenantMembership({ tenantId: 't1', fetchMe })
      expect(r.known).toBe(false)
    })
  })

  describe('handleTenantRouteGuard', () => {
    beforeEach(() => {
      localStorage.clear()
    })

    it('非租户路径放行', async () => {
      const to = { path: '/auth/login/', params: {} }
      expect(await handleTenantRouteGuard(to, { fetchMe: vi.fn() })).toBeUndefined()
    })

    it('URL tenant 在列表内 → 放行', async () => {
      const to = { path: '/tenant/t1/work-panel/', params: { tenant: 't1' } }
      const fetchMe = vi.fn(async () => meResponse([{ id: 't1' }]))
      expect(await handleTenantRouteGuard(to, { fetchMe })).toBeUndefined()
      expect(fetchMe).toHaveBeenCalled()
    })

    it('陈旧 tenant 且有其它公司 → 清 lastActiveTenantId 并按子路径跳转首个公司', async () => {
      localStorage.setItem('lastActiveTenantId', 'stale-999')
      const to = { path: '/tenant/stale-999/billing/orders/', params: { tenant: 'stale-999' } }
      const fetchMe = vi.fn(async () => meResponse([{ id: 'c1', name: 'A' }, { id: 'c2' }]))
      const redirect = await handleTenantRouteGuard(to, { fetchMe })
      expect(redirect).toBe('/tenant/c1/billing/orders/')
      expect(localStorage.getItem('lastActiveTenantId')).toBeNull()
    })

    it('陈旧 tenant 且无公司 → 清 lastActiveTenantId 并跳 /onboarding/', async () => {
      localStorage.setItem('lastActiveTenantId', 'stale-999')
      const to = { path: '/tenant/stale-999/work-panel/', params: { tenant: 'stale-999' } }
      const fetchMe = vi.fn(async () => meResponse([]))
      const redirect = await handleTenantRouteGuard(to, { fetchMe })
      expect(redirect).toBe('/onboarding/')
      expect(localStorage.getItem('lastActiveTenantId')).toBeNull()
    })

    it('guest 路由即使 tenant 不在列表也放行', async () => {
      const to = { path: '/tenant/t2/people/join/', params: { tenant: 't2' } }
      const fetchMe = vi.fn(async () => meResponse([{ id: 't1' }]))
      expect(await handleTenantRouteGuard(to, { fetchMe })).toBeUndefined()
      expect(fetchMe).not.toHaveBeenCalled()
    })

    it('非成员打开他公司项目详情落到本公司项目列表（不保留 proj id）', async () => {
      const to = {
        path: '/tenant/877397588196749312/projects/proj_880498883115905024/',
        params: { tenant: '877397588196749312' },
      }
      const fetchMe = vi.fn(async () => meResponse([{ id: '879433200802230272' }]))
      const redirect = await handleTenantRouteGuard(to, { fetchMe })
      expect(redirect).toBe('/tenant/879433200802230272/projects/')
      expect(redirect).not.toContain('proj_880498883115905024')
    })

    it('非成员带 accessCode 打开 projects 仍被重定向（不放行）', async () => {
      const to = {
        path: '/tenant/t2/projects/proj-1/',
        params: { tenant: 't2' },
        query: { accessCode: 'x' },
      }
      const fetchMe = vi.fn(async () => meResponse([{ id: 't1' }]))
      const redirect = await handleTenantRouteGuard(to, { fetchMe })
      expect(redirect).toBe('/tenant/t1/projects/')
      expect(fetchMe).toHaveBeenCalled()
    })

    it('非成员带 accessCode 打开 work-panel 被重定向到 onboarding（无公司）', async () => {
      const to = {
        path: '/tenant/stale-1/work-panel/',
        params: { tenant: 'stale-1' },
        query: { accessCode: 'x' },
      }
      const fetchMe = vi.fn(async () => meResponse([]))
      expect(await handleTenantRouteGuard(to, { fetchMe })).toBe('/onboarding/')
    })

    it('fetch 失败（fail-open）→ 放行', async () => {
      const to = { path: '/tenant/stale-1/work-panel/', params: { tenant: 'stale-1' } }
      const fetchMe = vi.fn(async () => {
        throw new Error('boom')
      })
      expect(await handleTenantRouteGuard(to, { fetchMe })).toBeUndefined()
    })
  })
}

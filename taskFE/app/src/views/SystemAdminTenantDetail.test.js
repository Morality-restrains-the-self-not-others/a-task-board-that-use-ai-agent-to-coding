// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] SystemAdminTenantDetail.test.js requires vitest runtime')
} else {
  const { describe, it, expect, vi, beforeEach } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { createRouter, createMemoryHistory } = await import('vue-router')
  const { adminRoutes } = await import('../router/adminRoutes.js')

  const { apiFetch } = vi.hoisted(() => ({
    apiFetch: vi.fn(),
  }))

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch,
  }))

  vi.mock('vue-router', async () => {
    const actual = await vi.importActual('vue-router')
    return {
      ...actual,
      useRoute: () => ({ params: { tenantId: 'c1' } }),
    }
  })

  const { default: Page } = await import('./SystemAdminTenantDetail.vue')

  function okJson(body) {
    return {
      ok: true,
      json: async () => body,
      headers: { get: () => null },
    }
  }

  function failJson(status, body, traceId) {
    return {
      ok: false,
      status,
      json: async () => body,
      headers: { get: (h) => (String(h).toLowerCase() === 'x-trace-id' ? traceId : null) },
    }
  }

  describe('SystemAdminTenantDetail', () => {
    beforeEach(() => {
      apiFetch.mockReset()
      apiFetch.mockImplementation(async (url) => {
        const u = String(url)
        if (u.includes('/accounts/admin/tenants/c1/')) {
          return okJson({
            id: 'c1',
            name: '我的公司',
            creator_id: 'u1',
            email: 'a@x.com',
            phone: '13800000000',
            created_at: '2026-08-01T00:00:00Z',
          })
        }
        if (u.includes('/tenant-quotas/tenant_id/c1/')) {
          return okJson({
            task_post_quota: 12,
            task_post_quota_gifted: 2,
            task_post_quota_purchased: 10,
            gitlab_disk_gb: 5,
            gitlab_disk_used_gb: 1,
            gitlab_traffic_prepaid_gb: 8,
            gitlab_traffic_used_gb: 0.5,
          })
        }
        if (u.includes('/tenant-workspaces/tenant_id/c1/')) {
          return okJson({
            items: [{ id: 'ws1', name: '默认空间', is_default: true, created_at: '2026-08-02T00:00:00Z' }],
            total: 1,
          })
        }
        if (u.includes('/api/system-admin/orders/')) {
          return okJson({
            orders: [{
              id: 'o1',
              order_number: 'ORD-1',
              status: 'paid',
              total_yuan: '9.90',
              payment_method: 'wechat',
              created_at: '2026-08-03T00:00:00Z',
            }],
            total: 1,
          })
        }
        return failJson(404, { detail: 'unexpected ' + u }, 'x')
      })
    })

    it('loads header, quotas, workspaces and orders in parallel', async () => {
      const wrapper = mount(Page)
      await flushPromises()
      expect(wrapper.get('[data-testid="system-admin-tenant-header"]').text()).toContain('我的公司')
      expect(wrapper.get('[data-testid="system-admin-tenant-header"]').text()).toContain('c1')
      expect(wrapper.get('[data-testid="task-post-quota-card"]').text()).toContain('12')
      expect(wrapper.get('[data-testid="system-admin-tenant-workspaces-table"]').text()).toContain('默认空间')
      expect(wrapper.get('[data-testid="system-admin-tenant-orders-table"]').text()).toContain('ORD-1')
      expect(wrapper.get('[data-testid="system-admin-tenant-orders-table"]').text()).toContain('已支付')
      expect(wrapper.get('[data-testid="system-admin-tenant-detail-back"]').attributes('href')).toBe(
        '/system-admin/users/?tab=tenants',
      )
      expect(wrapper.get('[data-testid="system-admin-tenant-orders-all"]').attributes('href')).toBe(
        '/system-admin/order-records/?tenant_id=c1',
      )
      const urls = apiFetch.mock.calls.map((c) => String(c[0]))
      expect(urls.some((u) => u.includes('/accounts/admin/tenants/c1/'))).toBe(true)
      expect(urls.some((u) => u.includes('/tenant-quotas/tenant_id/c1/'))).toBe(true)
      expect(urls.some((u) => u.includes('/tenant-workspaces/tenant_id/c1/'))).toBe(true)
      expect(urls.some((u) => u.includes('/api/system-admin/orders/?') && u.includes('tenant_id=c1'))).toBe(true)
    })

    it('shows per-block error with data-traceId without hiding other blocks', async () => {
      apiFetch.mockImplementation(async (url) => {
        const u = String(url)
        if (u.includes('/accounts/admin/tenants/')) {
          return okJson({ id: 'c1', name: '我的公司', creator_id: 'u1' })
        }
        if (u.includes('/tenant-quotas/')) {
          return failJson(502, { detail: 'quota upstream failed', trace_id: 'web-quota-fail-01' }, 'web-quota-fail-01')
        }
        if (u.includes('/tenant-workspaces/')) {
          return okJson({ items: [], total: 0 })
        }
        if (u.includes('/api/system-admin/orders/')) {
          return okJson({ orders: [], total: 0 })
        }
        return failJson(404, { detail: 'no' }, 'x')
      })
      const wrapper = mount(Page)
      await flushPromises()
      const err = wrapper.get('[data-testid="system-admin-tenant-quotas-error"]')
      expect(err.text()).toContain('quota upstream failed')
      expect(err.attributes('data-traceid') || err.attributes('data-traceId')).toBe('web-quota-fail-01')
      expect(wrapper.get('[data-testid="system-admin-tenant-header"]').text()).toContain('我的公司')
      expect(wrapper.get('[data-testid="system-admin-tenant-workspaces-empty"]').exists()).toBe(true)
      expect(wrapper.get('[data-testid="system-admin-tenant-orders-empty"]').exists()).toBe(true)
    })

    it('shows header error with data-traceId when tenant is missing', async () => {
      apiFetch.mockImplementation(async (url) => {
        const u = String(url)
        if (u.includes('/accounts/admin/tenants/')) {
          return failJson(404, { detail: 'tenant not found', trace_id: 'web-tenant-404-01' }, 'web-tenant-404-01')
        }
        if (u.includes('/tenant-quotas/')) return okJson({})
        if (u.includes('/tenant-workspaces/')) return okJson({ items: [], total: 0 })
        if (u.includes('/api/system-admin/orders/')) return okJson({ orders: [], total: 0 })
        return failJson(404, { detail: 'no' }, 'x')
      })
      const wrapper = mount(Page)
      await flushPromises()
      const err = wrapper.get('[data-testid="system-admin-tenant-header-error"]')
      expect(err.text()).toContain('tenant not found')
      expect(err.attributes('data-traceid') || err.attributes('data-traceId')).toBe('web-tenant-404-01')
      expect(wrapper.find('[data-testid="system-admin-tenant-header"]').exists()).toBe(false)
    })

    it('shows workspace pagination when total exceeds limit (OPT-20260825-028)', async () => {
      apiFetch.mockImplementation(async (url) => {
        const u = String(url)
        if (u.includes('/accounts/admin/tenants/c1/')) {
          return okJson({ id: 'c1', name: '我的公司', creator_id: 'u1' })
        }
        if (u.includes('/tenant-quotas/tenant_id/c1/')) {
          return okJson({ task_post_quota: 12 })
        }
        if (u.includes('/tenant-workspaces/tenant_id/c1/')) {
          const m = u.match(/offset=(\d+)/)
          const offset = m ? Number(m[1]) : 0
          return okJson({
            items: Array.from({ length: 50 }, (_, i) => ({
              id: `ws${offset + i}`,
              name: `空间${offset + i}`,
              is_default: i === 0,
              created_at: '2026-08-02T00:00:00Z',
            })),
            total: 80,
          })
        }
        if (u.includes('/api/system-admin/orders/')) {
          return okJson({ orders: [{ id: 'o1', order_number: 'ORD-1', status: 'paid', total_yuan: '9.90', payment_method: 'wechat', created_at: '2026-08-03T00:00:00Z' }], total: 1 })
        }
        return failJson(404, { detail: 'no' }, 'x')
      })
      const wrapper = mount(Page)
      await flushPromises()
      expect(wrapper.get('[data-testid="system-admin-tenant-workspaces"]').text()).toContain('共 80 个')
      const next = wrapper.get('[data-testid="system-admin-tenant-workspaces-next"]')
      expect(next.attributes('disabled')).toBeUndefined()
      expect(wrapper.get('[data-testid="system-admin-tenant-workspaces-page"]').text()).toContain('第 1 页')
      await next.trigger('click')
      await flushPromises()
      const urls = apiFetch.mock.calls.map((c) => String(c[0]))
      expect(urls.some((u) => u.includes('/tenant-workspaces/') && u.includes('offset=50'))).toBe(true)
      expect(wrapper.get('[data-testid="system-admin-tenant-workspaces-page"]').text()).toContain('第 2 页')
      expect(wrapper.find('[data-testid="system-admin-tenant-workspaces-prev"]').exists()).toBe(true)
      wrapper.unmount()
    })

    it('registers /system-admin/tenants/:tenantId/ route', async () => {
      const router = createRouter({
        history: createMemoryHistory(),
        routes: adminRoutes,
      })
      await router.push('/system-admin/tenants/c1/')
      expect(router.currentRoute.value.name).toBe('system_admin_tenant_detail')
      expect(router.currentRoute.value.params.tenantId).toBe('c1')
    })
  })
}

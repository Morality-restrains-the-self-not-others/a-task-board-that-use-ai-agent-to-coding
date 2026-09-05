// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] SystemAdminTenantsPanel.test.js requires vitest runtime')
} else {
  const { describe, it, expect, vi, beforeEach } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')

  const { apiFetch } = vi.hoisted(() => ({
    apiFetch: vi.fn(),
  }))

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch,
  }))

  const { default: Panel } = await import('./SystemAdminTenantsPanel.vue')

  function okJson(body) {
    return {
      ok: true,
      json: async () => body,
      headers: { get: () => null },
    }
  }

  describe('SystemAdminTenantsPanel', () => {
    beforeEach(() => {
      apiFetch.mockReset()
    })

    it('renders tenant rows from items', async () => {
      apiFetch.mockResolvedValue(okJson({
        items: [{ id: 'c1', name: 'Acme', creator_id: 'u1', email: 'a@x.com', phone: '13800000000', created_at: '2026-08-01T00:00:00Z' }],
        total: 1,
      }))
      const wrapper = mount(Panel)
      await flushPromises()
      expect(wrapper.get('[data-testid="system-admin-tenants-table"]').text()).toContain('Acme')
      expect(wrapper.get('[data-testid="system-admin-tenants-table"]').text()).toContain('c1')
      const link = wrapper.get('[data-testid="system-admin-tenant-name-link"]')
      expect(link.text()).toBe('Acme')
      expect(link.attributes('href')).toBe('/system-admin/tenants/c1/')
      expect(apiFetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/system-admin/accounts/admin/tenants/'),
        expect.any(Object),
      )
    })

    it('renders a grant-points href per row (OPT-20260825-026)', async () => {
      apiFetch.mockResolvedValue(okJson({
        items: [{ id: 'c1', name: 'Acme', creator_id: 'u1', email: 'a@x.com', phone: '13800000000', created_at: '2026-08-01T00:00:00Z' }],
        total: 1,
      }))
      const wrapper = mount(Panel)
      await flushPromises()
      const grant = wrapper.get('[data-testid="system-admin-tenant-grant-link"]')
      expect(grant.text()).toBe('赠送资源')
      expect(grant.attributes('href')).toBe('/system-admin/grant-points/?tenant_id=c1')
      wrapper.unmount()
    })

    it('renders em dash without a link when name is empty', async () => {
      apiFetch.mockResolvedValue(okJson({
        items: [{ id: 'c2', name: '', creator_id: 'u1', email: '', phone: '', created_at: '2026-08-01T00:00:00Z' }],
        total: 1,
      }))
      const wrapper = mount(Panel)
      await flushPromises()
      expect(wrapper.find('[data-testid="system-admin-tenant-name-link"]').exists()).toBe(false)
      expect(wrapper.get('[data-testid="system-admin-tenants-table"]').text()).toContain('—')
    })

    it('shows empty state when items is empty', async () => {
      apiFetch.mockResolvedValue(okJson({ items: [], total: 0 }))
      const wrapper = mount(Panel)
      await flushPromises()
      expect(wrapper.get('[data-testid="system-admin-tenants-empty"]').text()).toContain('暂无租户')
    })

    it('shows error with data-traceId on failure', async () => {
      apiFetch.mockResolvedValue({
        ok: false,
        status: 502,
        json: async () => ({ detail: 'upstream failed', trace_id: 'web-tenants-fail-01' }),
        headers: { get: (h) => (String(h).toLowerCase() === 'x-trace-id' ? 'web-tenants-fail-01' : null) },
      })
      const wrapper = mount(Panel)
      await flushPromises()
      const err = wrapper.get('[data-testid="system-admin-tenants-error"]')
      expect(err.text()).toContain('upstream failed')
      expect(err.attributes('data-traceid') || err.attributes('data-traceId')).toBe('web-tenants-fail-01')
    })
  })
}

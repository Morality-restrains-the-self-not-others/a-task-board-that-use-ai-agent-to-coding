// @vitest-environment jsdom
/**
 * 清库/删公司后访问陈旧 /tenant/:id/settings/company/：
 * companies/current 404 → 清 lastActiveTenantId → 跳转 onboarding 或其它公司设置。
 */
if (!process.env.VITEST) {
  console.log('[skip] TenantCompanySettings.stale404.test.js requires vitest runtime')
} else {
  const { beforeEach, describe, expect, it, vi } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')

  const { apiFetchMock, routeMock, routerMock } = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
    routeMock: { params: { tenant: '874599492341493760' } },
    routerMock: { replace: vi.fn(), push: vi.fn() },
  }))

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch: (...args) => apiFetchMock(...args),
  }))

  vi.mock('vue-router', () => ({
    useRoute: () => routeMock,
    useRouter: () => routerMock,
  }))

  describe('TenantCompanySettings — companies/current 404 陈旧租户恢复', () => {
    beforeEach(() => {
      vi.clearAllMocks()
      localStorage.clear()
      routeMock.params = { tenant: '874599492341493760' }
      localStorage.setItem('lastActiveTenantId', '874599492341493760')
    })

    it('无公司时：清 lastActiveTenantId 并 replace /onboarding/', async () => {
      apiFetchMock.mockImplementation(async (url) => {
        if (String(url).includes('/accounts/companies/current/')) {
          return {
            ok: false,
            status: 404,
            json: async () => ({ detail: 'not found', trace_id: '256779af-09fa-45b1-93f5-b7e26787b976' }),
          }
        }
        if (String(url).includes('/accounts/users/me/')) {
          return {
            ok: true,
            status: 200,
            json: async () => ({ companies: [], current_company: null }),
          }
        }
        return { ok: false, status: 500, json: async () => ({}) }
      })

      const { default: TenantCompanySettings } = await import('./TenantCompanySettings.vue')
      mount(TenantCompanySettings)
      await flushPromises()

      expect(localStorage.getItem('lastActiveTenantId')).toBeNull()
      expect(routerMock.replace).toHaveBeenCalledWith('/onboarding/')
    })

    it('仍有其它公司时：跳到首个公司的 settings/company', async () => {
      apiFetchMock.mockImplementation(async (url) => {
        if (String(url).includes('/accounts/companies/current/')) {
          return { ok: false, status: 404, json: async () => ({ detail: 'not found' }) }
        }
        if (String(url).includes('/accounts/users/me/')) {
          return {
            ok: true,
            status: 200,
            json: async () => ({
              companies: [{ id: '874600000000000001', name: '新公司' }],
              current_company: { id: '874600000000000001' },
            }),
          }
        }
        return { ok: false, status: 500, json: async () => ({}) }
      })

      const { default: TenantCompanySettings } = await import('./TenantCompanySettings.vue')
      mount(TenantCompanySettings)
      await flushPromises()

      expect(localStorage.getItem('lastActiveTenantId')).toBeNull()
      expect(routerMock.replace).toHaveBeenCalledWith(
        '/tenant/874600000000000001/settings/company/',
      )
    })

    it('非 404 错误仍展示错误文案并带 data-traceId，不强制跳转', async () => {
      apiFetchMock.mockImplementation(async (url) => {
        if (String(url).includes('/accounts/companies/current/')) {
          return {
            ok: false,
            status: 500,
            json: async () => ({ detail: 'db down', trace_id: 'trace-500' }),
          }
        }
        return { ok: false, status: 500, json: async () => ({}) }
      })

      const { default: TenantCompanySettings } = await import('./TenantCompanySettings.vue')
      const wrapper = mount(TenantCompanySettings)
      await flushPromises()

      expect(routerMock.replace).not.toHaveBeenCalled()
      expect(wrapper.text()).toMatch(/db down|加载公司信息失败/)
      const errEl = wrapper.find('p.text-red-600')
      expect(errEl.attributes('data-traceid') || errEl.attributes('data-traceId')).toBe('trace-500')
    })
  })
}

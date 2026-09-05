// @vitest-environment jsdom
/**
 * OPT-20260819-042：账单首页「购买资源」「查看全部」必须是真实 <a href>，
 * 不得用 router-link（元规则 49 禁止点击拦截，避免守卫回弹打架）。
 */
if (!process.env.VITEST) {
  console.log('[skip] BillingDashboard.realHref.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { describe, expect, it, vi, beforeEach } = await import('vitest')

  const hoistedMocks = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
    routeMock: {
      params: { tenant: '877397588196749312' },
      path: '/tenant/877397588196749312/billing/',
      query: {},
    },
  }))

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch: hoistedMocks.apiFetchMock,
  }))

  vi.mock('../utils/cookieUtils.js', () => ({
    getCookie: () => '',
  }))

  vi.mock('vue-router', () => ({
    useRoute: () => hoistedMocks.routeMock,
    useRouter: () => ({ push: vi.fn() }),
  }))

  const BillingDashboard = (await import('./BillingDashboard.vue')).default

  function jsonOk(body) {
    return { ok: true, status: 200, json: async () => body }
  }

  function stubApis() {
    hoistedMocks.apiFetchMock.mockImplementation(async (url) => {
      const u = String(url || '')
      if (u.includes('/balance/')) return jsonOk({ cents: '0' })
      if (u.includes('/statistics/')) return jsonOk({ user_recharge_points: 55 })
      if (u.includes('/quotas/')) return jsonOk({ task_post_quota: 0 })
      if (u.includes('/transactions/')) return jsonOk({ results: [], total: 0, page: 1, page_size: 5 })
      return jsonOk([])
    })
  }

  function mountDashboard() {
    return mount(BillingDashboard, {
      global: {
        stubs: {
          // 若误用 router-link，测试将断言失败（真实 a 不应出现）
          'router-link': { props: ['to'], template: '<a data-router-link-stub="1"><slot /></a>' },
        },
      },
    })
  }

  describe('BillingDashboard 导航用真实 a href', () => {
    beforeEach(() => {
      hoistedMocks.apiFetchMock.mockReset()
      stubApis()
    })

    it('「购买资源」为原生 <a href> 指向订单创建页', async () => {
      const wrapper = mountDashboard()
      await flushPromises()
      const buy = wrapper.find('a[href="/tenant/877397588196749312/billing/orders/create/"]')
      expect(buy.exists()).toBe(true)
      expect(buy.text()).toContain('购买资源')
      expect(buy.attributes('href')).toBe('/tenant/877397588196749312/billing/orders/create/')
      expect(buy.attributes('data-router-link-stub')).toBeUndefined()
    })

    it('「查看全部」为原生 <a href> 指向交易列表页', async () => {
      const wrapper = mountDashboard()
      await flushPromises()
      const all = wrapper.find('a[href="/tenant/877397588196749312/billing/transactions/"]')
      expect(all.exists()).toBe(true)
      expect(all.text()).toContain('查看全部')
      expect(all.attributes('href')).toBe('/tenant/877397588196749312/billing/transactions/')
      expect(all.attributes('data-router-link-stub')).toBeUndefined()
    })
  })
}

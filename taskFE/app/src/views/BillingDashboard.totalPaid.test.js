// @vitest-environment jsdom
/**
 * 账单首页「累计支付」：用户订单实付后不得显示 0.00。
 * 回归：统计接口仍发 *_points 时，前端须读入并按元展示。
 */
if (!process.env.VITEST) {
  console.log('[skip] BillingDashboard.totalPaid.test.js requires vitest runtime')
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

  function stubApis(stats) {
    hoistedMocks.apiFetchMock.mockImplementation(async (url) => {
      const u = String(url || '')
      if (u.includes('/balance/')) return jsonOk({ cents: '0' })
      if (u.includes('/statistics/')) return jsonOk(stats)
      if (u.includes('/quotas/')) return jsonOk({ task_post_quota: 0 })
      if (u.includes('/transactions/')) {
        return jsonOk({
          results: [{
            id: 'txn-pay-1',
            created_at: '2026-08-19 21:00:00',
            transaction_type: 'consumption',
            points_source_type: 'resource_purchase',
            amount_points: 55,
            change_display: '-0.55 元',
            description: '订单资源购买',
          }],
          total: 1,
          page: 1,
          page_size: 5,
        })
      }
      return jsonOk([])
    })
  }

  function mountDashboard() {
    return mount(BillingDashboard, {
      global: {
        stubs: {
          'router-link': {
            props: ['to'],
            template: '<a :href="typeof to === \'string\' ? to : \'\'"><slot /></a>',
          },
        },
      },
    })
  }

  describe('BillingDashboard 累计支付', () => {
    beforeEach(() => {
      hoistedMocks.apiFetchMock.mockReset()
    })

    it('统计仅返回 user_recharge_points=55 时累计支付显示 0.55', async () => {
      stubApis({
        total_consumption_points: 55,
        monthly_consumption_points: 55,
        user_recharge_points: 55,
      })
      const wrapper = mountDashboard()
      await flushPromises()
      expect(wrapper.text()).toContain('累计支付')
      expect(wrapper.text()).toContain('0.55')
      expect(wrapper.text()).toContain('-0.55 元')
    })

    it('统计返回 user_recharge_cents=55 时累计支付显示 0.55', async () => {
      stubApis({
        total_consumption_cents: 55,
        monthly_consumption_cents: 55,
        user_recharge_cents: 55,
      })
      const wrapper = mountDashboard()
      await flushPromises()
      expect(wrapper.text()).toContain('0.55')
    })
  })
}

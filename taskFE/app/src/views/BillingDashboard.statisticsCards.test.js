// @vitest-environment jsdom
/**
 * 账单首页累计消耗 / 累计支付：按分转元展示，并标明口径差异。
 */
if (!process.env.VITEST) {
  console.log('[skip] BillingDashboard.statisticsCards.test.js requires vitest runtime')
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
    return {
      ok: true,
      status: 200,
      json: async () => body,
    }
  }

  function stubApis(stats) {
    hoistedMocks.apiFetchMock.mockImplementation(async (url) => {
      const u = String(url || '')
      if (u.includes('/balance/')) return jsonOk({ cents: '0' })
      if (u.includes('/statistics/')) return jsonOk(stats)
      if (u.includes('/quotas/')) return jsonOk({ task_post_quota: 0 })
      if (u.includes('/transactions/')) return jsonOk({ results: [], total: 0, page: 1, page_size: 5 })
      return jsonOk({})
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

  describe('BillingDashboard 累计消耗与累计支付', () => {
    beforeEach(() => {
      hoistedMocks.apiFetchMock.mockReset()
    })

    it('将 points（分）展示为元，且两卡口径文案不同', async () => {
      stubApis({
        total_consumption_points: 12345,
        monthly_consumption_points: 100,
        user_recharge_points: 20000,
      })
      const wrapper = mountDashboard()
      await flushPromises()
      const consume = wrapper.get('[data-testid="stat-total-consumption"]')
      const pay = wrapper.get('[data-testid="stat-total-payment"]')
      expect(consume.text()).toContain('累计消耗')
      expect(consume.text()).toContain('123.45')
      expect(consume.text()).toMatch(/用量|配额|扣费/)
      expect(consume.text()).toContain('不含已退款')
      expect(consume.text()).toContain('已取消')
      expect(pay.text()).toContain('累计支付')
      expect(pay.text()).toContain('200.00')
      expect(pay.text()).toContain('不含后台赠送')
      expect(pay.text()).toMatch(/订单|PayPal|微信/)
    })

    it('仅有遗留 cents 字段时也不再恒为 0.00', async () => {
      stubApis({
        total_consumption_cents: 55,
        monthly_consumption_cents: 55,
        user_recharge_cents: 155,
      })
      const wrapper = mountDashboard()
      await flushPromises()
      expect(wrapper.get('[data-testid="stat-total-consumption"]').text()).toContain('0.55')
      expect(wrapper.get('[data-testid="stat-total-payment"]').text()).toContain('1.55')
    })
  })
}

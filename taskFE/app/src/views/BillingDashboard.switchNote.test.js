// @vitest-environment jsdom
/**
 * BillingDashboard：资源单价卡片已从账单首页移除（回归）
 * - 加载中/加载后均不渲染 BillingDashboardPricing
 * - 单价文案不出现在账单首页；配额/消耗/最近交易卡片不受影响
 */
if (!process.env.VITEST) {
  console.log('[skip] BillingDashboard.switchNote.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { describe, expect, it, vi, beforeEach, afterEach } = await import('vitest')

  const hoistedMocks = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
    routeMock: {
      params: { tenant: '850256677331562496' },
      path: '/tenant/850256677331562496/billing/',
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

  function stubDefaultApis(pricingBody) {
    hoistedMocks.apiFetchMock.mockImplementation(async (url) => {
      const u = String(url || '')
      if (u.includes('tenant_pricing_view') || u.includes('order-pricing')) {
        return jsonOk({ pricing: pricingBody })
      }
      if (u.includes('/balance/')) {
        return jsonOk({ points: '100' })
      }
      if (u.includes('/statistics/')) {
        return jsonOk({
          total_consumption_points: 0,
          monthly_consumption_points: 0,
          user_recharge_points: 0,
        })
      }
      if (u.includes('/transactions/')) {
        return jsonOk([])
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

  describe('BillingDashboard 定价卡片移除（资源单价不再展示）', () => {
    beforeEach(() => {
      hoistedMocks.apiFetchMock.mockReset()
    })

    afterEach(() => {
      vi.restoreAllMocks()
    })

    it('加载中与加载后均不渲染定价卡片', async () => {
      hoistedMocks.apiFetchMock.mockImplementation(() => new Promise(() => {}))
      const wrapper = mountDashboard()
      expect(wrapper.find('[data-alias="BillingDashboardPricing"]').exists()).toBe(false)

      stubDefaultApis({
        task_post: { price_yuan: '3' },
      })
      await flushPromises()
      expect(wrapper.find('[data-alias="BillingDashboardPricing"]').exists()).toBe(false)
    })

    it('单价文案（当前资源单价/元每帖/同区域内网不计费）不出现在账单首页', async () => {
      stubDefaultApis({
        task_post: { price_yuan: '3' },
        gitlab_disk: { price_yuan: '2' },
        gitlab_traffic: { price_yuan: '5' },
      })

      const wrapper = mountDashboard()
      await flushPromises()

      const text = wrapper.text()
      expect(text).not.toContain('当前资源单价')
      expect(text).not.toContain('元/帖')
      expect(text).not.toContain('同区域内网流量不计费')
      // 资源配额卡片仍在（只移除单价卡，配额/消耗/交易不受影响）
      expect(text).toContain('资源配额')
      expect(text).toContain('最近交易')
    })

    it('does not show frozen cash wallet on billing home even when frozen_balance > 0', async () => {
      hoistedMocks.apiFetchMock.mockImplementation(async (url) => {
        const u = String(url || '')
        if (u.includes('/balance/')) {
          return jsonOk({ cents: '0', frozen_balance: 500 })
        }
        if (u.includes('/statistics/')) {
          return jsonOk({
            total_consumption_points: 0,
            monthly_consumption_points: 0,
            user_recharge_points: 0,
          })
        }
        if (u.includes('/transactions/')) return jsonOk([])
        if (u.includes('/quotas/')) return jsonOk({ task_post_quota: 0 })
        return jsonOk({})
      })
      const wrapper = mountDashboard()
      await flushPromises()
      expect(wrapper.text()).not.toContain('冻结金额')
      expect(wrapper.text()).not.toContain('账户余额')
    })
  })
}

// @vitest-environment jsdom
/**
 * 账单页任务帖配额区分赠送 / 购买。
 */
if (!process.env.VITEST) {
  console.log('[skip] BillingDashboard.quotaSplit.test.js requires vitest runtime')
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

  describe('BillingDashboard 任务帖赠送/购买拆分', () => {
    beforeEach(() => {
      hoistedMocks.apiFetchMock.mockReset()
      hoistedMocks.apiFetchMock.mockImplementation(async (url) => {
        const u = String(url || '')
        if (u.includes('/balance/')) return jsonOk({ cents: '0' })
        if (u.includes('/statistics/')) {
          return jsonOk({
            total_consumption_points: 0,
            monthly_consumption_points: 0,
            user_recharge_points: 0,
          })
        }
        if (u.includes('/transactions/')) return jsonOk({ results: [] })
        if (u.includes('/quotas/')) {
          return jsonOk({
            task_post_quota: 83,
            task_post_quota_gifted: 50,
            task_post_quota_purchased: 33,
          })
        }
        return jsonOk({})
      })
    })

    it('展示合计以及赠送与购买剩余', async () => {
      const wrapper = mount(BillingDashboard)
      await flushPromises()
      const card = wrapper.find('[data-testid="task-post-quota-card"]')
      expect(card.text()).toContain('83')
      const split = wrapper.find('[data-testid="task-post-quota-split"]')
      expect(split.text()).toContain('赠送 50 帖')
      expect(split.text()).toContain('购买 33 帖')
      wrapper.unmount()
    })

    it('无拆分字段时不渲染拆分行', async () => {
      hoistedMocks.apiFetchMock.mockImplementation(async (url) => {
        const u = String(url || '')
        if (u.includes('/balance/')) return jsonOk({ cents: '0' })
        if (u.includes('/statistics/')) {
          return jsonOk({
            total_consumption_points: 0,
            monthly_consumption_points: 0,
            user_recharge_points: 0,
          })
        }
        if (u.includes('/transactions/')) return jsonOk({ results: [] })
        if (u.includes('/quotas/')) return jsonOk({ task_post_quota: 10 })
        return jsonOk({})
      })
      const wrapper = mount(BillingDashboard)
      await flushPromises()
      expect(wrapper.find('[data-testid="task-post-quota-split"]').exists()).toBe(false)
      wrapper.unmount()
    })
  })
}

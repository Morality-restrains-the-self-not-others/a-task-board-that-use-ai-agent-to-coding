// @vitest-environment jsdom
/**
 * BillingDashboard 资源配额：GitLab 磁盘/流量按区展示「赠送/购买」拆分（OPT-20260820-041）。
 */
if (!process.env.VITEST) {
  console.log('[skip] BillingDashboard.gitlabQuotaSplit.test.js requires vitest runtime')
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

  describe('BillingDashboard GitLab 配额赠送/购买拆分', () => {
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
        if (u.includes('/quotas/')) {
          return jsonOk({
            task_post_quota: 10,
            gitlab_resources: [
              {
                region: 'tencent-sh-1',
                region_name: '腾讯上海一区',
                disk_gb: 100,
                traffic_prepaid_gb: 500,
                disk_gifted_gb: 30,
                disk_purchased_gb: 70,
                traffic_gifted_gb: 100,
                traffic_purchased_gb: 400,
                gitlab_web_url: 'https://gitlab-tencent-sh-1.daydaymoney.com',
              },
            ],
          })
        }
        if (u.includes('/transactions/')) {
          return jsonOk({ results: [], total: 0, page: 1, page_size: 5 })
        }
        return jsonOk({})
      })
    })

    it('区域卡片展示磁盘/流量「赠送/购买」拆分', async () => {
      const wrapper = mount(BillingDashboard)
      await flushPromises()

      const rows = wrapper.get('[data-testid="gitlab-region-quota-rows"]')
      const text = rows.text()
      expect(text).toContain('0 / 100')
      expect(text).toContain('赠送 30 · 购买 70')
      expect(text).toContain('0 / 500')
      expect(text).toContain('赠送 100 · 购买 400')

      wrapper.unmount()
    })

    it('后端未下发拆分字段（旧版本）时不渲染赠送/购买文案', async () => {
      hoistedMocks.apiFetchMock.mockImplementation(async (url) => {
        const u = String(url || '')
        if (u.includes('/balance/')) return jsonOk({ cents: '0' })
        if (u.includes('/statistics/')) {
          return jsonOk({ total_consumption_points: 0, monthly_consumption_points: 0, user_recharge_points: 0 })
        }
        if (u.includes('/quotas/')) {
          return jsonOk({
            task_post_quota: 10,
            gitlab_resources: [
              { region: 'tencent-sh-1', region_name: '腾讯上海一区', disk_gb: 100, traffic_prepaid_gb: 500 },
            ],
          })
        }
        if (u.includes('/transactions/')) return jsonOk({ results: [], total: 0, page: 1, page_size: 5 })
        return jsonOk({})
      })

      const wrapper = mount(BillingDashboard)
      await flushPromises()

      const rows = wrapper.get('[data-testid="gitlab-region-quota-rows"]')
      expect(rows.text()).toContain('0 / 100')
      expect(rows.text()).not.toContain('赠送')

      wrapper.unmount()
    })
  })
}

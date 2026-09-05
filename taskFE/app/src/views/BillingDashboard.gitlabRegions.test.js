// @vitest-environment jsdom
/**
 * BillingDashboard 资源配额：多区域租户按区展示磁盘/流量（OPT-20260818-022）。
 */
if (!process.env.VITEST) {
  console.log('[skip] BillingDashboard.gitlabRegions.test.js requires vitest runtime')
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

  describe('BillingDashboard 按区域配额', () => {
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
            gitlab_disk_gb: 2,
            gitlab_traffic_prepaid_gb: 4,
            gitlab_disk_used_gb: 0.5,
            gitlab_traffic_used_gb: 0.3,
            gitlab_resources: [
              {
                region: 'tencent-shanghai-5',
                region_name: '腾讯上海区',
                disk_gb: 1,
                disk_used_gb: 0.25,
                disk_expires_at: '2026-12-31',
                traffic_prepaid_gb: 2,
                traffic_used_gb: 0.1,
                gitlab_web_url: 'https://gitlab.daydaymoney.com',
              },
              {
                region: 'tencent-sh-1',
                region_name: '腾讯上海一区',
                disk_gb: 1,
                disk_used_gb: 0.4,
                traffic_prepaid_gb: 2,
                traffic_used_gb: 0.2,
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

    it('两区域均按区展示磁盘/流量已用与配额', async () => {
      const wrapper = mount(BillingDashboard)
      await flushPromises()

      const rows = wrapper.get('[data-testid="gitlab-region-quota-rows"]')
      const text = rows.text()
      expect(text).toContain('腾讯上海区')
      expect(text).toContain('腾讯上海一区')
      expect(text).toContain('0.25 / 1')
      expect(text).toContain('0.4 / 1')
      expect(text).toContain('0.1 / 2')
      expect(text).toContain('0.2 / 2')
      expect(text).toContain('到期：2026-12-31')
      expect(wrapper.find('[data-testid="gitlab-aggregate-disk-card"]').exists()).toBe(false)
      expect(wrapper.find('[data-testid="gitlab-aggregate-traffic-card"]').exists()).toBe(false)
      const links = rows.findAll('a')
      expect(links.some((a) => a.attributes('href') === 'https://gitlab-tencent-sh-1.daydaymoney.com')).toBe(true)

      wrapper.unmount()
    })

    it('无 gitlab_resources 时不渲染按区区块，回退卡显示已用/配额', async () => {
      hoistedMocks.apiFetchMock.mockImplementation(async (url) => {
        const u = String(url || '')
        if (u.includes('/balance/')) return jsonOk({ cents: '0' })
        if (u.includes('/statistics/')) {
          return jsonOk({ total_consumption_points: 0, monthly_consumption_points: 0, user_recharge_points: 0 })
        }
        if (u.includes('/quotas/')) {
          return jsonOk({
            task_post_quota: 10,
            gitlab_disk_gb: 1,
            gitlab_disk_used_gb: 0.2,
            gitlab_disk_expires_at: '2027-03-18',
            gitlab_traffic_prepaid_gb: 1,
            gitlab_traffic_used_gb: 0.05,
          })
        }
        if (u.includes('/transactions/')) return jsonOk({ results: [], total: 0, page: 1, page_size: 5 })
        return jsonOk({})
      })
      const wrapper = mount(BillingDashboard)
      await flushPromises()
      expect(wrapper.find('[data-testid="gitlab-region-quota-rows"]').exists()).toBe(false)
      const disk = wrapper.get('[data-testid="gitlab-aggregate-disk-card"]')
      expect(disk.text()).toContain('0.2 / 1')
      expect(disk.text()).toContain('到期：2027-03-18')
      // OPT-20260825-009: 回退卡复用 usagePercent 进度条，aria-valuenow 与 used/quota 一致
      const diskBar = disk.get('[role="progressbar"]')
      expect(diskBar.attributes('aria-valuenow')).toBe('20')
      const trafficCard = wrapper.get('[data-testid="gitlab-aggregate-traffic-card"]')
      expect(trafficCard.text()).toContain('0.05 / 1')
      const trafficBar = trafficCard.get('[role="progressbar"]')
      expect(trafficBar.attributes('aria-valuenow')).toBe('5')
      wrapper.unmount()
    })
  })
}

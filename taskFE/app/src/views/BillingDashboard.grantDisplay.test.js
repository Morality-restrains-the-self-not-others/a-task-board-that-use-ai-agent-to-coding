// @vitest-environment jsdom
/**
 * BillingDashboard 最近交易：配额入账须展示来源与具体资源变动，禁止 +0.00 元空壳。
 */
if (!process.env.VITEST) {
  console.log('[skip] BillingDashboard.grantDisplay.test.js requires vitest runtime')
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

  const grantTxn = {
    id: 'txn-grant-1',
    created_at: '2026-08-18 11:19:00',
    transaction_type: 'recharge',
    points_source_type: 'admin_grant',
    points_source_type_display: '后台赠送',
    amount_points: 0,
    change_display: '任务帖 +10 帖',
    ledger_snapshot: { display_lines: ['余额 0.00 元', '任务帖剩余 10'] },
    description: '管理员后台赠送：任务帖 +10 帖',
  }

  function stubApis(transactions) {
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
      if (u.includes('/quotas/')) return jsonOk({ task_post_quota: 10 })
      if (u.includes('/transactions/')) {
        return jsonOk({ results: transactions, total: transactions.length, page: 1, page_size: 5 })
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

  describe('BillingDashboard 最近交易入账内容', () => {
    beforeEach(() => {
      hoistedMocks.apiFetchMock.mockReset()
    })

    it('后台赠送行展示来源、资源变动和描述，不出现 +0.00', async () => {
      stubApis([grantTxn])
      const wrapper = mountDashboard()
      await flushPromises()
      const text = wrapper.text()
      expect(text).toContain('变动明细')
      expect(text).toContain('瞬时账目')
      expect(text).toContain('后台赠送')
      expect(text).toContain('任务帖 +10 帖')
      expect(text).toContain('管理员后台赠送：任务帖 +10 帖')
      expect(text).not.toContain('+0.00')
    })

    it('无 API display 时仍能从来源码映射为后台赠送', async () => {
      stubApis([{
        ...grantTxn,
        points_source_type_display: '',
        change_display: '',
        resource_changes: [{ display: '任务帖 +10 帖' }],
      }])
      const wrapper = mountDashboard()
      await flushPromises()
      const text = wrapper.text()
      expect(text).toContain('后台赠送')
      expect(text).toContain('任务帖 +10 帖')
    })
  })
}

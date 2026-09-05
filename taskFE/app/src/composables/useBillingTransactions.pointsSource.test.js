// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] useBillingTransactions.pointsSource.test.js requires vitest runtime')
} else {
  const { describe, expect, it, vi, beforeEach } = await import('vitest')

  vi.mock('vue-router', () => ({
    useRoute: () => ({ params: { tenant: 't1' } }),
  }))

  // vi.mock 工厂引用的变量必须经 vi.hoisted() 定义（Vitest 会把工厂 hoist 到模块顶部）。
  const hoisted = vi.hoisted(() => ({ apiFetch: vi.fn() }))
  const apiFetch = hoisted.apiFetch

  // useBillingTransactions.js 显式 import apiFetch（OPT-20260807-039），
  // 经模块 mock 拦截，无需再绑 window.apiFetch 全局。
  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch: (...args) => hoisted.apiFetch(...args),
    parseCompanyMembersResponse: () => ({ members: [] }),
  }))

  const { useBillingTransactions } = await import('./useBillingTransactions.js')

  describe('useBillingTransactions 支付来源过滤参数', () => {
    beforeEach(() => {
      apiFetch.mockReset()
      apiFetch.mockImplementation(async () => ({
        ok: true,
        json: async () => ({ results: [], total: 0, page_size: 20 }),
      }))
    })

    it('设置支付来源后请求携带 points_source_type 参数（后端契约）', async () => {
      const { filters, applyFilters } = useBillingTransactions()
      filters.value.pointsSourceType = 'user_recharge_paypal'
      await applyFilters()
      expect(apiFetch).toHaveBeenCalledTimes(1)
      const url = apiFetch.mock.calls[0][0]
      expect(url).toContain('/billing/transactions/list_filtered/')
      expect(url).toContain('points_source_type=user_recharge_paypal')
      // 不携带已废弃的 cents_source_type
      expect(url).not.toContain('cents_source_type')
    })

    it('重置后支付来源为空（pointsSourceType 参与重置）', async () => {
      const { filters, applyFilters, resetFilters } = useBillingTransactions()
      filters.value.pointsSourceType = 'admin_grant'
      await applyFilters()
      await resetFilters()
      expect(filters.value.pointsSourceType).toBe('')
    })
  })
}

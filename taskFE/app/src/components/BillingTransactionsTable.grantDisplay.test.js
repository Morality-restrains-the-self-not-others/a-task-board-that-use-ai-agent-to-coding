// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] BillingTransactionsTable.grantDisplay.test.js requires vitest runtime')
} else {
  const { mount } = await import('@vue/test-utils')
  const { describe, expect, it } = await import('vitest')
  const { default: BillingTransactionsTable } = await import('./BillingTransactionsTable.vue')

  const baseProps = {
    tenantId: 't1',
    totalCount: 1,
    currentPage: 1,
    totalPages: 1,
    formatDate: (d) => d,
    getTransactionTypeText: (t) => (t === 'recharge' ? '入账' : t),
    paymentSource: '',
    paymentSourceOptions: [],
    transactionTypeFilter: '',
  }

  describe('BillingTransactionsTable 入账资源内容', () => {
    it('后台赠送行在分类列显示来源、变动列显示资源数量', () => {
      const wrapper = mount(BillingTransactionsTable, {
        props: {
          ...baseProps,
          transactions: [{
            id: '1',
            created_at: '2026-08-18 11:19:00',
            transaction_type: 'recharge',
            points_source_type: 'admin_grant',
            points_source_type_display: '后台赠送',
            amount_points: 0,
            change_display: '任务帖 +10 帖',
            ledger_snapshot: {
              display_lines: ['余额 0.00 元', '任务帖剩余 10'],
            },
            description: '管理员后台赠送：任务帖 +10 帖',
          }],
        },
        global: {
          stubs: { RouterLink: { template: '<a><slot /></a>' }, Teleport: true },
        },
      })
      const text = wrapper.text()
      expect(text).toContain('变动明细')
      expect(text).toContain('瞬时账目')
      expect(text).toContain('后台赠送')
      expect(text).toContain('任务帖 +10 帖')
      expect(text).toContain('任务帖剩余 10')
    })
  })
}

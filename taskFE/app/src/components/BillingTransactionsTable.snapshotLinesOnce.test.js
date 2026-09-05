// @vitest-environment jsdom
/**
 * OPT-20260819-016 回归：行级 ledgerSnapshotLines 只计算一次，
 * 模板 v-for 与空态 v-if 不得各自调用造成重复解析。
 */
if (!process.env.VITEST) {
  console.log('[skip] BillingTransactionsTable.snapshotLinesOnce.test.js requires vitest runtime')
} else {
  const { mount } = await import('@vue/test-utils')
  const { describe, expect, it, vi, beforeEach } = await import('vitest')

  const hoistedMocks = vi.hoisted(() => ({
    ledgerSnapshotLinesMock: vi.fn(),
    pointsSourceTypeDisplayOrDashMock: vi.fn((t) => ''),
    transactionChangeDisplayMock: vi.fn((t) => ''),
  }))

  vi.mock('../utils/transactionChangeDisplay.js', () => ({
    ledgerSnapshotLines: hoistedMocks.ledgerSnapshotLinesMock,
    pointsSourceTypeDisplayOrDash: hoistedMocks.pointsSourceTypeDisplayOrDashMock,
    transactionChangeDisplay: hoistedMocks.transactionChangeDisplayMock,
  }))

  const { default: BillingTransactionsTable } = await import('./BillingTransactionsTable.vue')

  const baseProps = {
    tenantId: 't1',
    totalCount: 1,
    currentPage: 1,
    totalPages: 1,
    formatDate: (d) => d,
    getTransactionTypeText: (t) => t,
    paymentSource: '',
    paymentSourceOptions: [],
    transactionTypeFilter: '',
  }

  describe('BillingTransactionsTable ledgerSnapshotLines 单次计算', () => {
    beforeEach(() => {
      hoistedMocks.ledgerSnapshotLinesMock.mockReset()
      hoistedMocks.ledgerSnapshotLinesMock.mockReturnValue(['余额 10 元', '任务帖剩余 5'])
    })

    it('每个 transaction 只调用一次 ledgerSnapshotLines', () => {
      const txn = {
        id: 't1',
        created_at: '2026-08-18 11:19:00',
        transaction_type: 'consumption',
        points_source_type: 'resource_purchase',
        amount_points: 55,
        ledger_snapshot: { display_lines: ['余额 10 元', '任务帖剩余 5'] },
      }
      mount(BillingTransactionsTable, {
        props: { ...baseProps, transactions: [txn] },
        global: { stubs: { RouterLink: { template: '<a><slot /></a>' }, Teleport: true } },
      })
      expect(hoistedMocks.ledgerSnapshotLinesMock).toHaveBeenCalledTimes(1)
      expect(hoistedMocks.ledgerSnapshotLinesMock).toHaveBeenCalledWith(txn)
    })

    it('空快照渲染 — 占位符且不重复调用', () => {
      hoistedMocks.ledgerSnapshotLinesMock.mockReturnValue([])
      const wrapper = mount(BillingTransactionsTable, {
        props: {
          ...baseProps,
          transactions: [{
            id: 't2',
            created_at: '2026-08-18 11:19:00',
            transaction_type: 'consumption',
            points_source_type: 'consumption',
            amount_points: 10,
            ledger_snapshot: null,
          }],
        },
        global: { stubs: { RouterLink: { template: '<a><slot /></a>' }, Teleport: true } },
      })
      expect(wrapper.text()).toContain('—')
      expect(hoistedMocks.ledgerSnapshotLinesMock).toHaveBeenCalledTimes(1)
    })
  })
}

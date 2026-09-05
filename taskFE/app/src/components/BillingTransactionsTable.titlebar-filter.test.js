// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] BillingTransactionsTable.titlebar-filter.test.js requires vitest runtime')
} else {
  const { mount } = await import('@vue/test-utils')
  const { describe, expect, it } = await import('vitest')

  const { default: BillingTransactionsTable } = await import('./BillingTransactionsTable.vue')

  const baseProps = {
    tenantId: 't1',
    transactions: [],
    totalCount: 3,
    currentPage: 1,
    totalPages: 1,
    formatDate: (d) => d,
    getTransactionTypeText: (t) => t,
  }

  const paymentSourceOptions = [
    { value: 'user_recharge_paypal', label: 'PayPal 支付' },
    { value: 'user_recharge_wechat', label: '微信支付' },
    { value: 'admin_grant', label: '后台赠送' },
  ]

  const mountTable = (props = {}) =>
    mount(BillingTransactionsTable, {
      props: { ...baseProps, ...props },
      global: { stubs: { RouterLink: { template: '<a><slot /></a>' } } },
    })

  describe('BillingTransactionsTable 标题栏支付来源过滤器', () => {
    it('支付来源 select 渲染在列表标题栏内（交易记录标题 + 共 N 条记录之间）', () => {
      const wrapper = mountTable({ paymentSourceOptions })
      const header = wrapper.get('.flex.justify-between.items-center')
      expect(header.text()).toContain('交易记录')
      expect(header.text()).toContain('共 3 条记录')
      const select = header.find('select')
      expect(select.exists()).toBe(true)
      // 过滤卡片（过滤条件）不再拥有该 select —— 表格内共 3 处：标题栏支付来源 + 类型列头交易类型 + 分类列头消耗（元）分类
      expect(wrapper.findAll('select')).toHaveLength(3)
    })

    it('渲染全部支付来源选项，且回显当前选中值', () => {
      const wrapper = mountTable({
        paymentSourceOptions,
        paymentSource: 'user_recharge_paypal',
      })
      const options = wrapper
        .find('select')
        .findAll('option')
        .map((o) => o.text())
      expect(options).toEqual(['全部', 'PayPal 支付', '微信支付', '后台赠送'])
      expect(wrapper.find('select').element.value).toBe('user_recharge_paypal')
    })

    it('切换支付来源时发出 update:filter(pointsSourceType, value) 与 apply', async () => {
      const wrapper = mountTable({ paymentSourceOptions })
      await wrapper.find('select').setValue('user_recharge_wechat')
      expect(wrapper.emitted('update:filter')).toEqual([['pointsSourceType', 'user_recharge_wechat']])
      expect(wrapper.emitted('apply')).toHaveLength(1)
    })

    it('交易类型 select 渲染在「类型」列头中，选项与回显正确', () => {
      const wrapper = mountTable({ transactionTypeFilter: 'consumption' })
      const typeSelect = wrapper.find('select[aria-label="交易类型"]')
      expect(typeSelect.exists()).toBe(true)
      const options = typeSelect.findAll('option').map((o) => o.text())
      expect(options).toEqual(['全部类型', '入账', '消耗', '退款'])
      expect(typeSelect.element.value).toBe('consumption')
    })

    it('切换交易类型时发出 update:filter(transactionType, value) 与 apply', async () => {
      const wrapper = mountTable()
      await wrapper.find('select[aria-label="交易类型"]').setValue('recharge')
      expect(wrapper.emitted('update:filter')).toEqual([['transactionType', 'recharge']])
      expect(wrapper.emitted('apply')).toHaveLength(1)
    })
  })
}

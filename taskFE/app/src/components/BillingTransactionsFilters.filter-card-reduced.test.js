// @vitest-environment jsdom
/**
 * BillingTransactionsFilters：顶部过滤卡已缩减 —— 过滤器逐步移至表格区（67d4a7d 支付来源→标题栏、交易类型→类型列头；
 * 本次 时间范围→交易时间列头、消耗（元）分类→分类列头、项目名称→项目列头、成员名称→成员列头）
 * 卡片仅保留无对应表格列的 工作空间名称/任务名称 + 应用/重置
 */
if (!process.env.VITEST) {
  console.log('[skip] BillingTransactionsFilters.filter-card-reduced.test.js requires vitest runtime')
} else {
  const { mount } = await import('@vue/test-utils')
  const { describe, expect, it } = await import('vitest')

  const { default: BillingTransactionsFilters } = await import('./BillingTransactionsFilters.vue')

  const mountFilters = (props = {}) =>
    mount(BillingTransactionsFilters, {
      props: {
        filters: { startDate: '', endDate: '', workspaceId: '', taskId: '' },
        workspaceSearch: '',
        taskSearch: '',
        ...props,
      },
    })

  describe('BillingTransactionsFilters 已移至表头/标题栏的过滤器', () => {
    it('不再渲染「支付来源」「交易类型」', () => {
      const wrapper = mountFilters()
      expect(wrapper.text()).not.toContain('支付来源')
      expect(wrapper.text()).not.toContain('交易类型')
    })

    it('不再渲染「时间范围」「消耗（元）分类」「项目名称」「成员名称」（均已移至表格列头）', () => {
      const wrapper = mountFilters()
      for (const moved of ['时间范围', '消耗（元）分类', '项目名称', '成员名称']) {
        expect(wrapper.text()).not.toContain(moved)
      }
    })

    it('仍保留无对应表格列的过滤项：工作空间名称 / 任务名称', () => {
      const wrapper = mountFilters()
      const text = wrapper.text()
      expect(text).toContain('工作空间名称')
      expect(text).toContain('任务名称')
    })

    it('仍渲染应用过滤/重置按钮', () => {
      const wrapper = mountFilters()
      expect(wrapper.text()).toContain('应用过滤')
      expect(wrapper.text()).toContain('重置')
    })
  })
}

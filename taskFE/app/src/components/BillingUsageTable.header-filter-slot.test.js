// @vitest-environment jsdom
/**
 * BillingUsageTable：表头过滤行插槽
 * - thead 在表头行之后渲染第二行（data-alias="BillingUsageFiltersRow"），内容来自 #filters 插槽
 * - 未传 #filters 插槽时不渲染过滤行（向后兼容）
 */
if (!process.env.VITEST) {
  console.log('[skip] BillingUsageTable.header-filter-slot.test.js requires vitest runtime')
} else {
  const { mount } = await import('@vue/test-utils')
  const { describe, expect, it } = await import('vitest')

  const { default: BillingUsageTable } = await import('./BillingUsageTable.vue')

  const baseProps = {
    tenantId: 't1',
    usageRecords: [],
    totalCount: 0,
    currentPage: 1,
    totalPages: 1,
    formatDate: (d) => d,
  }

  const mountTable = (options = {}) =>
    mount(BillingUsageTable, {
      props: { ...baseProps, ...(options.props || {}) },
      slots: options.slots || {},
      global: { stubs: { RouterLink: { template: '<a><slot /></a>' } } },
    })

  describe('BillingUsageTable 表头过滤行', () => {
    it('#filters 插槽内容渲染在 thead 第二行（表头行之后）', () => {
      const wrapper = mountTable({
        slots: {
          filters: '<td data-test="filter-cell">过滤控件</td>',
        },
      })
      const thead = wrapper.get('thead')
      const rows = thead.findAll('tr')
      expect(rows).toHaveLength(2)
      expect(rows[0].text()).toContain('使用时间')
      expect(rows[1].get('[data-test="filter-cell"]').text()).toBe('过滤控件')
    })

    it('过滤行带 data-alias="BillingUsageFiltersRow" 便于定位', () => {
      const wrapper = mountTable({
        slots: { filters: '<td>f</td>' },
      })
      expect(wrapper.get('tr[data-alias="BillingUsageFiltersRow"]').exists()).toBe(true)
    })

    it('未传 #filters 插槽时不渲染过滤行', () => {
      const wrapper = mountTable()
      const rows = wrapper.get('thead').findAll('tr')
      expect(rows).toHaveLength(1)
      expect(wrapper.find('[data-alias="BillingUsageFiltersRow"]').exists()).toBe(false)
    })

    it('过滤行单元格数量与表头列数一致（8 列）', () => {
      const cells = Array.from({ length: 8 }, (_, i) => `<td data-cell="${i}"></td>`).join('')
      const wrapper = mountTable({ slots: { filters: cells } })
      const filterRow = wrapper.get('[data-alias="BillingUsageFiltersRow"]')
      expect(filterRow.findAll('td')).toHaveLength(8)
      const headerRow = wrapper.get('thead tr:first-child')
      expect(headerRow.findAll('th')).toHaveLength(8)
    })
  })
}

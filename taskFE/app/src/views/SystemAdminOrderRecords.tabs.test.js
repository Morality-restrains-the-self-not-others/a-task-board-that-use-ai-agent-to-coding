// @vitest-environment jsdom
/**
 * 订单与退款页：Tab 切换写入/读取 ?tab=refund。
 */
if (!process.env.VITEST) {
  console.log('[skip] SystemAdminOrderRecords.tabs.test.js requires vitest runtime')
} else {
  const { describe, expect, it, vi, beforeEach } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')

  const hoisted = vi.hoisted(() => ({
    route: { path: '/system-admin/order-records/', query: {} },
    replace: vi.fn(),
  }))

  vi.mock('vue-router', () => ({
    useRoute: () => hoisted.route,
    useRouter: () => ({ replace: hoisted.replace }),
  }))

  vi.mock('../components/system-admin/SystemAdminOrderListPanel.vue', () => ({
    default: { name: 'SystemAdminOrderListPanel', template: '<div data-testid="order-list-panel" />' },
  }))
  vi.mock('../components/system-admin/SystemAdminRefundPanel.vue', () => ({
    default: { name: 'SystemAdminRefundPanel', template: '<div data-testid="refund-panel" />' },
  }))
  vi.mock('../components/system-admin/SystemAdminProfitSharingPanel.vue', () => ({
    default: { name: 'SystemAdminProfitSharingPanel', template: '<div data-testid="profit-sharing-panel" />' },
  }))
  vi.mock('../components/system-admin/SystemAdminInvoicePanel.vue', () => ({
    default: { name: 'SystemAdminInvoicePanel', template: '<div data-testid="invoice-panel" />' },
  }))

  const Page = (await import('./SystemAdminOrderRecords.vue')).default

  describe('SystemAdminOrderRecords tabs', () => {
    beforeEach(() => {
      hoisted.route.query = {}
      hoisted.replace.mockReset()
    })

    it('默认展示订单 Tab', async () => {
      const wrapper = mount(Page)
      await flushPromises()
      expect(wrapper.find('[data-testid="order-list-panel"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="refund-panel"]').exists()).toBe(false)
    })

    it('query.tab=refund 时展示退款面板', async () => {
      hoisted.route.query = { tab: 'refund' }
      const wrapper = mount(Page)
      await flushPromises()
      expect(wrapper.find('[data-testid="refund-panel"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="order-list-panel"]').exists()).toBe(false)
    })

    it('点击退款 Tab 写入 query.tab=refund', async () => {
      const wrapper = mount(Page)
      await flushPromises()
      await wrapper.get('[data-testid="order-records-tab-refund"]').trigger('click')
      expect(hoisted.replace).toHaveBeenCalledWith({
        path: '/system-admin/order-records/',
        query: { tab: 'refund' },
      })
    })

    it('query.tab=profit-sharing 时展示待分账面板', async () => {
      hoisted.route.query = { tab: 'profit-sharing' }
      const wrapper = mount(Page)
      await flushPromises()
      expect(wrapper.find('[data-testid="profit-sharing-panel"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="order-list-panel"]').exists()).toBe(false)
      expect(wrapper.find('[data-testid="refund-panel"]').exists()).toBe(false)
    })

    it('点击待分账 Tab 写入 query.tab=profit-sharing', async () => {
      const wrapper = mount(Page)
      await flushPromises()
      await wrapper.get('[data-testid="order-records-tab-profit-sharing"]').trigger('click')
      expect(hoisted.replace).toHaveBeenCalledWith({
        path: '/system-admin/order-records/',
        query: { tab: 'profit-sharing' },
      })
    })

    it('query.tab=invoice 时展示发票审批面板', async () => {
      hoisted.route.query = { tab: 'invoice' }
      const wrapper = mount(Page)
      await flushPromises()
      expect(wrapper.find('[data-testid="invoice-panel"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="order-list-panel"]').exists()).toBe(false)
    })

    it('点击发票 Tab 写入 query.tab=invoice', async () => {
      const wrapper = mount(Page)
      await flushPromises()
      await wrapper.get('[data-testid="order-records-tab-invoice"]').trigger('click')
      expect(hoisted.replace).toHaveBeenCalledWith({
        path: '/system-admin/order-records/',
        query: { tab: 'invoice' },
      })
    })
  })
}

// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] BillingOrders.tradeNoSearch.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { describe, expect, it, vi, beforeEach } = await import('vitest')

  const mocks = vi.hoisted(() => {
    const routeQuery = {}
    return {
      apiFetch: vi.fn(),
      routeQuery,
      useRoute: () => ({ params: { tenant: '881024523581812736' }, query: routeQuery, name: 'billing_orders', fullPath: '/', path: '/' }),
      useRouter: () => ({
        push: vi.fn(),
        replace: vi.fn(async (args) => {
          Object.keys(routeQuery).forEach((k) => delete routeQuery[k])
          Object.assign(routeQuery, args?.query || {})
        }),
      }),
    }
  })

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch: (...args) => mocks.apiFetch(...args),
    clearCachedAuthToken: () => {},
  }))
  vi.mock('vue-router', () => ({
    useRoute: () => mocks.useRoute(),
    useRouter: () => mocks.useRouter(),
  }))
  vi.mock('../composables/useBillingOrderActions.js', () => ({
    useBillingOrderActions: () => ({
      payModalVisible: false, payTarget: null, payCodeUrl: '', payMode: 'wechat', payPollingErr: '',
      openPayModal: () => {}, closePayModal: () => {},
      phoneGateActive: false, phoneVerificationStatus: { has_phone: false, phone_masked: '' }, onPhoneVerified: () => {},
      cancelConfirmVisible: false, cancelTarget: null, cancelling: false,
      confirmCancelOrder: () => {}, doCancelOrder: () => {},
    }),
  }))
  vi.mock('../composables/useBillingRefund.js', () => ({
    useBillingRefund: () => ({
      refundEnabled: false, applyError: { value: '' }, applyErrorTraceId: { value: '' },
      applying: { value: false }, applyRefund: async () => true, refresh: async () => {},
    }),
  }))
  vi.mock('../components/PayOrderModal.vue', () => ({ default: { template: '<div />' } }))
  vi.mock('../components/CancelOrderModal.vue', () => ({ default: { template: '<div />' } }))

  const { default: BillingOrders } = await import('./BillingOrders.vue')

  describe('BillingOrders 交易单号 / 商户单号查询', () => {
    beforeEach(() => {
      mocks.apiFetch.mockReset()
      Object.keys(mocks.routeQuery).forEach((k) => delete mocks.routeQuery[k])
      mocks.apiFetch.mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => ({ orders: [], total: 0 }),
      })
    })

    it('首次加载不带 order_number', async () => {
      const wrapper = mount(BillingOrders)
      await flushPromises()
      const ordersUrl = mocks.apiFetch.mock.calls.map((c) => String(c[0])).find((u) => u.includes('/billing/orders'))
      expect(ordersUrl).toMatch(/\/api\/tenant\/881024523581812736\/billing\/orders\//)
      expect(ordersUrl).not.toMatch(/order_number=/)
      wrapper.unmount()
    })

    it('交易单号查询请求带 order_number', async () => {
      const wrapper = mount(BillingOrders)
      await flushPromises()
      mocks.apiFetch.mockClear()
      await wrapper.get('[data-testid="billing-order-txn-input"]').setValue('4500000359202608221274536815')
      await wrapper.get('[data-testid="billing-order-number-search-btn"]').trigger('click')
      await flushPromises()
      const ordersUrl = mocks.apiFetch.mock.calls.map((c) => String(c[0])).find((u) => u.includes('/billing/orders'))
      expect(ordersUrl).toContain('order_number=4500000359202608221274536815')
      wrapper.unmount()
    })

    it('商户单号查询请求带 order_number', async () => {
      const wrapper = mount(BillingOrders)
      await flushPromises()
      mocks.apiFetch.mockClear()
      await wrapper.get('[data-testid="billing-order-out-trade-input"]').setValue('WX878981209491800064')
      await wrapper.get('[data-testid="billing-order-number-search-btn"]').trigger('click')
      await flushPromises()
      const ordersUrl = mocks.apiFetch.mock.calls.map((c) => String(c[0])).find((u) => u.includes('/billing/orders'))
      expect(ordersUrl).toContain('order_number=WX878981209491800064')
      wrapper.unmount()
    })

    it('清空后请求不再带 order_number', async () => {
      const wrapper = mount(BillingOrders)
      await flushPromises()
      await wrapper.get('[data-testid="billing-order-txn-input"]').setValue('4500000359202608221274536815')
      await wrapper.get('[data-testid="billing-order-number-search-btn"]').trigger('click')
      await flushPromises()
      mocks.apiFetch.mockClear()
      await wrapper.get('[data-testid="billing-order-number-clear-btn"]').trigger('click')
      await flushPromises()
      const ordersUrl = mocks.apiFetch.mock.calls.map((c) => String(c[0])).find((u) => u.includes('/billing/orders'))
      expect(ordersUrl).not.toMatch(/order_number=/)
      wrapper.unmount()
    })

    it('查询后把 order_number 写入 URL query（便于分享/刷新保持）', async () => {
      const wrapper = mount(BillingOrders)
      await flushPromises()
      await wrapper.get('[data-testid="billing-order-txn-input"]').setValue('4500000359202608221274536815')
      await wrapper.get('[data-testid="billing-order-number-search-btn"]').trigger('click')
      await flushPromises()
      expect(mocks.routeQuery.order_number).toBe('4500000359202608221274536815')
      wrapper.unmount()
    })

    it('清空后从 URL query 移除 order_number', async () => {
      const wrapper = mount(BillingOrders)
      await flushPromises()
      await wrapper.get('[data-testid="billing-order-txn-input"]').setValue('WX878981209491800064')
      await wrapper.get('[data-testid="billing-order-number-search-btn"]').trigger('click')
      await flushPromises()
      expect(mocks.routeQuery.order_number).toBe('WX878981209491800064')
      await wrapper.get('[data-testid="billing-order-number-clear-btn"]').trigger('click')
      await flushPromises()
      expect(mocks.routeQuery.order_number).toBeUndefined()
      wrapper.unmount()
    })

    it('URL 带 order_number 时预填查询框并请求', async () => {
      mocks.routeQuery.order_number = '4500000359202608221274536815'
      const wrapper = mount(BillingOrders)
      await flushPromises()
      const ordersUrl = mocks.apiFetch.mock.calls.map((c) => String(c[0])).find((u) => u.includes('/billing/orders'))
      expect(ordersUrl).toContain('order_number=4500000359202608221274536815')
      expect(wrapper.get('[data-testid="billing-order-txn-input"]').element.value).toBe('4500000359202608221274536815')
      wrapper.unmount()
    })
  })
}

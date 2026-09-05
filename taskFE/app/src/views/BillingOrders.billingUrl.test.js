// @vitest-environment jsdom
// 回归测试：订单列表 API 必须使用尾部斜杠（/api/tenant/{tid}/billing/orders/）。
// 根因：1508f24 将 URL 改为无斜杠 /billing/orders，而 taskBill 路由器按
// strings.HasSuffix(p, "/billing/orders/") 匹配（见 taskBill/src/handlers.go），
// 无斜杠 → 默认 404 {"detail":"not found"} → 页面红框 "not found"。
if (!process.env.VITEST) {
  console.log('[skip] BillingOrders.billingUrl.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { describe, expect, it, vi, beforeEach } = await import('vitest')

  const mocks = vi.hoisted(() => ({
    apiFetch: vi.fn(),
    useRoute: () => ({ params: { tenant: '873472655125147648' }, query: {}, name: 'billing_orders', fullPath: '/', path: '/' }),
    useRouter: () => ({ push: vi.fn(), replace: vi.fn(async () => {}) }),
  }))

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch: (...args) => mocks.apiFetch(...args),
    clearCachedAuthToken: () => {},
  }))
  vi.mock('vue-router', () => ({
    useRoute: () => mocks.useRoute(),
    useRouter: () => mocks.useRouter(),
  }))
  // 订单操作/退款/弹窗与本回归无关，置空避免真实网络调用
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

  describe('BillingOrders 订单列表 API URL（尾部斜杠回归）', () => {
    beforeEach(() => {
      mocks.apiFetch.mockReset()
    })

    it('fetchOrders 请求 /api/tenant/{tid}/billing/orders/（带尾部斜杠），不渲染错误红框', async () => {
      mocks.apiFetch.mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => ({ orders: [], total: 0 }),
      })
      const wrapper = mount(BillingOrders)
      await flushPromises()

      const called = mocks.apiFetch.mock.calls.map((c) => String(c[0]))
      expect(called.length).toBeGreaterThan(0)
      const ordersUrl = called.find((u) => u.includes('/billing/orders'))
      expect(ordersUrl).toBeDefined()
      expect(ordersUrl).toMatch(/\/api\/tenant\/873472655125147648\/billing\/orders\//)
      expect(ordersUrl).not.toMatch(/\/billing\/orders\?/)
      expect(wrapper.find('.bg-red-50').exists()).toBe(false)
      wrapper.unmount()
    })
  })
}

// @vitest-environment jsdom
// 回归：订单已有 pending 退款申请时，操作列显示「退款中」而非「申请退款」。
if (!process.env.VITEST) {
  console.log('[skip] BillingOrders.refund-pending.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { describe, expect, it, vi, beforeEach } = await import('vitest')
  const { ref } = await import('vue')

  const mocks = vi.hoisted(() => ({
    apiFetch: vi.fn(),
    refundApplications: null,
    applyError: null,
    applyErrorTraceId: null,
    applying: null,
    useRoute: () => ({
      params: { tenant: '877397588196749312' },
      query: {},
      name: 'billing_orders',
      fullPath: '/',
      path: '/',
    }),
    useRouter: () => ({ push: vi.fn(), replace: vi.fn(async () => {}) }),
  }))

  mocks.refundApplications = ref([])
  mocks.applyError = ref('')
  mocks.applyErrorTraceId = ref('')
  mocks.applying = ref(false)

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
      payModalVisible: false,
      payTarget: null,
      payCodeUrl: '',
      payMode: 'wechat',
      payPollingErr: '',
      openPayModal: () => {},
      closePayModal: () => {},
      phoneGateActive: false,
      phoneVerificationStatus: { has_phone: false, phone_masked: '' },
      onPhoneVerified: () => {},
      cancelConfirmVisible: false,
      cancelTarget: null,
      cancelling: false,
      confirmCancelOrder: () => {},
      doCancelOrder: () => {},
    }),
  }))
  vi.mock('../composables/useBillingRefund.js', () => ({
    useBillingRefund: () => ({
      refundEnabled: true,
      applications: mocks.refundApplications,
      applyError: mocks.applyError,
      applyErrorTraceId: mocks.applyErrorTraceId,
      applying: mocks.applying,
      applyRefund: async () => true,
      refresh: async () => {},
    }),
  }))
  vi.mock('../components/PayOrderModal.vue', () => ({ default: { template: '<div />' } }))
  vi.mock('../components/CancelOrderModal.vue', () => ({ default: { template: '<div />' } }))

  const { default: BillingOrders } = await import('./BillingOrders.vue')

  const paidOrder = {
    id: '900001',
    order_number: 'ORD-R1',
    status: 'paid',
    total_yuan: '10.00',
    total_yuan_cents: 1000,
    payment_method: 'wechat',
    created_at: '2026-08-01T00:00:00Z',
    paid_at: '2026-08-01T01:00:00Z',
  }

  describe('BillingOrders 退款中订单不显示申请按钮', () => {
    beforeEach(() => {
      mocks.apiFetch.mockReset()
      mocks.refundApplications.value = []
      mocks.applyError.value = ''
      mocks.applyErrorTraceId.value = ''
      mocks.applying.value = false
      mocks.apiFetch.mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => ({ orders: [paidOrder], total: 1 }),
      })
    })

    it('无退款申请时显示「申请退款」', async () => {
      const wrapper = mount(BillingOrders)
      await flushPromises()
      expect(wrapper.find('[data-testid="billing-refund-apply-btn"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="billing-refund-pending-label"]').exists()).toBe(false)
      wrapper.unmount()
    })

    it('订单 pending 退款申请时显示「退款中」并隐藏申请按钮', async () => {
      mocks.refundApplications.value = [{ order_id: '900001', status: 'pending' }]
      const wrapper = mount(BillingOrders)
      await flushPromises()
      expect(wrapper.find('[data-testid="billing-refund-pending-label"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="billing-refund-pending-label"]').text()).toContain('退款中')
      expect(wrapper.find('[data-testid="billing-refund-apply-btn"]').exists()).toBe(false)
      wrapper.unmount()
    })
  })
}

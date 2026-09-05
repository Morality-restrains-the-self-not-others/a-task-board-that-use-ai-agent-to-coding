// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] OrderDetail.refund.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { describe, expect, it, vi, beforeEach } = await import('vitest')
  const { ref } = await import('vue')
  const { BILLING_REFUND_CONSUMED_NOTICE } = await import('../utils/billingRefundCopy.js')

  const mocks = vi.hoisted(() => ({
    apiFetch: vi.fn(),
    refundEnabled: null,
    refundApplications: null,
    applyError: null,
    applyErrorTraceId: null,
    applying: null,
    applyRefund: vi.fn(async () => true),
    useRoute: () => ({
      params: { tenant: '878619773850644480', orderId: '878981177317294080' },
      query: {},
      fullPath: '/',
      path: '/',
    }),
  }))

  mocks.refundEnabled = ref(true)
  mocks.refundApplications = ref([])
  mocks.applyError = ref('')
  mocks.applyErrorTraceId = ref('')
  mocks.applying = ref(false)

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch: (...args) => mocks.apiFetch(...args),
    extractErrorMessage: () => 'mock-err',
  }))
  vi.mock('../composables/useQrCodeCanvas.js', () => ({
    renderQrToCanvas: vi.fn(),
  }))
  vi.mock('../utils/cookieUtils.js', () => ({ getCookie: () => '' }))
  vi.mock('vue-router', () => ({ useRoute: () => mocks.useRoute() }))
  vi.mock('../composables/useBillingRefund.js', () => ({
    useBillingRefund: () => ({
      refundEnabled: mocks.refundEnabled,
      applications: mocks.refundApplications,
      applyError: mocks.applyError,
      applyErrorTraceId: mocks.applyErrorTraceId,
      applying: mocks.applying,
      applyRefund: (...args) => mocks.applyRefund(...args),
      refresh: async () => {},
    }),
  }))

  const { default: OrderDetail } = await import('./OrderDetail.vue')

  const PAID_ORDER = {
    id: '878981177317294080',
    order_number: 'ORD-PAID-1',
    status: 'paid',
    total_yuan: '5.50',
    total_yuan_cents: 550,
    payment_method: 'wechat',
    items: [
      { id: 'i1', resource_type: 'task_post', quantity: 10, unit_price_yuan: '0.55', subtotal_yuan: '5.50' },
    ],
    resource_consumption: {
      task_post: { granted: 10, consumed: 2, remaining: 8, events: [] },
    },
  }

  describe('OrderDetail 支付成功退款入口', () => {
    beforeEach(() => {
      mocks.apiFetch.mockReset()
      mocks.applyRefund.mockClear()
      mocks.refundEnabled.value = true
      mocks.refundApplications.value = []
      mocks.applyError.value = ''
      mocks.applyErrorTraceId.value = ''
      mocks.applying.value = false
      mocks.apiFetch.mockImplementation((url) => {
        const u = String(url || '')
        if (u.includes('/billing/orders/')) {
          return Promise.resolve({ ok: true, status: 200, json: async () => PAID_ORDER })
        }
        if (u.includes('phone-verification-status')) {
          return Promise.resolve({
            ok: true,
            status: 200,
            json: async () => ({ required: false, has_phone: true, sms_verified: true }),
          })
        }
        return Promise.resolve({ ok: true, status: 200, json: async () => ({}) })
      })
    })

    it('已支付横幅显示退款按钮与已消耗无法退回说明', async () => {
      const wrapper = mount(OrderDetail, { global: { stubs: { PhoneVerificationGate: true } } })
      await flushPromises()
      expect(wrapper.text()).toContain('支付成功，资源已发放到您的账户')
      expect(wrapper.text()).toContain(BILLING_REFUND_CONSUMED_NOTICE)
      expect(wrapper.find('[data-testid="billing-refund-apply-btn"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="billing-refund-pending-label"]').exists()).toBe(false)
      wrapper.unmount()
    })

    it('本单 pending 时显示退款中且无申请按钮', async () => {
      mocks.refundApplications.value = [{ order_id: '878981177317294080', status: 'pending' }]
      const wrapper = mount(OrderDetail, { global: { stubs: { PhoneVerificationGate: true } } })
      await flushPromises()
      expect(wrapper.find('[data-testid="billing-refund-pending-label"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="billing-refund-apply-btn"]').exists()).toBe(false)
      wrapper.unmount()
    })

    it('已退款订单仍展示电子发票与72小时冲红确认提醒', async () => {
      mocks.apiFetch.mockImplementation((url) => {
        const u = String(url || '')
        if (u.includes('/billing/orders/')) {
          return Promise.resolve({
            ok: true,
            status: 200,
            json: async () => ({
              ...PAID_ORDER,
              status: 'refunded',
              invoices: [{
                id: 'r1',
                kind: 'red',
                purpose: 'reverse',
                status: 'reverse_pending',
                amount_yuan: '5.50',
                reverse_confirm_hours: 72,
                reverse_confirm_deadline: '2026-08-26T00:00:00Z',
                reverse_confirm_expired: false,
              }],
              invoice_reverse_confirm: {
                required: true,
                hours: 72,
                deadline: '2026-08-26T00:00:00Z',
                expired: false,
                message: '请在微信卡包于72小时内确认冲红，逾期冲红将失效',
              },
            }),
          })
        }
        if (u.includes('phone-verification-status')) {
          return Promise.resolve({
            ok: true,
            status: 200,
            json: async () => ({ required: false, has_phone: true, sms_verified: true }),
          })
        }
        return Promise.resolve({ ok: true, status: 200, json: async () => ({}) })
      })
      const wrapper = mount(OrderDetail, { global: { stubs: { PhoneVerificationGate: true } } })
      await flushPromises()
      expect(wrapper.find('[data-testid="order-status-label"]').text()).toBe('已退款')
      expect(wrapper.find('[data-testid="order-refunded-invoice-panel"]').exists()).toBe(true)
      const hint = wrapper.find('[data-testid="order-invoice-reverse-confirm-hint"]')
      expect(hint.exists()).toBe(true)
      expect(hint.text()).toContain('72小时')
      wrapper.unmount()
    })
  })
}

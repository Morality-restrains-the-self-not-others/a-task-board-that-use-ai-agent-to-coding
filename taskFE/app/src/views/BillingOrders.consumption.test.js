// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] BillingOrders.consumption.test.js requires vitest runtime')
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
      refundEnabled: false,
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
    total_yuan: '0.55',
    payment_method: 'wechat',
    created_at: '2026-08-01T00:00:00Z',
    paid_at: '2026-08-01T01:00:00Z',
  }

  describe('BillingOrders 展开展示资源消耗', () => {
    beforeEach(() => {
      mocks.apiFetch.mockReset()
      mocks.refundApplications.value = []
      mocks.apiFetch.mockImplementation(async (url) => {
        const u = String(url || '')
        if (u.includes('/billing/orders/900001/')) {
          return {
            ok: true,
            json: async () => ({
              items: [{ id: 'i1', resource_type: 'task_post', quantity: 10, unit_price_yuan: '0.55', subtotal_yuan: '5.50' }],
              resource_consumption: {
                task_post: {
                  source_kind: 'purchase',
                  granted: 10,
                  consumed: 3,
                  remaining: 7,
                  events: [{ created_at: '2026-08-20', task_id: 't-1', action: 'create', quantity: 1 }],
                },
              },
            }),
          }
        }
        if (u.includes('/billing/orders/')) {
          return { ok: true, json: async () => ({ orders: [paidOrder], total: 1 }) }
        }
        return { ok: true, json: async () => ({}) }
      })
    })

    it('展开行显示已消耗与任务 id', async () => {
      const wrapper = mount(BillingOrders)
      await flushPromises()
      await wrapper.get('[data-order-id="900001"]').trigger('click')
      await flushPromises()
      const box = wrapper.get('[data-testid="order-resource-consumption"]')
      expect(box.text()).toContain('已消耗 3')
      expect(box.text()).toContain('剩余 7')
      expect(box.text()).toContain('t-1')
      wrapper.unmount()
    })
  })
}

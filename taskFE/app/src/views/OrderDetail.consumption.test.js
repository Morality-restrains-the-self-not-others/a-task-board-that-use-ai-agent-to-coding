// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] OrderDetail.consumption.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { describe, expect, it, vi, beforeEach } = await import('vitest')

  const mocks = vi.hoisted(() => ({
    apiFetch: vi.fn(),
    useRoute: () => ({
      params: { tenant: '873472655125147648', orderId: '1' },
      query: {},
      fullPath: '/',
      path: '/',
    }),
  }))

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch: (...args) => mocks.apiFetch(...args),
    extractErrorMessage: () => 'mock-err',
  }))
  vi.mock('../composables/useQrCodeCanvas.js', () => ({
    renderQrToCanvas: vi.fn(),
  }))
  vi.mock('../utils/cookieUtils.js', () => ({ getCookie: () => '' }))
  vi.mock('vue-router', () => ({ useRoute: () => mocks.useRoute() }))

  const { default: OrderDetail } = await import('./OrderDetail.vue')

  const PAID_ORDER = {
    id: '1',
    order_number: 'ORD-TEST-001',
    status: 'paid',
    total_yuan: '5.50',
    items: [
      { id: 'i1', resource_type: 'task_post', quantity: 10, unit_price_yuan: '0.55', subtotal_yuan: '5.50' },
    ],
    resource_consumption: {
      task_post: {
        source_kind: 'purchase',
        granted: 10,
        consumed: 2,
        remaining: 8,
        events: [{ created_at: '2026-08-20T01:00:00Z', task_id: 'task-abc', action: 'create', quantity: 1 }],
      },
    },
  }

  describe('OrderDetail 资源消耗', () => {
    beforeEach(() => {
      mocks.apiFetch.mockReset()
      mocks.apiFetch.mockImplementation((url) => {
        const u = String(url || '')
        if (u.includes('/billing/orders/1/')) {
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

    it('已支付订单展示消耗区块与任务 id', async () => {
      const wrapper = mount(OrderDetail, { global: { stubs: { PhoneVerificationGate: true } } })
      await flushPromises()
      const box = wrapper.get('[data-testid="order-resource-consumption"]')
      expect(box.text()).toContain('已消耗 2')
      expect(box.text()).toContain('剩余 8')
      expect(box.text()).toContain('task-abc')
      expect(box.text()).toContain('购买')
      wrapper.unmount()
    })
  })
}

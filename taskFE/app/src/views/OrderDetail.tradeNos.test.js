// @vitest-environment jsdom
// 订单详情摘要须展示微信支付交易单号与商户单号（GET 已返回字段）。
if (!process.env.VITEST) {
  console.log('[skip] OrderDetail.tradeNos.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { describe, expect, it, vi, beforeEach } = await import('vitest')

  const mocks = vi.hoisted(() => ({
    apiFetch: vi.fn(),
    useRoute: () => ({
      params: { tenant: '877397588196749312', orderId: '879747355690172416' },
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
    id: '879747355690172416',
    order_number: 'ORD-20260824-877397588196749312-879747355690172416',
    status: 'paid',
    total_yuan: '0.00',
    total_yuan_cents: 0,
    payment_method: 'wechat',
    out_trade_no: 'WX20260824879747355690172416',
    wechat_transaction_id: '4200001234202608241234567890',
    items: [],
  }

  function mockOrder(body) {
    mocks.apiFetch.mockImplementation((url) => {
      const u = String(url || '')
      if (u.includes('/billing/phone-verification-status/')) {
        return Promise.resolve({
          ok: true,
          status: 200,
          json: () => Promise.resolve({ required: false, has_phone: true, sms_verified: true }),
        })
      }
      if (u.includes('/billing/orders/')) {
        return Promise.resolve({
          ok: true,
          status: 200,
          json: () => Promise.resolve(body),
        })
      }
      return Promise.resolve({ ok: false, status: 404, json: () => Promise.resolve({}) })
    })
  }

  describe('OrderDetail 交易单号与商户单号', () => {
    beforeEach(() => {
      mocks.apiFetch.mockReset()
    })

    it('已支付订单展示交易单号与商户单号', async () => {
      mockOrder(PAID_ORDER)
      const wrapper = mount(OrderDetail, { global: { stubs: { PhoneVerificationGate: true } } })
      await flushPromises()
      const txn = wrapper.get('[data-testid="order-wechat-transaction-id"]')
      const merchant = wrapper.get('[data-testid="order-out-trade-no"]')
      expect(txn.text()).toContain('交易单号')
      expect(txn.text()).toContain('4200001234202608241234567890')
      expect(merchant.text()).toContain('商户单号')
      expect(merchant.text()).toContain('WX20260824879747355690172416')
      wrapper.unmount()
    })

    it('缺凭证时仍展示标签并以 — 占位，不虚构单号', async () => {
      mockOrder({
        ...PAID_ORDER,
        out_trade_no: '',
        wechat_transaction_id: '',
      })
      const wrapper = mount(OrderDetail, { global: { stubs: { PhoneVerificationGate: true } } })
      await flushPromises()
      expect(wrapper.get('[data-testid="order-wechat-transaction-id"]').text()).toMatch(/交易单号：\s*—/)
      expect(wrapper.get('[data-testid="order-out-trade-no"]').text()).toMatch(/商户单号：\s*—/)
      expect(wrapper.text()).not.toContain('4200001234202608241234567890')
      wrapper.unmount()
    })
  })
}

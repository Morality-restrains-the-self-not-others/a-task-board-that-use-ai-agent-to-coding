// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] useBillingOrderActions.trailingSlash.test.js requires vitest runtime')
} else {
const { describe, expect, it, vi, beforeEach } = await import('vitest')
const { computed } = await import('vue')

vi.mock('../utils/cookieUtils', () => ({
  getCookie: () => 'csrf-test',
}))

vi.mock('../utils/config.js', () => ({
  getApiUrl: (p) => `http://test${p}`,
}))

// vi.mock 工厂引用的变量必须经 vi.hoisted() 定义（Vitest 会把工厂 hoist 到模块顶部）。
const hoisted = vi.hoisted(() => ({
  apiFetch: vi.fn(),
}))
const apiFetch = hoisted.apiFetch

vi.mock('../utils/apiUtils', () => ({
  apiFetch: (...args) => hoisted.apiFetch(...args),
}))

const { useBillingOrderActions } = await import('./useBillingOrderActions.js')

function jsonResponse(body, { ok = true, status = 200 } = {}) {
  return { ok, status, json: async () => body }
}

describe('useBillingOrderActions billing URL 尾部斜杠（回归：not found 事故）', () => {
  beforeEach(() => {
    apiFetch.mockReset()
  })

  it('openPayModal 先查 phone-verification-status/（带尾部斜杠），无需验证时调 pay/（带尾部斜杠）', async () => {
    const urls = []
    apiFetch.mockImplementation(async (url) => {
      urls.push(String(url))
      if (String(url).includes('phone-verification-status')) {
        return jsonResponse({ has_phone: false, sms_verified: false, required: false })
      }
      return jsonResponse({ code_url: 'weixin://wxpay', out_trade_no: 'OT-1', mode: 'qrcode', status: 'pending' })
    })

    const c = useBillingOrderActions({ tenantId: computed(() => '850256677331562496'), onOrderChanged: vi.fn() })
    await c.openPayModal({ id: 'order-1' })
    c.closePayModal() // 停止支付轮询定时器

    const statusUrl = urls.find((u) => u.includes('phone-verification-status'))
    expect(statusUrl).toBe('/api/tenant/850256677331562496/billing/phone-verification-status/')

    const payUrl = urls.find((u) => u.includes('/pay'))
    expect(payUrl).toBe('/api/tenant/850256677331562496/billing/orders/order-1/pay/')
  })

  it('doCancelOrder 请求 cancel/ 带尾部斜杠', async () => {
    const urls = []
    apiFetch.mockImplementation(async (url) => {
      urls.push(String(url))
      return jsonResponse({ ok: true })
    })

    const onOrderChanged = vi.fn()
    const c = useBillingOrderActions({ tenantId: computed(() => '850256677331562496'), onOrderChanged })
    c.confirmCancelOrder({ id: 'order-2' })
    await c.doCancelOrder()

    const cancelUrl = urls.find((u) => u.includes('/cancel'))
    expect(cancelUrl).toBe('/api/tenant/850256677331562496/billing/orders/order-2/cancel/')
    expect(onOrderChanged).toHaveBeenCalledTimes(1)
  })
})

}

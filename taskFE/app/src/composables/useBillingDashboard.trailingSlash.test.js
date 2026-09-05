// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] useBillingDashboard.trailingSlash.test.js requires vitest runtime')
} else {
const { describe, expect, it, vi, beforeEach } = await import('vitest')
const { mount, flushPromises } = await import('@vue/test-utils')

vi.mock('vue-router', () => ({
  useRoute: () => ({ params: { tenant: '850256677331562496' } }),
}))

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

vi.mock('../utils/apiUtils.js', () => ({
  apiFetch: (...args) => hoisted.apiFetch(...args),
}))

const { useBillingDashboard } = await import('./useBillingDashboard.js')

function jsonResponse(body, { ok = true, status = 200 } = {}) {
  return { ok, status, json: async () => body }
}

describe('useBillingDashboard billing URL 尾部斜杠（回归：not found 事故）', () => {
  beforeEach(() => {
    apiFetch.mockReset()
  })

  it('账单首页不再请求 order-pricing（定价卡片已移除，防回归）', async () => {
    const urls = []
    apiFetch.mockImplementation(async (url) => {
      urls.push(String(url))
      return jsonResponse({ pricing: {}, cents: '0', frozen_balance: 0, results: [], total_consumption_cents: 0 })
    })

    const wrapper = mount({ setup() { useBillingDashboard(); return () => null } })
    await flushPromises()

    const pricingUrl = urls.find((u) => u.includes('/billing/order-pricing'))
    expect(pricingUrl).toBeUndefined()

    wrapper.unmount()
  })

  it('onMounted 交易与统计请求带尾部斜杠（查询串前）', async () => {
    const urls = []
    apiFetch.mockImplementation(async (url) => {
      urls.push(String(url))
      return jsonResponse({ pricing: {}, cents: '0', frozen_balance: 0, results: [], total_consumption_cents: 0 })
    })

    const wrapper = mount({ setup() { useBillingDashboard(); return () => null } })
    await flushPromises()

    const txUrl = urls.find((u) => u.includes('/billing/transactions'))
    expect(txUrl).toBeTruthy()
    expect(txUrl.startsWith('/api/tenant/850256677331562496/billing/transactions/')).toBe(true)
    expect(txUrl).not.toContain('/billing/transactions?')

    const statUrl = urls.find((u) => u.includes('/billing/statistics'))
    expect(statUrl).toBeTruthy()
    expect(statUrl.startsWith('/api/tenant/850256677331562496/billing/statistics/?')).toBe(true)
    expect(statUrl).not.toContain('/billing/statistics?month_start=') // 斜杠必须在 ? 之前

    wrapper.unmount()
  })
})

}

// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] useBillingDashboardStatistics.trailingSlash.test.js requires vitest runtime')
} else {
const { describe, expect, it, vi, beforeEach } = await import('vitest')

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

const { useBillingDashboardStatistics } = await import('./useBillingDashboardStatistics.js')

function jsonResponse(body, { ok = true, status = 200 } = {}) {
  return { ok, status, json: async () => body }
}

describe('useBillingDashboardStatistics billing URL 尾部斜杠（回归：not found 事故）', () => {
  beforeEach(() => {
    apiFetch.mockReset()
  })

  it('fetchStatistics 请求 billing/statistics/ 带尾部斜杠（查询串前）', async () => {
    let capturedUrl = ''
    apiFetch.mockImplementation(async (url) => {
      capturedUrl = String(url)
      return jsonResponse({ total_consumption_cents: 100, monthly_consumption_cents: 50, user_recharge_cents: 10 })
    })

    const c = useBillingDashboardStatistics({ value: '850256677331562496' })
    await c.fetchStatistics()

    expect(capturedUrl).toContain('/api/tenant/850256677331562496/billing/statistics/')
    expect(capturedUrl).not.toContain('/billing/statistics?')
    expect(c.totalConsumption.value).toBe(100)
    expect(c.monthlyConsumption.value).toBe(50)
    expect(c.totalRecharge.value).toBe(10)
  })

  it('仅有 *_points 时仍映射累计支付（回归：误读 *_cents 导致账单页 0.00）', async () => {
    apiFetch.mockImplementation(async () => jsonResponse({
      total_consumption_points: 55,
      monthly_consumption_points: 55,
      user_recharge_points: 55,
    }))
    const c = useBillingDashboardStatistics({ value: '877397588196749312' })
    await c.fetchStatistics()
    expect(c.totalRecharge.value).toBe(55)
    expect(c.totalConsumption.value).toBe(55)
    expect(c.monthlyConsumption.value).toBe(55)
  })
})

}

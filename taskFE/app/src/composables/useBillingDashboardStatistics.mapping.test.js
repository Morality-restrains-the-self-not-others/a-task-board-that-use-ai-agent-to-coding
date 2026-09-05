// @vitest-environment jsdom
/**
 * 账单统计字段映射：API 契约为 *_points（分）；误读 *_cents 会导致卡片恒 0.00。
 */
if (!process.env.VITEST) {
  console.log('[skip] useBillingDashboardStatistics.mapping.test.js requires vitest runtime')
} else {
  const { describe, expect, it, vi, beforeEach, afterEach } = await import('vitest')

  const hoisted = vi.hoisted(() => ({
    apiFetch: vi.fn(),
  }))

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch: (...args) => hoisted.apiFetch(...args),
  }))

  const {
    useBillingDashboardStatistics,
    localMonthStartISODate,
    readStatisticsCents,
  } = await import('./useBillingDashboardStatistics.js')

  function jsonResponse(body, { ok = true, status = 200 } = {}) {
    return { ok, status, json: async () => body }
  }

  describe('readStatisticsCents', () => {
    it('优先读取 *_points', () => {
      expect(readStatisticsCents({
        total_consumption_points: 12345,
        total_consumption_cents: 1,
      }, 'total_consumption')).toBe(12345)
    })

    it('无 *_points 时兼容遗留 *_cents', () => {
      expect(readStatisticsCents({
        user_recharge_cents: 20000,
      }, 'user_recharge')).toBe(20000)
    })

    it('字段缺失时为 0', () => {
      expect(readStatisticsCents({}, 'monthly_consumption')).toBe(0)
    })
  })

  describe('useBillingDashboardStatistics 字段契约', () => {
    beforeEach(() => {
      hoisted.apiFetch.mockReset()
    })

    it('将 total_consumption_points / user_recharge_points 写入卡片数值', async () => {
      let capturedUrl = ''
      hoisted.apiFetch.mockImplementation(async (url) => {
        capturedUrl = String(url)
        return jsonResponse({
          month_start: '2026-08-01',
          total_consumption_points: 12345,
          monthly_consumption_points: 100,
          user_recharge_points: 20000,
        })
      })

      const c = useBillingDashboardStatistics({ value: '877397588196749312' })
      await c.fetchStatistics()

      expect(c.totalConsumption.value).toBe(12345)
      expect(c.monthlyConsumption.value).toBe(100)
      expect(c.totalRecharge.value).toBe(20000)
      expect(capturedUrl).toContain('/billing/statistics/?month_start=')
    })

    it('仅返回遗留 *_cents 时仍映射，不恒为 0', async () => {
      hoisted.apiFetch.mockImplementation(async () => jsonResponse({
        total_consumption_cents: 55,
        monthly_consumption_cents: 55,
        user_recharge_cents: 55,
      }))

      const c = useBillingDashboardStatistics({ value: '877397588196749312' })
      await c.fetchStatistics()

      expect(c.totalConsumption.value).toBe(55)
      expect(c.totalRecharge.value).toBe(55)
    })
  })

  describe('localMonthStartISODate', () => {
    afterEach(() => {
      vi.useRealTimers()
    })

    it('使用本地日历月初，而非 UTC toISOString 日期', () => {
      vi.useFakeTimers()
      // 2026-08-01 00:30 GMT+8 == 2026-07-31T16:30:00.000Z
      vi.setSystemTime(new Date('2026-07-31T16:30:00.000Z'))
      const now = new Date()
      const utcDay = now.toISOString().slice(0, 10)
      const expected = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-01`
      expect(localMonthStartISODate(now)).toBe(expected)
      if (utcDay !== expected.slice(0, 10) && now.getDate() === 1) {
        expect(localMonthStartISODate(now)).not.toBe(`${utcDay.slice(0, 8)}01`)
      }
    })
  })
}

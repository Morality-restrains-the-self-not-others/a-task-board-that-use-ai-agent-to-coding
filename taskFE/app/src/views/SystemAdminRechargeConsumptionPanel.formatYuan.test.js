// @vitest-environment jsdom
/**
 * 超管充值消费汇总：*_points 为分，须按元展示。
 */
if (!process.env.VITEST) {
  console.log('[skip] SystemAdminRechargeConsumptionPanel.formatYuan.test.js requires vitest runtime')
} else {
  const { describe, expect, it, vi, beforeEach } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')

  const hoisted = vi.hoisted(() => ({
    apiFetch: vi.fn(),
  }))

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch: (...args) => hoisted.apiFetch(...args),
  }))

  const Page = (await import('./SystemAdminRechargeConsumptionPanel.vue')).default

  function jsonOk(body) {
    return {
      ok: true,
      status: 200,
      headers: { get: () => 'application/json' },
      json: async () => body,
      clone: function () {
        return this
      },
    }
  }

  describe('SystemAdminRechargeConsumptionPanel 积分列分转元', () => {
    beforeEach(() => {
      hoisted.apiFetch.mockReset()
      hoisted.apiFetch.mockResolvedValue(
        jsonOk({
          results: [
            {
              user_id: 'u1',
              email: 'a@b.com',
              recharge_amount_yuan: '10.00',
              commissionable_points: 55,
              unconsumed_points: 100,
              consumed_points: 55,
            },
          ],
          total: 1,
        })
      )
    })

    it('将 points=55 显示为 0.55', async () => {
      const wrapper = mount(Page)
      await flushPromises()
      const cells = wrapper.findAll('[data-testid="recharge-consumption-table"] td.text-right.tabular-nums')
      // 支付金额（已是元）+ 可分成 + 未消费 + 已消费
      expect(cells.map((c) => c.text().trim())).toEqual(['10.00', '0.55', '1.00', '0.55'])
    })
  })
}

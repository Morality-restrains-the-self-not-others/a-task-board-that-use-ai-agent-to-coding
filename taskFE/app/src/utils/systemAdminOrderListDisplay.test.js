// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] systemAdminOrderListDisplay.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const {
    SYSTEM_ADMIN_ORDER_STATUS_FILTERS,
    statusLabel,
    statusClass,
    paymentLabel,
    formatTime,
  } = await import('./systemAdminOrderListDisplay.js')

  describe('systemAdminOrderListDisplay', () => {
    it('statusLabel / statusClass 覆盖已知状态', () => {
      expect(statusLabel('paid')).toBe('已支付')
      expect(statusClass('pending')).toContain('yellow')
      expect(statusLabel('unknown')).toBe('unknown')
    })

    it('refunded 显示中文且筛选含已退款', () => {
      expect(statusLabel('refunded')).toBe('已退款')
      expect(statusClass('refunded')).toContain('gray')
      expect(
        SYSTEM_ADMIN_ORDER_STATUS_FILTERS.some(
          (s) => s.value === 'refunded' && s.label === '已退款',
        ),
      ).toBe(true)
      expect(SYSTEM_ADMIN_ORDER_STATUS_FILTERS.map((s) => s.label)).toEqual([
        '全部',
        '待支付',
        '已支付',
        '已取消',
        '已过期',
        '已退款',
      ])
    })

    it('paymentLabel 覆盖已知支付方式', () => {
      expect(paymentLabel('wechat')).toBe('微信支付')
      expect(paymentLabel('')).toBe('—')
    })

    it('formatTime 空值返回 —', () => {
      expect(formatTime('')).toBe('—')
      expect(formatTime(null)).toBe('—')
    })
  })
}

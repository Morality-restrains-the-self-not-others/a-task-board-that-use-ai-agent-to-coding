// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] billingOrderDisplay.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const {
    BILLING_ORDER_STATUS_FILTERS,
    billingOrderStatusLabel,
    isBillingOrderRefundable,
    billingOrderRefundApplicationStatus,
    billingOrderIsWechatChannel,
  } = await import('./billingOrderDisplay.js')

  describe('billingOrderDisplay', () => {
    it('statusLabel 映射常见状态', () => {
      expect(billingOrderStatusLabel('paid')).toBe('已支付')
      expect(billingOrderStatusLabel('refunded')).toBe('已退款')
      expect(billingOrderStatusLabel('weird')).toBe('weird')
    })

    it('筛选含已退款且与文案 SSOT 一致', () => {
      expect(
        BILLING_ORDER_STATUS_FILTERS.some(
          (s) => s.value === 'refunded' && s.label === '已退款',
        ),
      ).toBe(true)
      expect(BILLING_ORDER_STATUS_FILTERS.map((s) => s.label)).toEqual([
        '全部',
        '待支付',
        '已支付',
        '已取消',
        '已过期',
        '已退款',
      ])
    })

    it('isBillingOrderRefundable 仅微信/PayPal 正金额已支付', () => {
      expect(isBillingOrderRefundable({ status: 'paid', payment_method: 'wechat', total_yuan: 10 })).toBe(true)
      expect(isBillingOrderRefundable({ status: 'paid', payment_method: 'admin_grant', total_yuan: 10 })).toBe(false)
      expect(isBillingOrderRefundable({ status: 'pending', payment_method: 'wechat', total_yuan: 10 })).toBe(false)
      expect(isBillingOrderRefundable({ status: 'paid', payment_method: 'wechat', total_yuan: 0 })).toBe(false)
    })

    it('billingOrderRefundApplicationStatus 识别订单 pending/approved 退款申请', () => {
      const order = { id: '111' }
      expect(billingOrderRefundApplicationStatus(order, [])).toBe(null)
      expect(
        billingOrderRefundApplicationStatus(order, [
          { order_id: '111', status: 'pending' },
        ]),
      ).toBe('pending')
      expect(
        billingOrderRefundApplicationStatus(order, [
          { order_id: '111', status: 'approved' },
        ]),
      ).toBe('approved')
      expect(
        billingOrderRefundApplicationStatus(order, [
          { order_id: '222', status: 'pending' },
        ]),
      ).toBe(null)
      expect(
        billingOrderRefundApplicationStatus(order, [
          { order_id: '111', status: 'rejected' },
        ]),
      ).toBe(null)
    })

    it('isBillingOrderRefundable 在订单已有 pending/approved 退款申请时为 false', () => {
      const order = { id: '111', status: 'paid', payment_method: 'wechat', total_yuan: 10 }
      expect(isBillingOrderRefundable(order, [{ order_id: '111', status: 'pending' }])).toBe(false)
      expect(isBillingOrderRefundable(order, [{ order_id: '111', status: 'approved' }])).toBe(false)
      expect(isBillingOrderRefundable(order, [{ order_id: '999', status: 'pending' }])).toBe(false)
      expect(isBillingOrderRefundable(order, [{ order_id: '999', status: 'rejected' }])).toBe(true)
    })

    it('billingOrderIsWechatChannel 与后端 orderRefundChannel 对齐（开票仅微信）', () => {
      expect(billingOrderIsWechatChannel({ payment_method: 'wechat', payment_ref: 'wechat:wx123' })).toBe(true)
      expect(billingOrderIsWechatChannel({ payment_method: 'wechat' })).toBe(false)
      expect(billingOrderIsWechatChannel({ payment_ref: 'wechat:wx123' })).toBe(true)
      expect(billingOrderIsWechatChannel({ payment_method: 'paypal', payment_ref: 'paypal:pp1' })).toBe(false)
      expect(billingOrderIsWechatChannel({ payment_method: 'admin_grant', payment_ref: '' })).toBe(false)
      expect(billingOrderIsWechatChannel(null)).toBe(false)
    })
  })
}

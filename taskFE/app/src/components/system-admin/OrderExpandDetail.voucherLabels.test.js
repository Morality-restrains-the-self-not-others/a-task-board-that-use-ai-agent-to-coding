// @vitest-environment jsdom
// OPT-20260829-009 回归：OrderExpandDetail 凭证标签默认保持管理端文案，租户端可覆盖。
if (!process.env.VITEST) {
  console.log('[skip] OrderExpandDetail.voucherLabels.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const { mount } = await import('@vue/test-utils')
  const { default: OrderExpandDetail } = await import('./OrderExpandDetail.vue')

  const voucherProps = {
    items: [],
    outTradeNo: 'WX-MERCHANT-1',
    wechatTransactionId: '420000001',
  }

  describe('OrderExpandDetail 凭证标签', () => {
    it('默认保留管理端文案（商户订单号 / 微信支付单号）', () => {
      const wrapper = mount(OrderExpandDetail, { props: voucherProps })
      expect(wrapper.text()).toContain('商户订单号')
      expect(wrapper.text()).toContain('微信支付单号')
    })

    it('租户端可覆盖为商户单号 / 交易单号', () => {
      const wrapper = mount(OrderExpandDetail, {
        props: {
          ...voucherProps,
          outTradeNoLabel: '商户单号',
          wechatTransactionIdLabel: '交易单号',
        },
      })
      expect(wrapper.text()).toContain('商户单号')
      expect(wrapper.text()).toContain('交易单号')
      expect(wrapper.text()).not.toContain('商户订单号')
      expect(wrapper.text()).not.toContain('微信支付单号')
    })
  })
}

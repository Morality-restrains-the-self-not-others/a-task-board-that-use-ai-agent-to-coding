// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] OrderExpandDetail.profitSharing.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const { mount } = await import('@vue/test-utils')
  const { default: OrderExpandDetail } = await import('./OrderExpandDetail.vue')

  describe('OrderExpandDetail profit sharing isolation', () => {
    it('hides 分账 when profitSharing is null (tenant)', () => {
      const wrapper = mount(OrderExpandDetail, {
        props: { items: [], profitSharing: null },
      })
      expect(wrapper.text()).not.toContain('分账')
      expect(wrapper.find('[data-testid="admin-order-profit-sharing"]').exists()).toBe(false)
    })

    it('shows empty copy when profitSharing is []', () => {
      const wrapper = mount(OrderExpandDetail, {
        props: { items: [], profitSharing: [] },
      })
      expect(wrapper.get('[data-testid="admin-order-profit-sharing"]').text()).toContain('本订单无分账记录')
    })

    it('renders receiver and amount', () => {
      const wrapper = mount(OrderExpandDetail, {
        props: {
          items: [],
          profitSharing: [{ receiver_user_id: 'u-ref-1', amount_yuan: '0.55', status: 'pending' }],
        },
      })
      const text = wrapper.get('[data-testid="admin-order-profit-sharing"]').text()
      expect(text).toContain('u-ref-1')
      expect(text).toContain('0.55')
      expect(text).toContain('待分账')
    })

    it('renders AppID / OpenID pair for platform staff (OPT-20260826-012)', () => {
      const wrapper = mount(OrderExpandDetail, {
        props: {
          items: [],
          profitSharing: [
            { receiver_user_id: 'u-ref-1', app_id: 'wxapp', openid: 'o-open-1', amount_yuan: '0.55', status: 'pending' },
            { receiver_user_id: 'u-ref-2', app_id: '', openid: '', amount_yuan: '0.30', status: 'failed' },
          ],
        },
      })
      const text = wrapper.get('[data-testid="admin-order-profit-sharing"]').text()
      expect(text).toContain('wxapp')
      expect(text).toContain('o-open-1')
      expect(text).toContain('—')
    })

    it('prefers receiver_display over raw user id (OPT-20260822-024)', () => {
      const wrapper = mount(OrderExpandDetail, {
        props: {
          items: [],
          profitSharing: [
            { receiver_user_id: 'u-ref-2', receiver_display: '软刀', amount_yuan: '0.30', status: 'failed' },
          ],
        },
      })
      const text = wrapper.get('[data-testid="admin-order-profit-sharing"]').text()
      expect(text).toContain('软刀')
      expect(text).not.toContain('u-ref-2')
    })

    it('shows WeChat merchant and transaction ids from detail payload', () => {
      const wrapper = mount(OrderExpandDetail, {
        props: {
          items: [],
          outTradeNo: 'WX878981209491800064',
          wechatTransactionId: '4500000359202608221274536815',
        },
      })
      expect(wrapper.get('[data-testid="admin-order-out-trade-no"]').text()).toBe('WX878981209491800064')
      expect(wrapper.get('[data-testid="admin-order-wechat-transaction-id"]').text()).toBe(
        '4500000359202608221274536815',
      )
    })
  })
}

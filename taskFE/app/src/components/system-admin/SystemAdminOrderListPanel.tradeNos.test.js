// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] SystemAdminOrderListPanel.tradeNos.test.js requires vitest runtime')
} else {
  const { describe, expect, it, vi, beforeEach } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')

  const hoisted = vi.hoisted(() => ({
    apiFetch: vi.fn(),
    route: { query: {} },
    replace: vi.fn().mockResolvedValue(),
  }))

  vi.mock('../../utils/apiUtils.js', () => ({
    apiFetch: (...args) => hoisted.apiFetch(...args),
  }))

  vi.mock('vue-router', () => ({
    useRoute: () => hoisted.route,
    useRouter: () => ({ replace: hoisted.replace }),
  }))

  const Page = (await import('./SystemAdminOrderListPanel.vue')).default

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

  describe('SystemAdminOrderListPanel 展开展示微信凭证号', () => {
    beforeEach(() => {
      Element.prototype.scrollIntoView = vi.fn()
      hoisted.replace.mockReset()
      hoisted.replace.mockResolvedValue()
      hoisted.route.query = {
        tenant_id: 't1',
        order_id: '900001',
      }
      hoisted.apiFetch.mockReset()
      hoisted.apiFetch.mockImplementation(async (url) => {
        const u = String(url || '')
        if (u.includes('/tenant-options/')) {
          return jsonOk([{ id: 't1', name: 'Acme' }])
        }
        if (u.includes('/api/system-admin/orders/900001/')) {
          return jsonOk({
            items: [],
            out_trade_no: 'WX-ADMIN-900001',
            wechat_transaction_id: '420000ADMIN900001',
            profit_sharing: [],
          })
        }
        if (u.includes('/billing/orders/')) {
          return jsonOk({
            orders: [
              {
                id: '900001',
                order_number: 'ORD-A1',
                status: 'paid',
                total_yuan: '1.00',
                tenant_id: 't1',
              },
            ],
            total: 1,
            offset: 0,
            limit: 15,
          })
        }
        return jsonOk({ orders: [], total: 0 })
      })
    })

    it('展开行显示商户订单号与微信支付单号', async () => {
      const wrapper = mount(Page)
      await flushPromises()
      const row = wrapper.get('[data-order-id="900001"]')
      if (!wrapper.find('[data-testid="admin-order-out-trade-no"]').exists()) {
        await row.trigger('click')
        await flushPromises()
      }
      expect(wrapper.get('[data-testid="admin-order-out-trade-no"]').text()).toBe('WX-ADMIN-900001')
      expect(wrapper.get('[data-testid="admin-order-wechat-transaction-id"]').text()).toBe('420000ADMIN900001')
      wrapper.unmount()
    })
  })
}

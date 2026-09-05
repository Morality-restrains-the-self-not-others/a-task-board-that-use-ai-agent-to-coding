// @vitest-environment jsdom
/**
 * 超管订单列表：?tenant_id=&order_id= 深链应选中该租户，并在租户订单 API 带 order_id。
 */
if (!process.env.VITEST) {
  console.log('[skip] SystemAdminOrderListPanel.deeplink.test.js requires vitest runtime')
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

  describe('SystemAdminOrderListPanel 管理端订单深链', () => {
    beforeEach(() => {
      Element.prototype.scrollIntoView = vi.fn()
      hoisted.replace.mockReset()
      hoisted.replace.mockResolvedValue()
      hoisted.route.query = {
        tenant_id: '877397588196749312',
        order_id: '877596007691485184',
      }
      hoisted.apiFetch.mockReset()
      hoisted.apiFetch.mockImplementation(async (url) => {
        const u = String(url || '')
        if (u.includes('/tenant-options/')) {
          return jsonOk([{ id: '877397588196749312', name: 'Acme' }])
        }
        if (u.includes('/billing/orders/')) {
          return jsonOk({
            orders: [
              {
                id: '877596007691485184',
                order_number: 'ORD-1',
                status: 'paid',
                total_yuan: '1.00',
                tenant_id: '877397588196749312',
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

    it('请求租户订单列表时带 order_id，行带 data-order-id', async () => {
      const wrapper = mount(Page)
      await flushPromises()
      const orderCalls = hoisted.apiFetch.mock.calls
        .map((c) => String(c[0] || ''))
        .filter((u) => u.includes('/billing/orders/'))
      expect(orderCalls.some((u) => u.includes('order_id=877596007691485184'))).toBe(true)
      expect(orderCalls.some((u) => u.includes('/api/tenant/877397588196749312/'))).toBe(true)
      expect(wrapper.find('[data-order-id="877596007691485184"]').exists()).toBe(true)
    })

    it('?order_number= 深链自动按交易单号查询（可分享 URL）', async () => {
      // OPT-20260821-039: 刷新/转发 ?order_number= 应自动查询并命中该单。
      hoisted.route.query = { order_number: 'ORD-DEEPLINK-1' }
      hoisted.apiFetch.mockImplementation(async (url) => {
        const u = String(url || '')
        if (u.includes('/tenant-options/')) {
          return jsonOk([])
        }
        if (u.includes('/system-admin/orders/')) {
          return jsonOk({
            orders: [{ id: 'x1', order_number: 'ORD-DEEPLINK-1', status: 'paid', total_yuan: '1.00', tenant_id: 't1' }],
            total: 1,
            offset: 0,
            limit: 15,
          })
        }
        return jsonOk({ orders: [], total: 0 })
      })
      const wrapper = mount(Page)
      await flushPromises()
      const adminCalls = hoisted.apiFetch.mock.calls
        .map((c) => String(c[0] || ''))
        .filter((u) => u.includes('/system-admin/orders/'))
      expect(adminCalls.some((u) => u.includes('order_number=ORD-DEEPLINK-1'))).toBe(true)
      expect(wrapper.text()).toContain('ORD-DEEPLINK-1')
    })

    it('?wechat_account= 深链自动按微信关联账号查询', async () => {
      hoisted.route.query = { wechat_account: '微信昵称甲' }
      hoisted.apiFetch.mockImplementation(async (url) => {
        const u = String(url || '')
        if (u.includes('/tenant-options/')) {
          return jsonOk([])
        }
        if (u.includes('/system-admin/orders/')) {
          return jsonOk({
            orders: [{ id: 'w1', order_number: 'ORD-WX-1', status: 'paid', total_yuan: '1.00', tenant_id: 't1' }],
            total: 1,
            offset: 0,
            limit: 15,
          })
        }
        return jsonOk({ orders: [], total: 0 })
      })
      const wrapper = mount(Page)
      await flushPromises()
      const adminCalls = hoisted.apiFetch.mock.calls
        .map((c) => String(c[0] || ''))
        .filter((u) => u.includes('/system-admin/orders/'))
      expect(adminCalls.some((u) => u.includes('wechat_account='))).toBe(true)
      expect(adminCalls.some((u) => u.includes('order_number='))).toBe(false)
      expect(wrapper.text()).toContain('ORD-WX-1')
      expect(wrapper.get('[data-testid="admin-order-lookup-tab-wechat"]').attributes('aria-selected')).toBe('true')
    })

    it('全部租户模式（无 tenant_id）请求 /api/system-admin/orders/ 也带 order_id', async () => {
      // OPT-20260819-033：深链 order_id 在「全部租户」下走 doAdminListOrders 对齐。
      hoisted.route.query = { order_id: '877596007691485184' }
      hoisted.apiFetch.mockImplementation(async (url) => {
        const u = String(url || '')
        if (u.includes('/tenant-options/')) {
          return jsonOk([])
        }
        if (u.includes('/system-admin/orders/')) {
          return jsonOk({
            orders: [
              {
                id: '877596007691485184',
                order_number: 'ORD-1',
                status: 'paid',
                total_yuan: '1.00',
                tenant_id: '877397588196749312',
              },
            ],
            total: 1,
            offset: 0,
            limit: 15,
          })
        }
        return jsonOk({ orders: [], total: 0 })
      })
      const wrapper = mount(Page)
      await flushPromises()
      const adminCalls = hoisted.apiFetch.mock.calls
        .map((c) => String(c[0] || ''))
        .filter((u) => u.includes('/system-admin/orders/'))
      expect(adminCalls.some((u) => u.includes('order_id=877596007691485184'))).toBe(true)
      expect(wrapper.find('[data-order-id="877596007691485184"]').exists()).toBe(true)
    })

    it('状态筛选含已退款，refunded 订单徽章为中文', async () => {
      hoisted.route.query = { tenant_id: '877397588196749312' }
      hoisted.apiFetch.mockImplementation(async (url) => {
        const u = String(url || '')
        if (u.includes('/tenant-options/')) {
          return jsonOk([{ id: '877397588196749312', name: 'Acme' }])
        }
        if (u.includes('/billing/orders/')) {
          return jsonOk({
            orders: [
              {
                id: '877596007691485184',
                order_number: 'ORD-REF',
                status: 'refunded',
                total_yuan: '1.00',
                tenant_id: '877397588196749312',
              },
            ],
            total: 1,
            offset: 0,
            limit: 15,
          })
        }
        return jsonOk({ orders: [], total: 0 })
      })
      const wrapper = mount(Page)
      await flushPromises()
      const filterBar = wrapper.find('[data-testid="order-status-filters"]')
      expect(filterBar.text()).toContain('已退款')
      expect(filterBar.text()).toContain('全部')
      expect(wrapper.find('[data-testid="order-status-badge"]').text()).toBe('已退款')
    })
  })
}

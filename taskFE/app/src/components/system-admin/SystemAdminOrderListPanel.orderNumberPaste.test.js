// @vitest-environment jsdom
/**
 * 管理端订单记录：按交易单号 GET 列表精确查询。
 */
if (!process.env.VITEST) {
  console.log('[skip] SystemAdminOrderListPanel.orderNumberPaste.test.js requires vitest runtime')
} else {
  const { describe, expect, it, vi, beforeEach } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')

  const hoisted = vi.hoisted(() => ({
    apiFetch: vi.fn(),
    route: { path: '/system-admin/order-records/', query: {} },
    push: vi.fn().mockResolvedValue(undefined),
    replace: vi.fn().mockResolvedValue(undefined),
  }))

  vi.mock('vue-router', () => ({
    useRoute: () => hoisted.route,
    useRouter: () => ({ push: hoisted.push, replace: hoisted.replace }),
  }))
  vi.mock('../../composables/useSystemAdminOrderListDeepLink.js', () => ({
    useSystemAdminOrderListDeepLink: () => ({ applyFromRoute: vi.fn() }),
  }))
  vi.mock('../../utils/apiUtils.js', () => ({
    apiFetch: (...args) => hoisted.apiFetch(...args),
  }))

  const Panel = (await import('./SystemAdminOrderListPanel.vue')).default

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

  const sampleOrder = (overrides = {}) => ({
    id: '987654321',
    tenant_id: '1234567890',
    order_number: 'ORD-20260821-1234567890-987654321',
    status: 'paid',
    total_yuan: '10.00',
    payment_method: 'wechat',
    created_at: '2026-08-21 12:00:00',
    paid_at: '2026-08-21 12:01:00',
    ...overrides,
  })

  const listUrl = (url) => {
    if (String(url).includes('/api/system-admin/orders/')) {
      const q = new URL(String(url), 'http://local.test').searchParams.get('order_number') || ''
      if (q === 'MISSING-NO') {
        return jsonOk({ orders: [], total: 0, offset: 0 })
      }
      if (q) {
        return jsonOk({
          orders: [sampleOrder({ order_number: q, id: q.includes('wx') ? '111' : '987654321' })],
          total: 1,
          offset: 0,
        })
      }
      return jsonOk({ orders: [], total: 0, offset: 0 })
    }
    if (String(url).includes('/billing/orders/')) {
      return jsonOk({ items: [], buyer_note: '' })
    }
    return jsonOk([])
  }

  describe('SystemAdminOrderListPanel 交易单号查询', () => {
    beforeEach(() => {
      hoisted.apiFetch.mockReset()
      hoisted.apiFetch.mockImplementation(listUrl)
      hoisted.push.mockReset()
      hoisted.route.query = {}
    })

    it('展示号查询请求管理端列表并展示该单', async () => {
      const wrapper = mount(Panel)
      await flushPromises()
      const num = 'ORD-20260821-1234567890-987654321'
      await wrapper.get('[data-testid="order-number-paste-input"]').setValue(num)
      await wrapper.get('[data-testid="order-number-paste-jump"]').trigger('click')
      await flushPromises()

      const called = hoisted.apiFetch.mock.calls.map((c) => String(c[0]))
      expect(called.some((u) => u.includes('/api/system-admin/orders/') && u.includes('order_number='))).toBe(true)
      expect(wrapper.text()).toContain(num)
    })

    it('查询成功后 router.replace 写入 ?order_number=（可分享 URL）', async () => {
      // OPT-20260821-039: 查询结果写入可分享 URL，刷新/转发仍停留在这一单。
      const wrapper = mount(Panel)
      await flushPromises()
      hoisted.replace.mockClear()
      const num = 'ORD-20260821-1234567890-987654321'
      await wrapper.get('[data-testid="order-number-paste-input"]').setValue(num)
      await wrapper.get('[data-testid="order-number-paste-jump"]').trigger('click')
      await flushPromises()
      expect(hoisted.replace).toHaveBeenCalled()
      const arg = hoisted.replace.mock.calls[0][0]
      expect(arg && arg.query && arg.query.order_number).toBe(num)
    })

    it('主键与支付渠道号同样带 order_number query', async () => {
      const wrapper = mount(Panel)
      await flushPromises()
      await wrapper.get('[data-testid="order-number-paste-input"]').setValue('wx_txn_abc')
      await wrapper.get('[data-testid="order-number-paste-jump"]').trigger('click')
      await flushPromises()
      const called = hoisted.apiFetch.mock.calls.map((c) => String(c[0]))
      expect(called.some((u) => u.includes('order_number=wx_txn_abc'))).toBe(true)
    })

    it('微信支付单号导出反引号不进入 query', async () => {
      const wrapper = mount(Panel)
      await flushPromises()
      await wrapper.get('[data-testid="order-number-paste-input"]').setValue('`4500000359202608221274536815')
      await wrapper.get('[data-testid="order-number-paste-jump"]').trigger('click')
      await flushPromises()
      const called = hoisted.apiFetch.mock.calls.map((c) => String(c[0]))
      expect(called.some((u) => u.includes('order_number=4500000359202608221274536815'))).toBe(true)
      expect(called.some((u) => u.includes('%60'))).toBe(false)
    })

    it('空输入不请求列表', async () => {
      const wrapper = mount(Panel)
      await flushPromises()
      hoisted.apiFetch.mockClear()
      await wrapper.get('[data-testid="order-number-paste-jump"]').trigger('click')
      await flushPromises()
      expect(hoisted.apiFetch).not.toHaveBeenCalled()
      expect(wrapper.get('[data-testid="order-number-parse-error"]').text()).toContain('请输入交易单号')
    })

    it('未命中展示空态', async () => {
      const wrapper = mount(Panel)
      await flushPromises()
      await wrapper.get('[data-testid="order-number-paste-input"]').setValue('MISSING-NO')
      await wrapper.get('[data-testid="order-number-paste-jump"]').trigger('click')
      await flushPromises()
      expect(wrapper.get('[data-testid="order-trade-no-empty"]').text()).toContain('未找到该交易单号')
    })

    it('列表 502 展示错误并带 data-traceId', async () => {
      hoisted.apiFetch.mockImplementation((url) => {
        if (String(url).includes('/api/system-admin/orders/')) {
          return {
            ok: false,
            status: 502,
            headers: {
              get: (name) => (String(name).toLowerCase() === 'x-trace-id' ? 'gw-trace-502list' : null),
            },
            json: async () => {
              throw new SyntaxError("Unexpected token '<'")
            },
            clone: function () {
              return this
            },
          }
        }
        return jsonOk([])
      })
      const wrapper = mount(Panel)
      await flushPromises()
      await wrapper.get('[data-testid="order-number-paste-input"]').setValue('ORD-20260821-1-2')
      await wrapper.get('[data-testid="order-number-paste-jump"]').trigger('click')
      await flushPromises()
      const el = wrapper.get('[data-traceId], [data-traceid]')
      expect(el.text().length).toBeGreaterThan(0)
      expect(el.attributes('data-traceid') || el.attributes('data-traceId')).toBe('gw-trace-502list')
    })
  })
}

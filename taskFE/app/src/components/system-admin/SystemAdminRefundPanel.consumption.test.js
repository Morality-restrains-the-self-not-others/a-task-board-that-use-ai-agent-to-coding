// @vitest-environment jsdom
/**
 * 超管退款审批：批准/拒绝弹层须展示关联订单的资源消耗。
 */
if (!process.env.VITEST) {
  console.log('[skip] SystemAdminRefundPanel.consumption.test.js requires vitest runtime')
} else {
  const { describe, expect, it, vi, beforeEach } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')

  const hoisted = vi.hoisted(() => ({
    apiFetch: vi.fn(),
  }))

  vi.mock('../../utils/apiUtils.js', () => ({
    apiFetch: (...args) => hoisted.apiFetch(...args),
  }))

  vi.mock('../../utils/modalService.js', () => ({
    default: { confirm: vi.fn() },
  }))

  const Page = (await import('./SystemAdminRefundPanel.vue')).default

  const pendingRow = {
    id: '877617449518792704',
    tenant_id: '877397588196749312',
    order_id: '8775960076914851840',
    frozen_points: 55,
    status: 'pending',
    reason: '体验不满意，申请全额退款',
    created_at: '2026-08-18T18:18:00Z',
  }

  const consumptionPayload = {
    resource_consumption: {
      task_post: {
        source_kind: 'purchase',
        granted: 10,
        consumed: 3,
        remaining: 7,
        events: [
          {
            created_at: '2026-08-19T10:00:00Z',
            task_id: 'task-consumed-001',
            action: 'create',
            quantity: 1,
            source_kind: 'purchase',
          },
        ],
      },
    },
  }

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

  describe('SystemAdminRefundPanel 审批弹层资源消耗', () => {
    beforeEach(() => {
      hoisted.apiFetch.mockReset()
      hoisted.apiFetch.mockImplementation(async (url, opts = {}) => {
        const u = String(url || '')
        if (u.includes('/refund-policy/') && (!opts.method || opts.method === 'GET')) {
          return jsonOk({ enabled: true })
        }
        if (u.includes('/billing/orders/')) {
          return jsonOk(consumptionPayload)
        }
        if (u.includes('/refund-applications/')) {
          return jsonOk({ results: [pendingRow] })
        }
        return jsonOk({})
      })
    })

    it('点击批准后弹层展示已消耗/剩余及 task_id', async () => {
      const wrapper = mount(Page)
      await flushPromises()

      await wrapper.get('[data-testid="refund-approve-btn"]').trigger('click')
      await flushPromises()

      const modal = wrapper.get('[data-testid="refund-approve-modal"]')
      const block = modal.get('[data-testid="order-resource-consumption"]')
      expect(block.text()).toContain('已消耗 3')
      expect(block.text()).toContain('剩余 7')
      expect(block.text()).toContain('task-consumed-001')
      expect(hoisted.apiFetch.mock.calls.some(([url]) =>
        String(url).includes('/api/tenant/877397588196749312/billing/orders/8775960076914851840/'),
      )).toBe(true)
    })

    it('点击拒绝后弹层同样展示资源消耗', async () => {
      const wrapper = mount(Page)
      await flushPromises()

      await wrapper.get('[data-testid="refund-reject-btn"]').trigger('click')
      await flushPromises()

      const modal = wrapper.get('[data-testid="refund-reject-modal"]')
      expect(modal.get('[data-testid="order-resource-consumption"]').text()).toContain('已消耗 3')
    })
  })
}

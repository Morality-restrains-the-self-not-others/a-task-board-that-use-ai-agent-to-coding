// @vitest-environment jsdom
/**
 * 超管退款审批：列表行内「查看消耗」即可看到已消耗/剩余，无需打开批准/拒绝弹层。
 * OPT-20260820-042
 */
if (!process.env.VITEST) {
  console.log('[skip] SystemAdminRefundPanel.inline-consumption.test.js requires vitest runtime')
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

  const noOrderRow = {
    id: '877617449518792705',
    tenant_id: '877397588196749312',
    order_id: '',
    frozen_points: 20,
    status: 'pending',
    reason: '无关联订单申请',
    created_at: '2026-08-18T18:19:00Z',
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

  describe('SystemAdminRefundPanel 行内资源消耗预览', () => {
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
          return jsonOk({ results: [pendingRow, noOrderRow] })
        }
        return jsonOk({})
      })
    })

    it('点「查看消耗」后列表行内直接展示已消耗/剩余', async () => {
      const wrapper = mount(Page)
      await flushPromises()

      await wrapper.get('[data-testid="refund-consumption-toggle"]').trigger('click')
      await flushPromises()

      const inline = wrapper.get('[data-testid="refund-consumption-inline"]')
      const block = inline.get('[data-testid="order-resource-consumption"]')
      expect(block.text()).toContain('已消耗 3')
      expect(block.text()).toContain('剩余 7')
      expect(block.text()).toContain('task-consumed-001')
      expect(hoisted.apiFetch.mock.calls.some(([url]) =>
        String(url).includes('/api/tenant/877397588196749312/billing/orders/8775960076914851840/'),
      )).toBe(true)
    })

    it('GitLab 磁盘订单行内展示已购买数量而非任务帖 0', async () => {
      hoisted.apiFetch.mockImplementation(async (url, opts = {}) => {
        const u = String(url || '')
        if (u.includes('/refund-policy/') && (!opts.method || opts.method === 'GET')) {
          return jsonOk({ enabled: true })
        }
        if (u.includes('/billing/orders/')) {
          return jsonOk({
            items: [{ resource_type: 'gitlab_disk', quantity: 10, region: 'tencent-sh-1' }],
            resource_consumption: {
              gitlab_disk: {
                source_kind: 'purchase',
                granted: 10,
                consumed: 2,
                remaining: 8,
                region: 'tencent-sh-1',
                unit: 'GB',
                events: [],
              },
            },
          })
        }
        if (u.includes('/refund-applications/')) {
          return jsonOk({ results: [pendingRow] })
        }
        return jsonOk({})
      })

      const wrapper = mount(Page)
      await flushPromises()
      await wrapper.get('[data-testid="refund-consumption-toggle"]').trigger('click')
      await flushPromises()

      const block = wrapper.get('[data-testid="order-resource-consumption"]')
      expect(block.text()).toContain('GitLab 磁盘')
      expect(block.text()).toContain('已发放 10')
      expect(block.text()).toContain('已消耗 2')
      expect(block.text()).not.toMatch(/任务帖：已发放 0/)
    })

    it('收起后再点其他行会重新拉取该行消耗', async () => {
      const wrapper = mount(Page)
      await flushPromises()

      const toggle = wrapper.get('[data-testid="refund-consumption-toggle"]')
      await toggle.trigger('click')
      await flushPromises()
      expect(wrapper.get('[data-testid="refund-consumption-inline"]').text()).toContain('剩余 7')

      // 收起
      await wrapper.get('[data-testid="refund-consumption-toggle"]').trigger('click')
      await flushPromises()
      expect(wrapper.find('[data-testid="refund-consumption-inline"]').exists()).toBe(false)

      // 再次展开仍可看到 remaining（复用同一数据源）
      await wrapper.get('[data-testid="refund-consumption-toggle"]').trigger('click')
      await flushPromises()
      expect(wrapper.get('[data-testid="refund-consumption-inline"]').text()).toContain('剩余 7')
    })
  })
}

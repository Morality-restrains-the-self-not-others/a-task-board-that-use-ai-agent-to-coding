// @vitest-environment jsdom
/**
 * 退款审批表「冻结金额」列：API frozen_points 为分（yuan cents），须按元展示。
 * 回归：曾直接渲染 55，用户期望 0.55元。
 */
if (!process.env.VITEST) {
  console.log('[skip] SystemAdminRefundPanel.frozen-amount.test.js requires vitest runtime')
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
    default: {
      confirm: vi.fn().mockResolvedValue(true),
    },
  }))

  const Page = (await import('./SystemAdminRefundPanel.vue')).default

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

  describe('SystemAdminRefundPanel 冻结金额分转元', () => {
    beforeEach(() => {
      hoisted.apiFetch.mockReset()
      hoisted.apiFetch.mockImplementation(async (url) => {
        const u = String(url || '')
        if (u.includes('/refund-policy/')) {
          return jsonOk({ enabled: true })
        }
        if (u.includes('/refund-applications/')) {
          return jsonOk({
            results: [
              {
                id: 'app-1',
                tenant_id: 't1',
                order_id: 'ord-1',
                frozen_points: 55,
                status: 'pending',
                created_at: '2026-08-19T08:00:00Z',
              },
            ],
          })
        }
        return jsonOk({})
      })
    })

    it('将 frozen_points=55 显示为 0.55元', async () => {
      const wrapper = mount(Page)
      await flushPromises()

      const amountCell = wrapper.find('td.px-3.py-2.text-right.tabular-nums')
      expect(amountCell.exists()).toBe(true)
      expect(amountCell.text().trim()).toBe('0.55元')
    })

    it('frozen_points 缺失时显示 —', async () => {
      hoisted.apiFetch.mockImplementation(async (url) => {
        const u = String(url || '')
        if (u.includes('/refund-policy/')) return jsonOk({ enabled: true })
        if (u.includes('/refund-applications/')) {
          return jsonOk({
            results: [
              {
                id: 'app-2',
                tenant_id: 't1',
                status: 'pending',
                created_at: '2026-08-19T08:00:00Z',
              },
            ],
          })
        }
        return jsonOk({})
      })

      const wrapper = mount(Page)
      await flushPromises()

      const amountCell = wrapper.find('td.px-3.py-2.text-right.tabular-nums')
      expect(amountCell.text().trim()).toBe('—')
    })
  })
}

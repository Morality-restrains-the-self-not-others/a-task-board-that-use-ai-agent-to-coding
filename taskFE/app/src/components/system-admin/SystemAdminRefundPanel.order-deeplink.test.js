// @vitest-environment jsdom
/**
 * 超管退款审批「关联订单」须链到管理端订单 Tab，而非租户 /billing/orders/。
 */
if (!process.env.VITEST) {
  console.log('[skip] SystemAdminRefundPanel.order-deeplink.test.js requires vitest runtime')
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

  describe('SystemAdminRefundPanel 关联订单深链', () => {
    beforeEach(() => {
      hoisted.apiFetch.mockReset()
      hoisted.apiFetch.mockImplementation(async (url) => {
        const u = String(url || '')
        if (u.includes('/refund-policy/')) return jsonOk({ enabled: true })
        if (u.includes('/refund-applications/')) {
          return jsonOk({
            results: [
              {
                id: 'app-1',
                tenant_id: '877397588196749312',
                order_id: '877596007691485184',
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

    it('href 指向 /system-admin/order-records/ 且带 tenant_id 与 order_id', async () => {
      const wrapper = mount(Page)
      await flushPromises()
      const link = wrapper.get('[data-testid="refund-related-order-link"]')
      expect(link.attributes('href')).toBe(
        '/system-admin/order-records/?tenant_id=877397588196749312&order_id=877596007691485184',
      )
      expect(link.attributes('href')).not.toContain('/tenant/')
      expect(link.text()).toBe('877596007691485184')
    })
  })
}

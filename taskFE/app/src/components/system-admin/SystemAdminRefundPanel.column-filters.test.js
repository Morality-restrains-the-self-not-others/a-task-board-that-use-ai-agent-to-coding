// @vitest-environment jsdom
/**
 * OPT-20260823-045 回归：退款审批表头列过滤（租户 ID / 关联订单）。
 * - 表头第二行渲染 `[data-testid=refund-list-column-filters]`
 * - 输入租户 ID / 订单 ID 后 debounce 400ms 自动带 query 拉列表
 * - 「重置」清空列过滤并重拉（不带 tenant_id/order_id）
 */
if (!process.env.VITEST) {
  console.log('[skip] SystemAdminRefundPanel.column-filters.test.js requires vitest runtime')
} else {
  const { describe, expect, it, vi, beforeEach, afterEach } = await import('vitest')
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

  const refundAppUrlCalls = () =>
    hoisted.apiFetch.mock.calls
      .map(([url]) => String(url || ''))
      .filter((u) => u.includes('/refund-applications/'))

  const mountPanel = async () => {
    const wrapper = mount(Page)
    await flushPromises()
    return wrapper
  }

  describe('SystemAdminRefundPanel 表头列过滤（OPT-20260823-045）', () => {
    beforeEach(() => {
      hoisted.apiFetch.mockReset()
      hoisted.apiFetch.mockImplementation(async (url) => {
        const u = String(url || '')
        if (u.includes('/refund-policy/')) return jsonOk({ enabled: true })
        if (u.includes('/refund-applications/')) {
          return jsonOk({ results: [] })
        }
        return jsonOk({})
      })
      vi.useFakeTimers()
    })

    afterEach(() => {
      vi.useRealTimers()
    })

    it('表头渲染列过滤行（tenant_id / order_id 输入 + 重置按钮）', async () => {
      const wrapper = await mountPanel()
      const row = wrapper.find('[data-testid="refund-list-column-filters"]')
      expect(row.exists()).toBe(true)
      expect(wrapper.find('[data-alias="RefundFilterTenantId"]').exists()).toBe(true)
      expect(wrapper.find('[data-alias="RefundFilterOrderId"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="refund-list-column-filters-reset"]').exists()).toBe(true)
    })

    it('输入租户 ID 后 debounce 自动带 tenant_id 拉列表', async () => {
      const wrapper = await mountPanel()
      const lastBefore = refundAppUrlCalls().length

      await wrapper.find('[data-alias="RefundFilterTenantId"]').setValue('877397588196749312')
      await vi.advanceTimersByTimeAsync(400)
      await flushPromises()

      const urls = refundAppUrlCalls()
      expect(urls.length).toBeGreaterThan(lastBefore)
      const latest = urls[urls.length - 1]
      expect(latest).toContain('tenant_id=877397588196749312')
    })

    it('输入关联订单后 debounce 自动带 order_id 拉列表', async () => {
      const wrapper = await mountPanel()
      const lastBefore = refundAppUrlCalls().length

      await wrapper.find('[data-alias="RefundFilterOrderId"]').setValue('878000000000000001')
      await vi.advanceTimersByTimeAsync(400)
      await flushPromises()

      const urls = refundAppUrlCalls()
      expect(urls.length).toBeGreaterThan(lastBefore)
      expect(urls[urls.length - 1]).toContain('order_id=878000000000000001')
    })

    it('重置按钮清空列过滤并重拉（不再带 tenant_id/order_id）', async () => {
      const wrapper = await mountPanel()

      await wrapper.find('[data-alias="RefundFilterTenantId"]').setValue('111')
      await vi.advanceTimersByTimeAsync(400)
      await flushPromises()
      const filtered = refundAppUrlCalls().pop()
      expect(filtered).toContain('tenant_id=111')

      await wrapper.find('[data-testid="refund-list-column-filters-reset"]').trigger('click')
      await flushPromises()

      const latest = refundAppUrlCalls().pop()
      expect(latest).not.toContain('tenant_id=111')
      expect(latest).not.toContain('order_id=')
    })
  })
}

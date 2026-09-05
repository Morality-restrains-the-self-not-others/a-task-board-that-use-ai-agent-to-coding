// @vitest-environment jsdom
/**
 * OPT-20260823-045 回归：开票申请表头列过滤（申请 ID / 租户 / 订单 / 抬头）。
 * - 表头第二行渲染 `[data-testid=invoice-list-column-filters]`
 * - 输入过滤值后 debounce 400ms 自动带 query 拉列表
 * - 「重置」清空列过滤并重拉（不再带 id/tenant_id/order_id/buyer_name）
 */
if (!process.env.VITEST) {
  console.log('[skip] SystemAdminInvoicePanel.column-filters.test.js requires vitest runtime')
} else {
  const { describe, expect, it, vi, beforeEach, afterEach } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')

  const hoisted = vi.hoisted(() => ({
    apiFetch: vi.fn(),
  }))

  vi.mock('../../utils/apiUtils.js', () => ({
    apiFetch: (...args) => hoisted.apiFetch(...args),
  }))

  vi.mock('../../utils/traceId.js', () => ({
    extractTraceId: () => '',
  }))

  const Page = (await import('./SystemAdminInvoicePanel.vue')).default

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

  const invoiceAppUrlCalls = () =>
    hoisted.apiFetch.mock.calls
      .map(([url]) => String(url || ''))
      .filter((u) => u.includes('/invoice-applications/'))

  const mountPanel = async () => {
    const wrapper = mount(Page)
    await flushPromises()
    return wrapper
  }

  describe('SystemAdminInvoicePanel 表头列过滤（OPT-20260823-045）', () => {
    beforeEach(() => {
      hoisted.apiFetch.mockReset()
      hoisted.apiFetch.mockImplementation(async () => jsonOk({ results: [] }))
      vi.useFakeTimers()
    })

    afterEach(() => {
      vi.useRealTimers()
    })

    it('表头渲染列过滤行（id/tenant/order/buyer_name 输入 + 重置按钮）', async () => {
      const wrapper = await mountPanel()
      const row = wrapper.find('[data-testid="invoice-list-column-filters"]')
      expect(row.exists()).toBe(true)
      expect(wrapper.find('[data-alias="InvoiceFilterId"]').exists()).toBe(true)
      expect(wrapper.find('[data-alias="InvoiceFilterTenantId"]').exists()).toBe(true)
      expect(wrapper.find('[data-alias="InvoiceFilterOrderId"]').exists()).toBe(true)
      expect(wrapper.find('[data-alias="InvoiceFilterBuyerName"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="invoice-list-column-filters-reset"]').exists()).toBe(true)
    })

    it('输入申请 ID 后 debounce 自动带 id 拉列表', async () => {
      const wrapper = await mountPanel()
      const lastBefore = invoiceAppUrlCalls().length

      await wrapper.find('[data-alias="InvoiceFilterId"]').setValue('1001')
      await vi.advanceTimersByTimeAsync(400)
      await flushPromises()

      const urls = invoiceAppUrlCalls()
      expect(urls.length).toBeGreaterThan(lastBefore)
      expect(urls[urls.length - 1]).toContain('id=1001')
    })

    it('输入租户 ID 后 debounce 自动带 tenant_id 拉列表', async () => {
      const wrapper = await mountPanel()
      const lastBefore = invoiceAppUrlCalls().length

      await wrapper.find('[data-alias="InvoiceFilterTenantId"]').setValue('877397588196749312')
      await vi.advanceTimersByTimeAsync(400)
      await flushPromises()

      const urls = invoiceAppUrlCalls()
      expect(urls.length).toBeGreaterThan(lastBefore)
      expect(urls[urls.length - 1]).toContain('tenant_id=877397588196749312')
    })

    it('输入关联订单后 debounce 自动带 order_id 拉列表', async () => {
      const wrapper = await mountPanel()
      const lastBefore = invoiceAppUrlCalls().length

      await wrapper.find('[data-alias="InvoiceFilterOrderId"]').setValue('878000000000000001')
      await vi.advanceTimersByTimeAsync(400)
      await flushPromises()

      const urls = invoiceAppUrlCalls()
      expect(urls.length).toBeGreaterThan(lastBefore)
      expect(urls[urls.length - 1]).toContain('order_id=878000000000000001')
    })

    it('输入抬头名称后 debounce 自动带 buyer_name 拉列表', async () => {
      const wrapper = await mountPanel()
      const lastBefore = invoiceAppUrlCalls().length

      await wrapper.find('[data-alias="InvoiceFilterBuyerName"]').setValue('张三科技')
      await vi.advanceTimersByTimeAsync(400)
      await flushPromises()

      const urls = invoiceAppUrlCalls()
      expect(urls.length).toBeGreaterThan(lastBefore)
      expect(decodeURIComponent(urls[urls.length - 1])).toContain('buyer_name=张三科技')
    })

    it('重置按钮清空列过滤并重拉（不再带 id/tenant_id/order_id/buyer_name）', async () => {
      const wrapper = await mountPanel()

      await wrapper.find('[data-alias="InvoiceFilterTenantId"]').setValue('111')
      await vi.advanceTimersByTimeAsync(400)
      await flushPromises()
      const filtered = invoiceAppUrlCalls().pop()
      expect(filtered).toContain('tenant_id=111')

      await wrapper.find('[data-testid="invoice-list-column-filters-reset"]').trigger('click')
      await flushPromises()

      const latest = invoiceAppUrlCalls().pop()
      expect(latest).not.toContain('tenant_id=111')
      expect(latest).not.toContain('buyer_name=')
    })
  })
}

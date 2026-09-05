// @vitest-environment jsdom
/**
 * OPT-20260823-045 回归：待分账队列表头列过滤（订单号 / 租户 ID / 接收方 ID）。
 * - 表头第二行渲染 `[data-testid=profit-sharing-list-column-filters]`
 * - 输入过滤值后 debounce 400ms 自动带 query 拉列表
 * - 「重置」清空列过滤并重拉（不再带 order_number/tenant_id/receiver_user_id）
 */
if (!process.env.VITEST) {
  console.log('[skip] SystemAdminProfitSharingPanel.column-filters.test.js requires vitest runtime')
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

  const Page = (await import('./SystemAdminProfitSharingPanel.vue')).default

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

  const psUrlCalls = () =>
    hoisted.apiFetch.mock.calls
      .map(([url]) => String(url || ''))
      .filter((u) => u.includes('/profit-sharing/'))

  const mountPanel = async () => {
    const wrapper = mount(Page)
    await flushPromises()
    return wrapper
  }

  describe('SystemAdminProfitSharingPanel 表头列过滤（OPT-20260823-045）', () => {
    beforeEach(() => {
      hoisted.apiFetch.mockReset()
      hoisted.apiFetch.mockImplementation(async () => jsonOk({ items: [], total: 0 }))
      vi.useFakeTimers()
    })

    afterEach(() => {
      vi.useRealTimers()
    })

    it('表头渲染列过滤行（订单号/商户单号/租户/接收方输入 + 重置按钮）', async () => {
      const wrapper = await mountPanel()
      const row = wrapper.find('[data-testid="profit-sharing-list-column-filters"]')
      expect(row.exists()).toBe(true)
      expect(wrapper.find('[data-alias="ProfitSharingFilterOrderNumber"]').exists()).toBe(true)
      expect(wrapper.find('[data-alias="ProfitSharingFilterOutTradeNo"]').exists()).toBe(true)
      expect(wrapper.find('[data-alias="ProfitSharingFilterTenantId"]').exists()).toBe(true)
      expect(wrapper.find('[data-alias="ProfitSharingFilterReceiver"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="profit-sharing-list-column-filters-reset"]').exists()).toBe(true)
    })

    it('输入商户单号后 debounce 自动带 out_trade_no 拉列表（OPT-20260825-023）', async () => {
      const wrapper = await mountPanel()
      const lastBefore = psUrlCalls().length

      await wrapper.find('[data-alias="ProfitSharingFilterOutTradeNo"]').setValue('WX1001')
      await vi.advanceTimersByTimeAsync(400)
      await flushPromises()

      const urls = psUrlCalls()
      expect(urls.length).toBeGreaterThan(lastBefore)
      expect(urls[urls.length - 1]).toContain('out_trade_no=WX1001')
    })

    it('输入订单号后 debounce 自动带 order_number 拉列表', async () => {
      const wrapper = await mountPanel()
      const lastBefore = psUrlCalls().length

      await wrapper.find('[data-alias="ProfitSharingFilterOrderNumber"]').setValue('ORD2001')
      await vi.advanceTimersByTimeAsync(400)
      await flushPromises()

      const urls = psUrlCalls()
      expect(urls.length).toBeGreaterThan(lastBefore)
      expect(urls[urls.length - 1]).toContain('order_number=ORD2001')
    })

    it('输入租户 ID 后 debounce 自动带 tenant_id 拉列表', async () => {
      const wrapper = await mountPanel()
      const lastBefore = psUrlCalls().length

      await wrapper.find('[data-alias="ProfitSharingFilterTenantId"]').setValue('877397588196749312')
      await vi.advanceTimersByTimeAsync(400)
      await flushPromises()

      const urls = psUrlCalls()
      expect(urls.length).toBeGreaterThan(lastBefore)
      expect(urls[urls.length - 1]).toContain('tenant_id=877397588196749312')
    })

    it('输入接收方 ID 后 debounce 自动带 receiver_user_id 拉列表', async () => {
      const wrapper = await mountPanel()
      const lastBefore = psUrlCalls().length

      await wrapper.find('[data-alias="ProfitSharingFilterReceiver"]').setValue('referrer-2')
      await vi.advanceTimersByTimeAsync(400)
      await flushPromises()

      const urls = psUrlCalls()
      expect(urls.length).toBeGreaterThan(lastBefore)
      expect(urls[urls.length - 1]).toContain('receiver_user_id=referrer-2')
    })

    it('重置按钮清空列过滤并重拉（不再带 order_number/tenant_id/receiver_user_id）', async () => {
      const wrapper = await mountPanel()

      await wrapper.find('[data-alias="ProfitSharingFilterTenantId"]').setValue('111')
      await vi.advanceTimersByTimeAsync(400)
      await flushPromises()
      const filtered = psUrlCalls().pop()
      expect(filtered).toContain('tenant_id=111')

      await wrapper.find('[data-testid="profit-sharing-list-column-filters-reset"]').trigger('click')
      await flushPromises()

      const latest = psUrlCalls().pop()
      expect(latest).not.toContain('tenant_id=111')
      expect(latest).not.toContain('order_number=')
      expect(latest).not.toContain('receiver_user_id=')
      expect(latest).not.toContain('out_trade_no=')
    })
  })
}

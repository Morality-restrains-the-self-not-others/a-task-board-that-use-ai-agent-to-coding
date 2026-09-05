// @vitest-environment jsdom
/**
 * 退款审批弹层：按租户+订单拉取 resource_consumption。
 */
if (!process.env.VITEST) {
  console.log('[skip] useAdminRefundOrderConsumption.test.js requires vitest runtime')
} else {
  const { describe, expect, it, vi, beforeEach } = await import('vitest')

  const mocks = vi.hoisted(() => ({ apiFetch: vi.fn() }))
  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch: (...args) => mocks.apiFetch(...args),
  }))

  const { useAdminRefundOrderConsumption } = await import('./useAdminRefundOrderConsumption.js')

  describe('useAdminRefundOrderConsumption', () => {
    beforeEach(() => {
      mocks.apiFetch.mockReset()
    })

    it('按 tenant_id+order_id GET 订单并写入 resource_consumption', async () => {
      mocks.apiFetch.mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => ({
          resource_consumption: {
            task_post: { granted: 10, consumed: 3, remaining: 7, source_kind: 'purchase' },
          },
        }),
      })
      const { loadFor, consumption, error } = useAdminRefundOrderConsumption()
      await loadFor({
        tenant_id: '877397588196749312',
        order_id: '8775960076914851840',
      })
      expect(mocks.apiFetch).toHaveBeenCalledWith(
        '/api/tenant/877397588196749312/billing/orders/8775960076914851840/',
        expect.objectContaining({ credentials: 'include' }),
      )
      expect(consumption.value.task_post.consumed).toBe(3)
      expect(error.value).toBe('')
    })

    it('缺少订单 ID 时不发请求并给出错误', async () => {
      const { loadFor, error } = useAdminRefundOrderConsumption()
      await loadFor({ tenant_id: 't1', order_id: '' })
      expect(mocks.apiFetch).not.toHaveBeenCalled()
      expect(error.value).toContain('无法加载资源消耗')
    })
  })
}

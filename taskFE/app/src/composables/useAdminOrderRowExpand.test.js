// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] useAdminOrderRowExpand.test.js requires vitest runtime')
} else {
  const { describe, expect, it, vi, beforeEach } = await import('vitest')
  const { ref, computed } = await import('vue')

  const mocks = vi.hoisted(() => ({ apiFetch: vi.fn() }))
  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch: (...args) => mocks.apiFetch(...args),
  }))

  const { useAdminOrderRowExpand } = await import('./useAdminOrderRowExpand.js')

  describe('useAdminOrderRowExpand', () => {
    beforeEach(() => {
      mocks.apiFetch.mockReset()
    })

    it('展开时写入 resource_consumption', async () => {
      mocks.apiFetch.mockResolvedValue({
        ok: true,
        json: async () => ({
          items: [{ id: 'i1' }],
          buyer_note: 'n',
          resource_consumption: { task_post: { granted: 10, consumed: 1, remaining: 9 } },
          profit_sharing: [{ receiver_user_id: 'u-ref', amount_yuan: '0.55', status: 'pending' }],
          out_trade_no: 'WX-EXPAND-1',
          wechat_transaction_id: '420000EXPAND1',
        }),
      })
      const selectedTenant = ref({ id: 't1' })
      const isAllTenants = computed(() => false)
      const { toggleExpand, expandConsumption, expandProfitSharing, expandOutTradeNo, expandWechatTransactionId, isExpanded } = useAdminOrderRowExpand({
        selectedTenant,
        isAllTenants,
      })
      await toggleExpand({ id: '900001' })
      expect(isExpanded('900001')).toBe(true)
      expect(expandConsumption.value['900001'].task_post.remaining).toBe(9)
      expect(expandProfitSharing.value['900001'][0].amount_yuan).toBe('0.55')
      expect(expandOutTradeNo.value['900001']).toBe('WX-EXPAND-1')
      expect(expandWechatTransactionId.value['900001']).toBe('420000EXPAND1')
      expect(mocks.apiFetch.mock.calls[0][0]).toBe('/api/system-admin/orders/900001/')
    })
  })
}

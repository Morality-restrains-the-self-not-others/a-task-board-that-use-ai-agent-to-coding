// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] BillingRefundConfirmModal.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { describe, expect, it, vi } = await import('vitest')
  const { BILLING_REFUND_CONSUMED_NOTICE } = await import('../utils/billingRefundCopy.js')
  const { default: BillingRefundConfirmModal } = await import('./BillingRefundConfirmModal.vue')

  describe('BillingRefundConfirmModal', () => {
    it('展示已消耗无法退回说明，确认时把原因与 Idempotency-Key 交给 submit', async () => {
      const submit = vi.fn(async () => true)
      const wrapper = mount(BillingRefundConfirmModal, {
        props: {
          order: { id: '9', order_number: 'ORD-1', total_yuan: '5.50' },
          consumption: {
            task_post: { granted: 10, consumed: 2, remaining: 8, events: [] },
          },
          applyError: '',
          applyErrorTraceId: '',
          applying: false,
          submit,
        },
      })
      expect(wrapper.text()).toContain(BILLING_REFUND_CONSUMED_NOTICE)
      expect(wrapper.text()).toContain('已消耗 2')
      await wrapper.get('textarea').setValue('不想用了')
      await wrapper.get('[data-testid="billing-refund-confirm-btn"]').trigger('click')
      await flushPromises()
      expect(submit).toHaveBeenCalledTimes(1)
      const [reason, idempotencyKey] = submit.mock.calls[0]
      expect(reason).toBe('不想用了')
      expect(String(idempotencyKey || '').length).toBeGreaterThan(8)
      wrapper.unmount()
    })
  })
}

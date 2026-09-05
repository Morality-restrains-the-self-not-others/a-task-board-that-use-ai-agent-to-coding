// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] ProfitSharingShareReasonForm.test.js requires vitest runtime')
} else {
  const { describe, expect, it, vi } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')
  const Form = (await import('./ProfitSharingShareReasonForm.vue')).default

  describe('ProfitSharingShareReasonForm', () => {
    it('挂载后滚入视口并聚焦缘由输入', async () => {
      const scrollSpy = vi.fn()
      HTMLElement.prototype.scrollIntoView = scrollSpy
      const wrapper = mount(Form, {
        props: {
          orderLabel: 'ORD1',
          modelValue: '',
          formId: 'referral-ps-share-form',
          testIdPrefix: 'referral-ps',
        },
        attachTo: document.body,
      })
      await flushPromises()
      expect(wrapper.get('[data-testid="referral-ps-share-form"]').exists()).toBe(true)
      expect(scrollSpy).toHaveBeenCalled()
      expect(document.activeElement).toBe(wrapper.get('[data-testid="referral-ps-share-reason"]').element)
      wrapper.unmount()
    })
  })
}

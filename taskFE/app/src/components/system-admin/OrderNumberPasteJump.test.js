// @vitest-environment jsdom
/**
 * 交易单号查询：空输入不发请求；非空 emit search。
 */
if (!process.env.VITEST) {
  console.log('[skip] OrderNumberPasteJump.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')

  const Comp = (await import('./OrderNumberPasteJump.vue')).default

  describe('OrderNumberPasteJump 交易单号查询', () => {
    it('空输入提示且不 emit search', async () => {
      const wrapper = mount(Comp)
      await wrapper.get('[data-testid="order-number-paste-jump"]').trigger('click')
      await flushPromises()
      expect(wrapper.emitted('search')).toBeFalsy()
      expect(wrapper.get('[data-testid="order-number-parse-error"]').text()).toContain('请输入交易单号')
    })

    it('非空输入 emit search 查询串', async () => {
      const wrapper = mount(Comp)
      await wrapper.get('[data-testid="order-number-paste-input"]').setValue('  ORD-20260821-1-2  ')
      await wrapper.get('[data-testid="order-number-paste-jump"]').trigger('click')
      await flushPromises()
      expect(wrapper.emitted('search')?.[0]?.[0]).toBe('ORD-20260821-1-2')
    })

    it('剥离微信账单导出前导反引号后 emit', async () => {
      const wrapper = mount(Comp)
      await wrapper.get('[data-testid="order-number-paste-input"]').setValue('`4500000359202608221274536815')
      await wrapper.get('[data-testid="order-number-paste-jump"]').trigger('click')
      await flushPromises()
      expect(wrapper.emitted('search')?.[0]?.[0]).toBe('4500000359202608221274536815')
    })

    it('剥离首尾成对引号后 emit', async () => {
      const wrapper = mount(Comp)
      await wrapper.get('[data-testid="order-number-paste-input"]').setValue('"ORD-WRAP"')
      await wrapper.get('[data-testid="order-number-paste-jump"]').trigger('click')
      await flushPromises()
      expect(wrapper.emitted('search')?.[0]?.[0]).toBe('ORD-WRAP')
    })
  })
}

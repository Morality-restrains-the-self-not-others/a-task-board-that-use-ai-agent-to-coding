// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] BillingOrderNumberSearch.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')

  const Comp = (await import('./BillingOrderNumberSearch.vue')).default

  describe('BillingOrderNumberSearch 租户订单单号查询', () => {
    it('空输入提示且不 emit search', async () => {
      const wrapper = mount(Comp)
      await wrapper.get('[data-testid="billing-order-number-search-btn"]').trigger('click')
      await flushPromises()
      expect(wrapper.emitted('search')).toBeFalsy()
      expect(wrapper.get('[data-testid="billing-order-number-search-error"]').text()).toContain('请输入交易单号或商户单号')
    })

    it('交易单号 trim 后 emit search', async () => {
      const wrapper = mount(Comp)
      await wrapper.get('[data-testid="billing-order-txn-input"]').setValue('  4500000359202608221274536815  ')
      await wrapper.get('[data-testid="billing-order-number-search-btn"]').trigger('click')
      await flushPromises()
      expect(wrapper.emitted('search')?.[0]?.[0]).toBe('4500000359202608221274536815')
    })

    it('商户单号剥离反引号后 emit search', async () => {
      const wrapper = mount(Comp)
      await wrapper.get('[data-testid="billing-order-out-trade-input"]').setValue('`WX878981209491800064')
      await wrapper.get('[data-testid="billing-order-number-search-btn"]').trigger('click')
      await flushPromises()
      expect(wrapper.emitted('search')?.[0]?.[0]).toBe('WX878981209491800064')
    })

    it('两项都填时以交易单号为准', async () => {
      const wrapper = mount(Comp)
      await wrapper.get('[data-testid="billing-order-txn-input"]').setValue('4500000359202608221274536815')
      await wrapper.get('[data-testid="billing-order-out-trade-input"]').setValue('WX878981209491800064')
      await wrapper.get('[data-testid="billing-order-number-search-btn"]').trigger('click')
      await flushPromises()
      expect(wrapper.emitted('search')?.[0]?.[0]).toBe('4500000359202608221274536815')
    })

    it('清空 emit clear 并去掉报错', async () => {
      const wrapper = mount(Comp)
      await wrapper.get('[data-testid="billing-order-number-search-btn"]').trigger('click')
      await flushPromises()
      await wrapper.get('[data-testid="billing-order-number-clear-btn"]').trigger('click')
      await flushPromises()
      expect(wrapper.emitted('clear')?.length).toBe(1)
      expect(wrapper.find('[data-testid="billing-order-number-search-error"]').exists()).toBe(false)
    })
  })
}

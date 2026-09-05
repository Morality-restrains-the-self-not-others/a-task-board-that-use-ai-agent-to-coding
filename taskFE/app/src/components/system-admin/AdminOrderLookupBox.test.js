// @vitest-environment jsdom
/**
 * 超管订单查询 Tab：交易单号 / 微信关联账号。
 */
if (!process.env.VITEST) {
  console.log('[skip] AdminOrderLookupBox.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const { mount } = await import('@vue/test-utils')

  const Comp = (await import('./AdminOrderLookupBox.vue')).default

  describe('AdminOrderLookupBox 查询 Tab', () => {
    it('默认展示交易单号查询', () => {
      const wrapper = mount(Comp)
      expect(wrapper.find('[data-testid="order-number-paste-box"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="wechat-linked-account-box"]').exists()).toBe(false)
    })

    it('切换到微信关联账号 Tab', async () => {
      const wrapper = mount(Comp)
      await wrapper.get('[data-testid="admin-order-lookup-tab-wechat"]').trigger('click')
      expect(wrapper.find('[data-testid="wechat-linked-account-box"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="order-number-paste-box"]').exists()).toBe(false)
    })
  })
}

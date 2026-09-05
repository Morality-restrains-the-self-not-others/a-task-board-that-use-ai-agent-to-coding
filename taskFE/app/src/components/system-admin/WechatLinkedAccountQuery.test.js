// @vitest-environment jsdom
/**
 * 微信关联账号查询：空输入不发请求；非空 emit search。
 */
if (!process.env.VITEST) {
  console.log('[skip] WechatLinkedAccountQuery.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')

  const Comp = (await import('./WechatLinkedAccountQuery.vue')).default

  describe('WechatLinkedAccountQuery 微信关联账号查询', () => {
    it('空输入提示且不 emit search', async () => {
      const wrapper = mount(Comp)
      await wrapper.get('[data-testid="wechat-linked-account-search"]').trigger('click')
      await flushPromises()
      expect(wrapper.emitted('search')).toBeFalsy()
      expect(wrapper.get('[data-testid="wechat-linked-account-error"]').text()).toContain('请输入微信关联账号')
    })

    it('非空输入 emit search 查询串', async () => {
      const wrapper = mount(Comp)
      await wrapper.get('[data-testid="wechat-linked-account-input"]').setValue('  微信昵称甲  ')
      await wrapper.get('[data-testid="wechat-linked-account-search"]').trigger('click')
      await flushPromises()
      expect(wrapper.emitted('search')?.[0]?.[0]).toBe('微信昵称甲')
    })
  })
}

// @vitest-environment jsdom
// OPT-20260829-027 回归：有效期下拉共用控件，三种邀请方式选项一致。
if (!process.env.VITEST) {
  console.log('[skip] InviteExpirationSelect.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const { mount } = await import('@vue/test-utils')
  const { default: InviteExpirationSelect } = await import('./InviteExpirationSelect.vue')

  describe('InviteExpirationSelect 有效期共用下拉', () => {
    it('选项含 90/180/365 且默认 90', () => {
      const wrapper = mount(InviteExpirationSelect)
      const values = wrapper.findAll('select option').map((opt) => Number(opt.attributes('value')))
      expect(values).toEqual([1, 3, 7, 14, 30, 90, 180, 365])
      expect(wrapper.find('select').element.value).toBe('90')
      expect(wrapper.text()).toContain('365天')
    })

    it('change 发出数值型 update:modelValue（可选中 365）', async () => {
      const wrapper = mount(InviteExpirationSelect, { props: { modelValue: 90 } })
      await wrapper.find('select').setValue('365')
      const emitted = wrapper.emitted('update:modelValue')
      expect(emitted?.flat()).toEqual([365])
    })
  })
}

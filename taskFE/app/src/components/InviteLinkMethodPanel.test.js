// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] InviteLinkMethodPanel.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const { mount } = await import('@vue/test-utils')
  const { default: InviteLinkMethodPanel } = await import('./InviteLinkMethodPanel.vue')

  describe('InviteLinkMethodPanel 开放链接', () => {
    it('默认单次：无成员名时生成按钮 disabled', () => {
      const wrapper = mount(InviteLinkMethodPanel, {
        props: { workspaceId: 'ws1', linkKind: 'single', companyMemberName: '' },
      })
      const btn = wrapper.get('[data-testid="invite-link-generate-btn"]')
      expect(btn.attributes('disabled')).toBeDefined()
      expect(wrapper.text()).toContain('只能使用一次')
    })

    it('开放模式：无成员名也可生成，展示人数不限文案', async () => {
      const wrapper = mount(InviteLinkMethodPanel, {
        props: { workspaceId: 'ws1', linkKind: 'open', companyMemberName: '' },
      })
      const btn = wrapper.get('[data-testid="invite-link-generate-btn"]')
      expect(btn.attributes('disabled')).toBeUndefined()
      expect(wrapper.text()).toContain('人数不限')
      await wrapper.get('[data-testid="invite-link-kind-single"]').setValue(true)
      expect(wrapper.emitted('update:linkKind')?.flat()).toContain('single')
    })

    it('开放模式填写上限时文案含人数', () => {
      const wrapper = mount(InviteLinkMethodPanel, {
        props: { workspaceId: 'ws1', linkKind: 'open', maxUses: '8', expirationDays: 7 },
      })
      expect(wrapper.text()).toContain('最多 8 人加入')
    })

    it('有效期选项含 90/180/365 天', () => {
      const wrapper = mount(InviteLinkMethodPanel, {
        props: { workspaceId: 'ws1' },
      })
      const values = wrapper.findAll('select option').map((opt) => Number(opt.attributes('value')))
      expect(values).toEqual([1, 3, 7, 14, 30, 90, 180, 365])
      expect(wrapper.text()).toContain('365天')
    })
  })
}

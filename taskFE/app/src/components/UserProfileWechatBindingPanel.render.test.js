// @vitest-environment jsdom
if (!process.env.VITEST) {
  // pre-commit 会以 node 直跑已暂存 *.test.js，非 Vitest 环境下跳过。
  console.log('[skip] UserProfileWechatBindingPanel.render.test.js requires vitest runtime')
} else {
  const { mount } = await import('@vue/test-utils')
  const { describe, expect, it } = await import('vitest')
  const { default: UserProfileWechatBindingPanel } = await import('./UserProfileWechatBindingPanel.vue')

  describe('UserProfileWechatBindingPanel', () => {
    it('已绑定 + 配置齐全 → 显示「绑定其他微信应用」和「解绑微信」', () => {
      const wrapper = mount(UserProfileWechatBindingPanel, {
        props: { hasWechat: true, wechatApps: ['web'], bindAvailable: true },
      })
      expect(wrapper.text()).toContain('绑定其他微信应用')
      expect(wrapper.text()).toContain('解绑微信')
      wrapper.unmount()
    })

    it('已绑定 + 未配置 appid/appSecret → 隐藏「绑定其他微信应用」，仍显示「解绑微信」', () => {
      const wrapper = mount(UserProfileWechatBindingPanel, {
        props: { hasWechat: true, wechatApps: ['web'], bindAvailable: false },
      })
      expect(wrapper.text()).not.toContain('绑定其他微信应用')
      expect(wrapper.text()).toContain('解绑微信')
      wrapper.unmount()
    })

    it('未绑定 → 显示「绑定微信」，不显示「绑定其他微信应用」', () => {
      const wrapper = mount(UserProfileWechatBindingPanel, {
        props: { hasWechat: false, wechatApps: [], bindAvailable: false },
      })
      expect(wrapper.text()).toContain('绑定微信')
      expect(wrapper.text()).not.toContain('绑定其他微信应用')
      wrapper.unmount()
    })

    it('未传 bindAvailable → 默认 true（向后兼容，不隐藏入口）', () => {
      const wrapper = mount(UserProfileWechatBindingPanel, {
        props: { hasWechat: true, wechatApps: ['web'] },
      })
      expect(wrapper.text()).toContain('绑定其他微信应用')
      wrapper.unmount()
    })
  })
}

// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] ServerConfigRelayDirectPanel.oauth-guide.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const { mount } = await import('@vue/test-utils')
  const { default: ServerConfigRelayDirectPanel } = await import('./ServerConfigRelayDirectPanel.vue')

  describe('ServerConfigRelayDirectPanel oauth guide', () => {
    it('unbound oauth guide points to create/edit task, not comment composer', () => {
      const wrapper = mount(ServerConfigRelayDirectPanel, {
        props: {
          visible: true,
          hasTaskId: true,
          hasImage: true,
          envItems: [],
          startBlockedByUnboundOAuth: true,
        },
      })
      const guide = wrapper.find('[data-testid="relay-to-trae-oauth-unbound-guide"]')
      expect(guide.exists()).toBe(true)
      expect(guide.text()).toMatch(/创建或编辑任务/)
      expect(guide.text()).not.toMatch(/添加评论区域/)
      expect(wrapper.find('[data-testid="relay-to-trae-start-disabled-hint"]').text())
        .toMatch(/创建或编辑任务/)
      wrapper.unmount()
    })
  })
}

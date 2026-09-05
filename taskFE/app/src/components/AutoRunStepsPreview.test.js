// @vitest-environment jsdom
/**
 * AutoRunStepsPreview：默认收起；仅在显式 defaultExpanded 时展开
 * pre-commit 会以 node 直跑已暂存 *.test.js，非 Vitest 环境下跳过。
 */
if (!process.env.VITEST) {
  console.log('[skip] AutoRunStepsPreview.test.js requires vitest runtime')
} else {
  const { describe, it, expect } = await import('vitest')
  const { mount } = await import('@vue/test-utils')
  const { default: AutoRunStepsPreview } = await import('./AutoRunStepsPreview.vue')

  describe('AutoRunStepsPreview', () => {
    it('默认收起：按钮为「查看自动运行说明」，正文不渲染', () => {
      const wrapper = mount(AutoRunStepsPreview, {
        props: { markdown: '# steps', extractStatus: 'ok' },
      })
      const toggle = wrapper.find('[data-testid="auto-run-steps-toggle"]')
      expect(toggle.exists()).toBe(true)
      expect(toggle.text()).toBe('查看自动运行说明')
      expect(wrapper.find('[data-testid="auto-run-steps-body"]').exists()).toBe(false)
    })

    it('defaultExpanded=true 时默认展开', () => {
      const wrapper = mount(AutoRunStepsPreview, {
        props: {
          markdown: '# steps',
          extractStatus: 'ok',
          defaultExpanded: true,
        },
      })
      expect(wrapper.find('[data-testid="auto-run-steps-toggle"]').text()).toBe('收起自动运行说明')
      expect(wrapper.find('[data-testid="auto-run-steps-body"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="auto-run-steps-body"]').text()).toContain('# steps')
    })

    it('点击切换展开/收起', async () => {
      const wrapper = mount(AutoRunStepsPreview, {
        props: { markdown: 'hello', extractStatus: 'ok' },
      })
      await wrapper.find('[data-testid="auto-run-steps-toggle"]').trigger('click')
      expect(wrapper.find('[data-testid="auto-run-steps-body"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="auto-run-steps-toggle"]').text()).toBe('收起自动运行说明')
      await wrapper.find('[data-testid="auto-run-steps-toggle"]').trigger('click')
      expect(wrapper.find('[data-testid="auto-run-steps-body"]').exists()).toBe(false)
      expect(wrapper.find('[data-testid="auto-run-steps-toggle"]').text()).toBe('查看自动运行说明')
    })
  })
}

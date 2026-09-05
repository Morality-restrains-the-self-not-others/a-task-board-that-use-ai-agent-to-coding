// @vitest-environment jsdom
/**
 * ImageAutoRunSteps：镜像市场空自动运行不渲染占位。
 * pre-commit 会以 node 直跑已暂存 *.test.js，非 Vitest 环境下跳过。
 */
if (!process.env.VITEST) {
  console.log('[skip] ImageAutoRunSteps.test.js requires vitest runtime')
} else {
  const { describe, it, expect } = await import('vitest')
  const { mount } = await import('@vue/test-utils')
  const { default: ImageAutoRunSteps } = await import('./ImageAutoRunSteps.vue')

  describe('ImageAutoRunSteps', () => {
    it('空说明时不渲染预览与「暂无自动运行说明」', () => {
      const wrapper = mount(ImageAutoRunSteps, {
        props: { image: { auto_run_steps_md: '', auto_run_steps_extract_status: '' } },
      })
      expect(wrapper.find('[data-testid="auto-run-steps-preview"]').exists()).toBe(false)
      expect(wrapper.text()).not.toContain('暂无自动运行说明')
    })

    it('有 markdown 时渲染预览入口', () => {
      const wrapper = mount(ImageAutoRunSteps, {
        props: { image: { auto_run_steps_md: '# steps', auto_run_steps_extract_status: 'ok' } },
      })
      expect(wrapper.find('[data-testid="auto-run-steps-preview"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="auto-run-steps-toggle"]').text()).toBe('查看自动运行说明')
    })
  })
}

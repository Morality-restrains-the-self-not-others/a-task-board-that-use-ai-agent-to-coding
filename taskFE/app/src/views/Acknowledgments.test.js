// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] Acknowledgments.test.js requires vitest runtime')
} else {
  const { mount } = await import('@vue/test-utils')
  const { describe, expect, it } = await import('vitest')
  const Acknowledgments = (await import('./Acknowledgments.vue')).default
  const { TRAE_AGENT_REPO_URL } = await import('../data/acknowledgmentsContent.js')

  describe('Acknowledgments.vue', () => {
    it('T2: 渲染 Trae Agent 名称与仓库链接', () => {
      const wrapper = mount(Acknowledgments, {
        global: {
          stubs: {
            'router-link': {
              props: ['to'],
              template: '<a :href="typeof to === \'string\' ? to : to.path"><slot /></a>',
            },
          },
        },
      })

      expect(wrapper.attributes('data-alias')).toBe('view-acknowledgments-page')
      expect(wrapper.text()).toContain('致谢')
      expect(wrapper.text()).toMatch(/Trae Agent/i)
      const repoLink = wrapper.find(`[data-testid="ack-repo-link-trae-agent"]`)
      expect(repoLink.exists()).toBe(true)
      expect(repoLink.attributes('href')).toBe(TRAE_AGENT_REPO_URL)
      expect(repoLink.attributes('target')).toBe('_blank')
      expect(repoLink.attributes('rel')).toContain('noopener')
    })
  })
}

// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] Home.feature-cards.test.js requires vitest runtime')
} else {
  const { mount } = await import('@vue/test-utils')
  const { describe, expect, it } = await import('vitest')
  const Home = (await import('./Home.vue')).default

  describe('Home.vue #projects 核心功能三卡（OPT-20260811-071）', () => {
    it('渲染三张特性卡且标题/了解更多完整', () => {
      const wrapper = mount(Home, {
        global: {
          stubs: {
            'router-link': {
              props: ['to'],
              template:
                '<a data-testid="stub-router-link" :data-to="typeof to === \'string\' ? to : (to.name || to.path)"><slot /></a>',
            },
          },
        },
      })

      const projects = wrapper.find('#projects')
      expect(projects.exists()).toBe(true)
      // #projects 内三张特性卡（Hero 区另有 1 个「了解更多」，不在此计数）
      expect(projects.findAll('a').filter((a) => a.text().includes('了解更多')).length).toBe(3)

      const text = projects.text()
      expect(text).toContain('项目管理')
      expect(text).toContain('云平台管理')
      expect(text).toContain('用户和权限管理')
    })
  })
}

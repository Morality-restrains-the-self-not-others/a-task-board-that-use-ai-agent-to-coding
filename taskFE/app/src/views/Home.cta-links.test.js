// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] Home.cta-links.test.js requires vitest runtime')
} else {
  const { mount } = await import('@vue/test-utils')
  const { describe, expect, it } = await import('vitest')
  const Home = (await import('./Home.vue')).default

  describe('Home.vue CTA 跳转（OPT-20260811-059/064/065）', () => {
    function mountHome() {
      return mount(Home, {
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
    }

    it('Hero「立即开始」指向 /auth/register/', () => {
      const wrapper = mountHome()
      const start = wrapper.findAll('a').find((a) => a.text().includes('立即开始'))
      expect(start).toBeTruthy()
      expect(start.attributes('href')).toBe('/auth/register/')
    })

    it('Hero「了解更多」锚到 #projects', () => {
      const wrapper = mountHome()
      const learnMore = wrapper.findAll('a').find((a) => a.text().includes('了解更多'))
      expect(learnMore).toBeTruthy()
      expect(learnMore.attributes('href')).toBe('#projects')
    })

    it('#projects 三张功能卡「了解更多」指向 /auth/register/', () => {
      const wrapper = mountHome()
      const projects = wrapper.find('#projects')
      expect(projects.exists()).toBe(true)
      const cardLinks = projects
        .findAll('a')
        .filter((a) => a.text().includes('了解更多'))
      expect(cardLinks.length).toBe(3)
      for (const link of cardLinks) {
        expect(link.attributes('href')).toBe('/auth/register/')
      }
    })
  })
}

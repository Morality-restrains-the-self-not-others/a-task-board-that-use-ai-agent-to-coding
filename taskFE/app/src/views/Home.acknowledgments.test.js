// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] Home.acknowledgments.test.js requires vitest runtime')
} else {
  const { mount } = await import('@vue/test-utils')
  const { describe, expect, it } = await import('vitest')
  const Home = (await import('./Home.vue')).default

  describe('Home.vue acknowledgments footer', () => {
    it('T3: 页脚致谢链接指向 /acknowledgments/', () => {
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

      const link = wrapper.find('[data-testid="home-footer-acknowledgments-link"]')
      expect(link.exists()).toBe(true)
      expect(link.text()).toContain('致谢')
      const to = link.attributes('data-to')
      expect(to === 'acknowledgments' || to === '/acknowledgments/').toBe(true)
    })
  })
}

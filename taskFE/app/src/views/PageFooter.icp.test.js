// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] PageFooter.icp.test.js requires vitest runtime')
} else {
  const { mount } = await import('@vue/test-utils')
  const { describe, expect, it } = await import('vitest')

  describe('PageFooter ICP filing number', () => {
    it('renders ICP from VITE_ICP_BEIAN linking to MIIT beian portal', async () => {
      const { default: PageFooter } = await import('./PageFooter.vue')
      const wrapper = mount(PageFooter)
      const beian = String(import.meta.env.VITE_ICP_BEIAN || '').trim()
      expect(beian.length, 'conf icpBeian required for PageFooter ICP').toBeGreaterThan(0)
      const link = wrapper.find('[data-testid="icp-link"]')
      expect(link.exists()).toBe(true)
      expect(link.attributes('href')).toBe('https://beian.miit.gov.cn/')
      expect(link.attributes('target')).toBe('_blank')
      expect(link.text()).toBe(beian)
      const src = (await import('./PageFooter.vue?raw')).default
      expect(src).not.toMatch(/闽ICP备\d+号/)
      wrapper.unmount()
    })
  })
}

// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] PageFooter.agpl-source.test.js requires vitest runtime')
} else {
  const { mount } = await import('@vue/test-utils')
  const { describe, expect, it } = await import('vitest')
  const { AGPL_SOURCE_URL, AGPL_SOURCE_LABEL } = await import('../utils/agplSourceInfo.js')

  describe('PageFooter AGPL source link', () => {
    it('renders a Source link reachable without login', async () => {
      const { default: PageFooter } = await import('./PageFooter.vue')
      const wrapper = mount(PageFooter)
      const link = wrapper.find('[data-testid="agpl-source-link"]')
      expect(link.exists()).toBe(true)
      expect(link.attributes('href')).toBe(AGPL_SOURCE_URL)
      expect(link.attributes('target')).toBe('_blank')
      expect(link.text()).toBe(AGPL_SOURCE_LABEL)
      wrapper.unmount()
    })
  })
}

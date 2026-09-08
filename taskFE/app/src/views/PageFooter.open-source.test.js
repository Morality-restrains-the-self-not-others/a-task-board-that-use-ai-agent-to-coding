if (typeof describe !== 'function') {
  console.log('[skip] PageFooter.open-source.test.js requires vitest runtime')
} else {
  const { mount } = await import('@vue/test-utils')
  const { OPEN_SOURCE_URL, OPEN_SOURCE_LABEL } = await import('../utils/openSourceInfo.js')

  describe('PageFooter open source link', () => {
    it('renders Source link to open source repository', async () => {
      const { default: PageFooter } = await import('./PageFooter.vue')
      const wrapper = mount(PageFooter)
      const link = wrapper.find('[data-testid="open-source-link"]')
      expect(link.exists()).toBe(true)
      expect(link.attributes('href')).toBe(OPEN_SOURCE_URL)
      expect(link.attributes('target')).toBe('_blank')
      expect(link.text()).toBe(OPEN_SOURCE_LABEL)
    })
  })
}

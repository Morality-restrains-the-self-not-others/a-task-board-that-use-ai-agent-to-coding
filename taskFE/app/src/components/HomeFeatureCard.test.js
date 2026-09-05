// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] HomeFeatureCard.test.js requires vitest runtime')
} else {
  const { mount } = await import('@vue/test-utils')
  const { describe, expect, it } = await import('vitest')
  const HomeFeatureCard = (await import('./HomeFeatureCard.vue')).default

  describe('HomeFeatureCard 视觉锚点（OPT-20260824-003）', () => {
    it('根节点含 bg-white 使 shadow-lg 在白底上有投影锚点', () => {
      const wrapper = mount(HomeFeatureCard, {
        props: {
          icon: 'M13 7l5 5m0 0l-5 5m5-5H6',
          title: '项目管理',
          description: 'desc',
          href: '#projects',
        },
      })
      const root = wrapper.element
      expect(root.classList.contains('bg-white')).toBe(true)
      expect(root.classList.contains('shadow-lg')).toBe(true)
    })
  })
}

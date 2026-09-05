// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] TenantPageAccessEmpty.test.js requires vitest runtime')
} else {
  const { mount } = await import('@vue/test-utils')
  const { describe, expect, it } = await import('vitest')
  const { default: TenantPageAccessEmpty } = await import('./TenantPageAccessEmpty.vue')

  describe('TenantPageAccessEmpty 空态', () => {
    it('渲染 page-key 与引导文案', () => {
      const wrapper = mount(TenantPageAccessEmpty, { props: { pageKey: 'settings.cloud' } })
      expect(wrapper.get('[data-testid="tenant-page-access-empty"]').exists()).toBe(true)
      expect(wrapper.text()).toContain('暂无访问权限')
      expect(wrapper.text()).toContain('page:settings.cloud')
    })
  })
}

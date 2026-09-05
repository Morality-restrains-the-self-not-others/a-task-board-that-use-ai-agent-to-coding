// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] SystemAdminSidebar.scroll.test.js requires vitest runtime')
} else {
  const { mount } = await import('@vue/test-utils')
  const { describe, expect, it, vi } = await import('vitest')

  vi.mock('vue-router', () => ({
    useRoute: () => ({ path: '/system-admin/', query: {} }),
  }))

  const { default: SystemAdminSidebar } = await import('./SystemAdminSidebar.vue')

  describe('SystemAdminSidebar 菜单栏滚动', () => {
    it('aside 在 App 壳层 overflow-hidden 下可纵向滚动（h-full + min-h-0 + overflow-y-auto）', () => {
      const wrapper = mount(SystemAdminSidebar, {
        global: {
          mocks: {
            $route: { path: '/system-admin/', query: {} },
          },
          stubs: {
            'router-link': { template: '<a><slot /></a>', props: ['to'] },
          },
        },
      })
      const aside = wrapper.find('aside')
      expect(aside.exists()).toBe(true)
      const cls = aside.classes()
      expect(cls).toContain('h-full')
      expect(cls).toContain('min-h-0')
      expect(cls).toContain('overflow-y-auto')
    })
  })
}

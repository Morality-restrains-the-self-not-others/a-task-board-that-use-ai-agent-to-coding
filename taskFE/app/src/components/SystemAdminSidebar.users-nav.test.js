// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] SystemAdminSidebar.users-nav.test.js requires vitest runtime')
} else {
  const { mount } = await import('@vue/test-utils')
  const { describe, expect, it, vi } = await import('vitest')

  vi.mock('vue-router', () => ({
    useRoute: () => ({ path: '/system-admin/tenants/c1/', query: {} }),
  }))

  const { default: SystemAdminSidebar } = await import('./SystemAdminSidebar.vue')

  describe('SystemAdminSidebar 用户管理高亮', () => {
    it('租户详情路径高亮用户管理且仍指向列表 href', () => {
      const wrapper = mount(SystemAdminSidebar, {
        global: {
          mocks: {
            $route: { path: '/system-admin/tenants/c1/', query: {} },
          },
          stubs: {
            'router-link': { template: '<a><slot /></a>', props: ['to'] },
          },
        },
      })
      const link = wrapper.find('a[href="/system-admin/users/"]')
      expect(link.exists()).toBe(true)
      expect(link.text()).toContain('用户管理')
      expect(link.classes()).toContain('text-primary')
    })
  })
}

// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] SystemAdminSidebar.marketplace.test.js requires vitest runtime')
} else {
  const { mount } = await import('@vue/test-utils')
  const { describe, expect, it, vi } = await import('vitest')

  vi.mock('vue-router', () => ({
    useRoute: () => ({ path: '/system-admin/container-images/', query: {} }),
  }))

  const { default: SystemAdminSidebar } = await import('./SystemAdminSidebar.vue')

  describe('SystemAdminSidebar 镜像市场入口迁移', () => {
    it('侧栏不再包含「镜像市场管理（SSO）」外链', () => {
      const wrapper = mount(SystemAdminSidebar, {
        global: {
          mocks: {
            $route: { path: '/system-admin/container-images/', query: {} },
          },
          stubs: {
            'router-link': { template: '<a><slot /></a>', props: ['to'] },
          },
        },
      })
      expect(wrapper.text()).toContain('容器镜像列表')
      expect(wrapper.text()).not.toContain('镜像市场管理（SSO）')
      expect(wrapper.find('a[href*="sso/ai-provider/admin"]').exists()).toBe(false)
    })
  })
}

// @vitest-environment jsdom
// AdminLogin.vue 品牌 logo 回归（OPT-20260824-012 扩展）：/favicon.png 未随 release 产出致线上 404，
// 改为 /img/icon128.png（源 app/static/img/icon128.png）。
if (!process.env.VITEST) {
  console.log('[skip] AdminLogin.logo.test.js requires vitest runtime')
} else {
  const { mount } = await import('@vue/test-utils')
  const { describe, expect, it, vi } = await import('vitest')

  vi.mock('vue-router', () => ({
    useRoute: () => ({ query: {}, params: {}, path: '/auth/admin-login/', fullPath: '/auth/admin-login/' }),
  }))

  // useLoginSubmit 会引用 apiFetch/extractErrorMessage，单测不真正请求
  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch: vi.fn(async () => ({ ok: true, status: 200, json: async () => ({}) })),
    extractErrorMessage: vi.fn(() => ''),
  }))
  vi.mock('../utils/modalService.js', () => ({
    default: { alert: vi.fn(), confirm: vi.fn(), show: vi.fn(), close: vi.fn() },
  }))

  const { default: AdminLogin } = await import('./AdminLogin.vue')

  const mountOptions = {
    global: {
      stubs: {
        LoginEmailPasswordFields: { template: '<div><slot /></div>' },
      },
    },
  }

  describe('AdminLogin.vue — 品牌 logo（OPT-20260824-012 扩展）', () => {
    it('管理员登录页 logo 指向 /img/icon128.png 而非失效的 /favicon.png', async () => {
      const wrapper = mount(AdminLogin, mountOptions)
      const img = wrapper.find('img[alt="云端开发"]')
      expect(img.exists()).toBe(true)
      expect(img.attributes('src')).toBe('/img/icon128.png')
      expect(img.attributes('src')).not.toBe('/favicon.png')
    })
  })
}

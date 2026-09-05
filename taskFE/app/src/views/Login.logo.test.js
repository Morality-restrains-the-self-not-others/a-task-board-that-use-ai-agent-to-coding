// @vitest-environment jsdom
// Login.vue 品牌 logo 回归（OPT-20260824-012 扩展）：/favicon.png 未随 release 产出致线上 404，
// 改为 /img/icon128.png（源 app/static/img/icon128.png）。
if (!process.env.VITEST) {
  console.log('[skip] Login.logo.test.js requires vitest runtime')
} else {
  const { mount } = await import('@vue/test-utils')
  const { describe, expect, it, vi } = await import('vitest')

  vi.mock('vue-router', () => ({
    useRoute: () => ({ query: {}, params: {}, path: '/auth/login/', fullPath: '/auth/login/' }),
  }))

  // 网络层桩：挂载期会拉公共策略/码表/法律文档，单测不真正请求
  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch: vi.fn(async () => ({ ok: true, status: 200, json: async () => ({}) })),
    extractErrorMessage: vi.fn(() => ''),
  }))
  vi.mock('../utils/modalService.js', () => ({
    default: { alert: vi.fn(), confirm: vi.fn(), show: vi.fn(), close: vi.fn() },
  }))

  const { default: Login } = await import('./Login.vue')

  const childStub = () => ({ template: '<div><slot /></div>' })
  const mountOptions = {
    global: {
      stubs: {
        LoginEmailPasswordFields: childStub(),
        LoginLegalConsentPanel: childStub(),
        LoginMarketingFooter: childStub(),
        LoginMethodSelector: childStub(),
        LoginPolicyReaderModals: childStub(),
        LoginRegisterInviteSection: childStub(),
        LoginPhonePasswordFields: childStub(),
        LoginAccessTokenFields: childStub(),
      },
    },
  }

  describe('Login.vue — 品牌 logo（OPT-20260824-012 扩展）', () => {
    it('登录页 logo 指向 /img/icon128.png 而非失效的 /favicon.png', async () => {
      const wrapper = mount(Login, mountOptions)
      const img = wrapper.find('img[alt="云端开发"]')
      expect(img.exists()).toBe(true)
      expect(img.attributes('src')).toBe('/img/icon128.png')
      expect(img.attributes('src')).not.toBe('/favicon.png')
    })
  })
}

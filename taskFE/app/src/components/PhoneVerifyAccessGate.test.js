// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] PhoneVerifyAccessGate.test.js requires vitest runtime')
} else {
  const { describe, expect, it, vi, beforeEach } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { createRouter, createMemoryHistory } = await import('vue-router')

  const apiFetchMock = vi.hoisted(() => vi.fn())
  vi.mock('../utils/apiUtils.js', () => ({ apiFetch: (...args) => apiFetchMock(...args) }))
  vi.mock('../utils/sessionUserIdUtils.js', () => ({ getStoredUserId: () => '999' }))
  vi.mock('../utils/cookieUtils.js', () => ({ getCookie: () => '' }))

  const { default: PhoneVerifyAccessGate } = await import('./PhoneVerifyAccessGate.vue')

  const meUnverified = {
    login_methods: [{ method_type: 'email', is_verified: true }],
  }
  const meVerified = {
    login_methods: [{ method_type: 'phone', is_verified: true }],
  }

  function jsonOk(body) {
    return { ok: true, json: async () => body }
  }

  async function mountAt(path, me = meUnverified) {
    apiFetchMock.mockResolvedValue(jsonOk(me))
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/tenant/:tenant/work-panel/', name: 'work_panel', component: { template: '<div />' } },
        { path: '/profile/', name: 'profile', component: { template: '<div />' } },
      ],
    })
    await router.push(path)
    await router.isReady()
    const wrapper = mount(PhoneVerifyAccessGate, {
      global: { plugins: [router] },
      attachTo: document.body,
    })
    await flushPromises()
    return wrapper
  }

  describe('PhoneVerifyAccessGate', () => {
    beforeEach(() => {
      apiFetchMock.mockReset()
      document.body.innerHTML = ''
    })

    it('工作面板未验证 → 展示阻断层且含去验证链接', async () => {
      const wrapper = await mountAt('/tenant/1/work-panel/')
      const gate = document.querySelector('[data-testid="phone-verify-access-gate"]')
      expect(gate).toBeTruthy()
      const link = gate.querySelector('a[href="/profile/#rg=profile.phone_binding"]')
      expect(link).toBeTruthy()
      expect(link.textContent).toContain('去验证')
      expect(gate.textContent).not.toContain('稍后再说')
      wrapper.unmount()
    })

    it('已验证工作面板不展示阻断层', async () => {
      const wrapper = await mountAt('/tenant/1/work-panel/', meVerified)
      expect(document.querySelector('[data-testid="phone-verify-access-gate"]')).toBeNull()
      wrapper.unmount()
    })

    it('资料主页不展示阻断层', async () => {
      const wrapper = await mountAt('/profile/', meUnverified)
      expect(document.querySelector('[data-testid="phone-verify-access-gate"]')).toBeNull()
      wrapper.unmount()
    })
  })
}

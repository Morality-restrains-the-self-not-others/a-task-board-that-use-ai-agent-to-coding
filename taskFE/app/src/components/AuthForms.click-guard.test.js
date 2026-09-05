// @vitest-environment jsdom
// OPT-20260819-038 回归：登录 POST / 注册 POST 写请求携带 Idempotency-Key 头。
if (!process.env.VITEST) {
  console.log('[skip] AuthForms.click-guard.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { describe, expect, it, vi, beforeEach } = await import('vitest')

  const mocks = vi.hoisted(() => ({
    apiFetch: vi.fn(),
  }))

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch: (...args) => mocks.apiFetch(...args),
    extractErrorMessage: (data, resp, fallback) => data?.error || fallback,
    setCachedAuthToken: vi.fn(),
  }))
  // OPT-20260819-038: 让 guard 直接放行并把幂等键透传给 mergeIdempotencyHeaders，
  // 断言 POST 携带 Idempotency-Key 头。
  vi.mock('../utils/clickGuard.js', () => ({
    createClickGuard: () => ({
      run: async (fn) => fn({ idempotencyKey: 'ik-test-auth' }),
      isBusy: () => false,
    }),
    mergeIdempotencyHeaders: (headers, key) => ({ ...(headers || {}), 'Idempotency-Key': key }),
  }))

  beforeEach(() => {
    vi.clearAllMocks()
    mocks.apiFetch.mockImplementation(async () => ({
      ok: true,
      json: async () => ({ token: '', user: null }),
    }))
  })

  const { default: AuthForms } = await import('./AuthForms.vue')

  describe('AuthForms 写操作 clickGuard 接线', () => {
    it('登录 POST 携带 Idempotency-Key', async () => {
      const wrapper = mount(AuthForms, {
        props: { loginTitle: '登录', registerTitle: '注册' },
      })
      await flushPromises()

      const username = wrapper.findAll('input').find((i) => i.attributes('type') === 'text')
      const password = wrapper.findAll('input').find((i) => i.attributes('type') === 'password')
      await username.setValue('alice')
      await password.setValue('secret')
      await flushPromises()

      const loginBtn = wrapper.findAll('button').find((b) => b.text().trim() === '登录')
      expect(loginBtn).toBeTruthy()
      await loginBtn.trigger('click')
      await flushPromises()

      const postCall = mocks.apiFetch.mock.calls.find(([, o]) => o?.method === 'POST')
      expect(postCall).toBeTruthy()
      expect(postCall[0]).toBe('/api/accounts/users/login/')
      expect(postCall[1].headers['Idempotency-Key']).toBe('ik-test-auth')
    })

    it('注册 POST 携带 Idempotency-Key', async () => {
      const wrapper = mount(AuthForms, {
        props: { loginTitle: '登录', registerTitle: '注册' },
      })
      await flushPromises()

      // 切到注册表单
      const registerTabBtn = wrapper.findAll('button').find((b) => b.text().includes('注册新账号'))
      expect(registerTabBtn).toBeTruthy()
      await registerTabBtn.trigger('click')
      await flushPromises()

      const inputs = wrapper.findAll('input')
      const emailInput = inputs.find((i) => i.attributes('type') === 'email') || inputs[0]
      const usernameInput = inputs.find((i) => i.attributes('type') === 'text')
      const passwordInputs = inputs.filter((i) => i.attributes('type') === 'password')
      await emailInput.setValue('a@b.com')
      await usernameInput.setValue('alice')
      await passwordInputs[0].setValue('secret1')
      await passwordInputs[1].setValue('secret1')
      await flushPromises()

      const registerBtn = wrapper.findAll('button').find((b) => b.text().trim() === '注册')
      expect(registerBtn).toBeTruthy()
      await registerBtn.trigger('click')
      await flushPromises()

      const postCall = mocks.apiFetch.mock.calls.find(([, o]) => o?.method === 'POST')
      expect(postCall).toBeTruthy()
      expect(postCall[0]).toBe('/api/accounts/users/email_register/')
      expect(postCall[1].headers['Idempotency-Key']).toBe('ik-test-auth')
    })
  })
}

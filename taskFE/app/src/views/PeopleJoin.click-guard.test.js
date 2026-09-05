// @vitest-environment jsdom
// OPT-20260819-038: 加入团队是写操作（POST），防连点双发。
if (!process.env.VITEST) {
  console.log('[skip] PeopleJoin.click-guard.test.js requires vitest runtime')
} else {
  const { describe, it, expect, vi, beforeEach } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')

  const { apiFetch } = vi.hoisted(() => ({
    apiFetch: vi.fn(),
  }))

  vi.mock('vue-router', () => ({
    useRoute: () => ({ params: { tenant: 'ten1' }, query: { token: 'tok123' } }),
    useRouter: () => ({ push: vi.fn() }),
  }))
  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch,
    clearCachedAuthToken: vi.fn(),
  }))
  vi.mock('../utils/sessionUserIdUtils.js', () => ({
    clearStoredUserId: vi.fn(),
    getStoredUserId: () => 'user-1',
  }))
  vi.mock('../utils/authReturnUrl.js', () => ({
    savePostLoginRedirect: vi.fn(),
  }))
  // 认证守卫直接判定已登录，便于渲染「确认加入」按钮
  vi.mock('../domain/auth/services/auth_session_guard_service.js', () => ({
    AuthSessionGuardService: class {
      constructor() {}
      async isAuthenticated() {
        return true
      }
    },
  }))
  // OPT-20260819-038: 让 guard 直接放行并把幂等键透传给 mergeIdempotencyHeaders，
  // 断言 POST 携带 Idempotency-Key 头。
  vi.mock('../utils/clickGuard.js', () => ({
    createClickGuard: () => ({
      run: async (fn) => fn({ idempotencyKey: 'ik-test-people-join' }),
      isBusy: () => false,
    }),
    mergeIdempotencyHeaders: (headers, key) => ({ ...(headers || {}), 'Idempotency-Key': key }),
  }))

  const { default: View } = await import('./PeopleJoin.vue')

  function jsonOk(body) {
    return {
      ok: true,
      json: async () => body,
      headers: { get: () => 'application/json' },
    }
  }

  describe('PeopleJoin 写操作 clickGuard 接线', () => {
    beforeEach(() => {
      apiFetch.mockReset()
      apiFetch.mockImplementation(async (url, opts) => {
        if (opts?.method === 'POST') {
          return jsonOk({ success: true })
        }
        if (String(url).includes('validate-invite')) {
          return jsonOk({ valid: true })
        }
        return jsonOk({ name: '示例团队' })
      })
    })

    it('加入团队 POST 携带 Idempotency-Key', async () => {
      const wrapper = mount(View, {})
      await flushPromises()

      const joinBtn = wrapper.find('[data-testid="people-join-confirm-btn"]')
      expect(joinBtn.exists()).toBe(true)
      await joinBtn.trigger('click')
      await flushPromises()

      const postCall = apiFetch.mock.calls.find(([, opts]) => opts?.method === 'POST')
      expect(postCall).toBeTruthy()
      expect(postCall[0]).toBe('/api/tenant/ten1/accounts/members/join/')
      expect(postCall[1].headers['Idempotency-Key']).toBe('ik-test-people-join')
      wrapper.unmount()
    })
  })
}

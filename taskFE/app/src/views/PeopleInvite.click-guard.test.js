// @vitest-environment jsdom
// OPT-20260819-038 回归：邀请用户 POST / 生成邀请链接 POST 写请求携带 Idempotency-Key 头。
if (!process.env.VITEST) {
  console.log('[skip] PeopleInvite.click-guard.test.js requires vitest runtime')
} else {
  const { beforeEach, describe, expect, it, vi } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')

  const hoisted = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
    routeMock: { params: { tenant: 't1' } },
    routerMock: { push: vi.fn() },
    toast: { success: vi.fn(), error: vi.fn(), warning: vi.fn() },
  }))

  vi.mock('../utils/apiUtils', () => ({
    apiFetch: (...args) => hoisted.apiFetchMock(...args),
  }))
  vi.mock('../utils/toastService', () => ({
    default: hoisted.toast,
  }))
  vi.mock('../composables/usePermissions.js', () => ({
    usePermissions: () => ({
      load: vi.fn(async () => {}),
      reload: vi.fn(async () => {}),
      hasPerm: () => true,
      hasRegion: () => true,
      hasPage: () => true,
    }),
  }))
  vi.mock('vue-router', () => ({
    useRoute: () => hoisted.routeMock,
    useRouter: () => hoisted.routerMock,
  }))
  // OPT-20260819-038: 让 guard 直接放行并把幂等键透传给 mergeIdempotencyHeaders，
  // 断言 POST 携带 Idempotency-Key 头。
  vi.mock('../utils/clickGuard.js', () => ({
    createClickGuard: () => ({
      run: async (fn) => fn({ idempotencyKey: 'ik-test-invite' }),
      isBusy: () => false,
    }),
    mergeIdempotencyHeaders: (headers, key) => ({ ...(headers || {}), 'Idempotency-Key': key }),
  }))

  function okResponse(body) {
    return { ok: true, status: 200, json: async () => body }
  }

  beforeEach(() => {
    vi.clearAllMocks()
    // PeopleInvite 使用 window.apiFetch 全局（taskFE 全局契约），单测需手动绑定
    window.apiFetch = hoisted.apiFetchMock
    hoisted.apiFetchMock.mockImplementation(async (url, opts) => {
      if (url.includes('/accounts/members/invite/')) {
        return { ok: true, status: 201, json: async () => ({ message: '邀请已发送' }) }
      }
      if (url.includes('/api/projects/workspaces/tenant_id/')) {
        return okResponse({ data: [{ id: 'ws1', name: 'WS1' }] })
      }
      if (url.includes('/api/auth/roles/company_id/')) {
        return okResponse([])
      }
      if (url.includes('resource-groups')) {
        return okResponse({ pages: [] })
      }
      return okResponse({})
    })
  })

  const { default: PeopleInvite } = await import('./PeopleInvite.vue')

  describe('PeopleInvite 写操作 clickGuard 接线', () => {
    it('邀请用户 POST 携带 Idempotency-Key', async () => {
      const wrapper = mount(PeopleInvite)
      await flushPromises()

      const form = wrapper.find('form')
      expect(form.exists()).toBe(true)
      await form.trigger('submit')
      await flushPromises()

      const postCall = hoisted.apiFetchMock.mock.calls.find(([, o]) => o?.method === 'POST')
      expect(postCall).toBeTruthy()
      expect(postCall[0]).toBe('/api/tenant/t1/accounts/members/invite/')
      expect(postCall[1].headers['Idempotency-Key']).toBe('ik-test-invite')
    })
  })
}

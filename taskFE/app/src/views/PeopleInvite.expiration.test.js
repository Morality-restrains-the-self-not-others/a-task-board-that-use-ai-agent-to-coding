// @vitest-environment jsdom
// OPT-20260829-027 回归：邮箱/电话邀请表单也展示有效期下拉，可选项与复制链接一致。
if (!process.env.VITEST) {
  console.log('[skip] PeopleInvite.expiration.test.js requires vitest runtime')
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
  vi.mock('../utils/toastService', () => ({ default: hoisted.toast }))
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

  describe('PeopleInvite 邮箱/电话表单有效期下拉', () => {
    it('email 与 phone 表单均含 invite-expiration-days 且可选项含 365', async () => {
      const wrapper = mount(PeopleInvite)
      await flushPromises()

      const radios = wrapper.findAll('input[type="radio"]')
      const optionValues = () =>
        wrapper
          .find('[data-testid="invite-expiration-days"]')
          .findAll('option')
          .map((o) => Number(o.attributes('value')))

      await radios.find((r) => r.element.value === 'email').setValue(true)
      await flushPromises()
      expect(wrapper.find('[data-testid="invite-expiration-days"]').exists()).toBe(true)
      expect(optionValues()).toContain(365)

      await radios.find((r) => r.element.value === 'phone').setValue(true)
      await flushPromises()
      expect(wrapper.find('[data-testid="invite-expiration-days"]').exists()).toBe(true)
      expect(optionValues()).toContain(365)

      await radios.find((r) => r.element.value === 'link').setValue(true)
      await flushPromises()
      expect(wrapper.find('[data-testid="invite-expiration-days"]').exists()).toBe(true)
      expect(optionValues()).toContain(365)
    })
  })
}

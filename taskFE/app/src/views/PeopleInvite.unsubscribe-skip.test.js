// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] PeopleInvite.unsubscribe-skip.test.js requires vitest runtime')
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
        return {
          ok: true,
          status: 201,
          json: async () => ({
            email_skipped: true,
            code: 'email_unsubscribed',
            message: '该邮箱已退订邮件邀请，请手动复制邀请链接给对方',
            invitation_url: 'https://example.test/tenant/t1/people/join/?token=abc',
            invite_token: 'abc',
          }),
        }
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

  describe('PeopleInvite 退订跳过发信', () => {
    it('email_skipped 时展示复制链接提示，不 toast 已发送', async () => {
      const wrapper = mount(PeopleInvite, {
        global: { stubs: { InviteAccessGrants: true } },
      })
      await flushPromises()

      const form = wrapper.find('form')
      expect(form.exists()).toBe(true)
      await form.trigger('submit')
      await flushPromises()

      expect(hoisted.toast.success).not.toHaveBeenCalled()
      const prompt = wrapper.find('[data-testid="invite-copy-link-prompt"]')
      expect(prompt.exists()).toBe(true)
      expect(prompt.text()).toContain('手动复制邀请链接')
      expect(wrapper.find('[data-testid="invite-copy-link-input"]').element.value)
        .toContain('/people/join/?token=abc')
    })
  })
}

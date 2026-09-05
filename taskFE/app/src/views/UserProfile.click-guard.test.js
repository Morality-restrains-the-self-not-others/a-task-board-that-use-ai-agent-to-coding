// @vitest-environment jsdom
// OPT-20260819-038 回归：个人资料 保存 PATCH / 移除头像 DELETE 写请求携带 Idempotency-Key 头。
if (!process.env.VITEST) {
  console.log('[skip] UserProfile.click-guard.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { describe, expect, it, vi, beforeEach } = await import('vitest')

  const { apiFetchMock, routeMock, scrollToRgKeyMock } = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
    routeMock: { params: { tenant: 't1' }, query: {} },
    scrollToRgKeyMock: vi.fn(() => true),
  }))
  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch: apiFetchMock,
    clearCachedAuthToken: vi.fn(),
  }))
  vi.mock('vue-router', () => ({ useRoute: () => routeMock }))
  vi.mock('../utils/rgDeepLink.js', () => ({ scrollToRgKey: scrollToRgKeyMock }))
  vi.mock('../utils/toastService.js', () => ({
    default: { success: vi.fn(), error: vi.fn(), warning: vi.fn(), hide: vi.fn(), state: { show: false } },
  }))
  // OPT-20260819-038: 让 guard 直接放行并把幂等键透传给 mergeIdempotencyHeaders，
  // 断言 PATCH/DELETE 携带 Idempotency-Key 头。
  vi.mock('../utils/clickGuard.js', () => ({
    createClickGuard: () => ({
      run: async (fn) => fn({ idempotencyKey: 'ik-test-userprofile' }),
      isBusy: () => false,
    }),
    mergeIdempotencyHeaders: (headers, key) => ({ ...(headers || {}), 'Idempotency-Key': key }),
  }))

  const PROFILE = {
    user_id: 'u1',
    email: 'a@b.com',
    has_email: true,
    personal_nickname: 'alice',
    avatar_url: 'https://x/a.png',
    company_nicknames: [],
  }

  beforeEach(() => {
    vi.clearAllMocks()
    Element.prototype.scrollIntoView = vi.fn()
    apiFetchMock.mockImplementation(async (url, opts) => {
      if (opts?.method === 'PATCH' || opts?.method === 'DELETE' || opts?.method === 'POST') {
        return { ok: true, json: async () => PROFILE }
      }
      return { ok: true, json: async () => PROFILE }
    })
  })

  const { default: UserProfile } = await import('./UserProfile.vue')

  const STUBS = {
    UserCenterSidebar: { template: '<div />' },
    UserProfileEmailBindingPanel: { template: '<div />' },
    UserProfilePhoneBindingPanel: { template: '<div />' },
    UserProfileWechatBindingPanel: { template: '<div />' },
    UserProfilePersonalDataExportPanel: { template: '<div />' },
    UserProfileAccountDeletionPanel: { template: '<div />' },
    UserProfileCompanySettingsPanel: { template: '<div />' },
  }

  describe('UserProfile 写操作 clickGuard 接线', () => {
    it('保存个人资料 PATCH 携带 Idempotency-Key', async () => {
      const wrapper = mount(UserProfile, { global: { stubs: STUBS } })
      await flushPromises()

      const saveBtn = wrapper.findAll('button').find((b) => b.text().trim() === '保存')
      expect(saveBtn).toBeTruthy()
      await saveBtn.trigger('click')
      await flushPromises()

      const patchCall = apiFetchMock.mock.calls.find(([, o]) => o?.method === 'PATCH')
      expect(patchCall).toBeTruthy()
      expect(patchCall[0]).toBe('/api/accounts/users/profile/')
      expect(patchCall[1].headers['Idempotency-Key']).toBe('ik-test-userprofile')
    })

    it('移除头像 DELETE 携带 Idempotency-Key', async () => {
      const wrapper = mount(UserProfile, { global: { stubs: STUBS } })
      await flushPromises()

      const removeBtn = wrapper.findAll('button').find((b) => b.text().includes('移除头像'))
      expect(removeBtn).toBeTruthy()
      await removeBtn.trigger('click')
      await flushPromises()

      const delCall = apiFetchMock.mock.calls.find(([, o]) => o?.method === 'DELETE')
      expect(delCall).toBeTruthy()
      expect(delCall[0]).toBe('/api/accounts/users/profile/avatar/')
      expect(delCall[1].headers['Idempotency-Key']).toBe('ik-test-userprofile')
    })
  })
}

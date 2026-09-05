// @vitest-environment jsdom
// OPT-20260819-038 回归：公司设置面板 复制到公司 POST / 移除头像 DELETE 写请求携带 Idempotency-Key 头。
if (!process.env.VITEST) {
  console.log('[skip] UserProfileCompanySettingsPanel.click-guard.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { describe, expect, it, vi, beforeEach } = await import('vitest')

  const mocks = vi.hoisted(() => ({
    apiFetch: vi.fn(),
  }))

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch: (...args) => mocks.apiFetch(...args),
  }))
  // OPT-20260819-038: 让 guard 直接放行并把幂等键透传给 mergeIdempotencyHeaders，
  // 断言 POST/DELETE 携带 Idempotency-Key 头。
  vi.mock('../utils/clickGuard.js', () => ({
    createClickGuard: () => ({
      run: async (fn) => fn({ idempotencyKey: 'ik-test-company' }),
      isBusy: () => false,
    }),
    mergeIdempotencyHeaders: (headers, key) => ({ ...(headers || {}), 'Idempotency-Key': key }),
  }))

  const companyNicknames = [
    { company_id: 'c1', company_name: 'ACME', member_name: 'alice', member_avatar_url: 'https://x/a.png' },
  ]

  beforeEach(() => {
    vi.clearAllMocks()
    mocks.apiFetch.mockImplementation(async () => ({
      ok: true,
      json: async () => ({ company_nicknames: companyNicknames }),
    }))
  })

  const { default: Panel } = await import('./UserProfileCompanySettingsPanel.vue')

  describe('UserProfileCompanySettingsPanel 写操作 clickGuard 接线', () => {
    it('复制到公司 POST 携带 Idempotency-Key', async () => {
      const wrapper = mount(Panel, {
        props: { companyNicknames, personalNickname: 'alice', isSaving: false },
      })
      await flushPromises()

      const copyBtn = wrapper.findAll('button').find((b) => b.text().includes('从个人昵称与头像复制'))
      expect(copyBtn).toBeTruthy()
      await copyBtn.trigger('click')
      await flushPromises()

      const postCall = mocks.apiFetch.mock.calls.find(([, o]) => o?.method === 'POST')
      expect(postCall).toBeTruthy()
      expect(postCall[0]).toBe('/api/accounts/users/profile/copy-personal-to-company/')
      expect(postCall[1].headers['Idempotency-Key']).toBe('ik-test-company')
    })

    it('移除公司头像 DELETE 携带 Idempotency-Key', async () => {
      const wrapper = mount(Panel, {
        props: { companyNicknames, personalNickname: 'alice', isSaving: false },
      })
      await flushPromises()

      const removeBtn = wrapper.findAll('button').find((b) => b.text().includes('移除头像'))
      expect(removeBtn).toBeTruthy()
      await removeBtn.trigger('click')
      await flushPromises()

      const delCall = mocks.apiFetch.mock.calls.find(([, o]) => o?.method === 'DELETE')
      expect(delCall).toBeTruthy()
      expect(delCall[0]).toBe('/api/accounts/users/profile/company-avatar/?company_id=c1')
      expect(delCall[1].headers['Idempotency-Key']).toBe('ik-test-company')
    })
  })
}

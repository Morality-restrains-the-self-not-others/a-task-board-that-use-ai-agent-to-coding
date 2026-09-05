// @vitest-environment jsdom
// 回归：资料页 bind-phone 假成功 {ok:true} 不得跳转；占用 409 露出 reclaim。
if (!process.env.VITEST) {
  console.log('[skip] useUserProfilePhoneBindingPanel.test.js requires vitest runtime')
} else {
  const { describe, expect, it, vi, beforeEach } = await import('vitest')
  const { flushPromises } = await import('@vue/test-utils')
  const { effectScope } = await import('vue')

  const apiFetchMock = vi.hoisted(() => vi.fn())
  const navigateMock = vi.hoisted(() => vi.fn(() => false))

  vi.mock('../../utils/apiUtils.js', () => ({ apiFetch: apiFetchMock }))
  vi.mock('../../utils/phoneBindingDeepLink.js', async (importOriginal) => {
    const actual = await importOriginal()
    return {
      ...actual,
      maybeNavigateToPhoneVerifyRedirect: (...args) => navigateMock(...args),
    }
  })
  vi.mock('./useAllowedCountryCodes.js', () => ({
    useAllowedCountryCodes: () => ({
      filteredCountryOptions: { value: [{ prefix: '+86', label: '中国' }] },
      defaultCountryPrefix: { value: '+86' },
      syncPrefixWithAllowedOptions: () => {},
      fetchAllowedCodes: async () => {},
      loading: { value: false },
      error: { value: '' },
    }),
  }))

  const { useUserProfilePhoneBindingPanel } = await import('./useUserProfilePhoneBindingPanel.js')

  function setupPanel() {
    const emit = vi.fn()
    const scope = effectScope()
    const api = scope.run(() => useUserProfilePhoneBindingPanel(emit))
    api.bindPhoneNational.value = '18959264502'
    api.bindSmsCode.value = '123456'
    return { emit, api, scope }
  }

  beforeEach(() => {
    apiFetchMock.mockReset()
    navigateMock.mockReset()
    navigateMock.mockReturnValue(false)
  })

  describe('useUserProfilePhoneBindingPanel 绑定结果', () => {
    it('假成功 {ok:true} 且无 bound 不得跳转', async () => {
      apiFetchMock.mockResolvedValue({
        ok: true,
        json: async () => ({ ok: true }),
      })
      navigateMock.mockReturnValue(true)
      const { api, emit, scope } = setupPanel()
      await api.submitBindPhone(false)
      await flushPromises()
      expect(navigateMock).not.toHaveBeenCalled()
      expect(api.bindInlineError.value).toContain('绑定未确认')
      expect(emit).not.toHaveBeenCalledWith('profile-updated', expect.anything())
      scope.stop()
    })

    it('409 phone_taken 露出 reclaim 且不跳转', async () => {
      apiFetchMock.mockResolvedValue({
        ok: false,
        status: 409,
        json: async () => ({
          code: 'phone_taken',
          reclaim_available: true,
          detail: '该手机号已绑定其他账号',
        }),
      })
      const { api, scope } = setupPanel()
      await api.submitBindPhone(false)
      await flushPromises()
      expect(api.bindReclaimAvailable.value).toBe(true)
      expect(navigateMock).not.toHaveBeenCalled()
      scope.stop()
    })

    it('409 phone_bind_limit 不露出 reclaim', async () => {
      apiFetchMock.mockResolvedValue({
        ok: false,
        status: 409,
        json: async () => ({
          code: 'phone_bind_limit',
          reclaim_available: false,
          detail: '该手机号已绑定 5 个账号，无法再绑定。',
        }),
      })
      const { api, scope } = setupPanel()
      await api.submitBindPhone(false)
      await flushPromises()
      expect(api.bindReclaimAvailable.value).toBe(false)
      expect(api.bindInlineError.value).toContain('5 个账号')
      expect(navigateMock).not.toHaveBeenCalled()
      scope.stop()
    })
  })
}

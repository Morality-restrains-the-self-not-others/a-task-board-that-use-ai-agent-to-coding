// @vitest-environment jsdom
// OPT-20260825-002：登录后未绑手机引导「去验证」→ 绑定成功后回到原 next 落点。
// 纯逻辑（sessionStorage 暂存/清除）在 phoneBindingDeepLink.test.js 覆盖；
// 此处仅锁住面板「绑定/更换成功时调用回跳 helper 且成功文案让位于跳转」的接线。
if (!process.env.VITEST) {
  console.log('[skip] UserProfilePhoneBindingPanel.navigate-back.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { describe, expect, it, vi, beforeEach } = await import('vitest')

  const apiFetchMock = vi.hoisted(() => vi.fn())
  vi.mock('../utils/apiUtils', () => ({ apiFetch: apiFetchMock }))

  vi.mock('../utils/phoneBindingDeepLink.js', async (importOriginal) => {
    const actual = await importOriginal()
    return {
      ...actual,
      maybeNavigateToPhoneVerifyRedirect: vi.fn(() => false),
    }
  })

  const { maybeNavigateToPhoneVerifyRedirect } = await import('../utils/phoneBindingDeepLink.js')
  const { default: UserProfilePhoneBindingPanel } = await import('./UserProfilePhoneBindingPanel.vue')

  beforeEach(() => {
    apiFetchMock.mockReset()
    maybeNavigateToPhoneVerifyRedirect.mockReset()
    maybeNavigateToPhoneVerifyRedirect.mockReturnValue(false)
  })

  const mountPanel = (props = {}) =>
    mount(UserProfilePhoneBindingPanel, { props: { hasPhone: false, phoneMasked: '', ...props } })

  const okApiFetch = (url) => {
    if (url.includes('bind-phone')) {
      return { ok: true, json: async () => ({ bound: true, phone_masked: '138****8000' }) }
    }
    return { ok: true, json: async () => ({}) }
  }

  describe('UserProfilePhoneBindingPanel 绑定后回跳原落点', () => {
    it('绑定成功且存在暂存原落点 → 调 helper 且不发成功消息（让位于跳转）', async () => {
      apiFetchMock.mockImplementation(okApiFetch)
      maybeNavigateToPhoneVerifyRedirect.mockReturnValue(true)
      const wrapper = mountPanel()
      await wrapper.find('input[autocomplete="tel-national"]').setValue('13800138000')
      await wrapper.find('input[autocomplete="one-time-code"]').setValue('123456')
      await wrapper.findAll('button').find((b) => b.text().includes('验证并绑定')).trigger('click')
      await flushPromises()

      expect(apiFetchMock).toHaveBeenCalledWith(
        '/api/accounts/users/profile/bind-phone/',
        expect.objectContaining({ method: 'POST' }),
      )
      expect(maybeNavigateToPhoneVerifyRedirect).toHaveBeenCalled()
      // 仅初始清空 message，不应出现「已成功绑定手机号」成功文案
      expect(wrapper.emitted('message')).toEqual([['']])
    })

    it('绑定成功且无暂存原落点 → 正常发成功消息', async () => {
      apiFetchMock.mockImplementation(okApiFetch)
      const wrapper = mountPanel()
      await wrapper.find('input[autocomplete="tel-national"]').setValue('13800138000')
      await wrapper.find('input[autocomplete="one-time-code"]').setValue('123456')
      await wrapper.findAll('button').find((b) => b.text().includes('验证并绑定')).trigger('click')
      await flushPromises()

      expect(maybeNavigateToPhoneVerifyRedirect).toHaveBeenCalled()
      expect(wrapper.emitted('message')?.at(-1)?.[0]).toContain('已成功绑定手机号')
    })

    it('假成功 {ok:true} 且无 bound 不得回跳', async () => {
      apiFetchMock.mockImplementation((url) => {
        if (url.includes('bind-phone')) {
          return { ok: true, json: async () => ({ ok: true }) }
        }
        return { ok: true, json: async () => ({}) }
      })
      maybeNavigateToPhoneVerifyRedirect.mockReturnValue(true)
      const wrapper = mountPanel()
      await wrapper.find('input[autocomplete="tel-national"]').setValue('13800138000')
      await wrapper.find('input[autocomplete="one-time-code"]').setValue('123456')
      await wrapper.findAll('button').find((b) => b.text().includes('验证并绑定')).trigger('click')
      await flushPromises()
      expect(maybeNavigateToPhoneVerifyRedirect).not.toHaveBeenCalled()
      expect(wrapper.text()).toContain('绑定未确认')
    })

    it('409 phone_taken 展示转移按钮，确认后带 reclaim', async () => {
      apiFetchMock.mockImplementation((url, opts) => {
        if (url.includes('bind-phone')) {
          const body = JSON.parse(opts.body)
          if (body.reclaim) {
            return { ok: true, json: async () => ({ bound: true }) }
          }
          return {
            ok: false,
            json: async () => ({
              code: 'phone_taken',
              reclaim_available: true,
              error: '该手机号已绑定其他账号',
              detail: '该手机号已绑定其他账号。若这是您本人正在使用的号码，请确认转移到本账号。',
            }),
          }
        }
        return { ok: true, json: async () => ({}) }
      })
      const wrapper = mountPanel()
      await wrapper.find('input[autocomplete="tel-national"]').setValue('13800138000')
      await wrapper.find('input[autocomplete="one-time-code"]').setValue('123456')
      await wrapper.findAll('button').find((b) => b.text().includes('验证并绑定')).trigger('click')
      await flushPromises()
      expect(wrapper.text()).toContain('确认将号码转移到本账号')
      expect(maybeNavigateToPhoneVerifyRedirect).not.toHaveBeenCalled()
      await new Promise((resolve) => setTimeout(resolve, 350))
      await wrapper.findAll('button').find((b) => b.text().includes('确认将号码转移到本账号')).trigger('click')
      await flushPromises()
      const reclaimCall = apiFetchMock.mock.calls.find(([, opts]) => {
        try {
          return JSON.parse(opts.body).reclaim === true
        } catch {
          return false
        }
      })
      expect(reclaimCall).toBeTruthy()
      expect(maybeNavigateToPhoneVerifyRedirect).toHaveBeenCalled()
    })
  })
}

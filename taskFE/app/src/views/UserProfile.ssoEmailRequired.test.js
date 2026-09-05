// @vitest-environment jsdom
// OPT-20260806-065/066：SSO bridge 对无邮箱用户重定向到个人中心 ?sso_error=email_required，
// UserProfile 须 toast 引导，并在资料加载完成后滚动定位到邮箱绑定区域。
if (!process.env.VITEST) {
  console.log('[skip] UserProfile.ssoEmailRequired.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { describe, expect, it, vi, beforeEach, afterEach } = await import('vitest')
  const toastService = (await import('../utils/toastService.js')).default
  const { EMAIL_BINDING_RG_KEY } = await import('../utils/emailBindingDeepLink.js')

  const { apiFetchMock, routeMock, scrollToRgKeyMock } = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
    routeMock: { params: { tenant: 't1' }, query: {} },
    scrollToRgKeyMock: vi.fn(() => true),
  }))
  vi.mock('../utils/apiUtils.js', () => ({ apiFetch: apiFetchMock }))
  vi.mock('vue-router', () => ({ useRoute: () => routeMock }))
  vi.mock('../utils/rgDeepLink.js', () => ({
    scrollToRgKey: scrollToRgKeyMock,
  }))

  const { default: UserProfile } = await import('./UserProfile.vue')

  const mountProfile = async () => {
    apiFetchMock.mockResolvedValue({
      ok: true,
      json: async () => ({
        user_id: 'u1',
        email: '',
        has_email: false,
        personal_nickname: '',
      }),
    })
    return mount(UserProfile, {
      global: {
        stubs: {
          UserCenterSidebar: true,
          UserProfileEmailBindingPanel: true,
          UserProfilePhoneBindingPanel: true,
          UserProfileWechatBindingPanel: true,
        },
      },
    })
  }

  describe('UserProfile.vue sso_error=email_required 引导', () => {
    beforeEach(() => {
      scrollToRgKeyMock.mockClear()
      scrollToRgKeyMock.mockReturnValue(true)
      Element.prototype.scrollIntoView = vi.fn()
      window.location.hash = ''
    })
    afterEach(() => {
      toastService.hide()
    })

    it('携带 sso_error=email_required 时 toast 提示绑定邮箱', async () => {
      routeMock.query = { sso_error: 'email_required' }
      toastService.hide()
      await mountProfile()
      await flushPromises()
      expect(toastService.state.show).toBe(true)
      expect(toastService.state.message).toContain('绑定邮箱')
      expect(toastService.state.type).toBe('warning')
    })

    it('无 sso_error 时不弹出提示', async () => {
      routeMock.query = {}
      toastService.hide()
      await mountProfile()
      await flushPromises()
      expect(toastService.state.show).toBe(false)
    })

    it('资料加载完成后定位邮箱绑定区域（profile.email_binding）', async () => {
      routeMock.query = { sso_error: 'email_required' }
      await mountProfile()
      // 加载中时不应定位（邮箱区块尚未挂载）
      expect(scrollToRgKeyMock).not.toHaveBeenCalled()
      await flushPromises()
      expect(scrollToRgKeyMock).toHaveBeenCalledWith(
        EMAIL_BINDING_RG_KEY,
        expect.objectContaining({ scrollOptions: expect.objectContaining({ block: 'center' }) }),
      )
    })

    it('邮箱绑定区块暴露 data-rg-key=profile.email_binding 供深链定位', async () => {
      routeMock.query = {}
      const wrapper = await mountProfile()
      await flushPromises()
      const anchor = wrapper.find(`[data-rg-key="${EMAIL_BINDING_RG_KEY}"]`)
      expect(anchor.exists()).toBe(true)
      expect(anchor.attributes('id')).toBe('email-binding')
    })

    it('手机绑定区块暴露 data-rg-key=profile.phone_binding 供登录引导深链定位', async () => {
      routeMock.query = {}
      const wrapper = await mountProfile()
      await flushPromises()
      const { PHONE_BINDING_RG_KEY } = await import('../utils/phoneBindingDeepLink.js')
      const anchor = wrapper.find(`[data-rg-key="${PHONE_BINDING_RG_KEY}"]`)
      expect(anchor.exists()).toBe(true)
      expect(anchor.attributes('id')).toBe('phone-binding')
    })

    it('hash 为 #rg=profile.phone_binding 时资料加载后定位手机绑定区', async () => {
      routeMock.query = {}
      window.location.hash = '#rg=profile.phone_binding'
      await mountProfile()
      expect(scrollToRgKeyMock).not.toHaveBeenCalled()
      await flushPromises()
      const { PHONE_BINDING_RG_KEY } = await import('../utils/phoneBindingDeepLink.js')
      expect(scrollToRgKeyMock).toHaveBeenCalledWith(
        PHONE_BINDING_RG_KEY,
        expect.objectContaining({ scrollOptions: expect.objectContaining({ block: 'center' }) }),
      )
      window.location.hash = ''
    })
  })
}

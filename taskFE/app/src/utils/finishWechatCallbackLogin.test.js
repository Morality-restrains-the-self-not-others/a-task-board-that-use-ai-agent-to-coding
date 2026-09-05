// @vitest-environment jsdom
import { describe, expect, it, vi } from 'vitest'
import { finishWechatCallbackLogin } from './finishWechatCallbackLogin.js'

describe('finishWechatCallbackLogin', () => {
  it('无手机号且用户确认 → 资料页绑定深链', async () => {
    const persist = vi.fn().mockResolvedValue(undefined)
    const resolveIdentity = vi.fn().mockResolvedValue({
      ok: true,
      profile: { user_id: 'u1', has_phone: false },
    })
    const resolveRedirect = vi.fn().mockReturnValue('/onboarding/')
    const promptPhoneVerify = vi.fn().mockResolvedValue({
      action: 'verify',
      href: '/profile/#rg=profile.phone_binding',
    })
    const navigate = vi.fn()
    const replaceHistory = vi.fn()

    await finishWechatCallbackLogin({
      wechatToken: 'tok',
      next: '/onboarding/',
      persist,
      resolveIdentity,
      resolveRedirect,
      promptPhoneVerify,
      navigate,
      replaceHistory,
    })

    expect(persist).toHaveBeenCalledWith({ token: 'tok' })
    expect(promptPhoneVerify).toHaveBeenCalledWith(
      expect.objectContaining({
        skipPrompt: false,
        source: { user_id: 'u1', has_phone: false },
        redirectUrl: '/onboarding/',
      }),
    )
    expect(navigate).toHaveBeenCalledWith('/profile/#rg=profile.phone_binding')
  })

  it('已有手机号 → skip 后走原落点', async () => {
    const promptPhoneVerify = vi.fn().mockResolvedValue({
      action: 'redirect',
      href: '/tenant/1/work-panel/',
    })
    const navigate = vi.fn()
    await finishWechatCallbackLogin({
      wechatToken: 'tok',
      next: '',
      persist: vi.fn().mockResolvedValue(undefined),
      resolveIdentity: vi.fn().mockResolvedValue({
        ok: true,
        profile: { has_phone: true },
      }),
      resolveRedirect: () => '/tenant/1/work-panel/',
      promptPhoneVerify,
      navigate,
      replaceHistory: vi.fn(),
    })
    expect(promptPhoneVerify).toHaveBeenCalledWith(
      expect.objectContaining({ skipPrompt: false, source: { has_phone: true } }),
    )
    expect(navigate).toHaveBeenCalledWith('/tenant/1/work-panel/')
  })

  it('身份解析失败 fail-open：skipPrompt 且原落点', async () => {
    const promptPhoneVerify = vi.fn().mockResolvedValue({
      action: 'redirect',
      href: '/onboarding/',
    })
    await finishWechatCallbackLogin({
      wechatToken: 'tok',
      next: '',
      persist: vi.fn().mockResolvedValue(undefined),
      resolveIdentity: vi.fn().mockResolvedValue({ ok: false, profile: null }),
      resolveRedirect: () => '/onboarding/',
      promptPhoneVerify,
      navigate: vi.fn(),
      replaceHistory: vi.fn(),
    })
    expect(promptPhoneVerify).toHaveBeenCalledWith(
      expect.objectContaining({ skipPrompt: true }),
    )
  })
})

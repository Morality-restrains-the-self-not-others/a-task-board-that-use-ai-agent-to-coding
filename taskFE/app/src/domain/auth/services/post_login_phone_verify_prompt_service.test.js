// @vitest-environment jsdom
import { describe, expect, it, vi, beforeEach } from 'vitest'
import {
  LOGIN_PHONE_VERIFY_PROMPT_CONFIRM,
  LOGIN_PHONE_VERIFY_PROMPT_MESSAGE,
  LOGIN_PHONE_VERIFY_PROMPT_TITLE,
  promptPostLoginPhoneVerify,
  shouldSkipPhoneVerifyPrompt,
} from './post_login_phone_verify_prompt_service.js'
import { PHONE_VERIFY_REDIRECT_STORAGE_KEY } from '../../../utils/phoneBindingDeepLink.js'

describe('shouldSkipPhoneVerifyPrompt', () => {
  it('管理员入口 / 手机密码登录 / 模拟登录跳过', () => {
    expect(shouldSkipPhoneVerifyPrompt({ adminLogin: true })).toBe(true)
    expect(shouldSkipPhoneVerifyPrompt({ loginMethod: 'phonePassword' })).toBe(true)
    expect(shouldSkipPhoneVerifyPrompt({ isImpersonating: true })).toBe(true)
    expect(shouldSkipPhoneVerifyPrompt({ loginMethod: 'emailPassword' })).toBe(false)
  })
})

describe('promptPostLoginPhoneVerify 硬门禁', () => {
  const redirectUrl = '/tenant/1/work-panel/'

  beforeEach(() => {
    sessionStorage.removeItem(PHONE_VERIFY_REDIRECT_STORAGE_KEY)
  })

  it('skipPrompt 时不调用 acknowledge，直达原落点', async () => {
    const acknowledge = vi.fn()
    const decision = await promptPostLoginPhoneVerify({
      skipPrompt: true,
      source: {},
      redirectUrl,
      acknowledge,
    })
    expect(acknowledge).not.toHaveBeenCalled()
    expect(decision).toEqual({ action: 'redirect', href: redirectUrl })
  })

  it('已验证手机不弹窗', async () => {
    const acknowledge = vi.fn()
    const decision = await promptPostLoginPhoneVerify({
      source: { login_methods: [{ method_type: 'phone', is_verified: true }] },
      redirectUrl,
      acknowledge,
    })
    expect(acknowledge).not.toHaveBeenCalled()
    expect(decision).toEqual({ action: 'redirect', href: redirectUrl })
  })

  it('未验证且用户确认 → 资料页绑定深链，并暂存原落点', async () => {
    const acknowledge = vi.fn().mockResolvedValue(true)
    const decision = await promptPostLoginPhoneVerify({
      source: { login_methods: [] },
      redirectUrl,
      acknowledge,
    })
    expect(acknowledge).toHaveBeenCalledWith(
      LOGIN_PHONE_VERIFY_PROMPT_MESSAGE,
      LOGIN_PHONE_VERIFY_PROMPT_TITLE,
      LOGIN_PHONE_VERIFY_PROMPT_CONFIRM,
    )
    expect(decision).toEqual({ action: 'verify', href: '/profile/#rg=profile.phone_binding' })
    expect(sessionStorage.getItem(PHONE_VERIFY_REDIRECT_STORAGE_KEY)).toBe(redirectUrl)
  })

  it('未验证且用户关闭弹窗 → 仍去资料绑定（不可跳过）', async () => {
    const acknowledge = vi.fn().mockRejectedValue(false)
    const decision = await promptPostLoginPhoneVerify({
      source: { has_phone: false },
      redirectUrl,
      acknowledge,
    })
    expect(decision).toEqual({ action: 'verify', href: '/profile/#rg=profile.phone_binding' })
    expect(sessionStorage.getItem(PHONE_VERIFY_REDIRECT_STORAGE_KEY)).toBe(redirectUrl)
  })

  it('无 acknowledge 适配器时 fail-closed 去绑定', async () => {
    const decision = await promptPostLoginPhoneVerify({
      source: {},
      redirectUrl,
    })
    expect(decision).toEqual({ action: 'verify', href: '/profile/#rg=profile.phone_binding' })
  })

  it('判定抛错时 fail-closed 去绑定', async () => {
    const source = {}
    Object.defineProperty(source, 'login_methods', {
      get() {
        throw new Error('boom')
      },
    })
    const decision = await promptPostLoginPhoneVerify({
      source,
      redirectUrl,
      acknowledge: vi.fn(),
    })
    expect(decision).toEqual({ action: 'verify', href: '/profile/#rg=profile.phone_binding' })
  })
})

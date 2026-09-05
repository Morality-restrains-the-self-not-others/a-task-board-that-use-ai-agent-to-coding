// @vitest-environment jsdom
import { describe, expect, it, beforeEach, vi } from 'vitest'
import {
  PHONE_BINDING_RG_KEY,
  PHONE_VERIFY_REDIRECT_STORAGE_KEY,
  buildPhoneBindingRedirectUrl,
  consumePhoneVerifyRedirect,
  isPhoneBindingHash,
  maybeNavigateToPhoneVerifyRedirect,
  savePhoneVerifyRedirect,
} from './phoneBindingDeepLink.js'

describe('phoneBindingDeepLink 深链契约', () => {
  it('深链 key 与页面 data-rg-key 一致', () => {
    expect(PHONE_BINDING_RG_KEY).toBe('profile.phone_binding')
  })

  it('buildPhoneBindingRedirectUrl 指向当前用户资料手机绑定区', () => {
    expect(buildPhoneBindingRedirectUrl()).toBe('/profile/#rg=profile.phone_binding')
  })

  it('isPhoneBindingHash 识别 #rg=profile.phone_binding', () => {
    expect(isPhoneBindingHash('#rg=profile.phone_binding')).toBe(true)
    expect(isPhoneBindingHash('#rg=profile.email_binding')).toBe(false)
    expect(isPhoneBindingHash('')).toBe(false)
  })
})

describe('OPT-20260825-002 phone verify 回跳暂存', () => {
  beforeEach(() => {
    sessionStorage.removeItem(PHONE_VERIFY_REDIRECT_STORAGE_KEY)
  })

  it('savePhoneVerifyRedirect 写入 sessionStorage 原登录落点', () => {
    expect(savePhoneVerifyRedirect('/tenant/1/work-panel/')).toBe(true)
    expect(sessionStorage.getItem(PHONE_VERIFY_REDIRECT_STORAGE_KEY)).toBe('/tenant/1/work-panel/')
  })

  it('savePhoneVerifyRedirect 拒绝非法路径（外链/双斜杠/空）', () => {
    expect(savePhoneVerifyRedirect('https://evil.example')).toBe(false)
    expect(savePhoneVerifyRedirect('//evil.example')).toBe(false)
    expect(savePhoneVerifyRedirect('')).toBe(false)
    expect(sessionStorage.getItem(PHONE_VERIFY_REDIRECT_STORAGE_KEY)).toBeNull()
  })

  it('consumePhoneVerifyRedirect 读取并清除暂存落点', () => {
    sessionStorage.setItem(PHONE_VERIFY_REDIRECT_STORAGE_KEY, '/billing/orders/')
    expect(consumePhoneVerifyRedirect()).toBe('/billing/orders/')
    expect(sessionStorage.getItem(PHONE_VERIFY_REDIRECT_STORAGE_KEY)).toBeNull()
  })

  it('consumePhoneVerifyRedirect 无暂存时返回空串', () => {
    expect(consumePhoneVerifyRedirect()).toBe('')
  })

  it('maybeNavigateToPhoneVerifyRedirect 有暂存时跳回并返回 true', () => {
    sessionStorage.setItem(PHONE_VERIFY_REDIRECT_STORAGE_KEY, '/profile/referral/')
    const navigate = vi.fn()
    expect(maybeNavigateToPhoneVerifyRedirect(navigate)).toBe(true)
    expect(navigate).toHaveBeenCalledWith('/profile/referral/')
    expect(sessionStorage.getItem(PHONE_VERIFY_REDIRECT_STORAGE_KEY)).toBeNull()
  })

  it('maybeNavigateToPhoneVerifyRedirect 无暂存时不跳转返回 false', () => {
    const navigate = vi.fn()
    expect(maybeNavigateToPhoneVerifyRedirect(navigate)).toBe(false)
    expect(navigate).not.toHaveBeenCalled()
  })
})

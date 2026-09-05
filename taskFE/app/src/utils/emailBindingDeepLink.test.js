import { describe, it, expect } from 'vitest'
import {
  EMAIL_BINDING_RG_KEY,
  EMAIL_REQUIRED_SSO_ERROR,
  buildEmailBindingRedirectUrl,
  isEmailRequiredSsoError,
} from './emailBindingDeepLink.js'

// OPT-20260812-016：邮箱绑定深链 key / 引导 URL 跨模块共用，防止字面量漂移。
describe('emailBindingDeepLink 深链契约', () => {
  it('深链 key 与 SSO bridge / 页面 data-rg-key 一致', () => {
    expect(EMAIL_BINDING_RG_KEY).toBe('profile.email_binding')
  })

  it('sso_error 值与 taskAuth sso_bridge.go 重定向参数一致', () => {
    expect(EMAIL_REQUIRED_SSO_ERROR).toBe('email_required')
  })

  it('buildEmailBindingRedirectUrl 与 taskAuth sso_bridge.go Location 后缀一致', () => {
    expect(buildEmailBindingRedirectUrl()).toBe(
      '/profile/?sso_error=email_required#rg=profile.email_binding',
    )
  })

  it('isEmailRequiredSsoError 仅识别 email_required', () => {
    expect(isEmailRequiredSsoError({ sso_error: 'email_required' })).toBe(true)
    expect(isEmailRequiredSsoError({ sso_error: 'other' })).toBe(false)
    expect(isEmailRequiredSsoError({})).toBe(false)
    expect(isEmailRequiredSsoError(null)).toBe(false)
  })
})

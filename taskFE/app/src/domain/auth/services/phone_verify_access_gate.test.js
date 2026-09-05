import { describe, expect, it } from 'vitest'
import {
  isPhoneVerifyGateExemptPath,
  isPhoneVerifyStaffBypass,
  shouldBlockForUnverifiedPhone,
} from './phone_verify_access_gate.js'

describe('isPhoneVerifyGateExemptPath', () => {
  it('资料主页与登录注册激活放行', () => {
    expect(isPhoneVerifyGateExemptPath('/profile/')).toBe(true)
    expect(isPhoneVerifyGateExemptPath('/profile')).toBe(true)
    expect(isPhoneVerifyGateExemptPath('/auth/login/')).toBe(true)
    expect(isPhoneVerifyGateExemptPath('/auth/admin-login/')).toBe(true)
    expect(isPhoneVerifyGateExemptPath('/auth/reset-password-request/')).toBe(true)
  })

  it('资料子页与业务页不放行', () => {
    expect(isPhoneVerifyGateExemptPath('/profile/referral/')).toBe(false)
    expect(isPhoneVerifyGateExemptPath('/profile/access-tokens/')).toBe(false)
    expect(isPhoneVerifyGateExemptPath('/tenant/1/work-panel/')).toBe(false)
    expect(isPhoneVerifyGateExemptPath('/system-admin/')).toBe(false)
    expect(isPhoneVerifyGateExemptPath('/onboarding/')).toBe(false)
  })

  it('邀请加入与分享任务放行', () => {
    expect(isPhoneVerifyGateExemptPath('/tenant/1/people/join/')).toBe(true)
    expect(
      isPhoneVerifyGateExemptPath('/tenant/1/task-detail/task_1/', { query: { accessCode: 'abc' } }),
    ).toBe(true)
    expect(isPhoneVerifyGateExemptPath('/tenant/1/task-detail/task_1/')).toBe(false)
  })

  it('模拟登录放行', () => {
    expect(
      isPhoneVerifyGateExemptPath('/tenant/1/work-panel/', { isImpersonating: true }),
    ).toBe(true)
  })
})

describe('shouldBlockForUnverifiedPhone', () => {
  const unverified = { login_methods: [{ method_type: 'email', is_verified: true }] }

  it('会话未知 fail-open', () => {
    expect(
      shouldBlockForUnverifiedPhone({
        sessionKnown: false,
        source: unverified,
        path: '/tenant/1/work-panel/',
      }),
    ).toBe(false)
  })

  it('已验证不阻断', () => {
    expect(
      shouldBlockForUnverifiedPhone({
        sessionKnown: true,
        source: { login_methods: [{ method_type: 'phone', is_verified: true }] },
        path: '/tenant/1/work-panel/',
      }),
    ).toBe(false)
  })

  it('未验证业务页阻断', () => {
    expect(
      shouldBlockForUnverifiedPhone({
        sessionKnown: true,
        source: unverified,
        path: '/tenant/1/work-panel/',
      }),
    ).toBe(true)
  })

  it('超管/员工不阻断', () => {
    expect(isPhoneVerifyStaffBypass({ is_superuser: true })).toBe(true)
    expect(
      shouldBlockForUnverifiedPhone({
        sessionKnown: true,
        source: { ...unverified, platform_roles: ['employee'] },
        path: '/system-admin/',
      }),
    ).toBe(false)
  })
})

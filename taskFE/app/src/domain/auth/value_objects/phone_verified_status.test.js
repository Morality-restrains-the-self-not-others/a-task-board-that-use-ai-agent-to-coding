import { describe, expect, it } from 'vitest'
import { userHasVerifiedPhone } from './phone_verified_status.js'

describe('userHasVerifiedPhone', () => {
  it('login_methods 含已验证 phone 为 true', () => {
    expect(
      userHasVerifiedPhone({
        login_methods: [
          { method_type: 'email', is_verified: true },
          { method_type: 'phone', is_verified: true },
        ],
      }),
    ).toBe(true)
  })

  it('phone 存在但 is_verified=false 为 false', () => {
    expect(
      userHasVerifiedPhone({
        login_methods: [{ method_type: 'phone', is_verified: false }],
      }),
    ).toBe(false)
  })

  it('仅 email 为 false', () => {
    expect(
      userHasVerifiedPhone({
        login_methods: [{ method_type: 'email', is_verified: true }],
      }),
    ).toBe(false)
  })

  it('profile.has_phone true 为 true', () => {
    expect(userHasVerifiedPhone({ has_phone: true })).toBe(true)
  })

  it('profile.has_phone false 为 false', () => {
    expect(userHasVerifiedPhone({ has_phone: false })).toBe(false)
  })

  it('空对象 / null / 非对象为 false', () => {
    expect(userHasVerifiedPhone({})).toBe(false)
    expect(userHasVerifiedPhone(null)).toBe(false)
    expect(userHasVerifiedPhone(undefined)).toBe(false)
  })
})

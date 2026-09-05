// @vitest-environment jsdom
import { describe, it, expect, beforeEach } from 'vitest'
import {
  captureReferralAccessCodeFromSearch,
  normalizeUrlAccessCodeToOwn,
  readStoredReferralAccessCode,
  writeStoredReferralAccessCode,
} from './referralAccessCodeUtils.js'

describe('referralAccessCodeUtils', () => {
  beforeEach(() => {
    sessionStorage.clear()
    localStorage.clear()
  })

  it('persists opaque accessCode from register query', () => {
    expect(captureReferralAccessCodeFromSearch('?accessCode=DR2AKvP9J9')).toBe('DR2AKvP9J9')
    expect(readStoredReferralAccessCode()).toBe('DR2AKvP9J9')
  })

  it('keeps previously stored code when query has none', () => {
    writeStoredReferralAccessCode('KeptCode12')
    expect(captureReferralAccessCodeFromSearch('/auth/register/')).toBe('KeptCode12')
  })

  it('does not treat empty query as clearing a stored code', () => {
    writeStoredReferralAccessCode('KeptCode12')
    expect(captureReferralAccessCodeFromSearch('')).toBe('KeptCode12')
  })

  // OPT-20260827-014: 项目分享链接的入站 accessCode 是分享/授权码，登录后不得被
  // 自己的推荐码覆盖（否则入站分享溯源归属丢失）。
  describe('normalizeUrlAccessCodeToOwn project-share guard', () => {
    it('preserves inbound share code on project detail route', async () => {
      window.history.replaceState({}, '', '/tenant/850256677331562496/projects/822447075168653312/?accessCode=9aaHjbryhL')
      const rewrote = await normalizeUrlAccessCodeToOwn({ ownCode: 'DR2AKvP9J9' })
      expect(rewrote).toBe(false)
      expect(window.location.search).toContain('accessCode=9aaHjbryhL')
      expect(window.location.search).not.toContain('DR2AKvP9J9')
    })

    it('preserves inbound share code on public projects list route', async () => {
      window.history.replaceState({}, '', '/projects/?accessCode=9aaHjbryhL')
      const rewrote = await normalizeUrlAccessCodeToOwn({ ownCode: 'DR2AKvP9J9' })
      expect(rewrote).toBe(false)
      expect(window.location.search).toContain('accessCode=9aaHjbryhL')
    })

    it('still normalizes own code on non-project route', async () => {
      window.history.replaceState({}, '', '/profile/referral/?accessCode=9aaHjbryhL')
      const rewrote = await normalizeUrlAccessCodeToOwn({ ownCode: 'DR2AKvP9J9' })
      expect(rewrote).toBe(true)
      expect(window.location.search).toContain('accessCode=DR2AKvP9J9')
    })
  })
})

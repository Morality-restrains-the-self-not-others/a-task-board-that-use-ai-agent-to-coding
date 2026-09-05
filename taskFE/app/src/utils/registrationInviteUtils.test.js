// @vitest-environment jsdom
import { describe, it, expect, beforeEach, vi } from 'vitest'
import {
  readStoredInviteCode,
  writeStoredInviteCode,
  fetchRegistrationInvitePolicy,
  REGISTRATION_INVITE_POLICY_PUBLIC_API,
} from './registrationInviteUtils.js'

describe('registrationInviteUtils', () => {
  beforeEach(() => {
    sessionStorage.clear()
  })

  it('normalizes and persists invite codes in sessionStorage', () => {
    expect(writeStoredInviteCode(' ab12cd34 ')).toBe('AB12CD34')
    expect(readStoredInviteCode()).toBe('AB12CD34')
    writeStoredInviteCode('')
    expect(readStoredInviteCode()).toBe('')
  })

  it('fetches public registration invite policy', async () => {
    const apiFetch = vi.fn(async () => ({
      ok: true,
      headers: { get: () => null },
      json: async () => ({ enabled: true, daily_quota: 10, remaining_today: 7 }),
    }))
    const policy = await fetchRegistrationInvitePolicy(apiFetch)
    expect(apiFetch).toHaveBeenCalledWith(REGISTRATION_INVITE_POLICY_PUBLIC_API, expect.any(Object))
    expect(policy).toEqual({ enabled: true, daily_quota: 10, remaining_today: 7 })
  })
})

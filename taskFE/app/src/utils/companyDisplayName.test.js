// @vitest-environment node
import { describe, expect, it, vi } from 'vitest'
import {
  fetchTenantCompanyDisplayName,
  pickCompanyDisplayName,
} from './companyDisplayName.js'

describe('pickCompanyDisplayName', () => {
  it('returns trimmed company name from payload', () => {
    expect(pickCompanyDisplayName({ name: '  测试公司  ' })).toBe('测试公司')
  })

  it('returns empty string when name is missing or blank', () => {
    expect(pickCompanyDisplayName(null)).toBe('')
    expect(pickCompanyDisplayName({})).toBe('')
    expect(pickCompanyDisplayName({ name: '   ' })).toBe('')
  })
})

describe('fetchTenantCompanyDisplayName', () => {
  it('does not fetch when tenant id is empty', async () => {
    const apiFetch = vi.fn()
    await expect(fetchTenantCompanyDisplayName(apiFetch, '')).resolves.toBe('')
    expect(apiFetch).not.toHaveBeenCalled()
  })

  it('returns name from companies/current', async () => {
    const apiFetch = vi.fn(async () => ({
      ok: true,
      json: async () => ({ name: 'Acme Corp', member_is_admin: true }),
    }))
    await expect(
      fetchTenantCompanyDisplayName(apiFetch, '881024523581812736'),
    ).resolves.toBe('Acme Corp')
    expect(apiFetch).toHaveBeenCalledWith(
      '/api/tenant/881024523581812736/accounts/companies/current/',
      expect.objectContaining({
        credentials: 'include',
      }),
    )
  })

  it('returns empty string when companies/current is not ok', async () => {
    const apiFetch = vi.fn(async () => ({
      ok: false,
      status: 404,
      json: async () => ({ detail: 'company not found' }),
    }))
    await expect(
      fetchTenantCompanyDisplayName(apiFetch, '881024523581812736'),
    ).resolves.toBe('')
  })

  it('returns empty string when fetch throws', async () => {
    const apiFetch = vi.fn(async () => {
      throw new Error('network down')
    })
    await expect(
      fetchTenantCompanyDisplayName(apiFetch, '881024523581812736'),
    ).resolves.toBe('')
  })
})

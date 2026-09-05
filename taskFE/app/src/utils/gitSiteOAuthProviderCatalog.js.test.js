import { describe, it, expect } from 'vitest'
import {
  normalizeProviderCatalog,
  buildProviderCatalogRequest,
} from './gitSiteOAuthProviderCatalog.js'

describe('normalizeProviderCatalog', () => {
  it('normalizes rows and drops invalid', () => {
    const out = normalizeProviderCatalog([
      { provider: 'GitLab', service_provider: 'tenant-1', label: 'Mine', website: 'https://g.example' },
      { provider: '', service_provider: 'x' },
    ])
    expect(out).toHaveLength(1)
    expect(out[0].provider_key).toBe('gitlab:tenant-1')
    expect(out[0].website).toBe('https://g.example')
  })
})

describe('buildProviderCatalogRequest', () => {
  it('adds company_id and tenant header when present', () => {
    const { path, headers } = buildProviderCatalogRequest('850256677331562496')
    expect(path).toContain('company_id=850256677331562496')
    expect(headers['X-Tenant-Id']).toBe('850256677331562496')
  })

  it('omits query when no company', () => {
    const { path, headers } = buildProviderCatalogRequest('')
    expect(path).toBe('/api/git-oauth/providers/')
    expect(headers['X-Tenant-Id']).toBeUndefined()
  })
})

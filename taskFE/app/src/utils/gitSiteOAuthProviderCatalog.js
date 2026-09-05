/**
 * Git OAuth provider catalog helpers for UserGitSiteOAuthSettings.
 */

export function normalizeProviderCatalog(rows = []) {
  if (!Array.isArray(rows)) return []
  return rows
    .map((row) => {
      const provider = String(row?.provider || '').trim().toLowerCase()
      const serviceProvider = String(row?.service_provider || 'default').trim().toLowerCase() || 'default'
      const providerKey = String(row?.provider_key || `${provider}:${serviceProvider}`).trim().toLowerCase()
      const label = String(row?.label || '').trim() || providerKey
      if (!provider || !providerKey) return null
      return {
        provider,
        service_provider: serviceProvider,
        provider_key: providerKey,
        website: String(row?.website || '').trim(),
        clientId: String(row?.client_id || '').trim(),
        label,
      }
    })
    .filter(Boolean)
}

/** Build providers catalog URL + headers, optionally scoped to a company/tenant. */
export function buildProviderCatalogRequest(companyId) {
  const headers = { Accept: 'application/json' }
  const qs = new URLSearchParams()
  const cid = String(companyId || '').trim()
  if (cid) {
    qs.set('company_id', cid)
    headers['X-Tenant-Id'] = cid
  }
  const path = qs.toString()
    ? `/api/git-oauth/providers/?${qs.toString()}`
    : '/api/git-oauth/providers/'
  return { path, headers }
}

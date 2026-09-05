import { buildProviderCatalogRequest } from './gitSiteOAuthProviderCatalog.js'

export const GENERIC_REPO_BRANCHES_WITHOUT_CREDENTIALS_ERROR =
  'Branches cannot be retrieved from generic Git repositories without credentials'

let cachedCatalog = null
let cachedCatalogCompanyId = ''
let catalogLoadPromise = null

const parseHttpRepoUrl = (repoUrl) => {
  const raw = String(repoUrl || '').trim()
  if (!raw) return null
  try {
    const parsed = new URL(raw)
    if (!parsed.protocol || !parsed.hostname) return null
    if (parsed.protocol !== 'http:' && parsed.protocol !== 'https:') return null
    return parsed
  } catch {
    return null
  }
}

const parseSshRepoHost = (repoUrl) => {
  const raw = String(repoUrl || '').trim()
  if (!raw) return ''

  const scpLikeMatch = raw.match(/^[^@]+@([^:]+):.+$/)
  if (scpLikeMatch?.[1]) {
    return String(scpLikeMatch[1]).toLowerCase()
  }

  try {
    const parsed = new URL(raw)
    if (parsed.protocol === 'ssh:' && parsed.hostname) {
      return String(parsed.hostname).toLowerCase()
    }
  } catch {
    // ignore
  }

  return ''
}

const resolveRepoHost = (repoUrl) => {
  const parsed = parseHttpRepoUrl(repoUrl)
  if (parsed?.hostname) return String(parsed.hostname).toLowerCase()
  return parseSshRepoHost(repoUrl)
}

export const resolveRepoOrigin = (repoUrl) => {
  const parsed = parseHttpRepoUrl(repoUrl)
  if (!parsed) return ''
  return `${parsed.protocol}//${parsed.host}`.toLowerCase()
}

const normalizeCatalogOrigin = (website) => {
  const raw = String(website || '').trim()
  if (!raw) return ''
  try {
    const parsed = new URL(raw)
    if (!parsed.protocol || !parsed.host) return ''
    return `${parsed.protocol}//${parsed.host}`.toLowerCase()
  } catch {
    return ''
  }
}

export const normalizeGitOAuthProviderCatalog = (rows = []) => {
  if (!Array.isArray(rows)) return []
  return rows
    .map((row) => {
      const provider = String(row?.provider || '').trim().toLowerCase()
      const website = String(row?.website || '').trim()
      if (!provider || !website) return null
      return {
        provider,
        service_provider: String(row?.service_provider || 'default').trim().toLowerCase() || 'default',
        provider_key: String(row?.provider_key || '').trim().toLowerCase(),
        website,
        website_origin: normalizeCatalogOrigin(website),
      }
    })
    .filter(Boolean)
}

export const setGitOAuthProviderCatalogForTests = (catalog = null) => {
  cachedCatalog = catalog
  cachedCatalogCompanyId = ''
  catalogLoadPromise = null
}

export const getGitOAuthProviderCatalog = () => cachedCatalog

export const loadGitOAuthProviderCatalog = async (fetchFn, companyId = '') => {
  const cid = String(companyId || '').trim()
  if (catalogLoadPromise && cachedCatalogCompanyId === cid) return catalogLoadPromise
  if (cachedCatalog && cachedCatalogCompanyId === cid) return cachedCatalog
  const { path, headers } = buildProviderCatalogRequest(cid)
  cachedCatalogCompanyId = cid
  cachedCatalog = null
  catalogLoadPromise = (async () => {
    try {
      const response = await fetchFn(path, {
        credentials: 'include',
        headers,
      })
      const data = await response.json().catch(() => ({}))
      if (!response.ok) {
        cachedCatalog = []
        return cachedCatalog
      }
      cachedCatalog = normalizeGitOAuthProviderCatalog(data?.providers)
    } catch {
      cachedCatalog = []
    } finally {
      catalogLoadPromise = null
    }
    return cachedCatalog
  })()
  return catalogLoadPromise
}

const resolveCatalogEntryByRepoUrl = (repoUrl, catalog = cachedCatalog) => {
  if (!Array.isArray(catalog) || !catalog.length) return null
  const origin = resolveRepoOrigin(repoUrl)
  if (origin) {
    const byOrigin = catalog.find((entry) => entry.website_origin === origin)
    if (byOrigin) return byOrigin
  }
  // SSH / 无 scheme 仓库只有 host：与 catalog website 的 hostname 对齐
  // （例如 git@<gitlab-host>:group/repo ↔ http://<gitlab-host>:8012）
  const host = resolveRepoHost(repoUrl)
  if (!host) return null
  return (
    catalog.find((entry) => {
      const website = String(entry?.website || entry?.website_origin || '').trim()
      if (!website) return false
      try {
        return String(new URL(website).hostname || '').toLowerCase() === host
      } catch {
        return false
      }
    }) || null
  )
}

export const resolveRepoOAuthProviderFromCatalog = (repoUrl, catalog = cachedCatalog) => {
  return resolveCatalogEntryByRepoUrl(repoUrl, catalog)?.provider || ''
}

export const resolveRepoOAuthProviderByHeuristic = (repoUrl) => {
  const host = resolveRepoHost(repoUrl)
  if (!host) return ''
  if (host === 'github.com') return 'github'
  if (host === 'localhost' || host === '127.0.0.1') return 'gitlab'
  if (host.includes('gitlab')) return 'gitlab'
  return ''
}

export const shouldShowRepoOAuthAuthorizeButton = (errorMessage) => {
  const msg = String(errorMessage || '').trim()
  if (!msg) return false
  if (msg.includes(GENERIC_REPO_BRANCHES_WITHOUT_CREDENTIALS_ERROR)) return true
  const oauthHintPatterns = [
    'OAuth 授权',
    'OAuth 令牌',
    'GitLab 会话无效',
    '未检测到可用授权',
    '需要完成 Git',
    '需要 GitLab 授权',
    '重新 OAuth 授权',
    '重新绑定',
    'git oauth not connected',
    '尚未绑定 Git 网站 OAuth',
    '授权已失效',
    'refresh http',
    '无法获取 Git 访问令牌',
  ]
  return oauthHintPatterns.some((pattern) => msg.includes(pattern))
}

export const resolveRepoOAuthProvider = (repoUrl, catalog = cachedCatalog) => {
  const heuristic = resolveRepoOAuthProviderByHeuristic(repoUrl)
  if (heuristic) return heuristic
  return resolveRepoOAuthProviderFromCatalog(repoUrl, catalog)
}

/**
 * 同时解析 provider 和 service_provider：
 * - 优先从 catalog 精确匹配（拿到 service_provider，如 tencent-gitlab）
 * - 兜底 heuristic（只有 provider，service_provider 用 'default'）
 * @returns {{ provider: string, service_provider: string } | null}
 */
export const resolveRepoOAuthProviderInfo = (repoUrl, catalog = cachedCatalog) => {
  // 先查 catalog（origin 精确匹配，再 SSH/host 回退）→ 拿到 provider + service_provider
  const hit = resolveCatalogEntryByRepoUrl(repoUrl, catalog)
  if (hit) return { provider: hit.provider, service_provider: hit.service_provider }
  // 兜底：heuristic（只有 provider，service_provider 用 default）
  const provider = resolveRepoOAuthProviderByHeuristic(repoUrl)
  if (provider) return { provider, service_provider: 'default' }
  return null
}

export const supportsRepoOAuthAuthorize = (repoUrl, catalog = cachedCatalog) =>
  Boolean(resolveRepoOAuthProvider(repoUrl, catalog))

export const resolveRepoOAuthAuthorizeLabel = (repoUrl) => {
  const parsed = parseHttpRepoUrl(repoUrl)
  if (parsed?.hostname) return parsed.hostname
  return parseSshRepoHost(repoUrl) || '仓库站点'
}

export const resolveRepoOAuthAuthorizeUrl = (provider) => {
  if (provider === 'github') return '/api/git-oauth/github-start-from-gateway/'
  if (provider === 'gitlab') return '/api/git-oauth/gitlab-start-from-gateway/'
  return ''
}

/**
 * Real href for starting OAuth for a repo URL (github vs gitlab start API + repo_url).
 * Browser GET with Accept: text/html is 302'd to the provider authorize page.
 * @param {string} repoUrl
 * @param {{ nextPath?: string, grantKind?: string, grantId?: string }} [opts]
 * @returns {string}
 */
export function buildRepoOAuthStartHref(repoUrl, opts = {}) {
  const info = resolveRepoOAuthProviderInfo(repoUrl)
  if (!info) return ''
  const startApiUrl = resolveRepoOAuthAuthorizeUrl(info.provider)
  if (!startApiUrl) return ''
  const params = new URLSearchParams()
  params.set('repo_url', String(repoUrl || '').trim())
  const nextPath = String(opts.nextPath || '').trim()
  if (nextPath) params.set('next', nextPath)
  if (info.service_provider) params.set('service_provider', info.service_provider)
  const grantKind = String(opts.grantKind || '').trim()
  const grantId = String(opts.grantId || '').trim()
  if (grantKind) params.set('grant_kind', grantKind)
  if (grantId) params.set('grant_id', grantId)
  if (!grantKind && !grantId) params.set('grant_kind', 'pending')
  return `${startApiUrl}?${params.toString()}`
}

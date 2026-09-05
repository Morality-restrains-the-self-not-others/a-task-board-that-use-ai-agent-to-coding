import { apiFetch } from './apiUtils.js'

const inflightByKey = new Map()
const cacheByKey = new Map()

const CACHE_TTL_MS = 30_000

function buildCacheKey(tenantId, workspaceId) {
  return `${String(tenantId || '').trim()}:${String(workspaceId || '').trim()}`
}

function normalizePlatforms(payload) {
  if (!payload || typeof payload !== 'object') return []
  if (Array.isArray(payload.platforms)) return payload.platforms
  return []
}

/**
 * 获取工作空间已绑定云平台列表；同页多组件/多 watcher 并发时合并为单次 HTTP 请求。
 */
export async function fetchWorkspaceCloudPlatforms(tenantId, workspaceId, { force = false } = {}) {
  const tid = String(tenantId || '').trim()
  const wid = String(workspaceId || '').trim()
  if (!tid || !wid) return []

  const key = buildCacheKey(tid, wid)

  if (!force) {
    const cached = cacheByKey.get(key)
    if (cached && Date.now() - cached.fetchedAt < CACHE_TTL_MS) {
      return cached.platforms
    }
    const inflight = inflightByKey.get(key)
    if (inflight) return inflight
  } else {
    inflightByKey.delete(key)
    cacheByKey.delete(key)
  }

  const request = (async () => {
    try {
      const response = await apiFetch(`/api/cloud/platforms/tenant_id/${tid}/workspace_id/${wid}/`, {
        credentials: 'include',
        headers: { Accept: 'application/json' },
      })
      const payload = response.ok ? await response.json().catch(() => ({})) : {}
      const platforms = normalizePlatforms(payload)
      cacheByKey.set(key, { platforms, fetchedAt: Date.now() })
      return platforms
    } catch {
      return []
    } finally {
      inflightByKey.delete(key)
    }
  })()

  inflightByKey.set(key, request)
  return request
}

export function invalidateWorkspaceCloudPlatforms(tenantId, workspaceId) {
  const key = buildCacheKey(tenantId, workspaceId)
  inflightByKey.delete(key)
  cacheByKey.delete(key)
}

export function resetWorkspaceCloudPlatformsCacheForTests() {
  inflightByKey.clear()
  cacheByKey.clear()
}

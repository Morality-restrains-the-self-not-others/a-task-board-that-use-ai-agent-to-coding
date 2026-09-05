/**
 * 个人资料页访问令牌列表：状态筛选与客户端分页。
 */

export const ACCESS_TOKEN_STATUS_FILTERS = Object.freeze(['active', 'revoked', 'all'])

export const ACCESS_TOKEN_PAGE_SIZE = 10

/**
 * @param {unknown} value
 * @returns {'active'|'revoked'|'all'}
 */
export function normalizeAccessTokenStatusFilter(value) {
  const v = String(value || '').trim().toLowerCase()
  if (v === 'revoked' || v === 'all') return v
  return 'active'
}

/**
 * @param {Array<{ is_revoked?: boolean, created_at?: string, last_used_at?: string }>} tokens
 * @param {'active'|'revoked'|'all'} statusFilter
 */
export function filterAccessTokensByStatus(tokens, statusFilter) {
  const list = Array.isArray(tokens) ? tokens : []
  const filter = normalizeAccessTokenStatusFilter(statusFilter)
  if (filter === 'active') {
    return list.filter((t) => !t?.is_revoked)
  }
  if (filter === 'revoked') {
    return list.filter((t) => Boolean(t?.is_revoked))
  }
  return list.slice()
}

/**
 * 有效优先，其次按最后使用/创建时间倒序。
 * @param {Array<{ is_revoked?: boolean, created_at?: string, last_used_at?: string }>} tokens
 */
export function sortAccessTokens(tokens) {
  const list = Array.isArray(tokens) ? tokens.slice() : []
  const ts = (t) => {
    const raw = t?.last_used_at || t?.created_at || ''
    const n = Date.parse(raw)
    return Number.isFinite(n) ? n : 0
  }
  list.sort((a, b) => {
    const ar = Boolean(a?.is_revoked)
    const br = Boolean(b?.is_revoked)
    if (ar !== br) return ar ? 1 : -1
    return ts(b) - ts(a)
  })
  return list
}

/**
 * @param {unknown[]} items
 * @param {number} page
 * @param {number} [pageSize]
 */
export function paginateItems(items, page, pageSize = ACCESS_TOKEN_PAGE_SIZE) {
  const list = Array.isArray(items) ? items : []
  const size = Math.max(1, Math.floor(Number(pageSize) || ACCESS_TOKEN_PAGE_SIZE))
  const total = list.length
  const totalPages = Math.max(1, Math.ceil(total / size) || 1)
  const currentPage = Math.min(Math.max(1, Math.floor(Number(page) || 1)), totalPages)
  const start = (currentPage - 1) * size
  return {
    items: list.slice(start, start + size),
    currentPage,
    totalPages,
    total,
    pageSize: size,
  }
}

/**
 * @param {Array<{ is_revoked?: boolean }>} tokens
 */
export function countAccessTokensByStatus(tokens) {
  const list = Array.isArray(tokens) ? tokens : []
  let active = 0
  let revoked = 0
  for (const t of list) {
    if (t?.is_revoked) revoked += 1
    else active += 1
  }
  return { active, revoked, all: list.length }
}

/**
 * @param {Array<{ is_revoked?: boolean, created_at?: string, last_used_at?: string }>} tokens
 * @param {{ statusFilter?: string, page?: number, pageSize?: number }} [options]
 */
export function buildAccessTokenListView(tokens, options = {}) {
  const statusFilter = normalizeAccessTokenStatusFilter(options.statusFilter)
  const filtered = sortAccessTokens(filterAccessTokensByStatus(tokens, statusFilter))
  const page = paginateItems(filtered, options.page, options.pageSize)
  return {
    statusFilter,
    counts: countAccessTokensByStatus(tokens),
    ...page,
  }
}

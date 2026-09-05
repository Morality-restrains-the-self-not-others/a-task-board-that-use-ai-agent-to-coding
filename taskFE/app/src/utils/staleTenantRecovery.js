/**
 * 陈旧租户恢复：清库、删公司或书签残留后，URL/localStorage 仍指向不存在的 company id。
 * 与 Navbar / Sidebar 共用的 lastActiveTenantId 键名保持一致。
 */

export const LAST_TENANT_STORAGE_KEY = 'lastActiveTenantId'

/**
 * 若 localStorage.lastActiveTenantId 等于给定 tenantId，则清除。
 * @param {unknown} tenantId
 * @returns {boolean} 是否执行了清除
 */
export function clearStaleLastActiveTenantId(tenantId) {
  const id = String(tenantId ?? '').trim()
  if (!id) return false
  try {
    const stored = String(localStorage.getItem(LAST_TENANT_STORAGE_KEY) || '').trim()
    if (stored && stored === id) {
      localStorage.removeItem(LAST_TENANT_STORAGE_KEY)
      return true
    }
  } catch (_) {
    /* ignore quota / private mode */
  }
  return false
}

/**
 * 陈旧租户改写时不得把原公司的资源 ID 带到另一家公司。
 * 项目详情/编辑 → 项目列表；workspace / task-detail → 工作面板。
 * 租户级列表与设置路径原样保留。
 * @param {unknown} suffix
 * @returns {string}
 */
export function sanitizeStaleTenantPathSuffix(suffix) {
  let s = String(suffix || '/').trim()
  if (!s.startsWith('/')) s = `/${s}`
  if (!s.endsWith('/')) s = `${s}/`
  if (/^\/projects\/.+/i.test(s)) return '/projects/'
  if (/\/task-detail\//i.test(s) || /^\/workspace\//i.test(s)) return '/work-panel/'
  return s
}

/**
 * 公司不存在（companies/current 404）时的安全落点。
 * - 用户仍有其它公司 → 首个有效公司路径
 * - 无公司 → /onboarding/（创建第一个公司）
 * 陈旧租户改写时剥掉原公司的项目/任务资源 ID（见 sanitizeStaleTenantPathSuffix）。
 *
 * @param {object} [options]
 * @param {Array<{id?: unknown}>|null|undefined} [options.companies]
 * @param {string} [options.targetPathSuffix='/work-panel/'] 租户下相对路径（须以 / 开头）
 * @returns {string}
 */
export function resolveMissingCompanyRedirectPath(options = {}) {
  const companies = Array.isArray(options.companies) ? options.companies : []
  let suffix = String(options.targetPathSuffix || '/work-panel/').trim()
  if (!suffix.startsWith('/')) suffix = `/${suffix}`
  if (!suffix.endsWith('/')) suffix = `${suffix}/`
  suffix = sanitizeStaleTenantPathSuffix(suffix)

  for (const c of companies) {
    const id = String(c?.id ?? '').trim()
    if (id) return `/tenant/${id}${suffix}`
  }
  return '/onboarding/'
}

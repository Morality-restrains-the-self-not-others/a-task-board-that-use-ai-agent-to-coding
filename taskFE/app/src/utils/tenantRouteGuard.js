// 路由级陈旧租户守卫（OPT-20260811-005）
//
// 背景：清库、删公司或书签残留后，URL/localStorage 仍可能指向不存在的 company id。
// 此前仅公司设置页在 companies/current 404 时恢复（TenantCompanySettings.vue），
// WorkPanel / People / Billing* 等其余 /tenant/:tenant/... 页面仍会展示失败态。
// 且后端 /me/ 忽略 tenant_id 参数（auth_users.go buildUserDetailJSON），
// /me/?tenant_id=stale 恒返回用户真实公司列表——URL 残留不会触发登出，但各页各显其错。
//
// 统一守卫：/tenant/:tenant/... 导航时若 URL tenant 不在用户 /me/ companies 列表中
// （或用户已无任何公司），则清 lastActiveTenantId 并按现有公司跳转（无公司→onboarding），
// 保留原子路径（如 /billing/orders/）。guest 路径（people/join、people/invite、
// 带 accessCode 的任务分享）不属于成员范围，跳过守卫。
//
// 性能：/me/ 结果带 30s 模块缓存；同一 tenant 在窗口内重复导航零额外请求。

import { apiFetch } from './apiUtils'
import {
  clearStaleLastActiveTenantId,
  resolveMissingCompanyRedirectPath,
} from './staleTenantRecovery.js'
import { setUserCompanies } from './sharedUserTenantCache.js'

const TTL_MS = 30_000

// companies 缓存：{ companies: Array|null, fetchedAt: number }
// known=false（fetch 失败/401）时不缓存，下次导航重试（fail-open）。
const companiesCache = { companies: null, fetchedAt: 0 }

const GUEST_ROUTE_MARKERS = ['/people/join/', '/people/invite/']
// accessCode 只放行任务分享页；若任意 /tenant/:id/... 带 accessCode 都跳过成员校验，
// 超管/非成员即可停留在项目、工作面板等全部租户链接（Navbar 会复制该 query）。
const ACCESS_CODE_GUEST_MARKERS = ['/task-detail/']

/**
 * guest 路径（URL tenant 可能为用户未加入的公司）是否跳过守卫。
 * @param {object} to - vue-router 导航目标（含 path/params/query）
 * @returns {boolean}
 */
export function isGuestTenantRoute(to) {
  const path = String(to?.path || '')
  if (to?.query && (to.query.accessCode || to.query.access_code)) {
    return ACCESS_CODE_GUEST_MARKERS.some((marker) => path.includes(marker))
  }
  return GUEST_ROUTE_MARKERS.some((marker) => path.includes(marker))
}

/**
 * 提取 /tenant/:tenant 之后的子路径（含尾斜杠语义）。
 * 例：/tenant/123/billing/orders/ → '/billing/orders/'；/tenant/123/ → '/'
 * @param {string} path
 * @returns {string}
 */
export function tenantSubPath(path) {
  const m = String(path || '').match(/^\/tenant\/[^/]+\/(.*)$/)
  return m ? `/${m[1]}` : '/'
}

/**
 * 判断 URL tenant 是否仍在用户公司列表内（30s 缓存）。
 * @param {object} opts
 * @param {string} opts.tenantId
 * @param {() => Promise<Response>} [opts.fetchMe]
 * @returns {Promise<{known: boolean, valid: boolean, companies: Array}>}
 *   known=false 表示无法确知（fetch 失败/401）→ 调用方应放行；valid=false 且 known=true → 陈旧。
 */
export async function checkTenantMembership({ tenantId, fetchMe }) {
  const now = Date.now()
  const cached = companiesCache.companies
  if (cached && now - companiesCache.fetchedAt < TTL_MS) {
    return {
      known: true,
      valid: cached.some((c) => String(c?.id) === String(tenantId)),
      companies: cached,
    }
  }

  let companies = []
  let known = false
  try {
    const resp = await fetchMe()
    if (resp.ok) {
      const me = await resp.json()
      companies = Array.isArray(me.companies) ? me.companies : []
      known = true
    }
  } catch (_) {
    /* fail-open：网络异常不拦截导航 */
  }

  if (known) {
    companiesCache.companies = companies
    companiesCache.fetchedAt = Date.now()
    if (companies.length) setUserCompanies(companies.map((c) => String(c.id)))
  }
  return {
    known,
    valid: known && companies.some((c) => String(c?.id) === String(tenantId)),
    companies,
  }
}

/**
 * 守卫主体：返回重定向路径或 undefined（放行）。
 * @param {object} to - vue-router 导航目标
 * @param {object} [deps] - 依赖注入（便于单测）
 * @param {() => Promise<Response>} [deps.fetchMe]
 * @returns {Promise<string|undefined>}
 */
export async function handleTenantRouteGuard(to, deps = {}) {
  const fetchMe =
    deps.fetchMe ||
    (() =>
      apiFetch('/api/accounts/users/me/', {
        credentials: 'include',
        headers: { Accept: 'application/json' },
      }))

  const tenantId = to?.params?.tenant
  const path = String(to?.path || '')
  if (!tenantId || !path.startsWith('/tenant/')) return undefined
  if (isGuestTenantRoute(to)) return undefined

  const { known, valid, companies } = await checkTenantMembership({ tenantId, fetchMe })
  if (!known || valid) return undefined

  // 陈旧 tenant：清 lastActiveTenantId，按现有公司跳转（无公司 → onboarding）
  clearStaleLastActiveTenantId(tenantId)
  if (!companies.length) return '/onboarding/'
  return resolveMissingCompanyRedirectPath({
    companies,
    targetPathSuffix: tenantSubPath(path),
  })
}

/**
 * 安装到主路由实例（router.js 调用一次）。
 * @param {import('vue-router').Router} router
 * @param {object} [deps]
 */
export function installTenantRouteGuard(router, deps = {}) {
  if (!router || typeof router.beforeEach !== 'function') return
  router.beforeEach(async (to) => handleTenantRouteGuard(to, deps))
}

/**
 * 重置模块级 companies 缓存。仅供单测隔离使用（生产代码不调用）。
 */
export function resetTenantRouteGuardCache() {
  companiesCache.companies = null
  companiesCache.fetchedAt = 0
}

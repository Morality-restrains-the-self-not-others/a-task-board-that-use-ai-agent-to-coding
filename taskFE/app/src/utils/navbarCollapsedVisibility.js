/**
 * 顶部导航栏是否因租户侧栏「控制台导航」收起而隐藏。
 *
 * 收起状态与侧栏共享 localStorage，但只有当前页挂载了租户控制台侧栏
 * （有「控制台导航」恢复入口）时才允许隐藏顶栏。系统管理、首页、定价、
 * 个人资料等没有该入口的页面必须始终显示顶栏。
 *
 * 禁止用 navbarType=system_admin 作为「始终显示」的生产路径：该类型
 * CSS 为 fixed 通栏，会盖住 App 整页侧栏布局。
 */

/**
 * @param {unknown} path
 * @returns {string}
 */
export function normalizeNavbarPath(path) {
  let p = String(path || '')
  const q = p.indexOf('?')
  if (q >= 0) p = p.slice(0, q)
  const h = p.indexOf('#')
  if (h >= 0) p = p.slice(0, h)
  p = p.trim()
  if (p.length > 1) p = p.replace(/\/+$/, '')
  return p || '/'
}

/**
 * 当前路径是否挂载租户控制台侧栏（含「控制台导航」恢复入口）。
 * @param {unknown} path
 * @returns {boolean}
 */
export function hasTenantConsoleNavRestore(path) {
  const p = normalizeNavbarPath(path)
  if (p === '/projects') return true
  const m = p.match(/^\/tenant\/[^/]+(\/.*)?$/)
  if (!m) return false
  const rest = m[1] || '/'
  if (rest.startsWith('/profile')) return false
  if (rest.startsWith('/people/join')) return false
  return true
}

/**
 * @param {{ collapsed?: boolean, navbarType?: string, path?: unknown }} opts
 * @returns {boolean}
 */
export function shouldHideNavbarWhenCollapsed({ collapsed, navbarType, path } = {}) {
  if (!collapsed) return false
  if (navbarType === 'system_admin') return false
  return hasTenantConsoleNavRestore(path)
}

/**
 * v72 遗留：扫描未绑定的自定义访问角色（v75 访问管理仍用于「清理未绑定访问角色」）。
 * 主体授权主路径已迁移至 assignSubjectRoles.js。
 */
import { isSystemTenantRole } from './tenantConsoleNav.js'

/**
 * 扫描未被成员/小组绑定的自定义访问角色（优先 display_name 以「访问·」开头）。
 * @param {object} args
 * @param {Array<{id:string,name:string,display_name?:string,is_system?:boolean}>} args.roles
 * @param {Array<{role_name?:string}>} [args.memberRoles]
 * @param {Array<{role_name?:string}>} [args.groupRoles]
 * @returns {Array<{id:string,name:string,display_name?:string,is_system?:boolean}>}
 */
export function listOrphanCustomAccessRoles({ roles, memberRoles, groupRoles } = {}) {
  const referenced = new Set()
  for (const row of memberRoles || []) {
    const name = row?.role_name
    if (name) referenced.add(String(name))
  }
  for (const row of groupRoles || []) {
    const name = row?.role_name
    if (name) referenced.add(String(name))
  }

  const orphans = (Array.isArray(roles) ? roles : []).filter((r) => {
    if (!r || !r.id || !r.name) return false
    if (r.is_system || isSystemTenantRole(r.name)) return false
    if (referenced.has(r.name)) return false
    return true
  })

  orphans.sort((a, b) => {
    const ap = String(a.display_name || '').startsWith('访问·') ? 0 : 1
    const bp = String(b.display_name || '').startsWith('访问·') ? 0 : 1
    if (ap !== bp) return ap - bp
    return String(a.display_name || a.name).localeCompare(String(b.display_name || b.name), 'zh')
  })
  return orphans
}

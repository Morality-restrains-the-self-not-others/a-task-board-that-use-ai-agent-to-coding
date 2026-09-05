/**
 * PeopleAccess 目录「打开页面 / 定位」链接与勾选解析。
 */
import {
  buildTenantResourceHref,
  permissionCodesToMenuKeys,
  resourceBindingsToGroupKeys,
} from '../domain/auth/tenantConsoleNav.js'

export const PEOPLE_ACCESS_SAVE_ACTIONS = 'people.access.save_actions'

/** 已并入 subject_list / region_matrix 的 operate，目录不再单独勾选。 */
export function isPeopleAccessSaveActionsRegion(key) {
  return String(key || '') === PEOPLE_ACCESS_SAVE_ACTIONS
}

/**
 * 从目录树去掉 people.access.save_actions，避免与 v73 operate 重复。
 * @param {Array<{children?: Array}>} pages
 */
export function omitPeopleAccessSaveActionsRegion(pages) {
  return (Array.isArray(pages) ? pages : []).map((page) => ({
    ...page,
    children: (page.children || []).filter(
      (c) => !isPeopleAccessSaveActionsRegion(c?.group_key),
    ),
  }))
}

/**
 * 访问管理保存：subject_list / region_matrix operate，或存量 save_actions，或 member:manage。
 * @param {{ hasRegionOperate?: Function, hasRegion?: Function, hasPerm: Function }} perms
 * @param {string} companyId
 */
export function hasPeopleAccessWrite(perms, companyId) {
  const cid = String(companyId || '')
  const operate = typeof perms.hasRegionOperate === 'function'
    ? (key) => perms.hasRegionOperate(cid, key)
    : (key) => perms.hasRegion(cid, key)
  return Boolean(
    operate('people.access.subject_list')
    || operate('people.access.region_matrix')
    || operate(PEOPLE_ACCESS_SAVE_ACTIONS)
    || perms.hasPerm(cid, 'member:manage'),
  )
}

/**
 * @param {string} tenantId
 * @param {{ group_key?: string, route_prefix?: string }} page
 */
export function pageJumpHref(tenantId, page) {
  return buildTenantResourceHref(tenantId, {
    groupKey: page?.group_key,
    routePrefix: page?.route_prefix,
  })
}

/**
 * @param {string} tenantId
 * @param {{ route_prefix?: string }} page
 * @param {{ group_key?: string, route_prefix?: string }} region
 */
export function regionJumpHref(tenantId, page, region) {
  return buildTenantResourceHref(tenantId, {
    groupKey: region?.group_key,
    routePrefix: page?.route_prefix || region?.route_prefix,
  })
}

/**
 * 按关键字过滤目录。
 *
 * - 页面名 / group_key 命中 → 保留整页（含全部 children）；
 * - 页面名未命中但某子区域命中 → 仅保留匹配的 children，便于快速定位目标勾选框；
 * - 均未命中 → 整页剔除。
 *
 * @param {Array<{display_name?:string,group_key?:string,children?:Array}>} pages
 * @param {string} filter
 */
export function filterCatalogPages(pages, filter) {
  const q = String(filter || '').trim().toLowerCase()
  const list = omitPeopleAccessSaveActionsRegion(pages)
  if (!q) return list
  const result = []
  for (const page of list) {
    const pageName = String(page.display_name || '').toLowerCase()
    const pageKey = String(page.group_key || '').toLowerCase()
    if (pageName.includes(q) || pageKey.includes(q)) {
      result.push(page)
      continue
    }
    const matchedChildren = (page.children || []).filter(
      (region) =>
        String(region.display_name || '').toLowerCase().includes(q) ||
        String(region.group_key || '').toLowerCase().includes(q),
    )
    if (matchedChildren.length) {
      result.push({ ...page, children: matchedChildren })
    }
  }
  return result
}

/**
 * 目录树 → 全部可勾选 group_key（page + 子 ui_region）。
 * @param {Array<{group_key?:string,children?:Array<{group_key?:string}>}>} pages
 * @returns {string[]}
 */
export function flattenCatalogGroupKeys(pages) {
  const keys = []
  for (const page of pages || []) {
    if (page?.group_key) keys.push(String(page.group_key))
    for (const child of page?.children || []) {
      if (child?.group_key) keys.push(String(child.group_key))
    }
  }
  return keys
}

/**
 * 解析访问管理右侧勾选：tenant_admin 展示为目录全量（与 PDP 全权限一致）。
 * @param {{
 *   roleName?: string,
 *   role?: { name?: string, permissions?: string[] },
 *   boundResourceGroups?: Array<{group_key?:string,kind?:string}>,
 *   catalogPages?: Array,
 * }} opts
 * @returns {string[]}
 */
export function resolveSubjectSelectedGroupKeys(opts = {}) {
  const roleName = String(opts.roleName || opts.role?.name || '').trim()
  if (roleName === 'tenant_admin') {
    return flattenCatalogGroupKeys(opts.catalogPages)
  }
  const fromRg = resourceBindingsToGroupKeys(opts.boundResourceGroups || [], [])
  if (fromRg.length) return fromRg
  return permissionCodesToMenuKeys(opts.role?.permissions || [])
}

/**
 * @param {string[]} selectedKeys
 * @param {{ group_key?: string, children?: Array<{group_key?:string}> }} page
 */
export function isPageFullySelected(selectedKeys, page) {
  const selected = selectedKeys || []
  const children = page?.children || []
  if (!children.length) return selected.includes(page?.group_key)
  return children.every((c) => selected.includes(c.group_key))
}

/**
 * @param {string[]} selectedKeys
 * @param {{ group_key?: string, children?: Array<{group_key?:string}> }} page
 * @param {boolean} checked
 * @returns {string[]}
 */
export function togglePageGroupKeys(selectedKeys, page, checked) {
  const keys = new Set(selectedKeys || [])
  const children = page?.children || []
  if (checked) {
    if (page?.group_key) keys.add(page.group_key)
    for (const c of children) {
      if (c?.group_key) keys.add(c.group_key)
    }
  } else {
    keys.delete(page?.group_key)
    for (const c of children) keys.delete(c?.group_key)
  }
  return [...keys]
}

/**
 * 拉取自定义角色已绑定资源组；系统角色 / 失败返回 []。
 * @param {{ apiFetch: Function, role?: { id?: string, is_system?: boolean }, companyId: string }} opts
 */
export async function fetchRoleBoundResourceGroups(opts = {}) {
  const role = opts.role
  if (!role?.id || role.is_system || typeof opts.apiFetch !== 'function') return []
  try {
    const resp = await opts.apiFetch(
      `/api/auth/roles/role_id/${role.id}/resource-groups/?company_id=${opts.companyId}`,
      { credentials: 'include', headers: { Accept: 'application/json' } },
    )
    if (!resp.ok) return []
    const body = await resp.json()
    return body.resource_groups || []
  } catch {
    return []
  }
}

/**
 * 租户控制台左侧菜单 ↔ 逻辑资源组 / 粗码（SSOT）
 * 权威：docs/superpowers/specs/2026-08-11-rbac-page-resource-group-v72-design.md
 * page group_key 与导航 key 对齐；侧栏优先 page:*，无则回退粗码。
 */

/** @typedef {{ key: string, label: string, section: string, pathSuffix: string, anyOfPerms: string[] }} NavMenuItem */

/** @type {NavMenuItem[]} */
export const TENANT_CONSOLE_NAV = [
  { key: 'nav.projects', label: '项目列表', section: 'nav', pathSuffix: '/projects', anyOfPerms: ['project:view', 'project:manage'] },
  { key: 'nav.work_panel', label: '工作面板', section: 'nav', pathSuffix: '/work-panel', anyOfPerms: ['task:view', 'task:manage'] },
  { key: 'nav.image_market', label: '镜像市场', section: 'nav', pathSuffix: '/image-market', anyOfPerms: ['cloud:view', 'cloud:manage'] },
  { key: 'nav.feedback', label: '意见与建议', section: 'feedback', pathSuffix: '', anyOfPerms: ['feedback:view'] },
  { key: 'people.invite', label: '邀请人', section: 'people', pathSuffix: '/people/invite/', anyOfPerms: ['member:manage'] },
  { key: 'people.manage', label: '管理人员', section: 'people', pathSuffix: '/people/manage/', anyOfPerms: ['member:manage'] },
  { key: 'people.groups', label: '管理分组', section: 'people', pathSuffix: '/people/groups/', anyOfPerms: ['group:manage', 'group-members:manage'] },
  { key: 'people.access', label: '访问管理', section: 'people', pathSuffix: '/people/access/', anyOfPerms: ['member:manage'] },
  { key: 'people.roles', label: '角色管理', section: 'people', pathSuffix: '/people/roles/', anyOfPerms: ['member:manage'] },
  { key: 'settings.company', label: '公司设置', section: 'settings', pathSuffix: '/settings/company/', anyOfPerms: ['company:view', 'company:manage'] },
  { key: 'settings.cloud', label: '云平台绑定', section: 'settings', pathSuffix: '/settings/cloud-platform/', anyOfPerms: ['cloud:manage'] },
  { key: 'settings.gitlab', label: 'GitLab', section: 'settings', pathSuffix: '/settings/gitlab-connection/', anyOfPerms: ['company:manage'] },
  { key: 'settings.task_panel', label: '工作空间管理', section: 'settings', pathSuffix: '/settings/task-panel/', anyOfPerms: ['workspace:manage'] },
  { key: 'settings.feature_params', label: '智能体资源配置', section: 'settings', pathSuffix: '/settings/feature-params/', anyOfPerms: ['cloud:manage'] },
  { key: 'settings.deliverable', label: '交付物体系设置', section: 'settings', pathSuffix: '/deliverable-systems/', anyOfPerms: ['company:manage'] },
  { key: 'settings.status', label: '进度体系设置', section: 'settings', pathSuffix: '/settings/status/', anyOfPerms: ['company:manage'] },
  { key: 'billing.overview', label: '概览', section: 'billing', pathSuffix: '/billing/', anyOfPerms: ['billing:view', 'billing:manage'] },
  { key: 'billing.orders', label: '订单列表', section: 'billing', pathSuffix: '/billing/orders/', anyOfPerms: ['billing:view', 'billing:manage'] },
  { key: 'billing.transactions', label: '交易流水', section: 'billing', pathSuffix: '/billing/transactions/', anyOfPerms: ['billing:view', 'billing:manage'] },
  { key: 'billing.usage', label: '使用明细', section: 'billing', pathSuffix: '/billing/usage/', anyOfPerms: ['billing:view', 'billing:manage'] },
]

export const SYSTEM_TENANT_ROLES = new Set(['tenant_admin', 'member', 'group_admin'])

/**
 * @param {string} menuKey
 * @param {(code: string) => boolean} hasCode
 */
export function canSeeMenuKey(menuKey, hasCode) {
  if (typeof hasCode !== 'function') return false
  // v72：page:<nav.key> 优先
  if (hasCode(`page:${menuKey}`)) return true
  const item = TENANT_CONSOLE_NAV.find((m) => m.key === menuKey)
  if (!item) return false
  // 过渡：无 page 注入时回退粗码
  return item.anyOfPerms.some((p) => hasCode(p))
}

/**
 * @param {string} section
 * @param {(code: string) => boolean} hasCode
 */
export function canSeeSection(section, hasCode) {
  return TENANT_CONSOLE_NAV.some((m) => m.section === section && canSeeMenuKey(m.key, hasCode))
}

/**
 * 勾选菜单 → 权限码并集（写入自定义角色粗码桥接）
 * @param {string[]} menuKeys
 * @returns {string[]}
 */
export function menuKeysToPermissionCodes(menuKeys) {
  const set = new Set()
  for (const key of menuKeys || []) {
    const item = TENANT_CONSOLE_NAV.find((m) => m.key === key)
    if (!item) continue
    if (item.anyOfPerms[0]) set.add(item.anyOfPerms[0])
  }
  return [...set].sort()
}

/**
 * 角色权限码 → 应勾选的菜单 keys（任一 anyOf 命中即勾选）
 * @param {string[]} permCodes
 * @returns {string[]}
 */
export function permissionCodesToMenuKeys(permCodes) {
  const codes = new Set(permCodes || [])
  return TENANT_CONSOLE_NAV.filter((m) => m.anyOfPerms.some((p) => codes.has(p))).map((m) => m.key)
}

/**
 * 勾选的 page/region group_key → 导航 page keys
 * @param {string[]} groupKeys
 * @returns {string[]}
 */
export function groupKeysToMenuKeys(groupKeys) {
  const pages = new Set()
  for (const k of groupKeys || []) {
    const nav = TENANT_CONSOLE_NAV.find((m) => m.key === k || String(k).startsWith(`${m.key}.`))
    if (nav) pages.add(nav.key)
  }
  return [...pages]
}

/**
 * 资源组绑定 / PDP 码 → 勾选的 group_key
 * @param {Array<{group_key:string,kind:string}>} bound
 * @param {string[]} permCodes 含 page:/region:
 */
export function resourceBindingsToGroupKeys(bound, permCodes) {
  const keys = new Set()
  for (const b of bound || []) {
    if (b?.group_key) keys.add(b.group_key)
  }
  for (const c of permCodes || []) {
    if (String(c).startsWith('page:')) keys.add(String(c).slice(5))
    if (String(c).startsWith('region:')) keys.add(String(c).slice(7))
  }
  return [...keys]
}

/**
 * 访问配置角色展示名
 * @param {'member'|'group'} subjectType
 * @param {string} subjectLabel
 */
export function accessRoleDisplayName(subjectType, subjectLabel) {
  const label = String(subjectLabel || '').trim() || '未命名'
  const prefix = subjectType === 'group' ? '访问·组·' : '访问·'
  return `${prefix}${label}`.slice(0, 64)
}

export function isSystemTenantRole(roleName) {
  return SYSTEM_TENANT_ROLES.has(String(roleName || ''))
}

/**
 * 目录/API 可能带 page: / region: 前缀，统一成裸 group_key。
 * @param {string} key
 * @returns {string}
 */
export function normalizeResourceGroupKey(key) {
  const s = String(key || '').trim()
  if (s.startsWith('page:')) return s.slice(5)
  if (s.startsWith('region:')) return s.slice(7)
  return s
}

/**
 * 解析租户控制台 pathSuffix（优先 catalog route_prefix，否则 TENANT_CONSOLE_NAV）。
 * @param {string} groupKey
 * @param {string} [routePrefix]
 * @returns {string} 如 `/settings/cloud-platform/`；无法解析则空串
 */
export function resolveNavPathSuffix(groupKey, routePrefix) {
  const prefix = String(routePrefix || '').trim()
  if (prefix) {
    return prefix.startsWith('/') ? prefix : `/${prefix}`
  }
  const key = normalizeResourceGroupKey(groupKey)
  if (!key) return ''
  const nav = TENANT_CONSOLE_NAV.find((m) => m.key === key || key.startsWith(`${m.key}.`))
  return nav?.pathSuffix || ''
}

/**
 * 访问管理「打开」链接：新标签打开对应页面；区域带 #rg=<group_key> 便于定位。
 * @param {string} tenantId
 * @param {{ groupKey: string, routePrefix?: string, includeRegionHash?: boolean }} opts
 * @returns {string}
 */
export function buildTenantResourceHref(tenantId, opts = {}) {
  const tid = String(tenantId || '').trim()
  const key = normalizeResourceGroupKey(opts.groupKey)
  if (!tid || !key) return ''
  const pathSuffix = resolveNavPathSuffix(key, opts.routePrefix)
  if (!pathSuffix) return ''
  const path = pathSuffix.startsWith('/') ? pathSuffix : `/${pathSuffix}`
  const base = `/tenant/${tid}${path}`
  const includeHash = opts.includeRegionHash !== false
  const isPageKey = TENANT_CONSOLE_NAV.some((m) => m.key === key)
  if (!includeHash || isPageKey) return base
  return `${base}#rg=${encodeURIComponent(key)}`
}

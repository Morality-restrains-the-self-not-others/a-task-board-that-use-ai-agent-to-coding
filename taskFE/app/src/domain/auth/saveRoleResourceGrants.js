/**
 * v75 — 保存自定义角色的 page/region 授予（真源 auth_role_resource_group）。
 * 粗码静默双写供存量 RequirePerm 过渡。
 */
import { groupKeysToMenuKeys, menuKeysToPermissionCodes } from './tenantConsoleNav.js'
import { grantsMapToKeys, grantsMapToList, normalizeGrantsMap } from './resourceGrantEffects.js'

/**
 * @param {object} deps
 * @param {(url:string,init?:RequestInit)=>Promise<Response>} deps.apiFetch
 * @param {string} deps.companyId
 * @param {string} deps.roleId
 * @param {string} [deps.displayName]
 * @param {Record<string,string>|Array} deps.selectedGrants
 */
export async function saveRoleResourceGrants(deps) {
  const { apiFetch, companyId, roleId, displayName, selectedGrants } = deps
  if (!companyId || !roleId) throw new Error('companyId and roleId required')
  const grantsMap = normalizeGrantsMap(selectedGrants || {})
  const groupKeys = grantsMapToKeys(grantsMap)
  const grants = grantsMapToList(grantsMap)
  if (!grants.length) throw new Error('至少勾选一个页面或资源区域')

  const menuKeys = groupKeysToMenuKeys(groupKeys)
  const permissions = menuKeysToPermissionCodes(menuKeys)
  const permPayload = permissions.length ? permissions : ['member:view']

  if (displayName) {
    const putRole = await apiFetch(`/api/auth/roles/role_id/${roleId}/`, {
      method: 'PUT',
      credentials: 'include',
      headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
      body: JSON.stringify({ display_name: displayName, permissions: permPayload }),
    })
    if (!putRole.ok) {
      const err = await putRole.json().catch(() => null)
      throw new Error(err?.detail || err?.message || `更新角色失败 (${putRole.status})`)
    }
  }

  const rgResp = await apiFetch(`/api/auth/roles/role_id/${roleId}/resource-groups/`, {
    method: 'PUT',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
    body: JSON.stringify({ grants }),
  })
  if (!rgResp.ok) {
    const err = await rgResp.json().catch(() => null)
    const e = new Error(err?.detail || err?.message || `绑定资源组失败 (${rgResp.status})`)
    e.traceId = err?.trace_id || err?.traceId
    throw e
  }
  return { roleId, grants }
}

/**
 * @param {object} deps
 * @param {(url:string,init?:RequestInit)=>Promise<Response>} deps.apiFetch
 * @param {string} deps.companyId
 * @param {string} deps.displayName
 */
export async function createTenantRole(deps) {
  const { apiFetch, companyId, displayName } = deps
  const name = String(displayName || '').trim()
  if (!companyId || !name) throw new Error('company_id and display_name required')
  const resp = await apiFetch('/api/auth/roles/', {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
    body: JSON.stringify({ company_id: companyId, display_name: name, permissions: ['member:view'] }),
  })
  if (!resp.ok) {
    const err = await resp.json().catch(() => null)
    throw new Error(err?.detail || err?.message || `创建角色失败 (${resp.status})`)
  }
  return resp.json()
}

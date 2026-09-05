/**
 * v75 — 主体多角色分配（replace-all），不再隐式创建「访问·…」角色。
 */

/**
 * @param {Array<{member_id?:string,group_id?:string,role_name?:string}>} rows
 * @param {'member'|'group'} subjectType
 * @param {string} subjectId
 * @returns {string[]}
 */
export function roleNamesForSubject(rows, subjectType, subjectId) {
  const id = String(subjectId || '')
  const key = subjectType === 'group' ? 'group_id' : 'member_id'
  const names = []
  const seen = new Set()
  for (const row of rows || []) {
    if (String(row?.[key] || '') !== id) continue
    const n = String(row?.role_name || '').trim()
    if (!n || seen.has(n)) continue
    seen.add(n)
    names.push(n)
  }
  return names
}

/**
 * @param {object} deps
 * @param {(url:string,init?:RequestInit)=>Promise<Response>} deps.apiFetch
 * @param {string} deps.companyId
 * @param {'member'|'group'} deps.subjectType
 * @param {string} deps.subjectId
 * @param {string[]} deps.roleNames
 */
export async function assignSubjectRoles(deps) {
  const { apiFetch, companyId, subjectType, subjectId, roleNames, idempotencyKey } = deps
  if (!companyId || !subjectId) {
    throw new Error('companyId and subjectId required')
  }
  const names = Array.isArray(roleNames)
    ? [...new Set(roleNames.map((n) => String(n || '').trim()).filter(Boolean))]
    : []
  const path =
    subjectType === 'group'
      ? `/api/tenant/group-role/company_id/${companyId}/group_id/${subjectId}/`
      : `/api/tenant/member-role/company_id/${companyId}/member_id/${subjectId}/`
  // OPT-20260819-038: 角色分配是写操作，透传 Idempotency-Key 防连点/超时重试双发 PUT
  const headers = { 'Content-Type': 'application/json', Accept: 'application/json' }
  if (idempotencyKey) {
    headers['Idempotency-Key'] = idempotencyKey
  }
  const resp = await apiFetch(path, {
    method: 'PUT',
    credentials: 'include',
    headers,
    body: JSON.stringify({ role_names: names }),
  })
  if (!resp.ok) {
    const err = await resp.json().catch(() => null)
    const e = new Error(err?.detail || err?.message || `分配角色失败 (${resp.status})`)
    e.traceId = err?.trace_id || err?.traceId
    throw e
  }
  const body = await resp.json().catch(() => ({}))
  return { roleNames: body.role_names || names }
}

/**
 * 可分配给主体的角色（系统 member/tenant_admin + 自定义；排除 group_admin 伪角色可选）。
 * @param {Array<{name:string,display_name?:string,is_system?:boolean}>} roles
 */
export function assignableRoles(roles) {
  return (Array.isArray(roles) ? roles : []).filter((r) => {
    if (!r?.name) return false
    if (r.name === 'group_admin') return false
    return true
  })
}

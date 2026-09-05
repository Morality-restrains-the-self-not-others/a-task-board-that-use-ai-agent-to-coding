/**
 * PeopleAccess 列表加载与主体授予解析（削行：从 PeopleAccess.vue 抽出）
 */
import { apiFetch } from '../utils/apiUtils'
import { fetchRoleBoundResourceGroups, resolveSubjectSelectedGroupKeys } from './peopleAccessCatalog.js'
import { EFFECT_OPERATE, normalizeGrantsMap } from '../domain/auth/resourceGrantEffects.js'
import { safeResponseJson } from '../utils/safeResponseJson.js'
import { extractTraceId } from '../utils/traceId.js'

export async function loadPeopleAccessLists({ tenantId, state }) {
  state.listLoading.value = true
  state.loadError.value = ''
  state.loadErrorTraceId.value = ''
  try {
    const cid = tenantId
    const [memResp, grpResp, mrResp, grResp, rolesResp, catResp] = await Promise.all([
      apiFetch(`/api/tenant/${cid}/accounts/members/company_members/`, {
        credentials: 'include',
        headers: { Accept: 'application/json' },
      }),
      apiFetch(`/api/tenant/${cid}/accounts/groups/`, {
        credentials: 'include',
        headers: { Accept: 'application/json' },
      }),
      apiFetch(`/api/tenant/member-role/company_id/${cid}/`, {
        credentials: 'include',
        headers: { Accept: 'application/json' },
      }),
      apiFetch(`/api/tenant/group-role/company_id/${cid}/`, {
        credentials: 'include',
        headers: { Accept: 'application/json' },
      }),
      apiFetch(`/api/auth/roles/company_id/${cid}/`, {
        credentials: 'include',
        headers: { Accept: 'application/json' },
      }),
      apiFetch(`/api/auth/resource-groups/?company_id=${cid}`, {
        credentials: 'include',
        headers: { Accept: 'application/json' },
      }),
    ])

    if (memResp.ok) {
      const body = await memResp.json()
      state.members.value = Array.isArray(body) ? body : body.members || body.data || []
    }
    if (grpResp.ok) {
      const body = await grpResp.json()
      state.groups.value = Array.isArray(body) ? body : body.groups || body.data || []
    }
    if (mrResp.ok) {
      const body = await mrResp.json()
      state.memberRoles.value = Array.isArray(body) ? body : []
    }
    if (grResp.ok) {
      const body = await grResp.json()
      state.groupRoles.value = Array.isArray(body) ? body : []
    }
    if (rolesResp.ok) {
      const body = await rolesResp.json()
      state.roles.value = Array.isArray(body) ? body : []
    }
    if (catResp.ok) {
      const body = await catResp.json()
      state.catalogPages.value = Array.isArray(body.pages) ? body.pages : []
    } else {
      const { data: errBody, traceId } = await safeResponseJson(catResp, { fallback: {} })
      state.loadErrorTraceId.value =
        traceId || extractTraceId(errBody) || extractTraceId(catResp) || ''
      const detail = String(errBody?.detail || errBody?.message || errBody?.error || '').trim()
      if (catResp.status === 403) {
        state.loadError.value =
          detail || '资源组目录加载失败：权限不足（需访问管理区域或 member:manage）'
      } else if (catResp.status === 500 || catResp.status === 503) {
        state.loadError.value =
          detail ||
          '资源组目录加载失败（请确认已执行 dataMigrate/taskAuth/032_logical_resource_groups.sql）'
      } else {
        state.loadError.value = detail || `资源组目录加载失败（HTTP ${catResp.status}）`
      }
    }
  } catch (e) {
    state.loadError.value = e?.message || '加载失败'
    state.loadErrorTraceId.value = e?.traceId || e?.trace_id || extractTraceId(e) || ''
  } finally {
    state.listLoading.value = false
  }
}

export async function resolveSubjectGrants({ row, roles, catalogPages, companyId }) {
  const role = roles.find((r) => r.name === row.roleName)
  const bound =
    String(row.roleName || role?.name || '') === 'tenant_admin'
      ? []
      : await fetchRoleBoundResourceGroups({
          apiFetch,
          role,
          companyId,
        })
  if (bound?.length) {
    return normalizeGrantsMap(bound)
  }
  const keys = resolveSubjectSelectedGroupKeys({
    roleName: row.roleName || role?.name,
    role,
    boundResourceGroups: bound,
    catalogPages,
  })
  return normalizeGrantsMap(keys.map((k) => ({ group_key: k, effect: EFFECT_OPERATE })))
}

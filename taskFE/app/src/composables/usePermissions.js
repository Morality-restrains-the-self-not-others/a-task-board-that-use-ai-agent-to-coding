/**
 * usePermissions — v63 RBAC 前端权限判定
 *
 * 数据源:
 *   GET /api/auth/user-permissions/ → { tenant_perms: { cid: ["task:view", ...] } }
 *   GET /api/auth/user-roles/       → { roles: [{ role, level, company_id }] }
 *
 * 用法:
 *   const { hasPerm, hasPlatformPerm, isPlatformRole, loading } = usePermissions()
 *   hasPerm('t1', 'task:manage')          // 租户权限码（含组继承，PDP 已展开）
 *   hasPlatformPerm('tenant:audit')       // 平台权限码（由角色静态映射）
 *   isPlatformRole('super_admin')         // 平台角色判断
 */
import { ref } from 'vue'

let shared = null

export function usePermissions() {
  if (shared) return shared
  shared = createPermissionsStore()
  return shared
}

function createPermissionsStore() {
  const loading = ref(false)
  const loaded = ref(false)
  const tenantPerms = ref({}) // { cid: [codes] }
  const platformRoles = ref([]) // ["super_admin", ...]
  const error = ref(null)

  // 平台角色 → 权限码静态映射（与 shareLib/authz roles.go 一致）
  const PLATFORM_ROLE_PERMS = {
    super_admin: [
      'platform:manage', 'tenant:audit', 'user:impersonate',
      'system:config', 'billing:audit', 'employee:manage',
    ],
    employee: ['tenant:audit', 'user:impersonate', 'billing:audit'],
  }

  async function load(apiFetch) {
    if (loaded.value) return
    loading.value = true
    try {
      const [permsResp, rolesResp] = await Promise.all([
        apiFetch('/api/auth/user-permissions/', {
          credentials: 'include',
          headers: { Accept: 'application/json' },
        }),
        apiFetch('/api/auth/user-roles/', {
          credentials: 'include',
          headers: { Accept: 'application/json' },
        }),
      ])
      if (permsResp.ok) {
        const body = await permsResp.json()
        tenantPerms.value = body?.tenant_perms || {}
      }
      if (rolesResp.ok) {
        const body = await rolesResp.json()
        const roles = body?.roles || []
        platformRoles.value = roles
          .filter((r) => r && r.role)
          .map((r) => r.role)
      }
      loaded.value = true
    } catch (e) {
      error.value = e
    } finally {
      loading.value = false
    }
  }

  function hasPerm(companyId, code) {
    const codes = tenantPerms.value?.[companyId]
    return Array.isArray(codes) && codes.includes(code)
  }

  // hasPage / hasRegion / hasRegionView / hasRegionOperate — v72/v73 logical
  // resource group checks against the injected code set. The PDP emits
  // region:<key> / page:<key> (legacy full) plus effect-qualified
  // region:<key>:view|operate / page:<key>:view|operate (ADR-0004).
  // operate ⊃ view; legacy bare code counts as operate.
  function hasPage(companyId, pageKey) {
    const codes = tenantPerms.value?.[companyId] || []
    return codes.includes(`page:${pageKey}`) ||
      codes.includes(`page:${pageKey}:view`) ||
      codes.includes(`page:${pageKey}:operate`)
  }

  function hasRegion(companyId, regionKey) {
    return hasRegionView(companyId, regionKey)
  }

  function hasRegionView(companyId, regionKey) {
    const codes = tenantPerms.value?.[companyId] || []
    return codes.includes(`region:${regionKey}`) ||
      codes.includes(`region:${regionKey}:view`) ||
      codes.includes(`region:${regionKey}:operate`)
  }

  function hasRegionOperate(companyId, regionKey) {
    const codes = tenantPerms.value?.[companyId] || []
    return codes.includes(`region:${regionKey}:operate`) ||
      codes.includes(`region:${regionKey}`)
  }

  // reload re-fetches the permission/role snapshot after role changes so the
  // current session's sidebar/access state reflects the update (OPT-043 语义).
  async function reload(apiFetch) {
    loaded.value = false
    await load(apiFetch)
  }

  function hasPlatformPerm(code) {
    return platformRoles.value.some((role) => {
      const perms = PLATFORM_ROLE_PERMS[role]
      return Array.isArray(perms) && perms.includes(code)
    })
  }

  function isPlatformRole(role) {
    return platformRoles.value.includes(role)
  }

  return { loading, loaded, error, load, reload, hasPerm, hasPage, hasRegion, hasRegionView, hasRegionOperate, hasPlatformPerm, isPlatformRole }
}

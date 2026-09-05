import { computed, ref } from 'vue'
import { getCookie } from '../utils/cookieUtils.js'
import { apiFetch } from '../utils/apiUtils.js'
import { extractTraceId } from '../utils/traceId.js'
import { activateSavedAccountSession, resolveSwitchHref } from '../domain/auth/services/activate_session_service.js'

/**
 * Projects 列表：租户解析与拉取。
 * @param {{ route: import('vue-router').RouteLocationNormalizedLoaded, router: import('vue-router').Router }} deps
 */
export function useProjectsListLoad({ route, router }) {
  const currentTenant = ref('')
  const projects = ref([])
  const loading = ref(true)
  const error = ref('')
  const errorTraceId = ref('')
  const otherAccounts = ref([]) // 403 时可切换的其他账号
  const accountSwitching = ref(false)

  const getTenantFromRoute = () => route.params.tenant || ''

  const effectiveTenantId = computed(() => getTenantFromRoute() || currentTenant.value)

  const tenantPath = computed(() =>
    (effectiveTenantId.value ? `/tenant/${effectiveTenantId.value}` : ''),
  )

  const sortProjectsByCreatedAtDesc = (list) =>
    [...list].sort((a, b) => {
      const ta = new Date(a.created_at || 0).getTime()
      const tb = new Date(b.created_at || 0).getTime()
      return tb - ta
    })

  const fetchCurrentTenant = async () => {
    try {
      const userId = getCookie('userId')
      if (!userId) {
        console.log('Cookie中不存在userId，跳过API请求')
        return
      }
      const response = await apiFetch(`/api/accounts/users/me/`, {
        method: 'GET',
        headers: { Accept: 'application/json' },
        credentials: 'include',
      })
      if (response.ok) {
        const userData = await response.json()
        if (userData.current_company && userData.current_company.id) {
          currentTenant.value = userData.current_company.id
        } else if (userData.companies && userData.companies.length > 0) {
          currentTenant.value = userData.companies[0].id
        }
      }
    } catch (e) {
      console.error('获取租户信息失败:', e)
    }
  }

  const loadProjects = async () => {
    loading.value = true
    error.value = ''
    errorTraceId.value = ''
    try {
      const workspaceId = route.query.workspace_id || ''
      const tenantId = effectiveTenantId.value
      if (!tenantId) {
        error.value = '缺少租户ID'
        loading.value = false
        return
      }
      let url = `/api/projects/tenant_id/${tenantId}`
      if (workspaceId) url += `?workspace_id=${workspaceId}`
      const response = await apiFetch(url)
      if (response.ok) {
        const raw = await response.json()
        projects.value = Array.isArray(raw) ? sortProjectsByCreatedAtDesc(raw) : []
        return
      }
      if (response.status === 401) {
        error.value = '请先登录'
        try {
          const body = await response.json()
          const apiMsg = (body && (body.message || body.error)) || ''
          if (String(apiMsg).trim()) {
            error.value = String(apiMsg).trim()
          }
        } catch {
          /* keep 请先登录 */
        }
        errorTraceId.value = extractTraceId(response) || ''
        return
      }
      if (response.status === 404 || response.status === 403) {
        await fetchCurrentTenant()
        const fallbackTenant = currentTenant.value
        if (fallbackTenant && String(fallbackTenant) !== String(tenantId)) {
          const query = workspaceId ? { workspace_id: workspaceId } : {}
          await router.replace({ path: `/tenant/${fallbackTenant}/projects/`, query })
          return
        }
        error.value =
          response.status === 404
            ? '租户不存在或无权访问，请从控制台重新进入'
            : '无权访问该租户'
        errorTraceId.value = extractTraceId(response) || ''
        // OPT-20260807-015：多账号槽位已彻底下线，项目列表不再跨账号聚合其他可切换账号。
        otherAccounts.value = []
        return
      }
      error.value = '加载项目失败'
      errorTraceId.value = extractTraceId(response) || ''
    } catch (err) {
      error.value = '网络错误，请稍后重试'
      errorTraceId.value = extractTraceId(err) || ''
    } finally {
      loading.value = false
    }
  }

  const switchToAccount = async (acc) => {
    if (accountSwitching.value) return
    accountSwitching.value = true
    try {
      await activateSavedAccountSession({
        userId: acc.userId,
        token: acc.token,
        apiFetch,
        username: acc.username,
        avatarUrl: acc.avatarUrl,
      })
      window.location.href = resolveSwitchHref(route.fullPath || window.location.pathname, acc.userId)
    } catch (e) {
      console.error('[useProjectsListLoad] switch account failed', e)
      accountSwitching.value = false
    }
  }

  return {
    currentTenant,
    projects,
    loading,
    error,
    errorTraceId,
    otherAccounts,
    accountSwitching,
    switchToAccount,
    getTenantFromRoute,
    effectiveTenantId,
    tenantPath,
    fetchCurrentTenant,
    loadProjects,
  }
}

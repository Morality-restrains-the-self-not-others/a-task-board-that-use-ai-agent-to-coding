<template>
  <ImpersonationBanner />
  <NavbarUI
    :navbarType="navbarType"
    :user="user"
    :currentUser="currentUser"
    :isUserAuthenticated="isUserAuthenticated"
    :tenantPath="tenantPath"
    :route="route"
    :userCompanies="userCompanies"
    :currentTenant="currentTenant"
    :membershipTier="membershipTier"
    :gitResources="gitResources"
    :gitResourcesStatus="gitResourcesStatus"
    @logout="handleLogout"
    @company-switched="switchCompany"
  />
</template>

<script setup>
/* @alias:cmp-navbar-logic */
import { ref, onMounted, onBeforeUnmount, computed, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getCookie, clearCookie } from '../utils/cookieUtils'
import { apiFetch } from '../utils/apiUtils.js'
import { safeJson } from '../utils/safeResponseJson.js'
import { syncUserIdCookieFromProfile, clearStoredUserId, getStoredUserId } from '../utils/sessionUserIdUtils.js'
import { removeSavedAccount, upsertSavedAccount, onAccountStateChanged, getActiveToken } from '../domain/auth/services/saved_accounts_store.js'
import { clearCachedAuthToken } from '../utils/apiUtils.js'
import { setUserCompanies } from '../utils/sharedUserTenantCache.js'
import { handleNavbarLogout } from './navbarHandleLogout.js'
import NavbarUI from './Navbar.ui.vue'
import ImpersonationBanner from './ImpersonationBanner.vue'

const route = useRoute()
const router = useRouter()

// localStorage key: 持久化最后活跃租户 ID，确保非租户页面（如 profile）上
// 「工作面板」链接仍能指向正确租户，而非回退到 /。
const LAST_TENANT_STORAGE_KEY = 'lastActiveTenantId'

// 同步初始化：从 localStorage 读取上次活跃租户，避免首帧渲染时
// workPanelPath 回退到 /，导致用户点击「工作面板」跳转到首页而非工作空间。
const getInitialTenant = () => {
  try { return localStorage.getItem(LAST_TENANT_STORAGE_KEY) || '' } catch (_) { return '' }
}

// 添加响应式变量存储租户ID
const currentTenant = ref(getInitialTenant())

// 修改tenantPath计算属性，优先从路由中获取租户ID
const tenantPath = computed(() => {
  // 优先从路由中获取租户ID，确保导航链接与当前页面使用相同的租户ID
  const routeTenant = route.params.tenant
  if (routeTenant) {
    return '/tenant/' + routeTenant
  }
  // 如果路由中没有租户ID，使用从API获取的租户ID
  const tenant = currentTenant.value
  if (tenant) {
    return '/tenant/' + tenant
  }
  // 最终回退：从 localStorage 读取上次活跃租户 ID，
  // 确保在非租户页面（如 profile）上也能构造正确的工作面板链接
  try {
    const stored = localStorage.getItem(LAST_TENANT_STORAGE_KEY)
    if (stored) return '/tenant/' + stored
  } catch (_) {}
  return ''
})

// 定义组件的属性
const props = defineProps({
  navbarType: {
    type: String,
    default: 'user'
  },
  user: {
    type: Object,
    default: () => ({
      isAuthenticated: false,
      isSuperuser: false,
      username: ''
    })
  }
})

// 定义响应式数据
const userData = ref({
  isAuthenticated: false,
  isSuperuser: false,
  username: '',
  avatarUrl: null,
  userId: '',
})

// 用户所属公司列表（用于公司切换器）
const userCompanies = ref([])

// 当前租户会员等级（'vip1' | 'normal' | '' 未知）— 仅驱动「代码仓库」VIP1 角标
const membershipTier = ref('')

// 当前租户可跳转的 GitLab 区域（购买或赠送），驱动「代码仓库」直链/下拉
const gitResources = ref([])
// unknown=未拉取；ready=成功；error=失败（空列表 fail-open 不跳价格页）
const gitResourcesStatus = ref('unknown')

// 已去掉多账号登录（导航栏不再渲染切换/添加账号入口）；以下保留：
// - accountSwitching：登出后自动切换剩余槽位账号（既有多账号用户）时防重入
// - syncCurrentUserIntoSlot：登录后把当前账号写入槽位，供项目列表跨账号聚合使用
const accountSwitching = ref(false)
let unsubAccountEvents = null // 账号状态事件监听取消函数

const syncCurrentUserIntoSlot = async () => {
  const uid = String(userData.value.userId || getCookie('userId') || '').trim()
  const token = String(await getActiveToken() || '').trim()
  if (!uid || !token || !userData.value.isAuthenticated) return
  try {
    await upsertSavedAccount({
      userId: uid,
      username: userData.value.username || '',
      avatarUrl: userData.value.avatarUrl || null,
      token,
    })
  } catch (e) {
    console.warn('[Navbar] upsertSavedAccount failed', e)
  }
}

/**
 * fetchMembershipTier — 异步拉取当前租户会员等级（/api/tenant/{tid}/billing/membership/），
 * 仅驱动「代码仓库」VIP1 角标。静默失败：等级未知时不显示角标。
 */
async function fetchMembershipTier() {
  const tid = String(currentTenant.value || '').trim()
  if (!tid || !userData.value.isAuthenticated) return
  try {
    const resp = await apiFetch(`/api/tenant/${encodeURIComponent(tid)}/billing/membership/`, {
      headers: { Accept: 'application/json' },
    })
    if (!resp.ok) return
    const body = await safeJson(resp, {})
    const tier = body?.membership?.tier || ''
    if (tier) {
      membershipTier.value = tier
    }
  } catch (e) {
    // 静默失败：保持未知状态
  }
}

/**
 * fetchGitResources — 拉取当前租户已开通/获赠的 GitLab 区域列表。
 * 失败时保持空数组 + status=error（fail-open：入口留在当前页，不跳转价格页）。
 */
async function fetchGitResources() {
  const tid = String(currentTenant.value || '').trim()
  if (!tid || !userData.value.isAuthenticated) {
    gitResources.value = []
    gitResourcesStatus.value = 'ready'
    return
  }
  gitResourcesStatus.value = 'unknown'
  try {
    const resp = await apiFetch(`/api/tenant/${encodeURIComponent(tid)}/billing/gitlab-resources/`, {
      headers: { Accept: 'application/json' },
    })
    if (!resp.ok) {
      gitResourcesStatus.value = 'error'
      console.warn('[Navbar] fetchGitResources failed', { status: resp.status, tenant_id: tid })
      return
    }
    const body = await safeJson(resp, {})
    const list = Array.isArray(body?.resources) ? body.resources : []
    gitResources.value = list
      .map((r) => ({
        region: String(r?.region || ''),
        region_name: String(r?.region_name || r?.region || ''),
        gitlab_web_url: String(r?.gitlab_web_url || '').trim(),
        provisioning_status: String(r?.provisioning_status || ''),
      }))
      .filter((r) => r.region)
    gitResourcesStatus.value = 'ready'
  } catch (e) {
    gitResourcesStatus.value = 'error'
    console.warn('[Navbar] fetchGitResources error', e)
  }
}

const setLoggedOutUser = () => {
  currentTenant.value = ''
  membershipTier.value = ''
  gitResources.value = []
  gitResourcesStatus.value = 'unknown'
  userCompanies.value = []
  userData.value = {
    isAuthenticated: false,
    isSuperuser: false,
    username: '',
    avatarUrl: null,
    userId: '',
  }
  if (typeof window !== 'undefined') {
    delete window.currentUser
  }
}

/** /me/ 鉴权失败（401/403/404）：清本地态并探测 profile 以触发服务端清 HttpOnly cookie。 */
const isAuthFailureStatus = (status) => [401, 403, 404].includes(Number(status))

const clearStaleSessionAfterAuthFailure = async () => {
  clearCachedAuthToken()
  clearStoredUserId()
  setLoggedOutUser()
  // OPT-20260807-006：profile 401 会 Set-Cookie Max-Age=-1 清残留 HttpOnly userId/token。
  // OPT-20260810-015 后有 localStorage 时不再默认打 profile，清库后须显式补一次。
  try {
    await apiFetch('/api/accounts/users/profile/', {
      method: 'GET',
      headers: {
        Accept: 'application/json',
        'X-Requested-With': 'XMLHttpRequest',
      },
    })
  } catch (_) {
    /* 探测失败不阻断登出 UI */
  }
}

const applyMePayload = (userInfo, userId = '') => {
  const emailLoginMethod = userInfo.login_methods?.find((method) => method.method_type === 'email')
  const username = userInfo.username || emailLoginMethod?.identifier || ''

  userData.value = {
    isAuthenticated: true,
    isSuperuser: userInfo.is_superuser === true || userInfo.is_superuser === 'True',
    // v63 RBAC: 平台角色（super_admin/employee）→ 显示系统管理入口
    isPlatformStaff: (userInfo.platform_roles || []).some((r) => ['super_admin', 'employee'].includes(r)),
    username,
    avatarUrl: userInfo.avatar_url || null,
    userId: String(userId || userData.value.userId || getCookie('userId') || '').trim(),
  }

  const routeTenant = route.params.tenant
  const companiesList = Array.isArray(userInfo.companies) ? userInfo.companies : []
  if (routeTenant && companiesList.some(c => String(c.id) === routeTenant)) {
    currentTenant.value = routeTenant
  } else if (userInfo.current_company && userInfo.current_company.id) {
    const cid = String(userInfo.current_company.id)
    // 确保 current_company 在 companies 列表中，防止 Navbar 生成无权限租户链接
    if (companiesList.some(c => String(c.id) === cid)) {
      currentTenant.value = cid
    } else if (companiesList.length > 0) {
      currentTenant.value = String(companiesList[0].id)
    } else {
      currentTenant.value = ''
    }
  } else if (companiesList.length > 0) {
    currentTenant.value = String(companiesList[0].id)
  } else {
    currentTenant.value = ''
  }

  // Persist last active tenant for cross-page fallback (e.g. profile → work panel)。
  // 无可用公司时必须清除：清库后 companies=[] 若保留陈旧 id，Navbar 会链到已删除租户。
  if (currentTenant.value) {
    try { localStorage.setItem(LAST_TENANT_STORAGE_KEY, currentTenant.value) } catch (_) {}
  } else {
    try { localStorage.removeItem(LAST_TENANT_STORAGE_KEY) } catch (_) {}
  }

  // 填充公司列表供切换器使用
  userCompanies.value = (userInfo.companies || []).map((c) => ({
    id: String(c.id),
    name: c.name || '',
  }))

  if (typeof window !== 'undefined') {
    window.currentUser = userData.value
  }
  setUserCompanies(userCompanies.value.map(c => c.id))
  syncCurrentUserIntoSlot().catch(() => {})
  fetchMembershipTier().catch(() => {})
  fetchGitResources().catch(() => {})
}

// AUTH_ROUTE_NAMES 仅用于「登录/注册 ↔ 业务页」穿越时触发重新拉会话（见下方 watch），
// 不再在这些路由上强制 setLoggedOutUser：已登录用户打开 /auth/login/ 时仍须展示昵称等导航。
// people_join 故意不列入：邀请页始终拉用户态；未登录时 profile 401 → setLoggedOutUser。
const AUTH_ROUTE_NAMES = new Set(['login', 'auth_login', 'register', 'auth_register'])

const fetchCurrentUser = async () => {
  try {
    // 优先 localStorage（getStoredUserId），HttpOnly cookie 场景下 JS 读不到
    // userId cookie，依赖 profile 回退会多一次请求；localStorage 有 id 时直接走 /me/。
    const userIdFromStorage = String(getStoredUserId() || '').trim()
    if (userIdFromStorage) {
      const tenantParam = route.params.tenant ? `?tenant_id=${encodeURIComponent(route.params.tenant)}` : ''
      const meResponse = await apiFetch(`/api/accounts/users/me/${tenantParam}`, {
        method: 'GET',
        headers: {
          Accept: 'application/json',
          'X-Requested-With': 'XMLHttpRequest'
        }
      })

      if (meResponse.ok) {
        applyMePayload(await safeJson(meResponse, {}), userIdFromStorage)
        await refreshPlatformStaffFlag()
        return
      }

      // 清库/会话失效：不得用 localStorage 残留 id 假装已登录（登录页会误显账号切换器）
      if (isAuthFailureStatus(meResponse.status)) {
        await clearStaleSessionAfterAuthFailure()
        return
      }

      // 非鉴权失败（如瞬时 5xx）：不保留假登录态；与 catch 路径一致
      setLoggedOutUser()
      return
    }

    const profileResponse = await apiFetch('/api/accounts/users/profile/', {
      method: 'GET',
      headers: {
        Accept: 'application/json',
        'X-Requested-With': 'XMLHttpRequest'
      }
    })

    if (!profileResponse.ok) {
      console.log('[Navbar] profile 未认证，设置用户为未登录状态')
      setLoggedOutUser()
      return
    }

    const profile = await safeJson(profileResponse, {})
    syncUserIdCookieFromProfile(profile)
    const userId = profile?.user_id ? String(profile.user_id) : getStoredUserId()
    if (!userId) {
      userData.value = {
        isAuthenticated: true,
        isSuperuser: false,
        username: profile.personal_nickname || profile.email || '',
        avatarUrl: profile.avatar_url || null,
        userId: '',
      }
      if (typeof window !== 'undefined') {
        window.currentUser = userData.value
      }
      return
    }

    const tenantParam = route.params.tenant ? `?tenant_id=${encodeURIComponent(route.params.tenant)}` : ''
    const meResponse = await apiFetch(`/api/accounts/users/me/${tenantParam}`, {
      method: 'GET',
      headers: {
        Accept: 'application/json',
        'X-Requested-With': 'XMLHttpRequest'
      }
    })

    if (meResponse.ok) {
      applyMePayload(await safeJson(meResponse, {}), userId)
      return
    }

    userData.value = {
      isAuthenticated: true,
      isSuperuser: false,
      username: profile.personal_nickname || profile.email || '',
      avatarUrl: profile.avatar_url || null,
      userId,
    }
    if (typeof window !== 'undefined') {
      window.currentUser = userData.value
    }
    // 同上：/me/ 失败兜底仍尝试平台角色刷新
    await refreshPlatformStaffFlag()
  } catch (error) {
    console.error('Error fetching current user:', error)
    setLoggedOutUser()
  }
}

// 公司切换：菜单项已是真实 <a href>（buildCompanySwitchHref），此处只持久化活跃租户。
// 禁止再写 window.location.href：否则 Ctrl/中键新标签时当前页也会被拽走。
const switchCompany = (companyId) => {
  const id = String(companyId).trim()
  if (!id) return
  try { localStorage.setItem(LAST_TENANT_STORAGE_KEY, id) } catch (_) {}
}

// 从全局变量获取用户信息，并支持动态更新
const currentUser = computed(() => {
  // 优先使用组件内的userData
  if (userData.value.isAuthenticated) {
    return userData.value
  }
  // 然后检查全局变量
  if (typeof window !== 'undefined' && window.currentUser) {
    return window.currentUser
  }
  // 最后检查props传递的用户信息
  return props.user
})

// 简化的认证状态检查，确保未登录状态下只显示登录按钮
const isUserAuthenticated = computed(() => {
  const user = currentUser.value
  // 只有当isAuthenticated明确为true时，才认为用户已登录
  return Boolean(user && user.isAuthenticated === true)
})

// 组件挂载时获取租户ID和用户信息
onMounted(async () => {
  await fetchCurrentUser()

  // 跨 Tab 账号同步：本机账号槽 storage 事件（其他 Tab 写入 savedAccounts /
  // activeAccountUserId 时触发）
  unsubAccountEvents = onAccountStateChanged(() => {
    if (!accountSwitching.value) {
      fetchCurrentUser()
    }
  })
})

// 监听路由变化，仅在穿越认证边界（登录页 ↔ 业务页）时重新获取用户信息，
// 避免每次页面导航都触发 /me/ 请求。
watch(() => route.name, async (newName, oldName) => {
  const wasAuth = oldName && AUTH_ROUTE_NAMES.has(oldName)
  const isAuth = newName && AUTH_ROUTE_NAMES.has(newName)
  // 仅在认证页面 ↔ 业务页面穿越时重新获取
  if (wasAuth !== isAuth) {
    await fetchCurrentUser()
  }
})

const handleLogout = () =>
  handleNavbarLogout({
    apiFetch,
    getCookie,
    clearCookie,
    clearCachedAuthToken,
    clearStoredUserId,
    removeSavedAccount,
    safeJson,
    userData,
    currentTenant,
    lastTenantStorageKey: LAST_TENANT_STORAGE_KEY,
    router,
  })

onBeforeUnmount(() => {
  if (typeof unsubAccountEvents === 'function') {
    unsubAccountEvents()
    unsubAccountEvents = null
  }
})

/**
 * refreshPlatformStaffFlag — v63 RBAC: 异步拉取平台角色判定系统管理入口。
 * /me/ 响应暂无 platform_roles 时兜底走 /api/auth/user-roles/。
 * 必须位于 script setup 块内：此前声明在 script 结束标签之后被 SFC 编译器丢弃，
 * 调用时抛 ReferenceError（/me/ 成功后导航栏整体失效）。
 */
async function refreshPlatformStaffFlag() {
  try {
    const rolesResp = await apiFetch('/api/auth/user-roles/', {
      credentials: 'include',
      headers: { Accept: 'application/json' },
    })
    if (!rolesResp.ok) return
    const body = await rolesResp.json()
    const roles = (body?.roles || []).map((r) => r?.role).filter(Boolean)
    userData.value = {
      ...userData.value,
      isPlatformStaff: roles.some((r) => ['super_admin', 'employee'].includes(r)),
    }
  } catch (e) {
    // 静默失败：保持 /me/ 推导值
  }
}
</script>

/**
 * 系统管理路由守卫：仅 is_superuser 可进入；否则跳转当前租户工作面板。
 */

const PROFILE_PATH = '/api/accounts/users/profile/'

export function isSystemSuperuser(userInfo) {
  return userInfo?.is_superuser === true || userInfo?.is_superuser === 'True'
}

/**
 * @param {object|null|undefined} userInfo /me/ 载荷
 * @returns {string} 工作面板路径或首页
 */
export function resolveWorkPanelPathFromUser(userInfo) {
  const fromCurrent = userInfo?.current_company?.id
  if (fromCurrent != null && String(fromCurrent).trim() !== '') {
    return `/tenant/${String(fromCurrent).trim()}/work-panel/`
  }
  const fromList = userInfo?.companies?.[0]?.id
  if (fromList != null && String(fromList).trim() !== '') {
    return `/tenant/${String(fromList).trim()}/work-panel/`
  }
  // 用户无公司时跳转到 onboarding 引导页创建公司
  // 禁止返回 '/'——会导致登录后跳转到公开首页
  return '/onboarding/'
}

export class SystemAdminRouteGuardService {
  /**
   * @param {{ apiFetch: Function, getCookie: Function }} deps
   */
  constructor({ apiFetch, getCookie }) {
    this._apiFetch = apiFetch
    this._getCookie = getCookie
  }

  /**
   * @returns {Promise<{ allowed: true } | { allowed: false, redirectPath: string }>}
   */
  async resolveAccess() {
    const me = await this._loadMe()
    if (!me) {
      // /me/ API 不可达时保持在当前页面，由页面级错误处理兜底
      return { allowed: false, redirectPath: '/system-admin/' }
    }
    // v63 RBAC: 平台角色判定（/api/auth/user-roles/），is_superuser 字段兜底过渡
    if (isSystemSuperuser(me) || await this._hasPlatformRole('super_admin')) {
      return { allowed: true }
    }
    return { allowed: false, redirectPath: resolveWorkPanelPathFromUser(me) }
  }

  async _hasPlatformRole(roleName) {
    const resp = await this._apiFetch('/api/auth/user-roles/', {
      credentials: 'include',
      headers: { Accept: 'application/json' },
    })
    if (!resp.ok) return false
    const body = await resp.json()
    return (body?.roles || []).some((r) => r && r.role === roleName)
  }

  async _loadMe() {
    let userId = String(this._getCookie('userId') || '').trim()
    if (!userId) {
      userId = await this._resolveUserIdFromProfile()
    }
    if (!userId) {
      return null
    }

    const meResponse = await this._apiFetch(`/api/accounts/users/me/`, {
      credentials: 'include',
      headers: { Accept: 'application/json' },
    })
    if (!meResponse.ok) {
      return null
    }
    return meResponse.json()
  }

  async _resolveUserIdFromProfile() {
    const profileResponse = await this._apiFetch(PROFILE_PATH, {
      credentials: 'include',
      headers: { Accept: 'application/json' },
    })
    if (!profileResponse.ok) {
      return ''
    }
    const profile = await profileResponse.json()
    return String(profile?.user_id ?? profile?.user?.id ?? '').trim()
  }
}

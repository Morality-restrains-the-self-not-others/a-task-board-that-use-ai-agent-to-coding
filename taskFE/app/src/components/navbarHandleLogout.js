/**
 * Navbar 登出：清本地态 → POST logout → SLO / 账号槽切换 / 回登录页。
 * 从 Navbar.logic.vue 抽出以压行数门禁。
 */
export async function handleNavbarLogout({
  apiFetch,
  getCookie,
  clearCookie,
  clearCachedAuthToken,
  clearStoredUserId,
  removeSavedAccount,
  safeJson,
  userData,
  currentTenant,
  lastTenantStorageKey,
  router,
}) {
  const clearLocalAuthState = () => {
    currentTenant.value = ''
    try {
      localStorage.removeItem(lastTenantStorageKey)
    } catch (_) {}
    clearCachedAuthToken()
    clearStoredUserId()
    clearCookie('userId')
    clearCookie('sessionid')
    if (typeof window !== 'undefined') {
      window.currentUser = {
        isAuthenticated: false,
        isSuperuser: false,
        username: '',
        avatarUrl: null,
      }
    }
    userData.value = {
      isAuthenticated: false,
      isSuperuser: false,
      username: '',
      avatarUrl: null,
      userId: '',
    }
  }

  const currentUid = String(userData.value.userId || getCookie('userId') || '').trim()

  try {
    const response = await apiFetch('/api/accounts/users/logout/', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-Requested-With': 'XMLHttpRequest',
        Accept: 'application/json',
      },
    })

    if (currentUid) {
      try {
        removeSavedAccount(currentUid)
      } catch (e) {
        console.warn('[Navbar] removeSavedAccount on logout', e)
      }
    }

    if (response.ok) {
      try {
        const data = await safeJson(response, {})
        if (data.slo_redirect_url) {
          clearLocalAuthState()
          const loginUrl = window.location.origin + '/auth/login/'
          const sloUrl = new URL(data.slo_redirect_url)
          sloUrl.searchParams.set('post_logout_redirect_uri', loginUrl)
          window.location.href = sloUrl.toString()
          return
        }
      } catch (_) {
        /* no JSON body */
      }

      // OPT-20260807-015：多账号槽位已彻底下线，登出不再自动切换剩余账号。
      clearLocalAuthState()
      router.push('/auth/login/')
    } else {
      console.warn('Logout API returned non-OK status, clearing local state anyway')
      clearLocalAuthState()
      router.push('/auth/login/')
    }
  } catch (error) {
    console.error('登出失败:', error)
    clearLocalAuthState()
    router.push('/auth/login/')
  }
}

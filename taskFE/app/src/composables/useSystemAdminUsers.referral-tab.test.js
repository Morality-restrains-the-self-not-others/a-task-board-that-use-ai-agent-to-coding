// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] useSystemAdminUsers.referral-tab.test.js requires vitest runtime')
} else {
  const { describe, it, expect, vi, beforeEach } = await import('vitest')

  const hoisted = vi.hoisted(() => ({
    apiFetch: vi.fn(),
    routeQuery: {},
    routerReplace: vi.fn(),
  }))

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch: hoisted.apiFetch,
  }))
  vi.mock('../utils/requestErrorDisplay.js', () => ({
    showRequestError: vi.fn(),
  }))
  vi.mock('vue-router', () => ({
    useRoute: () => ({ query: hoisted.routeQuery }),
    useRouter: () => ({ replace: hoisted.routerReplace }),
  }))

  const { useSystemAdminUsers } = await import('./useSystemAdminUsers.js')

  describe('useSystemAdminUsers referral-apps tab', () => {
    beforeEach(() => {
      hoisted.apiFetch.mockReset()
      hoisted.routerReplace.mockReset()
      hoisted.routeQuery = {}
      hoisted.apiFetch.mockResolvedValue({
        ok: true,
        json: async () => ({ users: [], total: 0 }),
      })
    })

    it('switchTab(referral-apps) does not fetch users list; other tabs do', async () => {
      const { switchTab, activeTab } = useSystemAdminUsers()
      hoisted.apiFetch.mockClear()

      switchTab('referral-apps')
      expect(activeTab.value).toBe('referral-apps')
      expect(hoisted.apiFetch).not.toHaveBeenCalled()

      switchTab('archived')
      expect(activeTab.value).toBe('archived')
      expect(hoisted.apiFetch).toHaveBeenCalled()
      expect(hoisted.apiFetch.mock.calls.some(([url]) => String(url).includes('is_archived=true'))).toBe(true)
    })

    it('switchTab(tenants) does not fetch users list', () => {
      const { switchTab, activeTab } = useSystemAdminUsers()
      hoisted.apiFetch.mockClear()
      switchTab('tenants')
      expect(activeTab.value).toBe('tenants')
      expect(hoisted.apiFetch).not.toHaveBeenCalled()
      expect(hoisted.routerReplace).toHaveBeenCalledWith({ query: { tab: 'tenants' } })
    })

    it('?tab=referral-apps 深链初始化 activeTab 为 referral-apps（OPT-20260819-024）', () => {
      hoisted.routeQuery = { tab: 'referral-apps' }
      const { activeTab } = useSystemAdminUsers()
      expect(activeTab.value).toBe('referral-apps')
    })

    it('?tab=tenants 深链初始化 activeTab 为 tenants', () => {
      hoisted.routeQuery = { tab: 'tenants' }
      const { activeTab, isUserListTab } = useSystemAdminUsers()
      expect(activeTab.value).toBe('tenants')
      expect(isUserListTab.value).toBe(false)
    })

    it('switchTab 同步 tab 到 route.query（OPT-20260819-024）', () => {
      const { switchTab } = useSystemAdminUsers()
      hoisted.routeQuery = { tab: 'active' }
      switchTab('referral-apps')
      expect(hoisted.routerReplace).toHaveBeenCalledWith({ query: { tab: 'referral-apps' } })
      switchTab('archived')
      expect(hoisted.routerReplace).toHaveBeenCalledWith({ query: {} })
    })
  })
}

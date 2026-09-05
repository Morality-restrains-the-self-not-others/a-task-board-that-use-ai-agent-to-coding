// @vitest-environment jsdom
/**
 * SystemAdminUsers：表头第二行列过滤器
 */
if (!process.env.VITEST) {
  console.log('[skip] SystemAdminUsers.columnFilters.test.js requires vitest runtime')
} else {
  const { describe, it, expect, vi, beforeEach, afterEach } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')

  const { apiFetch } = vi.hoisted(() => ({
    apiFetch: vi.fn(),
  }))

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch,
  }))
  vi.mock('../utils/requestErrorDisplay.js', () => ({
    showRequestError: vi.fn(),
  }))
  vi.mock('../utils/safeResponseJson.js', () => ({
    safeResponseJson: async () => ({ data: { invitations: [] }, traceId: '' }),
  }))
  vi.mock('../utils/traceId.js', () => ({
    extractTraceId: () => '',
  }))
  vi.mock('vue-router', () => ({
    useRoute: () => ({ query: {} }),
    useRouter: () => ({ replace: vi.fn() }),
  }))

  vi.mock('../components/SystemAdminUserRechargeDrawer.vue', () => ({
    default: { name: 'SystemAdminUserRechargeDrawer', template: '<div />' },
  }))
  vi.mock('../components/SystemAdminUserKycDrawer.vue', () => ({
    default: { name: 'SystemAdminUserKycDrawer', template: '<div />' },
  }))
  vi.mock('../components/SystemAdminReferralPerformanceDrawer.vue', () => ({
    default: { name: 'SystemAdminReferralPerformanceDrawer', template: '<div />' },
  }))
  vi.mock('../components/SystemAdminReferralApplicationsPanel.vue', () => ({
    default: { name: 'SystemAdminReferralApplicationsPanel', template: '<div />' },
  }))
  vi.mock('../components/SystemAdminEmailInvitationsSection.vue', () => ({
    default: { name: 'SystemAdminEmailInvitationsSection', template: '<div />' },
  }))

  const { default: SystemAdminUsers } = await import('../views/SystemAdminUsers.vue')

  function jsonOk(body) {
    return {
      ok: true,
      json: async () => body,
      headers: { get: () => null },
    }
  }

  function usersListUrls() {
    return apiFetch.mock.calls
      .map((c) => String(c[0] || ''))
      .filter((u) => u.includes('/api/system-admin/users/'))
  }

  async function mountPage() {
    const wrapper = mount(SystemAdminUsers, {
      global: { stubs: { Teleport: true } },
    })
    await flushPromises()
    return wrapper
  }

  describe('SystemAdminUsers 表头列过滤器', () => {
    beforeEach(() => {
      vi.useFakeTimers()
      apiFetch.mockReset()
      apiFetch.mockResolvedValue(jsonOk({
        users: [{
          id: 'u1',
          email: 'a@example.com',
          phone: '',
          login_methods: [],
          is_active: true,
          is_archived: false,
        }],
        total: 1,
      }))
    })

    afterEach(() => {
      vi.useRealTimers()
    })

    it('renders a filter control under each column header', async () => {
      const wrapper = await mountPage()
      const row = wrapper.get('[data-testid="user-list-column-filters"]')
      expect(row.findAll('td').length).toBe(12)
      expect(wrapper.find('[data-testid="user-list-column-filters-reset"]').exists()).toBe(true)
      const headers = wrapper.findAll('thead th')
      expect(headers).toHaveLength(12)
    })

    it('sends email= after typing in the email filter', async () => {
      const wrapper = await mountPage()
      const before = usersListUrls().length
      await wrapper.get('input[aria-label="邮箱"]').setValue('alice@')
      await vi.advanceTimersByTimeAsync(400)
      await flushPromises()
      const urls = usersListUrls()
      expect(urls.length).toBeGreaterThan(before)
      expect(urls[urls.length - 1]).toContain('email=alice')
    })

    it('sends role=tester when role filter is 测试', async () => {
      const wrapper = await mountPage()
      await wrapper.get('select[aria-label="角色"]').setValue('tester')
      await vi.advanceTimersByTimeAsync(400)
      await flushPromises()
      expect(usersListUrls().at(-1)).toContain('role=tester')
    })

    it('sends role=superuser when role filter changes', async () => {
      const wrapper = await mountPage()
      await wrapper.get('select[aria-label="角色"]').setValue('superuser')
      await vi.advanceTimersByTimeAsync(400)
      await flushPromises()
      expect(usersListUrls().at(-1)).toContain('role=superuser')
    })

    it('sends login_method=phone when login method filter changes', async () => {
      const wrapper = await mountPage()
      await wrapper.get('select[aria-label="登录方式"]').setValue('phone')
      await vi.advanceTimersByTimeAsync(400)
      await flushPromises()
      expect(usersListUrls().at(-1)).toContain('login_method=phone')
    })

    it('sends date_joined_from when registration start date changes', async () => {
      const wrapper = await mountPage()
      await wrapper.get('input[aria-label="注册开始日期"]').setValue('2026-08-01')
      await vi.advanceTimersByTimeAsync(400)
      await flushPromises()
      expect(usersListUrls().at(-1)).toContain('date_joined_from=2026-08-01')
    })

    it('sends is_active=false when status is 已禁用', async () => {
      const wrapper = await mountPage()
      await wrapper.get('select[aria-label="状态"]').setValue('false')
      await vi.advanceTimersByTimeAsync(400)
      await flushPromises()
      expect(usersListUrls().at(-1)).toContain('is_active=false')
    })

    it('sends has_profit_sharing=true when qualification is 是', async () => {
      const wrapper = await mountPage()
      await wrapper.get('select[aria-label="是否获得分账资格"]').setValue('true')
      await vi.advanceTimersByTimeAsync(400)
      await flushPromises()
      expect(usersListUrls().at(-1)).toContain('has_profit_sharing=true')
    })

    it('reset clears column filter query params but keeps is_archived', async () => {
      const wrapper = await mountPage()
      await wrapper.get('select[aria-label="角色"]').setValue('superuser')
      await vi.advanceTimersByTimeAsync(400)
      await flushPromises()
      expect(usersListUrls().at(-1)).toContain('role=superuser')

      await wrapper.get('[data-testid="user-list-column-filters-reset"]').trigger('click')
      await flushPromises()
      const url = usersListUrls().at(-1)
      expect(url).toContain('is_archived=false')
      expect(url).not.toContain('role=')
    })

    it('hides the filter row on the referral-apps tab', async () => {
      const wrapper = await mountPage()
      expect(wrapper.find('[data-testid="user-list-column-filters"]').exists()).toBe(true)
      await wrapper.get('[data-testid="system-admin-users-tab-referral-apps"]').trigger('click')
      await flushPromises()
      expect(wrapper.find('[data-testid="user-list-column-filters"]').exists()).toBe(false)
    })
  })
}

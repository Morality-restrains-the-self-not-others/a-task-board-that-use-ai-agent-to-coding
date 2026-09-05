// @vitest-environment jsdom
/**
 * SystemAdminUsers：活跃 Tab 搜索结果含归档用户时展示提示（OPT-20260824-091）。
 *
 * 后端在 q/phone/email 检索时不再套用 is_archived=false，活跃 Tab 能搜到归档占用者。
 * 行上虽有「已归档」徽标，但停留在活跃 Tab 可能让运营误以为筛选坏了，故页面在
 * activeTabHasArchivedResults 为真时展示提示条。
 */
if (!process.env.VITEST) {
  console.log('[skip] SystemAdminUsers.archived-in-active-results.test.js requires vitest runtime')
} else {
  const { describe, it, expect, vi, beforeEach } = await import('vitest')
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

  function userRow(id, overrides = {}) {
    return {
      id,
      email: `${id}@example.com`,
      phone: '',
      login_methods: [],
      is_active: true,
      is_archived: false,
      ...overrides,
    }
  }

  async function mountPage(users) {
    apiFetch.mockResolvedValue(jsonOk({ users, total: users.length }))
    const wrapper = mount(SystemAdminUsers, {
      global: { stubs: { Teleport: true } },
    })
    await flushPromises()
    return wrapper
  }

  beforeEach(() => {
    apiFetch.mockReset()
  })

  describe('SystemAdminUsers 活跃 Tab 命中归档用户提示', () => {
    it('活跃 Tab 搜索结果含归档用户时展示提示条', async () => {
      const wrapper = await mountPage([userRow('u1'), userRow('u2', { is_archived: true })])
      const hint = wrapper.find('[data-testid="archived-in-active-results-hint"]')
      expect(hint.exists()).toBe(true)
      expect(hint.text()).toContain('已归档用户')
    })

    it('活跃 Tab 无归档用户时不展示提示条', async () => {
      const wrapper = await mountPage([userRow('u1'), userRow('u2')])
      expect(wrapper.find('[data-testid="archived-in-active-results-hint"]').exists()).toBe(false)
    })

    it('切换到已归档 Tab 后提示条消失（activeTab 不再是 active）', async () => {
      const wrapper = await mountPage([userRow('u1', { is_archived: true })])
      expect(wrapper.find('[data-testid="archived-in-active-results-hint"]').exists()).toBe(true)
      await wrapper.get('[data-testid="system-admin-users-tab-archived"]').trigger('click')
      await flushPromises()
      expect(wrapper.find('[data-testid="archived-in-active-results-hint"]').exists()).toBe(false)
    })
  })
}

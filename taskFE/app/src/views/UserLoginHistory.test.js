// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] UserLoginHistory.test.js requires vitest runtime')
} else {
const { describe, expect, it, vi } = await import('vitest')
const { mount } = await import('@vue/test-utils')

vi.mock('../utils/apiUtils.js', () => ({
  apiFetch: (...args) => globalThis.__loginHistoryFetch(...args),
}))
vi.mock('../components/UserCenterSidebar.vue', () => ({
  default: { template: '<aside data-testid="user-center-sidebar" />' },
}))
vi.mock('vue-router', () => ({
  useRoute: () => ({ params: {}, query: {} }),
}))

const { default: UserLoginHistory } = await import('./UserLoginHistory.vue')

describe('UserLoginHistory', () => {
  it('loads self login history via /api/auth/login-history/', async () => {
    globalThis.__loginHistoryFetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      headers: { get: () => null },
      json: async () => ({
        results: [{
          id: '9',
          logged_in_at: '2026-08-25T12:00:00.000000Z',
          client_ip: '198.51.100.4',
          entry_label: '管理员入口',
          method_label: '邮箱',
        }],
        total: 1,
      }),
    })
    const wrapper = mount(UserLoginHistory)
    await vi.waitFor(() => expect(wrapper.text()).toContain('198.51.100.4'))
    expect(globalThis.__loginHistoryFetch.mock.calls[0][0]).toMatch(/^\/api\/auth\/login-history\//)
    expect(wrapper.find('[data-testid="user-center-sidebar"]').exists()).toBe(true)
  })
})
}

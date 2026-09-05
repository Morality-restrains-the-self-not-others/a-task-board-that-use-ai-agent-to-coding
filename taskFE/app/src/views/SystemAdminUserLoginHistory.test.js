// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] SystemAdminUserLoginHistory.test.js requires vitest runtime')
} else {
const { describe, expect, it, vi } = await import('vitest')
const { mount } = await import('@vue/test-utils')

vi.mock('../utils/apiUtils.js', () => ({
  apiFetch: (...args) => globalThis.__loginHistoryFetch(...args),
}))
vi.mock('vue-router', () => ({
  useRoute: () => ({ params: { userId: '4242' }, query: {} }),
}))

const { default: SystemAdminUserLoginHistory } = await import('./SystemAdminUserLoginHistory.vue')

describe('SystemAdminUserLoginHistory', () => {
  it('requests admin login-history for the route user id', async () => {
    globalThis.__loginHistoryFetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      headers: { get: () => null },
      json: async () => ({ results: [], total: 0 }),
    })
    const wrapper = mount(SystemAdminUserLoginHistory)
    await vi.waitFor(() => expect(globalThis.__loginHistoryFetch).toHaveBeenCalled())
    expect(globalThis.__loginHistoryFetch.mock.calls[0][0]).toContain('/api/system-admin/users/4242/login-history/')
    expect(wrapper.get('a[href="/system-admin/users/"]').exists()).toBe(true)
  })
})
}

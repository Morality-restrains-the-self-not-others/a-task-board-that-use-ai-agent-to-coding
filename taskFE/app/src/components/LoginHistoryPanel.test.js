// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] LoginHistoryPanel.test.js requires vitest runtime')
} else {
const { describe, expect, it, vi, beforeEach, afterEach } = await import('vitest')
const { mount, flushPromises } = await import('@vue/test-utils')

vi.mock('../utils/apiUtils.js', () => ({
  apiFetch: (...args) => globalThis.__loginHistoryFetch(...args),
}))

const { default: LoginHistoryPanel } = await import('./LoginHistoryPanel.vue')

describe('LoginHistoryPanel', () => {
  beforeEach(() => {
    globalThis.__loginHistoryFetch = vi.fn()
  })

  afterEach(() => {
    document.body.innerHTML = ''
  })

  it('renders IP and entry label from results', async () => {
    globalThis.__loginHistoryFetch.mockResolvedValue({
      ok: true,
      status: 200,
      headers: { get: () => null },
      json: async () => ({
        results: [{
          id: '1',
          logged_in_at: '2026-08-25T12:00:00.000000Z',
          client_ip: '203.0.113.9',
          user_agent: 'TestUA',
          entry: 'customer',
          method_type: 'email',
          entry_label: '用户入口',
          method_label: '邮箱',
        }],
        total: 1,
        limit: 20,
        offset: 0,
      }),
    })
    const wrapper = mount(LoginHistoryPanel, { props: { apiUrl: '/api/auth/login-history/' } })
    await vi.waitFor(() => expect(wrapper.find('[data-testid="login-history-row"]').exists()).toBe(true))
    expect(wrapper.text()).toContain('203.0.113.9')
    expect(wrapper.text()).toContain('用户入口')
    expect(wrapper.text()).toContain('邮箱')
    expect(globalThis.__loginHistoryFetch.mock.calls[0][0]).toContain('/api/auth/login-history/')
  })

  it('shows empty copy when there are no rows', async () => {
    globalThis.__loginHistoryFetch.mockResolvedValue({
      ok: true,
      status: 200,
      headers: { get: () => null },
      json: async () => ({ results: [], total: 0 }),
    })
    const wrapper = mount(LoginHistoryPanel, { props: { apiUrl: '/api/auth/login-history/' } })
    await flushPromises()
    expect(wrapper.get('[data-testid="login-history-empty"]').text()).toBe('暂无登录记录')
  })

  it('default list omits include_failures param', async () => {
    globalThis.__loginHistoryFetch.mockResolvedValue({
      ok: true,
      status: 200,
      headers: { get: () => null },
      json: async () => ({ results: [], total: 0 }),
    })
    const wrapper = mount(LoginHistoryPanel, { props: { apiUrl: '/api/auth/login-history/' } })
    await flushPromises()
    const firstUrl = globalThis.__loginHistoryFetch.mock.calls[0][0]
    expect(firstUrl).toContain('/api/auth/login-history/')
    expect(firstUrl).not.toContain('include_failures')
  })

  it('toggle to failures refetches with include_failures and renders outcome badge', async () => {
    globalThis.__loginHistoryFetch
      .mockResolvedValueOnce({
        ok: true,
        status: 200,
        headers: { get: () => null },
        json: async () => ({ results: [], total: 0 }),
      })
      .mockResolvedValueOnce({
        ok: true,
        status: 200,
        headers: { get: () => null },
        json: async () => ({
          results: [{
            id: '2',
            logged_in_at: '2026-08-25T12:01:00.000000Z',
            client_ip: '203.0.113.10',
            user_agent: 'FailUA',
            entry: 'customer',
            method_type: 'email',
            outcome: 'password_mismatch',
            outcome_label: '密码错误',
            entry_label: '用户入口',
            method_label: '邮箱',
          }],
          total: 1,
          limit: 20,
          offset: 0,
        }),
      })
    const wrapper = mount(LoginHistoryPanel, { props: { apiUrl: '/api/auth/login-history/' } })
    await flushPromises()
    await wrapper.get('[data-testid="login-history-filter-all"]').trigger('click')
    await vi.waitFor(() => expect(wrapper.find('[data-testid="login-history-outcome-failure"]').exists()).toBe(true))
    const secondUrl = globalThis.__loginHistoryFetch.mock.calls[1][0]
    expect(secondUrl).toContain('include_failures=1')
    const badge = wrapper.get('[data-testid="login-history-outcome-failure"]')
    expect(badge.text()).toContain('密码错误')
  })

  it('puts data-traceId on the error node when the request fails', async () => {
    globalThis.__loginHistoryFetch.mockResolvedValue({
      ok: false,
      status: 500,
      headers: { get: (name) => (String(name).toLowerCase() === 'x-trace-id' ? 'hist-fail-01' : null) },
      json: async () => ({ detail: 'db error', trace_id: 'hist-fail-01' }),
    })
    const wrapper = mount(LoginHistoryPanel, { props: { apiUrl: '/api/auth/login-history/' } })
    await vi.waitFor(() => expect(wrapper.find('[data-testid="login-history-error"]').exists()).toBe(true))
    const err = wrapper.get('[data-testid="login-history-error"]')
    expect(err.text()).toContain('db error')
    expect(err.element.getAttribute('data-traceId')).toBe('hist-fail-01')
  })
})
}

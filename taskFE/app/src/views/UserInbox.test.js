// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] UserInbox.test.js requires vitest runtime')
} else {
const { describe, expect, it, vi, beforeEach } = await import('vitest')
const { mount } = await import('@vue/test-utils')

vi.mock('../utils/apiUtils.js', () => ({
  apiFetch: (...args) => globalThis.__inboxFetch(...args),
}))
vi.mock('../components/UserCenterSidebar.vue', () => ({
  default: { template: '<aside />' },
}))
vi.mock('vue-router', () => ({
  useRoute: () => ({ params: {}, query: {} }),
}))

const { default: UserInbox } = await import('./UserInbox.vue')

describe('UserInbox', () => {
  beforeEach(() => {
    globalThis.__inboxFetch = vi.fn()
  })

  it('renders impersonation notice reason', async () => {
    globalThis.__inboxFetch.mockResolvedValue({
      ok: true,
      status: 200,
      headers: { get: () => null },
      json: async () => ({
        results: [{
          id: '1',
          title: '系统管理员以你的身份登录',
          body: 'body',
          reason: '排查线上工单问题',
          created_at: '2026-08-23T00:00:00Z',
        }],
      }),
    })
    const wrapper = mount(UserInbox)
    await vi.waitFor(() => expect(wrapper.text()).toContain('排查线上工单问题'))
    expect(wrapper.find('[data-testid="inbox-message"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="inbox-message-reason"]').text()).toContain('理由：排查线上工单问题')
    expect((wrapper.text().match(/理由：/g) || []).length).toBe(1)
  })

  it('does not repeat reason when body already embeds 理由', async () => {
    const reason = '请填写本次模拟登录的理由。该理由会写入审计并通知被模拟用户。'
    globalThis.__inboxFetch.mockResolvedValue({
      ok: true,
      status: 200,
      headers: { get: () => null },
      json: async () => ({
        results: [{
          id: '2',
          title: '系统管理员以你的身份登录',
          body: `系统管理员（用户 ID：bootstrap-admin）以你的身份登录了本平台。\n理由：${reason}`,
          reason,
          created_at: '2026-08-24T15:22:13Z',
        }],
      }),
    })
    const wrapper = mount(UserInbox)
    await vi.waitFor(() => expect(wrapper.find('[data-testid="inbox-message"]').exists()).toBe(true))
    const card = wrapper.find('[data-testid="inbox-message"]')
    expect(card.text()).toContain('系统管理员（用户 ID：bootstrap-admin）以你的身份登录了本平台。')
    expect((card.text().match(/理由：/g) || []).length).toBe(1)
    expect(card.find('[data-testid="inbox-message-reason"]').text()).toBe(`理由：${reason}`)
  })

  it('shows retry copy with data-traceId on gateway 502 HTML', async () => {
    globalThis.__inboxFetch.mockResolvedValue({
      ok: false,
      status: 502,
      traceId: 'tr-inbox-502',
      headers: { get: () => null },
      _errorData: {
        _rawErrorText: '<html><head><title>502 Bad Gateway</title></head><body><h1>502 Bad Gateway</h1></body></html>',
      },
      json: async () => { throw new Error('body is html') },
    })
    const wrapper = mount(UserInbox)
    await vi.waitFor(() => expect(wrapper.text()).toContain('服务暂时不可用，请稍后重试'))
    const err = wrapper.find('.text-red-600')
    expect(err.exists()).toBe(true)
    expect(err.attributes('data-trace-id')).toBe('tr-inbox-502')
  })
})
}

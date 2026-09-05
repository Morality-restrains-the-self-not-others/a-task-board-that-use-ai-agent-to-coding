// @vitest-environment jsdom
// 注册邀请面板回归：网关 forward-auth 会话失效（无法解析登录凭据）的跳转已统一收口到
// apiFetch（utils/apiUtils.js），面板自身不再自行导航；错误文案正常展示、正常响应渲染数据。
if (!process.env.VITEST) {
  console.log('[skip] SystemAdminRegistrationInvitePanel.test.js requires vitest runtime')
} else {
const { beforeEach, describe, expect, it, vi } = await import('vitest')
const { mount, flushPromises } = await import('@vue/test-utils')

// vi.mock 工厂引用的变量必须经 vi.hoisted() 定义（Vitest 会把工厂 hoist 到模块顶部）。
const hoisted = vi.hoisted(() => ({
  apiFetch: vi.fn(),
}))

vi.mock('../utils/apiUtils.js', () => ({
  apiFetch: (...args) => hoisted.apiFetch(...args),
}))

const { default: Panel } = await import('./SystemAdminRegistrationInvitePanel.vue')

const FORWARD_AUTH_DETAIL = '无法解析登录凭据，请重新登录'

const okPolicyResp = {
  ok: true,
  status: 200,
  headers: { get: () => null },
  json: async () => ({ enabled: true, daily_quota: 10, remaining_today: 3 }),
}

const okRelationsResp = {
  ok: true,
  status: 200,
  headers: { get: () => null },
  json: async () => ({ results: [{ code: 'ABC', status: 'pending' }] }),
}

describe('SystemAdminRegistrationInvitePanel 会话失效', () => {
  beforeEach(() => {
    vi.stubGlobal('location', { pathname: '/system-admin/', search: '', href: '' })
    hoisted.apiFetch.mockReset()
  })

  it('401 会话失效：面板不自行跳转（统一收口 apiFetch），错误文案展示', async () => {
    hoisted.apiFetch.mockResolvedValue({
      ok: false,
      status: 401,
      headers: { get: () => null },
      json: async () => ({ detail: FORWARD_AUTH_DETAIL }),
    })

    const wrapper = mount(Panel)
    await flushPromises()

    expect(window.location.href).toBe('') // 面板自身不再导航（此前旧接线会跳登录）
    expect(wrapper.text()).toContain(FORWARD_AUTH_DETAIL)
  })

  it('正常响应：策略与邀请关系渲染', async () => {
    hoisted.apiFetch.mockImplementation((url) =>
      url.includes('/registration-invite-policy/') ? okPolicyResp : okRelationsResp,
    )

    const wrapper = mount(Panel)
    await flushPromises()

    expect(wrapper.text()).toContain('今日已剩余：3')
    expect(wrapper.text()).toContain('ABC')
  })
})
}

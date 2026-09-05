// @vitest-environment jsdom
// SystemAdmin 回归：网关 forward-auth 会话失效（无法解析登录凭据）的跳转已统一收口到
// apiFetch（utils/apiUtils.js），组件自身不再自行导航；错误文案正常展示、正常响应渲染数据。
if (!process.env.VITEST) {
  console.log('[skip] SystemAdmin.test.js requires vitest runtime')
} else {
const { beforeEach, describe, expect, it, vi } = await import('vitest')
const { mount, flushPromises } = await import('@vue/test-utils')

// vi.mock 工厂引用的变量必须经 vi.hoisted() 定义（Vitest 会把工厂 hoist 到模块顶部）。
const hoisted = vi.hoisted(() => ({
  apiFetch: vi.fn(),
  safeJson: vi.fn(),
}))

vi.mock('../utils/apiUtils.js', () => ({
  apiFetch: (...args) => hoisted.apiFetch(...args),
}))

vi.mock('../utils/safeResponseJson.js', () => ({
  safeResponseJson: (...args) => hoisted.safeJson(...args),
}))

// 子面板行为由其自身测试覆盖（components/SystemAdminRegistrationInvitePanel.test.js）
vi.mock('../components/SystemAdminRegistrationInvitePanel.vue', () => ({
  default: { name: 'InvitePanelStub', template: '<div data-stub="invite-panel" />' },
}))

const { default: SystemAdmin } = await import('./SystemAdmin.vue')

const FORWARD_AUTH_DETAIL = '无法解析登录凭据，请重新登录'

describe('SystemAdmin 网关 forward-auth 会话失效', () => {
  beforeEach(() => {
    vi.stubGlobal('location', { pathname: '/system-admin/', search: '', href: '' })
    hoisted.apiFetch.mockReset()
    hoisted.safeJson.mockReset()
  })

  it('401 会话失效：组件不自行跳转（统一收口 apiFetch），错误文案与 traceId 照常展示', async () => {
    hoisted.apiFetch.mockResolvedValue({ ok: false, status: 401, traceId: 'tid-fa' })
    hoisted.safeJson.mockResolvedValue({
      data: { detail: FORWARD_AUTH_DETAIL },
      error: '',
      traceId: 'tid-fa',
    })

    const wrapper = mount(SystemAdmin)
    await flushPromises()

    expect(window.location.href).toBe('') // 组件自身不再导航（此前旧接线会跳登录）
    expect(wrapper.text()).toContain(FORWARD_AUTH_DETAIL)
    expect(wrapper.find('[data-traceid="tid-fa"]').exists()).toBe(true)
  })

  it('正常响应：统计卡片与系统信息渲染', async () => {
    hoisted.apiFetch.mockResolvedValue({ ok: true, status: 200, traceId: '' })
    hoisted.safeJson.mockResolvedValue({
      data: {
        stats: { total_users: 42, cloud_authorizations: 2, deliverable_systems: 5 },
        system_info: { version: 'v56', status: 'running' },
      },
      error: '',
      traceId: '',
    })

    const wrapper = mount(SystemAdmin)
    await flushPromises()

    expect(wrapper.text()).toContain('42')
    expect(wrapper.text()).toContain('v56')
    expect(wrapper.text()).toContain('正常运行')
  })
})
}

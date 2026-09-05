// @vitest-environment jsdom
// 待处理邀请「状态」列投递状态徽章渲染（taskTenantService delivery_status）：
// delivered=已投递 / queued=投递中 / failed=投递失败(含错误提示) / none=链接分享；
// 老数据无 delivery_status 时回退「待接受」。
if (!process.env.VITEST) {
  console.log('[skip] PendingInvitations.delivery.test.js requires vitest runtime')
} else {
const { describe, expect, it, vi } = await import('vitest')
const { mount, flushPromises } = await import('@vue/test-utils')

// PendingInvitations.vue 依赖 main.js 注入的 window.apiFetch 全局（无 import），
// 单测需手动绑定（与 useBillingUsage/useBillingTransactions 同模式）。
const hoisted = vi.hoisted(() => ({
  apiFetch: vi.fn(),
}))

vi.stubGlobal('apiFetch', hoisted.apiFetch)

const { default: PendingInvitations } = await import('./PendingInvitations.vue')

const okResp = (data) => ({
  ok: true,
  status: 200,
  headers: { get: () => null },
  json: async () => data,
})

const baseInvite = {
  id: 'inv1', token: 'tok1', expires_at: '2026-08-16 12:00:00',
  is_admin: false, created_at: '2026-08-09 10:00:00', is_accepted: false,
  workspace_name: 'ws1', company_member_name: '成员A',
  invite_method: 'email', invite_target: 'a@example.com',
}

const mountList = async (invites) => {
  hoisted.apiFetch.mockResolvedValue(okResp(invites))
  const wrapper = mount(PendingInvitations, { props: { tenantId: 'c1' } })
  await flushPromises()
  return wrapper
}

describe('PendingInvitations 投递状态列', () => {
  it('delivered → 已投递（成功徽章）+ 投递时间展示', async () => {
    const wrapper = await mountList([
      { ...baseInvite, delivery_status: 'delivered', email_sent_at: '2026-08-09 10:32:00' },
    ])
    const badge = wrapper.find('tbody tr td:nth-child(9) span')
    expect(badge.text()).toBe('已投递')
    expect(badge.classes()).toContain('bg-success/10')
    // 投递时间列（第 10 列）：email_sent_at 非空时展示格式化时间
    const deliveryTime = wrapper.find('tbody tr td:nth-child(10)')
    expect(deliveryTime.text()).toContain('2026/08/09')
    expect(deliveryTime.text()).toContain('10:32')
  })

  it('无 email_sent_at（link 渠道/老数据）→ 投递时间列显示 -', async () => {
    const wrapper = await mountList([
      { ...baseInvite, invite_method: 'link', invite_target: '', delivery_status: 'none' },
    ])
    const deliveryTime = wrapper.find('tbody tr td:nth-child(10)')
    expect(deliveryTime.text().trim()).toBe('-')
  })

  it('queued → 投递中（警告徽章）', async () => {
    const wrapper = await mountList([{ ...baseInvite, delivery_status: 'queued' }])
    expect(wrapper.find('tbody tr td:nth-child(9) span').text()).toBe('投递中')
    expect(wrapper.find('tbody tr td:nth-child(9) span').classes()).toContain('bg-warning/10')
  })

  it('failed → 投递失败（危险徽章 + 错误提示 title）', async () => {
    const wrapper = await mountList([
      { ...baseInvite, delivery_status: 'failed', delivery_error: 'SMTP 550 rejected' },
    ])
    const badge = wrapper.find('tbody tr td:nth-child(9) span')
    expect(badge.text()).toBe('投递失败')
    expect(badge.classes()).toContain('bg-danger/10')
    expect(badge.attributes('title')).toBe('SMTP 550 rejected')
  })

  it('none（link 渠道）→ 链接分享（中性徽章）', async () => {
    const wrapper = await mountList([
      { ...baseInvite, invite_method: 'link', invite_target: '', delivery_status: 'none' },
    ])
    const badge = wrapper.find('tbody tr td:nth-child(9) span')
    expect(badge.text()).toBe('链接分享')
    expect(badge.classes()).toContain('bg-gray-100')
  })

  it('skipped_unsubscribed → 已退订，请复制链接', async () => {
    const wrapper = await mountList([
      { ...baseInvite, delivery_status: 'skipped_unsubscribed' },
    ])
    const badge = wrapper.find('tbody tr td:nth-child(9) span')
    expect(badge.text()).toBe('已退订，请复制链接')
  })

  it('无 delivery_status（迁移前老数据）→ 回退「待接受」', async () => {
    const wrapper = await mountList([{ ...baseInvite, delivery_status: undefined }])
    expect(wrapper.find('tbody tr td:nth-child(9) span').text()).toBe('待接受')
    expect(wrapper.find('tbody tr td:nth-child(9) span').classes()).toContain('bg-warning/10')
  })
})
}

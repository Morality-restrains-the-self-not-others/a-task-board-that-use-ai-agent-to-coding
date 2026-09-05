// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] AccountSwitcherDropdown.test.js requires vitest runtime')
} else {
  const { describe, expect, it, vi, beforeEach } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')

  vi.mock('../../utils/publicClientIp.js', () => ({
    fetchPublicClientIp: vi.fn(async () => '203.0.113.77'),
  }))

  const { default: AccountSwitcherDropdown } = await import(
    '../../components/AccountSwitcherDropdown.vue'
  )
  const { fetchPublicClientIp } = await import('../../utils/publicClientIp.js')

  describe('AccountSwitcherDropdown', () => {
    beforeEach(() => {
      fetchPublicClientIp.mockClear()
      fetchPublicClientIp.mockResolvedValue('203.0.113.77')
    })

    it('单击展开，双击收起', async () => {
      const wrapper = mount(AccountSwitcherDropdown, {
        props: {
          displayName: 'alice',
          avatarSrc: 'data:image/svg+xml,<svg></svg>',
          profilePath: '/user/1/profile/',
        },
        global: {
          stubs: { 'router-link': { template: '<a><slot /></a>' } },
        },
      })
      expect(wrapper.find('[data-testid="account-switcher-menu"]').exists()).toBe(false)
      await wrapper.find('[data-testid="account-switcher-trigger"]').trigger('click')
      expect(wrapper.find('[data-testid="account-switcher-menu"]').exists()).toBe(true)
      await wrapper.find('[data-testid="account-switcher-trigger"]').trigger('dblclick')
      expect(wrapper.find('[data-testid="account-switcher-menu"]').exists()).toBe(false)
    })

    it('菜单不渲染多账号区块：无添加账号按钮、无账号切换列表，仅保留个人资料与退出当前', async () => {
      const wrapper = mount(AccountSwitcherDropdown, {
        props: {
          displayName: 'alice',
          avatarSrc: 'data:image/svg+xml,<svg></svg>',
          profilePath: '/user/1/profile/',
        },
        global: {
          stubs: { 'router-link': { template: '<a><slot /></a>' } },
        },
      })
      await wrapper.find('[data-testid="account-switcher-trigger"]').trigger('click')
      const menu = wrapper.find('[data-testid="account-switcher-menu"]')
      expect(menu.exists()).toBe(true)
      // 去掉多账号登录：添加账号按钮与账号切换列表不得渲染
      expect(wrapper.find('[data-testid="account-switcher-add"]').exists()).toBe(false)
      expect(wrapper.find('[data-testid^="account-switcher-item-"]').exists()).toBe(false)
      expect(menu.text()).not.toContain('切换账号')
      // 保留入口：个人资料与退出当前
      expect(wrapper.find('[data-testid="account-switcher-profile"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="account-switcher-logout"]').exists()).toBe(true)
    })

    it('昵称下方展示公网 IP', async () => {
      const wrapper = mount(AccountSwitcherDropdown, {
        props: {
          displayName: 'alice',
          avatarSrc: 'data:image/svg+xml,<svg></svg>',
          profilePath: '/user/1/profile/',
        },
        global: {
          stubs: { 'router-link': { template: '<a><slot /></a>' } },
        },
      })
      await flushPromises()
      const ipEl = wrapper.find('[data-testid="account-switcher-public-ip"]')
      expect(ipEl.exists()).toBe(true)
      expect(ipEl.text()).toBe('203.0.113.77')
      expect(fetchPublicClientIp).toHaveBeenCalled()
    })
  })
}

// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] UserListRow.test.js requires vitest runtime')
} else {
const { describe, expect, it } = await import('vitest')
const { mount } = await import('@vue/test-utils')
const { default: UserListRow } = await import('./UserListRow.vue')

function createUser(overrides = {}) {
  return {
    id: 1,
    username: 'testuser',
    email: 'test@example.com',
    email_verified: true,
    phone: '',
    login_methods: [],
    referrer_code: 'REF001',
    date_joined: '2025-01-15T10:30:00Z',
    is_active: true,
    is_superuser: false,
    is_staff: false,
    is_tenant: true,
    is_archived: false,
    tenant_companies: [],
    ...overrides,
  }
}

describe('UserListRow.vue', () => {
  it('renders email when available', () => {
    const wrapper = mount(UserListRow, {
      props: { user: createUser({ email: 'hello@example.com', email_verified: true }) },
    })
    const tds = wrapper.findAll('td')
    const emailCell = tds[1]
    expect(emailCell.text()).toContain('hello@example.com')
    expect(emailCell.text()).not.toContain('未验证')
    expect(emailCell.text()).not.toContain('—')
  })

  it('renders email with unverified tag when email_verified is false', () => {
    const wrapper = mount(UserListRow, {
      props: { user: createUser({ email: 'hello@example.com', email_verified: false }) },
    })
    const tds = wrapper.findAll('td')
    const emailCell = tds[1]
    expect(emailCell.text()).toContain('hello@example.com')
    expect(emailCell.text()).toContain('未验证')
  })

  it('falls back to phone when email is missing', () => {
    const wrapper = mount(UserListRow, {
      props: { user: createUser({ email: '', phone: '13800138000' }) },
    })
    const tds = wrapper.findAll('td')
    const emailCell = tds[1]
    expect(emailCell.text()).toContain('13800138000')
    expect(emailCell.text()).toContain('手机')
    expect(emailCell.text()).not.toContain('—')
  })

  it('falls back to username when both email and phone are missing', () => {
    const wrapper = mount(UserListRow, {
      props: { user: createUser({ email: '', phone: '', username: 'cooluser' }) },
    })
    const tds = wrapper.findAll('td')
    const emailCell = tds[1]
    expect(emailCell.text()).toContain('cooluser')
    expect(emailCell.text()).not.toContain('手机')
    expect(emailCell.text()).not.toContain('—')
  })

  it('shows user ID when all identifiers are missing', () => {
    const wrapper = mount(UserListRow, {
      props: { user: createUser({ email: '', phone: '', username: '' }) },
    })
    const tds = wrapper.findAll('td')
    const emailCell = tds[1]
    expect(emailCell.text()).toContain('1')
  })

  it('prefers email over phone when both are available', () => {
    const wrapper = mount(UserListRow, {
      props: { user: createUser({ email: 'hi@test.com', phone: '13800138000' }) },
    })
    const tds = wrapper.findAll('td')
    const emailCell = tds[1]
    expect(emailCell.text()).toContain('hi@test.com')
    expect(emailCell.text()).not.toContain('13800138000')
    expect(emailCell.text()).not.toContain('手机')
  })

  it('renders tenant company names after the email column', () => {
    const wrapper = mount(UserListRow, {
      props: { user: createUser({ tenant_companies: [{ id: 'c1', name: 'Acme' }, { id: 'c2', name: 'Zeta' }] }) },
    })
    const cell = wrapper.get('[data-testid="user-tenant-companies"]')
    expect(cell.text()).toBe('Acme、Zeta')
    const tds = wrapper.findAll('td')
    expect(tds[1].text()).toContain('test@example.com')
    expect(tds[2].text()).toBe('Acme、Zeta')
  })

  it('shows em dash when tenant_companies is empty', () => {
    const wrapper = mount(UserListRow, {
      props: { user: createUser({ tenant_companies: [] }) },
    })
    expect(wrapper.get('[data-testid="user-tenant-companies"]').text()).toBe('—')
  })

  it('falls back to company id when name is missing', () => {
    const wrapper = mount(UserListRow, {
      props: { user: createUser({ tenant_companies: [{ id: 'c-only' }] }) },
    })
    expect(wrapper.get('[data-testid="user-tenant-companies"]').text()).toBe('c-only')
  })

  it('shows 测试 badge above 租户 when is_tester is set', () => {
    const wrapper = mount(UserListRow, {
      props: { user: createUser({ is_tenant: true, is_tester: true }) },
    })
    expect(wrapper.get('[data-testid="user-role-badge"]').text()).toBe('测试')
  })

  it('shows 租户 when tester is unset', () => {
    const wrapper = mount(UserListRow, {
      props: { user: createUser({ is_tenant: true, is_tester: false }) },
    })
    expect(wrapper.get('[data-testid="user-role-badge"]').text()).toBe('租户')
  })

  it('renders last_login after the date_joined column', () => {
    const wrapper = mount(UserListRow, {
      props: { user: createUser({ last_login: '2026-08-21T10:30:00Z' }) },
    })
    const cell = wrapper.get('[data-testid="user-last-login"]')
    expect(cell.text()).not.toBe('—')
    expect(cell.text()).toMatch(/2026/)
  })

  it('shows 是 when has_profit_sharing_qualification is true', () => {
    const wrapper = mount(UserListRow, {
      props: { user: createUser({ has_profit_sharing_qualification: true }) },
    })
    expect(wrapper.get('[data-testid="user-profit-sharing-qualification"]').text()).toBe('是')
  })

  it('shows 否 when has_profit_sharing_qualification is false', () => {
    const wrapper = mount(UserListRow, {
      props: { user: createUser({ has_profit_sharing_qualification: false }) },
    })
    expect(wrapper.get('[data-testid="user-profit-sharing-qualification"]').text()).toBe('否')
  })

  it('shows em dash when has_profit_sharing_qualification is missing', () => {
    const wrapper = mount(UserListRow, {
      props: { user: createUser() },
    })
    expect(wrapper.get('[data-testid="user-profit-sharing-qualification"]').text()).toBe('—')
  })

  it('shows em dash when last_login is empty', () => {
    const wrapper = mount(UserListRow, {
      props: { user: createUser({ last_login: '' }) },
    })
    expect(wrapper.get('[data-testid="user-last-login"]').text()).toBe('—')
  })

  it('does not render impersonate button without user:impersonate permission', () => {
    const wrapper = mount(UserListRow, {
      props: { user: createUser(), canImpersonate: false },
    })
    expect(wrapper.find('[data-testid="user-row-impersonate-btn"]').exists()).toBe(false)
  })

  it('renders impersonate button when canImpersonate is true', () => {
    const wrapper = mount(UserListRow, {
      props: { user: createUser(), canImpersonate: true },
    })
    expect(wrapper.get('[data-testid="user-row-impersonate-btn"]').text()).toContain('以该用户身份登录')
  })

  it('emits impersonate with the user when impersonate button clicked', async () => {
    const user = createUser({ id: 42 })
    const wrapper = mount(UserListRow, {
      props: { user, canImpersonate: true },
    })
    await wrapper.get('[data-testid="user-row-impersonate-btn"]').trigger('click')
    expect(wrapper.emitted('impersonate')).toHaveLength(1)
    expect(wrapper.emitted('impersonate')[0][0]).toMatchObject({ id: 42 })
  })
})
}

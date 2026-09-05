// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] UserListRow.login-history.test.js requires vitest runtime')
} else {
const { describe, expect, it } = await import('vitest')
const { mount } = await import('@vue/test-utils')
const { default: UserListRow } = await import('./UserListRow.vue')

describe('UserListRow login history link', () => {
  it('exposes a real href to the admin login-history page', () => {
    const wrapper = mount(UserListRow, {
      props: {
        user: {
          id: '555',
          username: 'u',
          email: 'u@example.com',
          email_verified: true,
          phone: '',
          login_methods: [],
          referrer_code: '',
          date_joined: '2025-01-15T10:30:00Z',
          is_active: true,
          is_superuser: false,
          is_staff: false,
          is_tenant: true,
          is_archived: false,
          tenant_companies: [],
        },
      },
    })
    const link = wrapper.get('[data-testid="user-row-login-history"]')
    expect(link.element.tagName).toBe('A')
    expect(link.attributes('href')).toBe('/system-admin/users/555/login-history/')
    expect(link.text()).toContain('登录历史')
  })
})
}

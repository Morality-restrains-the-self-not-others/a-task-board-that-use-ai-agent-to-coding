// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] UserCenterSidebar.login-history.test.js requires vitest runtime')
} else {
const { describe, expect, it, vi } = await import('vitest')
const { mount } = await import('@vue/test-utils')

vi.mock('../utils/cookieUtils', () => ({
  getCookie: (name) => (name === 'userId' ? '99' : ''),
}))
vi.mock('vue-router', () => ({
  useRoute: () => ({ query: {} }),
}))

const { default: UserCenterSidebar } = await import('./UserCenterSidebar.vue')

describe('UserCenterSidebar login history', () => {
  it('links to the user login-history path and highlights the menu', () => {
    const wrapper = mount(UserCenterSidebar, {
      props: { activeMenu: 'login-history' },
      global: {
        stubs: {
          'router-link': {
            props: ['to'],
            template: '<a :href="typeof to === \'string\' ? to : to.path" :class="$attrs.class"><slot /></a>',
          },
        },
      },
    })
    const link = wrapper.get('[data-testid="user-center-nav-login-history"]')
    expect(link.text()).toContain('登录历史')
    expect(link.attributes('href')).toBe('/user/99/profile/login-history/')
    expect(link.classes().join(' ')).toContain('bg-primary/10')
  })
})
}

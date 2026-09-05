// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] UserGitIdentities.test.js requires vitest runtime')
} else {
  const { mount } = await import('@vue/test-utils')
  const { nextTick } = await import('vue')
  const { beforeEach, describe, expect, it, vi } = await import('vitest')

  const hoistedMocks = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
    resolveUserIdMock: vi.fn(async () => '827923618451263488'),
    routeMock: {
      params: { tenant: 'tenant-a' },
      query: {},
    },
  }))

  vi.mock('../utils/apiUtils', () => ({
    apiFetch: hoistedMocks.apiFetchMock,
  }))

  vi.mock('../utils/cookieUtils', () => ({
    getCookie: () => '',
  }))

  vi.mock('../utils/sessionUserIdUtils.js', () => ({
    resolveAuthenticatedUserId: hoistedMocks.resolveUserIdMock,
  }))

  vi.mock('vue-router', () => ({
    useRoute: () => hoistedMocks.routeMock,
  }))

  vi.mock('../components/UserCenterSidebar.vue', () => ({
    default: { template: '<div />' },
  }))

  async function flushRender() {
    for (let i = 0; i < 8; i += 1) {
      await Promise.resolve()
      await nextTick()
    }
  }

  describe('UserGitIdentities', () => {
    beforeEach(() => {
      vi.clearAllMocks()
      hoistedMocks.resolveUserIdMock.mockResolvedValue('827923618451263488')
      hoistedMocks.apiFetchMock.mockImplementation((url) => {
        if (String(url).includes('/api/accounts/users/profile/')) {
          return Promise.resolve({
            ok: true,
            json: async () => ({ company_nicknames: [] }),
          })
        }
        if (String(url).includes('/api/git-identities/user/')) {
          return Promise.resolve({
            ok: true,
            json: async () => ({ identities: [] }),
          })
        }
        return Promise.resolve({ ok: true, json: async () => ({}) })
      })
    })

    it('无 userId cookie 时应通过 resolver 拉取 Git 身份列表', async () => {
      const { default: UserGitIdentities } = await import('./UserGitIdentities.vue')
      mount(UserGitIdentities)
      await flushRender()

      expect(hoistedMocks.resolveUserIdMock).toHaveBeenCalled()
      expect(hoistedMocks.apiFetchMock).toHaveBeenCalledWith(
        '/api/git-identities/user/827923618451263488/',
        expect.objectContaining({ headers: { Accept: 'application/json' } }),
      )
    })

    it('公司名缺失时展示「未设置」而不是 company_id (OPT-20260828-025)', async () => {
      hoistedMocks.apiFetchMock.mockImplementation((url) => {
        if (String(url).includes('/api/accounts/users/profile/')) {
          return Promise.resolve({
            ok: true,
            json: async () => ({
              company_nicknames: [
                { company_id: 'snowflake-889933', company_name: '', member_name: 'u' },
              ],
            }),
          })
        }
        if (String(url).includes('/api/git-identities/user/')) {
          return Promise.resolve({ ok: true, json: async () => ({ identities: [] }) })
        }
        return Promise.resolve({ ok: true, json: async () => ({}) })
      })
      const { default: UserGitIdentities } = await import('./UserGitIdentities.vue')
      const wrapper = mount(UserGitIdentities)
      await flushRender()

      expect(wrapper.text()).not.toContain('snowflake-889933')
      expect(wrapper.text()).toContain('未设置')
    })
  })
}

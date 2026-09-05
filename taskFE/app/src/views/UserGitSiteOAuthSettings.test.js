// @vitest-environment jsdom
if (!process.env.VITEST) {
  // pre-commit 会以 node 直跑已暂存 *.test.js，非 Vitest 环境下跳过。
  console.log('[skip] UserGitSiteOAuthSettings.test.js requires vitest runtime')
} else {
  const { mount } = await import('@vue/test-utils')
  const { nextTick } = await import('vue')
  const { beforeEach, describe, expect, it, vi } = await import('vitest')

  const hoistedMocks = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
    getCookieMock: vi.fn(),
    replaceMock: vi.fn(async () => {}),
    routeMock: {
      params: {
        tenant: 'tenant-a',
        id: '827923618451263488',
      },
      query: {},
      path: '/user/827923618451263488/profile/git-site-oauth/',
    },
  }))

  vi.mock('../utils/apiUtils', () => ({
    apiFetch: hoistedMocks.apiFetchMock,
  }))

  vi.mock('../utils/cookieUtils', () => ({
    getCookie: hoistedMocks.getCookieMock,
  }))

  vi.mock('../utils/sessionUserIdUtils.js', () => ({
    resolveAuthenticatedUserId: vi.fn(async () => '827923618451263488'),
  }))

  vi.mock('../utils/githubAppReturnStorage', () => ({
    createGithubAppReturnKey: () => 'f'.repeat(32),
    setGithubAppReturnTarget: () => {},
  }))

  vi.mock('vue-router', () => ({
    useRoute: () => hoistedMocks.routeMock,
    useRouter: () => ({ replace: hoistedMocks.replaceMock }),
  }))

  async function flushRender() {
    for (let i = 0; i < 12; i += 1) {
      await Promise.resolve()
      await nextTick()
    }
  }

  describe('UserGitSiteOAuthSettings', () => {
    beforeEach(() => {
      vi.clearAllMocks()
      hoistedMocks.routeMock.query = {}
      hoistedMocks.getCookieMock.mockImplementation((name) =>
        name === 'userId' ? '827923618451263488' : '',
      )
    })

    it('连接接口 503 但返回 connected 字段时，页面应显示可操作状态而非加载失败占位文案', async () => {
      hoistedMocks.apiFetchMock.mockImplementation((url) => {
        if (String(url).includes('/api/git-oauth/providers/')) {
          return Promise.resolve({
            ok: true,
            json: async () => ({
              providers: [
                {
                  provider: 'github',
                  service_provider: 'github-official',
                  provider_key: 'github:github-official',
                  website: 'http://github.com',
                  label: 'GitHub',
                },
              ],
            }),
          })
        }
        return Promise.resolve({
          ok: false,
          json: async () => ({
            detail: '无法连接 gitOauth 或摘要响应无效；请稍后重试',
            connected: false,
            github_login: null,
            github_user_id: null,
            scope: null,
            authorize_scope: 'repo read:user',
          }),
        })
      })

      const component = await import('./UserGitSiteOAuthSettings.vue')
      const wrapper = mount(component.default, {
        global: {
          stubs: {
            UserCenterSidebar: { template: '<div data-test="sidebar-stub" />' },
          },
        },
      })
      await flushRender()

      expect(hoistedMocks.apiFetchMock).toHaveBeenCalledWith(
        '/api/git-oauth/user-app-connection/?service_provider=github-official',
        { headers: { Accept: 'application/json' } },
      )
      expect(wrapper.text()).toContain('无法连接 gitOauth 或摘要响应无效；请稍后重试')
      expect(wrapper.text()).toContain('尚未绑定 GitHub 账号。')
      expect(wrapper.text()).not.toContain('暂时无法获取绑定状态，请稍后重试。')
    })

    it('连接接口 404 非 JSON 响应时，「无法获取绑定状态」错误节点应带 data-traceId', async () => {
      hoistedMocks.apiFetchMock.mockImplementation((url) => {
        if (String(url).includes('/api/git-oauth/providers/')) {
          return Promise.resolve({
            ok: true,
            json: async () => ({
              providers: [
                {
                  provider: 'github',
                  service_provider: 'github-official',
                  provider_key: 'github:github-official',
                  website: 'http://github.com',
                  label: 'GitHub',
                },
              ],
            }),
          })
        }
        // 模拟网关/上游 404 非 JSON（真实 apiFetch 会把响应头 X-Trace-Id 挂到 response.traceId）
        return Promise.resolve({
          ok: false,
          traceId: 'gitoauth-conn-tid-0001',
          json: async () => {
            throw new Error('not json')
          },
        })
      })

      const component = await import('./UserGitSiteOAuthSettings.vue')
      const wrapper = mount(component.default, {
        global: {
          stubs: {
            UserCenterSidebar: { template: '<div data-test="sidebar-stub" />' },
          },
        },
      })
      await flushRender()

      expect(wrapper.text()).toContain('无法获取绑定状态')
      const err = wrapper.find('p.text-sm.text-danger')
      expect(err.exists()).toBe(true)
      expect(err.attributes('data-traceid') || err.attributes('data-traceId')).toBe(
        'gitoauth-conn-tid-0001',
      )
    })

    it('OAuth 回跳 exchange_failed 且带 trace_id 时，错误节点应有 data-traceId', async () => {
      hoistedMocks.routeMock.query = {
        github: 'exchange_failed',
        trace_id: 'gitoauth-exchg-tid-1',
      }
      hoistedMocks.apiFetchMock.mockImplementation((url) => {
        if (String(url).includes('/api/git-oauth/providers/')) {
          return Promise.resolve({
            ok: true,
            json: async () => ({
              providers: [
                {
                  provider: 'github',
                  service_provider: 'github-official',
                  provider_key: 'github:github-official',
                  website: 'http://github.com',
                  label: 'GitHub',
                },
              ],
            }),
          })
        }
        return Promise.resolve({
          ok: true,
          json: async () => ({
            connected: false,
            github_login: null,
            github_user_id: null,
            scope: null,
            authorize_scope: 'repo read:user',
            connections: [],
          }),
        })
      })

      const component = await import('./UserGitSiteOAuthSettings.vue')
      const wrapper = mount(component.default, {
        global: {
          stubs: {
            UserCenterSidebar: { template: '<div data-test="sidebar-stub" />' },
          },
        },
      })
      await flushRender()

      expect(wrapper.text()).toContain('授权失败：无法连接 GitHub 服务器，请检查网络后重试')
      const err = wrapper.find('p.text-sm.text-danger')
      expect(err.exists()).toBe(true)
      expect(err.attributes('data-traceid') || err.attributes('data-traceId')).toBe(
        'gitoauth-exchg-tid-1',
      )
      expect(hoistedMocks.replaceMock).toHaveBeenCalled()
      const replaceArg = hoistedMocks.replaceMock.mock.calls.at(-1)?.[0]
      expect(replaceArg?.query?.github).toBeUndefined()
      expect(replaceArg?.query?.trace_id).toBeUndefined()
    })
  })
}

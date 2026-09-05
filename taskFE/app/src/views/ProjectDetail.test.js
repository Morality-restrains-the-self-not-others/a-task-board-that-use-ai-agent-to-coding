// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] ProjectDetail.test.js requires vitest runtime')
} else {
  const { mount } = await import('@vue/test-utils')
  const { nextTick } = await import('vue')
  const { beforeEach, describe, expect, it, vi } = await import('vitest')

  const {
    setGitOAuthProviderCatalogForTests,
    normalizeGitOAuthProviderCatalog,
  } = await import('../utils/repoOAuthAuthorizeUtils.js')

  const TEST_CATALOG = normalizeGitOAuthProviderCatalog([
    { provider: 'github', website: 'https://github.com', service_provider: 'default' },
    { provider: 'gitlab', website: 'https://gitlab.com', service_provider: 'default' },
    { provider: 'gitlab', website: 'https://gitlab.daydaymoney.com', service_provider: 'default' },
  ])

  const hoistedMocks = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
    routeMock: {
      params: {
        tenant: '827923618468040704',
        id: '846027310833254400',
      },
      path: '/tenant/827923618468040704/projects/846027310833254400/',
      query: {},
    },
    pushMock: vi.fn(),
    setGithubAppReturnTargetMock: vi.fn(),
  }))

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch: hoistedMocks.apiFetchMock,
  }))

  vi.mock('../utils/githubAppReturnStorage.js', () => ({
    createGithubAppReturnKey: () => 'a'.repeat(32),
    setGithubAppReturnTarget: hoistedMocks.setGithubAppReturnTargetMock,
  }))

  vi.mock('vue-router', () => ({
    useRoute: () => hoistedMocks.routeMock,
    useRouter: () => ({ push: hoistedMocks.pushMock }),
  }))

  async function flushRender() {
    for (let i = 0; i < 4; i += 1) {
      await Promise.resolve()
      await nextTick()
    }
  }

  function mockProjectDetailApi(projectData, { companyName = '', companyCurrentOk = true } = {}) {
    hoistedMocks.apiFetchMock.mockImplementation(async (url) => {
      const path = String(url || '')
      if (path.includes('/accounts/companies/current')) {
        if (!companyCurrentOk) {
          return {
            ok: false,
            status: 404,
            json: async () => ({ detail: 'company not found' }),
          }
        }
        return {
          ok: true,
          json: async () => ({ name: companyName }),
        }
      }
      if (path.includes('/projects/846027310833254400/') && !path.includes('/branches/')) {
        return {
          ok: true,
          json: async () => projectData,
        }
      }
      if (path.includes('/workspaces/?ids=')) {
        return {
          ok: true,
          json: async () => [],
        }
      }
      if (path.endsWith('/workspaces/')) {
        return {
          ok: true,
          json: async () => [],
        }
      }
      if (path.includes('/api/accounts/git-oauth/providers')) {
        return {
          ok: true,
          json: async () => ({
            providers: [
              { provider: 'github', website: 'https://github.com', service_provider: 'default' },
              { provider: 'gitlab', website: 'https://gitlab.com', service_provider: 'default' },
              { provider: 'gitlab', website: 'https://gitlab.daydaymoney.com', service_provider: 'default' },
            ],
          }),
        }
      }
      if (path.includes('/projects/validate-git-repos/')) {
        return {
          ok: true,
          json: async () => ({
            results: [
              { url: 'https://github.com/demo/repo-a.git', token_status: 'not_bound' },
              { url: 'https://gitlab.daydaymoney.com/group/repo-b.git', token_status: 'not_bound' },
              { url: 'https://bitbucket.org/team/repo-c.git', token_status: 'not_applicable' },
            ],
          }),
        }
      }
      if (path.includes('/projects/validate-git-repo/')) {
        return {
          ok: true,
          json: async () => ({
            is_accessible: false,
            token_status: 'not_bound',
          }),
        }
      }
      if (path.includes('/installed-images/')) {
        return {
          ok: true,
          json: async () => [],
          text: async () => '[]',
        }
      }
      return {
        ok: true,
        json: async () => ({}),
      }
    })
  }

  describe('ProjectDetail OAuth row action', () => {
    beforeEach(() => {
      vi.clearAllMocks()
      setGitOAuthProviderCatalogForTests([...TEST_CATALOG])
    })

    it('仅对 GitHub/GitLab 仓库显示 OAuth 授权按钮', async () => {
      mockProjectDetailApi({
        id: '846027310833254400',
        name: 'Demo Project',
        description: '',
        git_repos: [
          'https://github.com/demo/repo-a.git',
          'https://gitlab.daydaymoney.com/group/repo-b.git',
          'https://bitbucket.org/team/repo-c.git',
        ],
        workspaces: [],
      })

      const component = await import('./ProjectDetail.vue')
      const wrapper = mount(component.default, {
        global: {
          stubs: {
            WorkspaceAssociation: { template: '<div data-test="workspace-association-stub" />' },
          },
        },
      })
      await flushRender()

      const oauthButtons = wrapper
        .findAll('button')
        .filter((btn) => btn.text().includes('OAuth 授权'))
      expect(oauthButtons).toHaveLength(2)
      expect(wrapper.text()).toContain('https://github.com/demo/repo-a.git')
      expect(wrapper.text()).toContain('https://gitlab.daydaymoney.com/group/repo-b.git')
      expect(wrapper.text()).toContain('https://bitbucket.org/team/repo-c.git')
    })

    it('点击 localhost 仓库行按钮应携带 next/return_key/repo_url 并跳转授权页', async () => {
      const oldLocation = window.location
      const locationMock = {
        href: 'http://localhost:4000/tenant/827923618468040704/projects/846027310833254400/',
        pathname: '/tenant/827923618468040704/projects/846027310833254400/',
        search: '',
      }
      Object.defineProperty(window, 'location', {
        value: locationMock,
        writable: true,
        configurable: true,
      })

      hoistedMocks.apiFetchMock.mockImplementation(async (url) => {
        const path = String(url || '')
        if (path.includes('/projects/846027310833254400/') && !path.includes('/branches/')) {
          return {
            ok: true,
            json: async () => ({
              id: '846027310833254400',
              name: 'Demo Project',
              description: '',
              git_repos: ['http://localhost:8012/ljy/somanyad'],
              workspaces: [],
            }),
          }
        }
        if (path.includes('/workspaces/')) {
          return {
            ok: true,
            json: async () => [],
          }
        }
        if (path.includes('/projects/validate-git-repos/')) {
          return {
            ok: true,
            json: async () => ({
              results: [
                { url: 'http://localhost:8012/ljy/somanyad', token_status: 'not_bound' },
              ],
            }),
          }
        }
        if (path.includes('/projects/validate-git-repo/')) {
          return {
            ok: true,
            json: async () => ({
              is_accessible: false,
              token_status: 'not_bound',
            }),
          }
        }
        if (path.includes('/api/accounts/git-oauth/providers')) {
          return {
            ok: true,
            json: async () => ({
              providers: [
                { provider: 'github', website: 'https://github.com', service_provider: 'default' },
                { provider: 'gitlab', website: 'https://gitlab.com', service_provider: 'default' },
                { provider: 'gitlab', website: 'http://localhost:8012', service_provider: 'default' },
              ],
            }),
          }
        }
        if (path.startsWith('/api/git-oauth/gitlab-start-from-gateway/')) {
          return {
            ok: true,
            json: async () => ({ authorize_url: 'https://oauth.example/gitlab-start' }),
          }
        }
        if (path.includes('/installed-images/')) {
          return {
            ok: true,
            json: async () => [],
            text: async () => '[]',
          }
        }
        return {
          ok: true,
          json: async () => ({}),
        }
      })

      try {
        const component = await import('./ProjectDetail.vue')
        const wrapper = mount(component.default, {
          global: {
            stubs: {
              WorkspaceAssociation: { template: '<div data-test="workspace-association-stub" />' },
            },
          },
        })
        await flushRender()

        const oauthButton = wrapper.findAll('button').find((btn) => btn.text().includes('OAuth 授权'))
        expect(oauthButton).toBeTruthy()
        await oauthButton.trigger('click')
        await flushRender()

        expect(hoistedMocks.setGithubAppReturnTargetMock).toHaveBeenCalledWith(
          'a'.repeat(32),
          '/tenant/827923618468040704/projects/846027310833254400/',
        )
        const startCall = hoistedMocks.apiFetchMock.mock.calls.find(([url]) =>
          String(url || '').startsWith('/api/git-oauth/gitlab-start-from-gateway/'),
        )
        expect(startCall).toBeTruthy()
        expect(String(startCall[0])).toContain(
          'repo_url=http%3A%2F%2Flocalhost%3A8012%2Fljy%2Fsomanyad',
        )
        expect(String(startCall[0])).toContain('return_key=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa')
        expect(String(startCall[0])).toContain('next=%2Ftenant%2F827923618468040704%2Fprojects%2F846027310833254400%2F')
        expect(window.location.href).toBe('https://oauth.example/gitlab-start')
      } finally {
        Object.defineProperty(window, 'location', {
          value: oldLocation,
          writable: true,
          configurable: true,
        })
      }
    })

    it('token_available 时不显示 OAuth 授权按钮', async () => {
      mockProjectDetailApi({
        id: '846027310833254400',
        name: 'Demo Project',
        description: '',
        git_repos: ['https://github.com/demo/repo-a.git'],
        git_repos_status: [
          {
            repo_url: 'https://github.com/demo/repo-a.git',
            token_status: 'token_available',
            oauth_provider: 'github',
            oauth_service_provider: 'default',
          },
        ],
        workspaces: [],
      })

      const component = await import('./ProjectDetail.vue')
      const wrapper = mount(component.default, {
        global: {
          stubs: {
            WorkspaceAssociation: { template: '<div data-test="workspace-association-stub" />' },
          },
        },
      })
      await flushRender()

      const oauthButtons = wrapper
        .findAll('button')
        .filter((btn) => btn.text().includes('OAuth 授权'))
      expect(oauthButtons).toHaveLength(0)
      const retryButtons = wrapper
        .findAll('button')
        .filter((btn) => btn.text().includes('重试'))
      expect(retryButtons).toHaveLength(0)
    })

    it('not_bound 时显示 OAuth 授权按钮', async () => {
      mockProjectDetailApi({
        id: '846027310833254400',
        name: 'Demo Project',
        description: '',
        git_repos: ['https://github.com/demo/repo-a.git'],
        git_repos_status: [
          {
            repo_url: 'https://github.com/demo/repo-a.git',
            token_status: 'not_bound',
            oauth_provider: 'github',
            oauth_service_provider: 'default',
          },
        ],
        workspaces: [],
      })

      const component = await import('./ProjectDetail.vue')
      const wrapper = mount(component.default, {
        global: {
          stubs: {
            WorkspaceAssociation: { template: '<div data-test="workspace-association-stub" />' },
          },
        },
      })
      await flushRender()

      const oauthButtons = wrapper
        .findAll('button')
        .filter((btn) => btn.text().includes('OAuth 授权'))
      expect(oauthButtons).toHaveLength(1)
      expect(oauthButtons[0].text()).toBe('OAuth 授权')
    })

    it('token_error 时显示重试按钮', async () => {
      mockProjectDetailApi({
        id: '846027310833254400',
        name: 'Demo Project',
        description: '',
        git_repos: ['https://github.com/demo/repo-a.git'],
        git_repos_status: [
          {
            repo_url: 'https://github.com/demo/repo-a.git',
            token_status: 'token_error',
            oauth_provider: 'github',
            oauth_service_provider: 'default',
          },
        ],
        workspaces: [],
      })

      const component = await import('./ProjectDetail.vue')
      const wrapper = mount(component.default, {
        global: {
          stubs: {
            WorkspaceAssociation: { template: '<div data-test="workspace-association-stub" />' },
          },
        },
      })
      await flushRender()

      const retryButtons = wrapper
        .findAll('button')
        .filter((btn) => btn.text().includes('重试'))
      expect(retryButtons).toHaveLength(1)
      const oauthButtons = wrapper
        .findAll('button')
        .filter((btn) => btn.text() === 'OAuth 授权')
      expect(oauthButtons).toHaveLength(0)
    })

    it('not_applicable 时不显示按钮（非 OAuth 仓库）', async () => {
      mockProjectDetailApi({
        id: '846027310833254400',
        name: 'Demo Project',
        description: '',
        git_repos: ['https://bitbucket.org/team/repo.git'],
        git_repos_status: [
          {
            repo_url: 'https://bitbucket.org/team/repo.git',
            token_status: 'not_applicable',
            oauth_provider: '',
            oauth_service_provider: '',
          },
        ],
        workspaces: [],
      })

      const component = await import('./ProjectDetail.vue')
      const wrapper = mount(component.default, {
        global: {
          stubs: {
            WorkspaceAssociation: { template: '<div data-test="workspace-association-stub" />' },
          },
        },
      })
      await flushRender()

      const oauthButtons = wrapper
        .findAll('button')
        .filter((btn) => btn.text().includes('OAuth 授权'))
      expect(oauthButtons).toHaveLength(0)
      const retryButtons = wrapper
        .findAll('button')
        .filter((btn) => btn.text().includes('重试'))
      expect(retryButtons).toHaveLength(0)
    })

    it('无 git_repos_status 字段时向后兼容（回退到 provider-only 逻辑）', async () => {
      mockProjectDetailApi({
        id: '846027310833254400',
        name: 'Demo Project',
        description: '',
        git_repos: ['https://github.com/demo/repo-a.git'],
        workspaces: [],
      })

      const component = await import('./ProjectDetail.vue')
      const wrapper = mount(component.default, {
        global: {
          stubs: {
            WorkspaceAssociation: { template: '<div data-test="workspace-association-stub" />' },
          },
        },
      })
      await flushRender()

      const oauthButtons = wrapper
        .findAll('button')
        .filter((btn) => btn.text().includes('OAuth 授权'))
      expect(oauthButtons).toHaveLength(1)
    })

    it('展示每个 Git 仓库的 OAuth 授权状态徽章', async () => {
      mockProjectDetailApi({
        id: '846027310833254400',
        name: 'Demo Project',
        description: '',
        git_repos: [
          'https://github.com/demo/repo-a.git',
          'https://bitbucket.org/team/repo-b.git',
        ],
        git_repos_status: [
          {
            repo_url: 'https://github.com/demo/repo-a.git',
            token_status: 'token_available',
            oauth_provider: 'github',
            oauth_service_provider: 'default',
          },
          {
            repo_url: 'https://bitbucket.org/team/repo-b.git',
            token_status: 'not_applicable',
            oauth_provider: '',
            oauth_service_provider: '',
          },
        ],
        workspaces: [],
      })

      const component = await import('./ProjectDetail.vue')
      const wrapper = mount(component.default, {
        global: {
          stubs: {
            WorkspaceAssociation: { template: '<div data-test="workspace-association-stub" />' },
          },
        },
      })
      await flushRender()

      expect(wrapper.get('[data-testid="git-repo-oauth-status-0"]').text()).toBe('已授权')
      expect(wrapper.get('[data-testid="git-repo-oauth-status-1"]').text()).toBe('无需 OAuth')
    })

    it('not_bound 状态显示未授权徽章与 OAuth 按钮', async () => {
      mockProjectDetailApi({
        id: '846027310833254400',
        name: 'Demo Project',
        description: '',
        git_repos: ['https://github.com/demo/repo-a.git'],
        git_repos_status: [
          {
            repo_url: 'https://github.com/demo/repo-a.git',
            token_status: 'not_bound',
            oauth_provider: 'github',
            oauth_service_provider: 'default',
          },
        ],
        workspaces: [],
      })

      const component = await import('./ProjectDetail.vue')
      const wrapper = mount(component.default, {
        global: {
          stubs: {
            WorkspaceAssociation: { template: '<div data-test="workspace-association-stub" />' },
          },
        },
      })
      await flushRender()

      expect(wrapper.get('[data-testid="git-repo-oauth-status-0"]').text()).toBe('需要授权')
      expect(wrapper.findAll('button').filter((btn) => btn.text().includes('OAuth 授权'))).toHaveLength(1)
    })
  })

  describe('ProjectDetail 所属公司展示', () => {
    beforeEach(() => {
      vi.clearAllMocks()
      setGitOAuthProviderCatalogForTests([...TEST_CATALOG])
    })

    it('展示公司名称而非租户 ID', async () => {
      const tenantId = '827923618468040704'
      mockProjectDetailApi(
        {
          id: '846027310833254400',
          name: 'Demo Project',
          company: tenantId,
          workspaces: [],
        },
        { companyName: '测试公司' },
      )

      const component = await import('./ProjectDetail.vue')
      const wrapper = mount(component.default, {
        global: {
          stubs: {
            WorkspaceAssociation: { template: '<div data-test="workspace-association-stub" />' },
          },
        },
      })
      await flushRender()

      const el = wrapper.get('[data-testid="project-company-name"]')
      expect(el.text()).toBe('测试公司')
      expect(el.text()).not.toContain(tenantId)
      expect(wrapper.text()).toContain('Demo Project')
    })

    it('名称接口失败时展示未设置且不回退租户 ID', async () => {
      const tenantId = '827923618468040704'
      mockProjectDetailApi(
        {
          id: '846027310833254400',
          name: 'Demo Project',
          company: tenantId,
          workspaces: [],
        },
        { companyCurrentOk: false },
      )

      const component = await import('./ProjectDetail.vue')
      const wrapper = mount(component.default, {
        global: {
          stubs: {
            WorkspaceAssociation: { template: '<div data-test="workspace-association-stub" />' },
          },
        },
      })
      await flushRender()

      const el = wrapper.get('[data-testid="project-company-name"]')
      expect(el.text()).toBe('未设置')
      expect(wrapper.text()).not.toContain(tenantId)
      expect(wrapper.text()).toContain('Demo Project')
    })

    it('GET 404 时展示项目不存在或已删除且带 data-testid', async () => {
      hoistedMocks.apiFetchMock.mockImplementation(async (url) => {
        const path = String(url || '')
        if (path.includes('/accounts/companies/current')) {
          return { ok: true, json: async () => ({ name: '' }) }
        }
        if (path.includes('/projects/846027310833254400/') && !path.includes('/branches/')) {
          return {
            ok: false,
            status: 404,
            json: async () => ({ detail: 'project not found' }),
            headers: { get: () => null },
          }
        }
        return { ok: true, json: async () => ({}) }
      })

      const component = await import('./ProjectDetail.vue')
      const wrapper = mount(component.default, {
        global: {
          stubs: {
            WorkspaceAssociation: { template: '<div data-test="workspace-association-stub" />' },
          },
        },
      })
      await flushRender()

      const el = wrapper.get('[data-testid="project-detail-error"]')
      expect(el.text()).toContain('项目不存在或已删除')
    })
  })
}

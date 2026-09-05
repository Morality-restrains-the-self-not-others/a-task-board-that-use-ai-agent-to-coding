// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] TaskDetailLinkedProjectsPanel.test.js requires vitest runtime')
} else {
  const { mount } = await import('@vue/test-utils')
  const { beforeEach, describe, expect, it, vi } = await import('vitest')

  const hoistedMocks = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
    replaceMock: vi.fn(async () => {}),
    routeMock: {
      path: '/tenant/t1/workspace/w1/task-detail/42/',
      query: { github: 'ok', keep: '1' },
      hash: '',
    },
  }))

  vi.mock('../../utils/apiUtils', () => ({
    apiFetch: hoistedMocks.apiFetchMock,
  }))

  vi.mock('../../utils/githubAppReturnStorage.js', () => ({
    createGithubAppReturnKey: () => 'a'.repeat(32),
    setGithubAppReturnTarget: () => {},
  }))

  vi.mock('vue-router', () => ({
    useRoute: () => hoistedMocks.routeMock,
    useRouter: () => ({ replace: hoistedMocks.replaceMock }),
  }))

  const baseProps = {
    tenantId: 't1',
    workspaceId: 'w1',
    taskId: '42',
    isEditing: false,
    taskProjectsWithDetails: [],
    editingTask: { linkedProjects: [] },
    workspaceProjects: [],
    workspaceProjectsLoading: false,
    getProjectRepos: () => [],
    getRepoBranchValue: () => '',
    getRepoBranches: () => [],
    getRepoBranchError: () => '',
    isLoadingRepoBranches: () => false,
    repoCloneFieldId: () => '',
    gitOAuthCatalogVersion: 0,
  }

  const githubProjectProps = {
    ...baseProps,
    taskProjectsWithDetails: [
      {
        id: 'proj-1',
        project_name: 'DemoProject',
        project: { git_repos: ['https://github.com/demo/repo-a.git'] },
        repo_branches: { 'https://github.com/demo/repo-a.git': 'main' },
      },
    ],
  }

  const flushMicrotasks = async (rounds = 4) => {
    for (let i = 0; i < rounds; i += 1) {
      await Promise.resolve()
    }
  }

  describe('TaskDetailLinkedProjectsPanel', () => {
    beforeEach(() => {
      vi.clearAllMocks()
      vi.useFakeTimers()
      document.cookie = 'userId=827923618451263488; path=/'
      hoistedMocks.routeMock.query = { github: 'ok', keep: '1' }
      hoistedMocks.apiFetchMock.mockResolvedValue({
        ok: true,
        json: async () => ({ github_connections: [], repo_bindings: [] }),
      })
    })

    it('非编辑态只展示仓库与分支，不含 OAuth/身份/进度操作', async () => {
      hoistedMocks.routeMock.query = {}
      const component = await import('./TaskDetailLinkedProjectsPanel.vue')
      const wrapper = mount(component.default, { props: githubProjectProps })
      await vi.runAllTimersAsync()
      expect(wrapper.text()).toContain('DemoProject')
      expect(wrapper.text()).toContain('https://github.com/demo/repo-a.git')
      expect(wrapper.text()).toContain('main')
      expect(wrapper.text()).toContain('添加评论时选择')
      expect(wrapper.find('[data-testid="task-repo-oauth-bind-btn"]').exists()).toBe(false)
      expect(wrapper.find('[data-testid="repo-git-commit-identity-section"]').exists()).toBe(false)
      expect(wrapper.text()).not.toContain('保存账号')
      expect(wrapper.text()).not.toContain('重新克隆')
      expect(wrapper.text()).not.toContain('Git 提交身份')
      expect(wrapper.text()).not.toContain('启动前需完成两步')
    })

    it('应在 OAuth 绑定状态变化时 emit repo-oauth-readiness', async () => {
      hoistedMocks.apiFetchMock.mockImplementation(async (url) => {
        const path = String(url || '')
        if (path.includes('/api/git-oauth/user-app-connection/')) {
          return { ok: true, json: async () => ({ connected: false, connections: [] }) }
        }
        return { ok: true, json: async () => ({ github_connections: [], repo_bindings: [] }) }
      })
      const component = await import('./TaskDetailLinkedProjectsPanel.vue')
      const wrapper = mount(component.default, { props: githubProjectProps })
      await vi.runAllTimersAsync()
      await flushMicrotasks(6)
      const events = wrapper.emitted('repo-oauth-readiness') || []
      expect(events.length).toBeGreaterThan(0)
      const latest = events[events.length - 1][0]
      expect(latest.hasOAuthRepos).toBe(true)
      expect(latest.startBlocked).toBe(true)
      expect(latest.unboundRepoUrls).toContain('https://github.com/demo/repo-a.git')
    })

    it('编辑态无关联项目时应展示项目选择下拉', async () => {
      const component = await import('./TaskDetailLinkedProjectsPanel.vue')
      const wrapper = mount(component.default, {
        props: {
          ...baseProps,
          isEditing: true,
          editingTask: { linkedProjects: [{ project_id: '', repo_branches: {} }] },
          workspaceProjects: [{ id: '100', name: 'Demo Project' }],
        },
      })
      expect(wrapper.text()).toContain('请选择项目')
      expect(wrapper.find('#task-linked-project-0').exists()).toBe(true)
      const options = wrapper.find('#task-linked-project-0').findAll('option')
      expect(options.some((opt) => opt.text().includes('Demo Project'))).toBe(true)
    })

    it('编辑态从分支候选列表选择后应失焦输入框避免下拉再次弹出', async () => {
      const component = await import('./TaskDetailLinkedProjectsPanel.vue')
      const repoUrl = 'http://localhost:8012/group/sdk.git'
      const wrapper = mount(component.default, {
        props: {
          ...baseProps,
          isEditing: true,
          editingTask: {
            linkedProjects: [{ project_id: '100', repo_branches: { [repoUrl]: '' } }],
          },
          workspaceProjects: [{ id: '100', name: 'Demo Project', git_repos: [repoUrl] }],
          getProjectRepos: () => [repoUrl],
          getRepoBranches: () => ['main', 'develop'],
          getRepoBranchValue: (project, url) => project.repo_branches?.[url] || '',
          repoCloneFieldId: () => 'sdk',
        },
      })
      const select = wrapper.find('#task-edit-base-branch-sdk')
      expect(select.exists()).toBe(true)
      await select.setValue('main')
      await select.trigger('change')
      const emitted = wrapper.emitted('set-repo-branch') || []
      expect(emitted.some((args) => args[2] === 'main')).toBe(true)
    })

    it('编辑态手动输入分支时应展示文本框并支持返回列表', async () => {
      const component = await import('./TaskDetailLinkedProjectsPanel.vue')
      const repoUrl = 'http://localhost:8012/group/sdk.git'
      const wrapper = mount(component.default, {
        props: {
          ...baseProps,
          isEditing: true,
          editingTask: {
            linkedProjects: [{ project_id: '100', repo_branches: { [repoUrl]: '' } }],
          },
          workspaceProjects: [{ id: '100', name: 'Demo Project', git_repos: [repoUrl] }],
          getProjectRepos: () => [repoUrl],
          getRepoBranches: () => ['main', 'develop'],
          getRepoBranchValue: (project, url) => project.repo_branches?.[url] || '',
          repoCloneFieldId: () => 'sdk',
        },
      })
      const select = wrapper.find('#task-edit-base-branch-sdk')
      await select.setValue('__custom__')
      await select.trigger('change')
      await flushMicrotasks(6)
      expect(wrapper.find('#task-edit-base-branch-sdk').element.tagName).toBe('INPUT')
      expect(wrapper.text()).toContain('返回列表')
      const backBtn = wrapper.findAll('button').find((btn) => btn.text().includes('返回列表'))
      expect(backBtn).toBeTruthy()
      await backBtn.trigger('click')
      await flushMicrotasks(6)
      expect(wrapper.find('#task-edit-base-branch-sdk').element.tagName).toBe('SELECT')
    })

    it('编辑态分支拉取 OAuth 失败时应展示 OAuth 授权按钮', async () => {
      const component = await import('./TaskDetailLinkedProjectsPanel.vue')
      const wrapper = mount(component.default, {
        props: {
          ...baseProps,
          isEditing: true,
          editingTask: {
            linkedProjects: [{ project_id: '100', repo_branches: {} }],
          },
          workspaceProjects: [{ id: '100', name: 'Demo Project', git_repos: ['http://localhost:8012/group/sdk.git'] }],
          getProjectRepos: () => ['http://localhost:8012/group/sdk.git'],
          getRepoBranchError: () =>
            '无法获取 GitLab 分支：GitLab 会话无效或无权访问该仓库，请重新登录 GitLab 或完成 OAuth 授权',
        },
      })
      expect(wrapper.find('[data-testid="task-edit-repo-oauth-btn"]').exists()).toBe(true)
      expect(wrapper.text()).toContain('OAuth 授权')
    })

    it('U7 非编辑态标题栏展示默认镜像且无镜像选择器', async () => {
      hoistedMocks.routeMock.query = {}
      const component = await import('./TaskDetailLinkedProjectsPanel.vue')
      const wrapper = mount(component.default, {
        props: {
          ...githubProjectProps,
          defaultImageId: '878236719185424384',
          defaultImageLabel: 'trae-agent:x86_64-latest',
        },
      })
      await vi.runAllTimersAsync()
      const el = wrapper.get('[data-testid="task-default-container-image"]')
      expect(el.text()).toContain('默认镜像')
      expect(el.text()).toContain('trae-agent:x86_64-latest')
      expect(wrapper.find('#task-container-image').exists()).toBe(false)
      expect(wrapper.find('select').exists()).toBe(false)
    })
  })
}

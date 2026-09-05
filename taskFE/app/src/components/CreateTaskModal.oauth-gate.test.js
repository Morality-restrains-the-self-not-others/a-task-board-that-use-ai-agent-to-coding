// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] CreateTaskModal.oauth-gate.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { afterEach, beforeEach, describe, expect, it, vi } = await import('vitest')

  const hoistedMocks = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
    oauthConnected: true,
    oauthStatus: 200,
    oauthTraceId: '',
    oauthDetail: '',
  }))

  vi.mock('../utils/apiUtils', () => ({
    apiFetch: hoistedMocks.apiFetchMock,
  }))

  vi.mock('../utils/cookieUtils', () => ({
    getCookie: () => 'user-1',
  }))

  vi.mock('./ProjectRepoAccessHintIcon.vue', () => ({
    default: { template: '<span />' },
  }))

  const makeEditingTask = (overrides = {}) => ({
    title: '测试任务',
    description: '',
    task_kind: '',
    progressColumn: { id: 1 },
    task_type: { id: 'type-1' },
    container_image: { id: 'img-1' },
    auto_run: true,
    priority: '1',
    due_date: '',
    owner: '',
    assignees: [],
    workBranchPreset: 'feature',
    workBranchName: 'feature/demo',
    mergeTargetPreset: 'custom',
    mergeTargetName: 'main',
    projectSelections: [{
      selectionRowKey: 'row-1',
      projectId: '100',
      repoBranches: [{ repoIndex: 0, baseBranch: 'main' }],
    }],
    ...overrides,
  })

  const fieldSettings = () => ({
    description: true,
    task_kind: true,
    code_lang: true,
    structured_fields: true,
    project_branch: true,
    container_image: true,
    feature_params: false,
    priority: true,
    due_date: true,
    auto_run: true,
    owner: true,
    assignees: true,
  })

  const baseProps = {
    show: true,
    editingTask: makeEditingTask(),
    taskStatuses: [{ id: 1, name: '待办' }],
    taskTypes: [{ id: 'type-1', name: '功能' }],
    projects: [{
      id: '100',
      name: 'Demo Project',
      git_repos: ['https://github.com/acme/demo.git'],
      container_image_id: 'img-1',
      server_run_template: {
        default_auto_run: true,
        platform: 'aliyun',
        region: 'cn-hangzhou',
      },
    }],
    installedImages: [{ id: 'img-1', name: 'ubuntu', version: '22.04' }],
    isProjectsLoading: false,
    projectsError: null,
    currentWorkspace: { id: 'ws-1' },
    tenantId: 't1',
    companyUserName: 'tester',
    fieldSettings: fieldSettings(),
  }

  const jsonResponse = (body, status = 200, headers = {}) => ({
    ok: status >= 200 && status < 300,
    status,
    headers: {
      get: (name) => headers[String(name || '').toLowerCase()] || null,
    },
    json: async () => body,
  })

  const mountModal = async () => {
    const component = await import('./CreateTaskModal.vue')
    const wrapper = mount(component.default, {
      props: baseProps,
      attachTo: document.body,
    })
    await flushPromises()
    await flushPromises()
    return wrapper
  }

  describe('CreateTaskModal oauth gate', () => {
    beforeEach(() => {
      vi.clearAllMocks()
      hoistedMocks.oauthConnected = true
      hoistedMocks.oauthStatus = 200
      hoistedMocks.oauthTraceId = ''
      hoistedMocks.oauthDetail = ''
      localStorage.setItem('currentUserId', 'user-1')
      hoistedMocks.apiFetchMock.mockImplementation(async (url) => {
        const path = String(url)
        if (path.includes('/git-identities/')) {
          return jsonResponse({
            identities: [{
              id: 'gid-1',
              git_user_name: 'Ann',
              git_user_email: 'ann@example.com',
              is_default: true,
            }],
          })
        }
        if (path.includes('/validate-git-repos/')) {
          if (hoistedMocks.oauthStatus !== 200) {
            return jsonResponse(
              { detail: hoistedMocks.oauthDetail || '无法检查 OAuth 绑定状态' },
              hoistedMocks.oauthStatus,
              hoistedMocks.oauthTraceId ? { 'x-trace-id': hoistedMocks.oauthTraceId } : {},
            )
          }
          return jsonResponse({
            results: [{
              url: 'https://github.com/acme/demo.git',
              token_status: hoistedMocks.oauthConnected ? 'token_available' : 'not_bound',
            }],
          })
        }
        if (path.includes('/user-app-connection/')) {
          return jsonResponse({ connected: hoistedMocks.oauthConnected })
        }
        if (path.includes('/workspace-collaborators/')) return jsonResponse([])
        if (path.includes('/repo-access-check/')) return jsonResponse({ accessible: true })
        if (path.includes('/branches/')) return jsonResponse({ branches: ['main'] })
        return jsonResponse({})
      })
    })

    afterEach(() => {
      document.body.innerHTML = ''
    })

    it('blocks create when selected github repo is not oauth bound and auto_run enabled', async () => {
      hoistedMocks.oauthConnected = false
      const wrapper = await mountModal()
      const reason = wrapper.find('[data-testid="create-task-submit-blocked-reason"]')
      expect(reason.exists()).toBe(true)
      expect(reason.text()).toMatch(/项目详情|下方绑定/)
      expect(wrapper.find('[data-testid="create-task-submit-btn"]').attributes('disabled')).toBeDefined()
      const bind = wrapper.find('[data-testid="create-task-repo-oauth-bind"]')
      expect(bind.exists()).toBe(true)
      expect(bind.element.tagName).toBe('A')
      expect(bind.attributes('href')).toContain('github-start-from-gateway')
      expect(bind.attributes('href')).toContain('repo_url=')
      wrapper.unmount()
    })

    it('does not oauth-block when auto_run is disabled even if repo is unbound', async () => {
      hoistedMocks.oauthConnected = false
      const component = await import('./CreateTaskModal.vue')
      const wrapper = mount(component.default, {
        props: {
          ...baseProps,
          editingTask: makeEditingTask({ auto_run: false }),
          projects: [{
            id: '100',
            name: 'Demo Project',
            git_repos: ['https://github.com/acme/demo.git'],
            container_image_id: 'img-1',
            server_run_template: {
              default_auto_run: false,
              platform: 'aliyun',
              region: 'cn-hangzhou',
            },
          }],
        },
        attachTo: document.body,
      })
      await flushPromises()
      await flushPromises()
      expect(wrapper.find('[data-testid="create-task-submit-blocked-reason"]').exists()).toBe(false)
      expect(wrapper.find('[data-testid="create-task-repo-oauth-bind"]').exists()).toBe(false)
      wrapper.unmount()
    })

    it('allows create when selected github repo is oauth bound', async () => {
      hoistedMocks.oauthConnected = true
      const wrapper = await mountModal()
      expect(wrapper.find('[data-testid="create-task-submit-blocked-reason"]').exists()).toBe(false)
      expect(wrapper.find('[data-testid="create-task-submit-btn"]').attributes('disabled')).toBeUndefined()
      expect(wrapper.find('[data-testid="create-task-repo-oauth-bound"]').exists()).toBe(true)
      wrapper.unmount()
    })

    it('treats git_repos {url} objects as bound when validate returns token_available', async () => {
      hoistedMocks.oauthConnected = true
      const component = await import('./CreateTaskModal.vue')
      const wrapper = mount(component.default, {
        props: {
          ...baseProps,
          projects: [{
            id: '100',
            name: 'Demo Project',
            git_repos: [{ url: 'https://github.com/acme/demo.git', clone_alias: 'demo' }],
            container_image_id: 'img-1',
            server_run_template: {
              default_auto_run: true,
              platform: 'aliyun',
              region: 'cn-hangzhou',
            },
          }],
        },
        attachTo: document.body,
      })
      await flushPromises()
      await flushPromises()
      expect(wrapper.find('[data-testid="create-task-repo-oauth-bound"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="create-task-repo-oauth-bind"]').exists()).toBe(false)
      wrapper.unmount()
    })

    it('blocks create and surfaces traceId when oauth check fails', async () => {
      hoistedMocks.oauthStatus = 503
      hoistedMocks.oauthDetail = 'git-oauth unavailable'
      hoistedMocks.oauthTraceId = 'trace-oauth-check-1'
      const wrapper = await mountModal()
      const reason = wrapper.find('[data-testid="create-task-submit-blocked-reason"]')
      expect(reason.exists()).toBe(true)
      expect(reason.text()).toContain('无法检查 Git OAuth 绑定')
      expect(reason.attributes('data-traceid') || reason.attributes('data-traceId')).toBe('trace-oauth-check-1')
      expect(wrapper.find('[data-testid="create-task-submit-btn"]').attributes('disabled')).toBeDefined()
      wrapper.unmount()
    })

    it('does not oauth-block when selected project has no github/gitlab repo', async () => {
      const component = await import('./CreateTaskModal.vue')
      const wrapper = mount(component.default, {
        props: {
          ...baseProps,
          projects: [{
            id: '100',
            name: 'Plain Git',
            git_repos: ['git://example.com/plain.git'],
            container_image_id: 'img-1',
          }],
        },
        attachTo: document.body,
      })
      await flushPromises()
      await flushPromises()
      expect(wrapper.find('[data-testid="create-task-submit-blocked-reason"]').exists()).toBe(false)
      expect(wrapper.find('[data-testid="create-task-repo-oauth-bind"]').exists()).toBe(false)
      wrapper.unmount()
    })
  })
}

// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] CreateTaskModal.git-identity-gate.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { afterEach, beforeEach, describe, expect, it, vi } = await import('vitest')

  const hoistedMocks = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
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
      git_repos: ['https://git.example/demo.git'],
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

  const jsonResponse = (body, status = 200) => ({
    ok: status >= 200 && status < 300,
    status,
    headers: { get: () => null },
    json: async () => body,
  })

  describe('CreateTaskModal git identity gate', () => {
    beforeEach(() => {
      vi.clearAllMocks()
      localStorage.setItem('currentUserId', 'user-1')
      hoistedMocks.apiFetchMock.mockImplementation(async (url) => {
        const path = String(url)
        if (path.includes('/git-identities/')) {
          return jsonResponse({
            identities: [{
              id: 'gid-1',
              git_user_name: 'Ann',
              git_user_email: 'ann@example.com',
              is_default: false,
            }],
          })
        }
        if (path.includes('/user-app-connection/')) return jsonResponse({ connected: true })
        if (path.includes('/workspace-collaborators/')) return jsonResponse([])
        if (path.includes('/repo-access-check/')) return jsonResponse({ accessible: true })
        if (path.includes('/branches/')) return jsonResponse({ branches: ['main'] })
        return jsonResponse({})
      })
    })

    afterEach(() => {
      document.body.innerHTML = ''
    })

    it('hides git identity select when auto_run is off', async () => {
      const component = await import('./CreateTaskModal.vue')
      const wrapper = mount(component.default, {
        props: {
          ...baseProps,
          editingTask: makeEditingTask({ auto_run: false }),
          projects: [{
            ...baseProps.projects[0],
            server_run_template: { default_auto_run: false, platform: 'aliyun', region: 'cn-hangzhou' },
          }],
        },
        attachTo: document.body,
      })
      await flushPromises()
      await flushPromises()
      expect(wrapper.find('[data-testid="create-task-repo-git-identity-select"]').exists()).toBe(false)
      expect(wrapper.find('[data-testid="create-task-submit-blocked-reason"]').exists()).toBe(false)
      wrapper.unmount()
    })

    it('blocks create when auto_run is on and identity is not selected', async () => {
      const component = await import('./CreateTaskModal.vue')
      const wrapper = mount(component.default, {
        props: {
          ...baseProps,
          editingTask: makeEditingTask({ repo_identities: [] }),
        },
        attachTo: document.body,
      })
      await flushPromises()
      await flushPromises()
      const select = wrapper.find('[data-testid="create-task-repo-git-identity-select"]')
      expect(select.exists()).toBe(true)
      const reason = wrapper.find('[data-testid="create-task-submit-blocked-reason"]')
      expect(reason.exists()).toBe(true)
      expect(reason.text()).toContain('Git 提交身份')
      expect(wrapper.find('[data-testid="create-task-submit-btn"]').attributes('disabled')).toBeDefined()
      wrapper.unmount()
    })

    it('places git identity section after auto-run checkbox', async () => {
      const component = await import('./CreateTaskModal.vue')
      const wrapper = mount(component.default, {
        props: {
          ...baseProps,
          editingTask: makeEditingTask({
            repo_identities: [{
              repo_url: 'https://git.example/demo.git',
              git_identity_id: 'gid-1',
            }],
          }),
        },
        attachTo: document.body,
      })
      await flushPromises()
      await flushPromises()
      const autoRun = wrapper.find('[data-testid="task-auto-run-field"]')
      const identity = wrapper.find('[data-testid="create-task-git-identity-section"]')
      expect(autoRun.exists()).toBe(true)
      expect(identity.exists()).toBe(true)
      const following = autoRun.element.compareDocumentPosition(identity.element)
        & Node.DOCUMENT_POSITION_FOLLOWING
      expect(following).toBe(Node.DOCUMENT_POSITION_FOLLOWING)
      const projectHeading = wrapper.find('[data-testid="create-task-project-section-heading"]')
      expect(
        projectHeading.element.contains(identity.element),
      ).toBe(false)
      wrapper.unmount()
    })

    it('places repo oauth row after auto-run checkbox', async () => {
      const component = await import('./CreateTaskModal.vue')
      const wrapper = mount(component.default, {
        props: {
          ...baseProps,
          projects: [{
            ...baseProps.projects[0],
            git_repos: ['https://gitlab.example/demo.git'],
          }],
        },
        attachTo: document.body,
      })
      await flushPromises()
      await flushPromises()
      const autoRun = wrapper.find('[data-testid="task-auto-run-field"]')
      const oauthSection = wrapper.find('[data-testid="create-task-repo-oauth-section"]')
      expect(autoRun.exists()).toBe(true)
      expect(oauthSection.exists()).toBe(true)
      const following = autoRun.element.compareDocumentPosition(oauthSection.element)
        & Node.DOCUMENT_POSITION_FOLLOWING
      expect(following).toBe(Node.DOCUMENT_POSITION_FOLLOWING)
      const projectHeading = wrapper.find('[data-testid="create-task-project-section-heading"]')
      expect(projectHeading.element.contains(oauthSection.element)).toBe(false)
      wrapper.unmount()
    })

    it('allows create when auto_run is on and identity is selected', async () => {
      const component = await import('./CreateTaskModal.vue')
      const wrapper = mount(component.default, {
        props: {
          ...baseProps,
          editingTask: makeEditingTask({
            repo_identities: [{
              repo_url: 'https://git.example/demo.git',
              git_identity_id: 'gid-1',
            }],
          }),
        },
        attachTo: document.body,
      })
      await flushPromises()
      await flushPromises()
      expect(wrapper.find('[data-testid="create-task-repo-git-identity-select"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="create-task-submit-blocked-reason"]').exists()).toBe(false)
      expect(wrapper.find('[data-testid="create-task-submit-btn"]').attributes('disabled')).toBeUndefined()
      wrapper.unmount()
    })
  })
}

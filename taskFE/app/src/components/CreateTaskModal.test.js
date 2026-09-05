// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] CreateTaskModal.test.js requires vitest runtime')
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

  vi.mock('../composables/useCreateTaskRepoOAuth.js', async () => {
    const { computed, ref } = await import('vue')
    return {
      useCreateTaskRepoOAuth: () => ({
        blockedReason: computed(() => ''),
        blockedTraceId: computed(() => ''),
        boundByUrl: ref({}),
        loadingByUrl: ref({}),
        errorByUrl: ref({}),
        errorTraceIdByUrl: ref({}),
      }),
    }
  })

  vi.mock('./ProjectRepoAccessHintIcon.vue', () => ({
    default: { template: '<span />' },
  }))

  const makeEditingTask = (overrides = {}) => ({
    title: '测试任务',
    description: '',
    task_kind: '',
    taskBackground: '',
    currentProblem: '',
    requestedChanges: '',
    targetLocation: '',
    preserveBehavior: '',
    forbidNewDeps: '',
    suggestedVerification: '',
    progressColumn: { id: 1 },
    task_type: { id: 'type-1' },
    container_image: { id: 'img-1' },
    auto_run: false,
    priority: '1',
    due_date: '',
    owner: '',
    assignees: [],
    workBranchPreset: 'feature',
    workBranchName: 'feature/2026-07-08_user_daydaymoney${taskId}_task',
    mergeTargetPreset: 'custom',
    mergeTargetName: '',
    projectSelections: [{
      selectionRowKey: 'row-1',
      projectId: '',
      repoBranches: [],
    }],
    ...overrides,
  })

  const allEnabledFieldSettings = () => ({
    description: true,
    task_kind: true,
    code_lang: true,
    structured_fields: true,
    project_branch: true,
    container_image: true,
    feature_params: true,
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
      git_repos: ['http://localhost:8012/group/sdk.git'],
      container_image_id: 'img-1',
    }],
    installedImages: [{ id: 'img-1', name: 'ubuntu', version: '22.04' }],
    isProjectsLoading: false,
    projectsError: null,
    isInstalledImagesLoading: false,
    installedImagesError: null,
    isTaskTypesLoading: false,
    taskTypesError: null,
    currentWorkspace: { id: 'ws-1' },
    tenantId: 't1',
    companyUserName: 'tester',
    // 模态用例显式全开，与产品默认（code_lang/structured_fields 关）解耦
    fieldSettings: allEnabledFieldSettings(),
  }

  const mountOpenedModal = async () => {
    const component = await import('./CreateTaskModal.vue')
    const wrapper = mount(component.default, {
      props: {
        ...baseProps,
        show: false,
        editingTask: makeEditingTask(),
      },
    })
    await flushPromises()
    await wrapper.setProps({
      show: true,
      editingTask: makeEditingTask({
        projectSelections: [{
          selectionRowKey: 'row-1',
          projectId: '100',
          repoBranches: [{ repoIndex: 0, baseBranch: '' }],
        }],
      }),
    })
    await flushPromises()
    await flushPromises()
    return wrapper
  }

  const jsonResponse = (body, status = 200) => ({
    ok: status >= 200 && status < 300,
    status,
    json: async () => body,
  })

  describe('CreateTaskModal branch datalist', () => {
    beforeEach(() => {
      vi.clearAllMocks()
      hoistedMocks.apiFetchMock.mockImplementation(async (url) => {
        if (String(url).includes('/workspace-collaborators/')) {
          return jsonResponse([])
        }
        if (String(url).includes('/repo-access-check/')) {
          return jsonResponse({ accessible: true })
        }
        if (String(url).includes('/branches/')) {
          return jsonResponse({ branches: ['main', 'develop'] })
        }
        return jsonResponse({})
      })
    })

    it('新建任务在共有分支加载后按 develop→release/*→main 填默认目标分支', async () => {
      const wrapper = await mountOpenedModal()
      expect(wrapper.props('editingTask').mergeTargetName).toBe('develop')
      wrapper.unmount()
    })

    it('项目选择应排在工作分支/目标分支之上', async () => {
      const wrapper = await mountOpenedModal()
      const projectSelect = wrapper.find('#task-project-row-1')
      const workBranch = wrapper.find('#work-branch-name')
      const mergeTarget = wrapper.find('#merge-target-name')
      expect(projectSelect.exists()).toBe(true)
      expect(workBranch.exists()).toBe(true)
      expect(mergeTarget.exists()).toBe(true)
      const projectPos = projectSelect.element.compareDocumentPosition(workBranch.element)
      const workBeforeMerge = workBranch.element.compareDocumentPosition(mergeTarget.element)
      expect(projectPos & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
      expect(workBeforeMerge & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
      wrapper.unmount()
    })

    it('共有分支无 develop/release/main 时目标分支保持为空', async () => {
      hoistedMocks.apiFetchMock.mockImplementation(async (url) => {
        if (String(url).includes('/workspace-collaborators/')) {
          return jsonResponse([])
        }
        if (String(url).includes('/repo-access-check/')) {
          return jsonResponse({ accessible: true })
        }
        if (String(url).includes('/branches/')) {
          return jsonResponse({ branches: ['feature/only', 'hotfix/x'] })
        }
        return jsonResponse({})
      })
      const wrapper = await mountOpenedModal()
      expect(wrapper.props('editingTask').mergeTargetName).toBe('')
      wrapper.unmount()
    })

    it('从目标分支 datalist 选中共有分支后应失焦并暂时摘掉 list', async () => {
      const blurSpy = vi.spyOn(HTMLInputElement.prototype, 'blur')
      const wrapper = await mountOpenedModal()

      const mergeInput = wrapper.find('#merge-target-name')
      expect(mergeInput.exists()).toBe(true)
      expect(mergeInput.attributes('list')).toBe('merge-target-preset-options')

      const datalistOptions = wrapper.findAll('#merge-target-preset-options option').map((n) => n.attributes('value'))
      expect(datalistOptions).toEqual(expect.arrayContaining(['main', 'develop']))

      await mergeInput.setValue('main')
      await mergeInput.trigger('change')
      await flushPromises()

      expect(blurSpy).toHaveBeenCalled()
      expect(mergeInput.attributes('list')).toBeUndefined()
      blurSpy.mockRestore()
    })

    it('多仓库时目标分支 datalist 仅展示共有分支', async () => {
      hoistedMocks.apiFetchMock.mockImplementation(async (url) => {
        if (String(url).includes('/workspace-collaborators/')) {
          return jsonResponse([])
        }
        if (String(url).includes('/repo-access-check/')) {
          return jsonResponse({ accessible: true })
        }
        if (String(url).includes('/branches/')) {
          if (String(url).includes(encodeURIComponent('http://localhost:8012/group/sdk.git'))) {
            return jsonResponse({ branches: ['main', 'develop', 'only-a'] })
          }
          return jsonResponse({ branches: ['main', 'develop', 'only-b'] })
        }
        return jsonResponse({})
      })

      const component = await import('./CreateTaskModal.vue')
      const wrapper = mount(component.default, {
        props: {
          ...baseProps,
          show: false,
          editingTask: makeEditingTask(),
          projects: [{
            id: '100',
            name: 'Demo Project',
            git_repos: [
              'http://localhost:8012/group/sdk.git',
              'http://localhost:8012/group/app.git',
            ],
            container_image_id: 'img-1',
          }],
        },
      })
      await flushPromises()
      await wrapper.setProps({
        show: true,
        editingTask: makeEditingTask({
          projectSelections: [{
            selectionRowKey: 'row-1',
            projectId: '100',
            repoBranches: [
              { repoIndex: 0, baseBranch: '' },
              { repoIndex: 1, baseBranch: '' },
            ],
          }],
        }),
      })
      await flushPromises()
      await flushPromises()

      const datalistOptions = wrapper.findAll('#merge-target-preset-options option').map((n) => n.attributes('value'))
      expect(datalistOptions).toEqual(['develop', 'main'])
      expect(datalistOptions).not.toContain('only-a')
      expect(datalistOptions).not.toContain('only-b')
      wrapper.unmount()
    })

    it('从基准分支 datalist 选中候选后应失焦并暂时摘掉 list', async () => {
      const blurSpy = vi.spyOn(HTMLInputElement.prototype, 'blur')
      const removeAttrSpy = vi.spyOn(HTMLInputElement.prototype, 'removeAttribute')
      const wrapper = await mountOpenedModal()

      const baseInput = wrapper.find('input[id^="task-base-branch-"]')
      expect(baseInput.exists()).toBe(true)
      expect(baseInput.attributes('list')).toMatch(/^task-base-branch-options-/)

      await baseInput.setValue('develop')
      await baseInput.trigger('change')
      await flushPromises()

      expect(blurSpy).toHaveBeenCalled()
      expect(removeAttrSpy).toHaveBeenCalledWith('list')
      expect(baseInput.attributes('list')).toBeUndefined()

      await baseInput.trigger('focus')
      await flushPromises()
      expect(baseInput.attributes('list')).toMatch(/^task-base-branch-options-/)

      blurSpy.mockRestore()
      removeAttrSpy.mockRestore()
    })

    it('datalist insertReplacementText 的 input 事件也应立即摘掉 list', async () => {
      const blurSpy = vi.spyOn(HTMLInputElement.prototype, 'blur')
      const wrapper = await mountOpenedModal()
      const baseInput = wrapper.find('input[id^="task-base-branch-"]')

      baseInput.element.value = 'develop'
      await baseInput.trigger('input', { inputType: 'insertReplacementText' })
      await flushPromises()

      expect(blurSpy).toHaveBeenCalled()
      expect(baseInput.attributes('list')).toBeUndefined()
      blurSpy.mockRestore()
    })

    it('手动输入非候选分支时不应触发失焦', async () => {
      const blurSpy = vi.spyOn(HTMLInputElement.prototype, 'blur')
      const wrapper = await mountOpenedModal()

      const mergeInput = wrapper.find('#merge-target-name')
      await mergeInput.setValue('my-custom-branch')
      await mergeInput.trigger('change')
      await flushPromises()

      expect(blurSpy).not.toHaveBeenCalled()
      blurSpy.mockRestore()
    })
  })

  describe('CreateTaskModal feature params selector', () => {
    beforeEach(() => {
      vi.clearAllMocks()
      hoistedMocks.apiFetchMock.mockImplementation(async (url) => {
        if (String(url).includes('/workspace-collaborators/')) {
          return jsonResponse([])
        }
        if (String(url).includes('/repo-access-check/')) {
          return jsonResponse({ accessible: true })
        }
        if (String(url).includes('/branches/')) {
          return jsonResponse({ branches: ['main', 'develop'] })
        }
        if (String(url).includes('/personal/feature-params-configs/')) {
          return jsonResponse({ configs: [{ id: 'cfg-1', name: '我的配置' }] })
        }
        return jsonResponse({})
      })
    })

    it('创建弹窗不渲染已安装镜像字段，智能体资源配置选择器正常渲染', async () => {
      const wrapper = await mountOpenedModal()
      // 已安装镜像独立字段已下线（镜像/技能选择收敛到任务描述 @ 弹层）
      expect(wrapper.find('#task-container-image').exists()).toBe(false)
      expect(wrapper.find('label[for="task-container-image"]').exists()).toBe(false)
      const sourceSelect = wrapper.find('[data-testid="feature-params-source-selector"]')
      expect(sourceSelect.exists()).toBe(true)
      expect(wrapper.find('[data-testid="feature-params-block"]').text()).toContain('智能体资源配置')
      expect(wrapper.find('[data-testid="feature-params-block"]').text()).not.toContain('环境变量参数')
      const blockHtml = wrapper.find('[data-testid="feature-params-block"]').element.outerHTML
      expect(blockHtml).toContain('feature-params-source-selector')
      wrapper.unmount()
    })

    it('新建任务选个人配置时展示二级下拉且无预览按钮', async () => {
      const wrapper = await mountOpenedModal()
      const sourceSelect = wrapper.find('[data-testid="feature-params-source-selector"]')
      await sourceSelect.setValue('personal')
      await flushPromises()
      expect(wrapper.props('editingTask').feature_params_source).toBe('personal')
      expect(wrapper.find('[data-testid="feature-params-personal-config-selector"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="feature-params-env-preview-btn"]').exists()).toBe(false)
      wrapper.unmount()
    })

    it('编辑任务回填 workspace 来源', async () => {
      const wrapper = await mountOpenedModal()
      const editing = {
        ...wrapper.props('editingTask'),
        id: 'task-9',
        feature_params_source: 'workspace',
        personal_feature_params_config_id: '',
      }
      await wrapper.setProps({ editingTask: editing })
      await flushPromises()
      const sourceSelect = wrapper.find('[data-testid="feature-params-source-selector"]')
      expect(sourceSelect.element.value).toBe('workspace')
      expect(wrapper.find('[data-testid="feature-params-env-preview-btn"]').exists()).toBe(false)
      wrapper.unmount()
    })

    it('未选智能体资源配置时创建按钮禁用并展示原因', async () => {
      const wrapper = await mountOpenedModal()
      const submitBtn = wrapper.find('[data-testid="create-task-submit-btn"]')
      expect(submitBtn.exists()).toBe(true)
      expect(submitBtn.attributes('disabled')).toBeDefined()
      expect(wrapper.find('[data-testid="create-task-submit-blocked-reason"]').text())
        .toContain('请先选择智能体资源配置后再创建')
      wrapper.unmount()
    })

    it('选中智能体资源配置后创建按钮可点击', async () => {
      const wrapper = await mountOpenedModal()
      await wrapper.find('[data-testid="feature-params-source-selector"]').setValue('company')
      await flushPromises()
      const submitBtn = wrapper.find('[data-testid="create-task-submit-btn"]')
      expect(submitBtn.attributes('disabled')).toBeUndefined()
      expect(wrapper.find('[data-testid="create-task-submit-blocked-reason"]').exists()).toBe(false)
      wrapper.unmount()
    })

    it('公司/工作空间/个人环境变量均未设置时选择器禁用', async () => {
      hoistedMocks.apiFetchMock.mockImplementation(async (url) => {
        if (String(url).includes('/workspace-collaborators/')) return jsonResponse([])
        if (String(url).includes('/repo-access-check/')) return jsonResponse({ accessible: true })
        if (String(url).includes('/branches/')) return jsonResponse({ branches: ['main'] })
        if (String(url).includes('/personal/feature-params-configs/')) return jsonResponse({ configs: [] })
        if (String(url).includes('/api/cloud/feature-params/')) {
          return jsonResponse({
            env_var_sources_available: { company: false, workspace: false },
          })
        }
        return jsonResponse({})
      })
      const wrapper = await mountOpenedModal()
      const sourceSelect = wrapper.find('[data-testid="feature-params-source-selector"]')
      expect(sourceSelect.attributes('disabled')).toBeDefined()
      expect(wrapper.find('[data-testid="feature-params-source-unavailable-hint"]').exists()).toBe(true)
      wrapper.unmount()
    })

    it('公司环境变量存在时选择器可点', async () => {
      hoistedMocks.apiFetchMock.mockImplementation(async (url) => {
        if (String(url).includes('/workspace-collaborators/')) return jsonResponse([])
        if (String(url).includes('/repo-access-check/')) return jsonResponse({ accessible: true })
        if (String(url).includes('/branches/')) return jsonResponse({ branches: ['main'] })
        if (String(url).includes('/personal/feature-params-configs/')) return jsonResponse({ configs: [] })
        if (String(url).includes('/api/cloud/feature-params/')) {
          return jsonResponse({
            env_var_sources_available: { company: true, workspace: false },
          })
        }
        return jsonResponse({})
      })
      const wrapper = await mountOpenedModal()
      expect(wrapper.find('[data-testid="feature-params-source-selector"]').attributes('disabled'))
        .toBeUndefined()
      wrapper.unmount()
    })

    it('工作空间自定义环境变量存在时选择器可点', async () => {
      hoistedMocks.apiFetchMock.mockImplementation(async (url) => {
        if (String(url).includes('/workspace-collaborators/')) return jsonResponse([])
        if (String(url).includes('/repo-access-check/')) return jsonResponse({ accessible: true })
        if (String(url).includes('/branches/')) return jsonResponse({ branches: ['main'] })
        if (String(url).includes('/personal/feature-params-configs/')) return jsonResponse({ configs: [] })
        if (String(url).includes('/api/cloud/feature-params/')) {
          return jsonResponse({
            env_var_sources_available: { company: false, workspace: true },
          })
        }
        return jsonResponse({})
      })
      const wrapper = await mountOpenedModal()
      expect(wrapper.find('[data-testid="feature-params-source-selector"]').attributes('disabled'))
        .toBeUndefined()
      wrapper.unmount()
    })

    it('仅个人配置存在时选择器仍可点', async () => {
      hoistedMocks.apiFetchMock.mockImplementation(async (url) => {
        if (String(url).includes('/workspace-collaborators/')) return jsonResponse([])
        if (String(url).includes('/repo-access-check/')) return jsonResponse({ accessible: true })
        if (String(url).includes('/branches/')) return jsonResponse({ branches: ['main'] })
        if (String(url).includes('/personal/feature-params-configs/')) {
          return jsonResponse({ configs: [{ id: 'cfg-1', name: '我的配置' }] })
        }
        if (String(url).includes('/api/cloud/feature-params/')) {
          return jsonResponse({
            env_var_sources_available: { company: false, workspace: false },
          })
        }
        return jsonResponse({})
      })
      const wrapper = await mountOpenedModal()
      expect(wrapper.find('[data-testid="feature-params-source-selector"]').attributes('disabled'))
        .toBeUndefined()
      wrapper.unmount()
    })

    it('提交时 editingTask 保留所选 feature_params_source', async () => {
      const wrapper = await mountOpenedModal()
      await wrapper.find('[data-testid="feature-params-source-selector"]').setValue('company')
      await flushPromises()
      const createBtn = wrapper.find('[data-testid="create-task-submit-btn"]')
      expect(createBtn.attributes('disabled')).toBeUndefined()
      await createBtn.trigger('click')
      await flushPromises()
      const submitPayload = wrapper.emitted('submit')?.[0]?.[0]
      expect(submitPayload?.feature_params_source).toBe('company')
      wrapper.unmount()
    })
  })

  describe('CreateTaskModal parent deliverable', () => {
    beforeEach(() => {
      vi.clearAllMocks()
      hoistedMocks.apiFetchMock.mockImplementation(async (url) => {
        if (String(url).includes('/workspace-collaborators/')) return jsonResponse([])
        if (String(url).includes('/repo-access-check/')) return jsonResponse({ accessible: true })
        if (String(url).includes('/branches/')) return jsonResponse({ branches: ['main'] })
        return jsonResponse({})
      })
    })

    const mountParentCase = async (taskTypeId, parentTask = '') => {
      const component = await import('./CreateTaskModal.vue')
      const taskTypes = [
        { id: 'c1', name: '价值流', order: 1 },
        { id: 'c2', name: '业务流程', order: 2 },
      ]
      const wrapper = mount(component.default, {
        props: {
          ...baseProps,
          show: false,
          taskTypes,
          todos: [{ id: 'p1', title: '价值流任务', deliverable_obj_id: 'c1' }],
          editingTask: makeEditingTask(),
        },
      })
      await flushPromises()
      await wrapper.setProps({
        show: true,
        editingTask: makeEditingTask({
          task_type: { id: taskTypeId },
          parent_task: parentTask,
          feature_params_source: 'company',
          projectSelections: [{
            selectionRowKey: 'row-1',
            projectId: '100',
            repoBranches: [{ repoIndex: 0, baseBranch: 'main' }],
          }],
        }),
      })
      await flushPromises()
      await flushPromises()
      return wrapper
    }

    it('顶层类别不展示上层交付物字段', async () => {
      const wrapper = await mountParentCase('c1')
      expect(wrapper.find('[data-testid="create-task-parent-deliverable"]').exists()).toBe(false)
      wrapper.unmount()
    })

    it('非顶层类别展示上层交付物且未选时禁用创建', async () => {
      const wrapper = await mountParentCase('c2')
      const parentField = wrapper.find('[data-testid="create-task-parent-deliverable"]')
      expect(parentField.exists()).toBe(true)
      expect(parentField.text()).toContain('上层交付物')
      expect(wrapper.find('[data-testid="create-task-submit-btn"]').attributes('disabled')).toBeDefined()
      expect(wrapper.find('[data-testid="create-task-submit-blocked-reason"]').text()).toContain('上层交付物')
      wrapper.unmount()
    })

    it('编辑模式非顶层类别同样展示上层交付物', async () => {
      const wrapper = await mountParentCase('c2')
      await wrapper.setProps({
        editingTask: makeEditingTask({
          id: 'edit-1',
          task_type: { id: 'c2' },
          parent_task: '',
          feature_params_source: 'company',
          projectSelections: [{
            selectionRowKey: 'row-1',
            projectId: '100',
            repoBranches: [{ repoIndex: 0, baseBranch: 'main' }],
          }],
        }),
      })
      await flushPromises()
      expect(wrapper.find('[data-testid="create-task-parent-deliverable"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="create-task-submit-btn"]').attributes('disabled')).toBeDefined()
      wrapper.unmount()
    })
  })

  describe('CreateTaskModal base branch commit check', () => {
    beforeEach(() => {
      vi.clearAllMocks()
      vi.useFakeTimers()
      hoistedMocks.apiFetchMock.mockImplementation(async (url) => {
        if (String(url).includes('/workspace-collaborators/')) {
          return jsonResponse([])
        }
        if (String(url).includes('/repo-access-check/')) {
          return jsonResponse({ accessible: true })
        }
        if (String(url).includes('/branches/')) {
          return jsonResponse({ branches: ['main', 'develop'] })
        }
        if (String(url).includes('/resolve-ref/')) {
          return jsonResponse({
            exists: true,
            sha: 'abcdef0123456789abcdef0123456789abcdef01',
          })
        }
        return jsonResponse({})
      })
    })

    afterEach(() => {
      vi.useRealTimers()
    })

    it('输入存在的 commit hash 后在基准分支 label 旁显示打勾', async () => {
      const wrapper = await mountOpenedModal()
      const input = wrapper.find('#task-base-branch-0-0')
      expect(input.exists()).toBe(true)
      expect(wrapper.find('[data-testid="base-branch-commit-ok"]').exists()).toBe(false)

      await input.setValue('abcdef0')
      await input.trigger('input')
      await vi.advanceTimersByTimeAsync(450)
      await flushPromises()

      expect(hoistedMocks.apiFetchMock.mock.calls.some((c) => String(c[0]).includes('/resolve-ref/'))).toBe(true)
      expect(wrapper.find('[data-testid="base-branch-commit-ok"]').exists()).toBe(true)
      wrapper.unmount()
    })

    it('输入普通分支名不显示打勾且不请求 resolve-ref', async () => {
      const wrapper = await mountOpenedModal()
      const input = wrapper.find('#task-base-branch-0-0')
      await input.setValue('main')
      await input.trigger('input')
      await vi.advanceTimersByTimeAsync(450)
      await flushPromises()

      expect(hoistedMocks.apiFetchMock.mock.calls.some((c) => String(c[0]).includes('/resolve-ref/'))).toBe(false)
      expect(wrapper.find('[data-testid="base-branch-commit-ok"]').exists()).toBe(false)
      wrapper.unmount()
    })

    it('校验中显示 spinner，不存在时显示弱提示并带 data-traceId', async () => {
      hoistedMocks.apiFetchMock.mockImplementation(async (url) => {
        if (String(url).includes('/workspace-collaborators/')) {
          return jsonResponse([])
        }
        if (String(url).includes('/repo-access-check/')) {
          return jsonResponse({ accessible: true })
        }
        if (String(url).includes('/branches/')) {
          return jsonResponse({ branches: ['main', 'develop'] })
        }
        if (String(url).includes('/resolve-ref/')) {
          return {
            ok: true,
            status: 200,
            traceId: 'tid-commit-missing',
            json: async () => ({ exists: false }),
          }
        }
        return jsonResponse({})
      })

      const wrapper = await mountOpenedModal()
      const input = wrapper.find('#task-base-branch-0-0')
      await input.setValue('deadbee')
      await input.trigger('input')
      await flushPromises()
      expect(wrapper.find('[data-testid="base-branch-commit-checking"]').exists()).toBe(true)

      await vi.advanceTimersByTimeAsync(450)
      await flushPromises()

      const missing = wrapper.find('[data-testid="base-branch-commit-missing"]')
      expect(missing.exists()).toBe(true)
      expect(missing.text()).toContain('未找到该 commit')
      expect(missing.attributes('data-traceid') || missing.attributes('data-traceId')).toBe('tid-commit-missing')
      expect(wrapper.find('[data-testid="base-branch-commit-ok"]').exists()).toBe(false)
      wrapper.unmount()
    })
  })

  describe('CreateTaskModal structured description fields', () => {
    beforeEach(() => {
      vi.clearAllMocks()
      hoistedMocks.apiFetchMock.mockImplementation(async (url) => {
        if (String(url).includes('/workspace-collaborators/')) return jsonResponse([])
        if (String(url).includes('/repo-access-check/')) return jsonResponse({ accessible: true })
        if (String(url).includes('/branches/')) return jsonResponse({ branches: ['main', 'develop'] })
        return jsonResponse({})
      })
    })

    it('创建弹窗展示任务类型下拉默认选项', async () => {
      const component = await import('./CreateTaskModal.vue')
      const wrapper = mount(component.default, {
        props: {
          ...baseProps,
          show: true,
          editingTask: makeEditingTask(),
          taskKindOptions: ['bug-fix', 'feature'],
        },
      })
      await flushPromises()
      const select = wrapper.find('[data-testid="create-task-field-taskKind"]')
      expect(select.exists()).toBe(true)
      const values = select.findAll('option').map((o) => o.element.value)
      expect(values).toEqual(expect.arrayContaining(['', 'bug-fix', 'feature']))
      wrapper.unmount()
    })

    it('T7 创建弹窗展示结构化可选字段标签', async () => {
      const wrapper = await mountOpenedModal()
      expect(wrapper.find('[data-testid="create-task-structured-fields"]').exists()).toBe(true)
      const text = wrapper.find('[data-testid="create-task-structured-fields"]').text()
      expect(text).toContain('任务背景')
      expect(text).toContain('当前问题是什么')
      expect(text).toContain('需要模型完成什么修改')
      expect(text).toContain('目标文件、模块或命令在哪里')
      expect(text).toContain('需要保持哪些原有行为不变')
      expect(text).toContain('是否禁止新增依赖')
      expect(text).toContain('建议验证方式')
      wrapper.unmount()
    })

    it('提交时将结构化字段拼入 description', async () => {
      const wrapper = await mountOpenedModal()
      await wrapper.find('[data-testid="feature-params-source-selector"]').setValue('company')
      await wrapper.find('[data-testid="create-task-field-taskBackground"]').setValue('背景X')
      await wrapper.find('[data-testid="create-task-field-currentProblem"]').setValue('问题Y')
      await wrapper.find('[data-testid="create-task-field-forbidNewDeps"]').setValue('yes')
      await flushPromises()
      await wrapper.find('[data-testid="create-task-submit-btn"]').trigger('click')
      await flushPromises()
      const payload = wrapper.emitted('submit')?.[0]?.[0]
      expect(payload?.description).toContain('## 任务背景\n背景X')
      expect(payload?.description).toContain('## 当前问题是什么\n问题Y')
      expect(payload?.description).toContain('## 是否禁止新增依赖\n是')
      wrapper.unmount()
    })

    it('编辑打开时从 description 回填结构化字段', async () => {
      const component = await import('./CreateTaskModal.vue')
      const md = [
        '补充说明',
        '',
        '## 任务背景',
        '旧背景',
        '',
        '## 是否禁止新增依赖',
        '否',
      ].join('\n')
      const wrapper = mount(component.default, {
        props: {
          ...baseProps,
          show: false,
          editingTask: makeEditingTask({
            id: 'task-structured-1',
            description: md,
            feature_params_source: 'company',
          }),
        },
      })
      await flushPromises()
      await wrapper.setProps({ show: true })
      await flushPromises()
      expect(wrapper.props('editingTask').description).toBe('补充说明')
      expect(wrapper.props('editingTask').taskBackground).toBe('旧背景')
      expect(wrapper.props('editingTask').forbidNewDeps).toBe('no')
      wrapper.unmount()
    })
  })

  describe('CreateTaskModal field settings visibility', () => {
    beforeEach(() => {
      vi.clearAllMocks()
      hoistedMocks.apiFetchMock.mockImplementation(async (url) => {
        if (String(url).includes('/workspace-collaborators/')) {
          return jsonResponse([])
        }
        if (String(url).includes('/repo-access-check/')) {
          return jsonResponse({ accessible: true })
        }
        if (String(url).includes('/branches/')) {
          return jsonResponse({ branches: ['main'] })
        }
        if (String(url).includes('/personal-feature-params')) {
          return jsonResponse([])
        }
        return jsonResponse({})
      })
    })

    afterEach(() => {
      vi.clearAllMocks()
    })

    it('隐藏 priority/due_date 时不渲染对应控件', async () => {
      const { defaultCreateTaskFieldSettings } = await import('../utils/createTaskFieldSettings.js')
      const component = await import('./CreateTaskModal.vue')
      const fields = defaultCreateTaskFieldSettings()
      fields.priority = false
      fields.due_date = false
      const wrapper = mount(component.default, {
        props: {
          ...baseProps,
          fieldSettings: fields,
          editingTask: makeEditingTask({ feature_params_source: 'company' }),
        },
      })
      await flushPromises()
      expect(wrapper.find('#task-priority').exists()).toBe(false)
      expect(wrapper.find('#task-deadline').exists()).toBe(false)
      expect(wrapper.find('#task-title').exists()).toBe(true)
      wrapper.unmount()
    })

    it('关闭 feature_params 时创建不再因环境变量禁用提交', async () => {
      const { defaultCreateTaskFieldSettings } = await import('../utils/createTaskFieldSettings.js')
      const component = await import('./CreateTaskModal.vue')
      const fields = defaultCreateTaskFieldSettings()
      fields.feature_params = false
      const wrapper = mount(component.default, {
        props: {
          ...baseProps,
          fieldSettings: fields,
          editingTask: makeEditingTask({ feature_params_source: '' }),
        },
      })
      await flushPromises()
      expect(wrapper.find('[data-testid="create-task-submit-btn"]').attributes('disabled')).toBeUndefined()
      expect(wrapper.find('[data-testid="create-task-submit-blocked-reason"]').exists()).toBe(false)
      wrapper.unmount()
    })
  })
}

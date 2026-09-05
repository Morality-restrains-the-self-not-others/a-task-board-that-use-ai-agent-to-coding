// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] taskDetailEditing.test.js requires vitest runtime')
} else {
  const { beforeEach, describe, expect, it, vi } = await import('vitest')
  const { ref } = await import('vue')
  const {
    buildTaskDetailRouteQuery,
    forkTask,
    openTaskDetailInNewTab,
    resolveForkProgressColumnId,
    saveEdit,
  } = await import('./taskDetailEditing.js')

  vi.mock('../../utils/apiUtils.js', () => ({
    apiFetch: vi.fn(),
  }))
  vi.mock('../../utils/requestErrorDisplay.js', () => ({
    showRequestError: vi.fn(),
    humanizeRequestErrorMessage: (m) => String(m ?? ''),
  }))
  vi.mock('../../utils/publicClientIp.js', () => ({
    queryClientPublicIpForAutoSg: vi.fn(async () => '203.0.113.10'),
  }))

  const { apiFetch } = await import('../../utils/apiUtils.js')
  const { queryClientPublicIpForAutoSg } = await import('../../utils/publicClientIp.js')
  const { showRequestError } = await import('../../utils/requestErrorDisplay.js')

  describe('buildTaskDetailRouteQuery', () => {
    it('preserves relayToTrae and accessCode', () => {
      expect(buildTaskDetailRouteQuery({
        accessCode: 'u123',
        relayToTrae: 'true',
        unrelated: 'drop-me',
      })).toEqual({
        accessCode: 'u123',
        relayToTrae: 'true',
      })
    })
  })

  describe('resolveForkProgressColumnId', () => {
    it('returns first option id from array or ref', () => {
      expect(resolveForkProgressColumnId([
        { id: 'col-todo', name: '待处理' },
        { id: 'col-wip', name: '进行中' },
      ])).toBe('col-todo')
      expect(resolveForkProgressColumnId(ref([{ id: 0, name: '待处理' }]))).toBe('0')
    })

    it('returns empty string when options missing', () => {
      expect(resolveForkProgressColumnId([])).toBe('')
      expect(resolveForkProgressColumnId(ref([]))).toBe('')
      expect(resolveForkProgressColumnId(undefined)).toBe('')
    })
  })

  describe('openTaskDetailInNewTab', () => {
    beforeEach(() => {
      vi.stubGlobal('window', {
        location: { origin: 'http://127.0.0.1:4000' },
        open: vi.fn(() => ({ closed: false })),
      })
    })

    it('opens resolved task detail href in a new tab', () => {
      const router = {
        resolve: vi.fn(() => ({ href: '/tenant/t1/workspace/w1/task-detail/task-2/?relayToTrae=true' })),
      }
      openTaskDetailInNewTab(router, {
        tenantId: 't1',
        workspaceId: 'w1',
        taskId: 'task-2',
        query: { relayToTrae: 'true' },
      })
      expect(window.open).toHaveBeenCalledWith(
        'http://127.0.0.1:4000/tenant/t1/workspace/w1/task-detail/task-2/?relayToTrae=true',
        '_blank',
        'noopener,noreferrer',
      )
    })
  })

  describe('forkTask', () => {
    beforeEach(() => {
      vi.clearAllMocks()
      queryClientPublicIpForAutoSg.mockResolvedValue('203.0.113.10')
      vi.stubGlobal('window', {
        alert: vi.fn(),
        location: { origin: 'http://127.0.0.1:4000' },
        open: vi.fn(() => ({ closed: false })),
      })
    })

    function baseForkDeps(overrides = {}) {
      return {
        effectiveTenantId: ref('t1'),
        effectiveWorkspaceId: ref('w1'),
        effectiveTaskId: ref('source-task'),
        localTask: ref({
          title: 'fork me',
          owner: '827923618472235008',
          assignees: [],
          progress_column_id: 'col-wip',
          projects: [{ project_id: 'p1', repo_index: 0, base_branch: 'dev', target_branch: 'feature/x' }],
        }),
        progressStatusOptions: ref([
          { id: 'col-todo', name: '待处理' },
          { id: 'col-wip', name: '进行中' },
        ]),
        isForking: ref(false),
        router: {
          resolve: vi.fn(() => ({ href: '/tenant/t1/workspace/w1/task-detail/847741372830752768/' })),
          push: vi.fn(),
        },
        detailQuery: ref({ relayToTrae: 'true' }),
        ...overrides,
      }
    }

    it('opens forked task in new tab and clears isForking', async () => {
      apiFetch.mockResolvedValue({
        ok: true,
        json: async () => ({ id: '847741372830752768' }),
      })
      const onForked = vi.fn()
      const deps = baseForkDeps({ onForked })
      const ok = await forkTask(deps)
      expect(ok).toBe(true)
      expect(deps.isForking.value).toBe(false)
      expect(deps.router.push).not.toHaveBeenCalled()
      expect(window.open).toHaveBeenCalled()
      expect(onForked).toHaveBeenCalledWith({ id: '847741372830752768' })
    })

    it('posts auto_run false when user chooses not to auto-run', async () => {
      apiFetch.mockResolvedValue({
        ok: true,
        json: async () => ({ id: '847741372830752768' }),
      })
      await forkTask(baseForkDeps({ autoRun: false }))
      const [, opts] = apiFetch.mock.calls[0]
      const body = JSON.parse(opts.body)
      expect(body.fork_from).toBe('source-task')
      expect(body.auto_run).toBe(false)
      expect(body.client_public_ip).toBeUndefined()
      expect(queryClientPublicIpForAutoSg).not.toHaveBeenCalled()
    })

    it('posts auto_run true and client_public_ip when user chooses auto-run', async () => {
      apiFetch.mockResolvedValue({
        ok: true,
        json: async () => ({ id: '847741372830752768' }),
      })
      await forkTask(baseForkDeps({ autoRun: true }))
      const [, opts] = apiFetch.mock.calls[0]
      const body = JSON.parse(opts.body)
      expect(body.auto_run).toBe(true)
      expect(body.client_public_ip).toBe('203.0.113.10')
      expect(queryClientPublicIpForAutoSg).toHaveBeenCalled()
    })

    it('posts current-user repo_identities when auto-run fork is confirmed', async () => {
      apiFetch.mockResolvedValue({
        ok: true,
        json: async () => ({ id: '847741372830752768' }),
      })
      await forkTask(baseForkDeps({
        autoRun: true,
        repoIdentities: [{
          repo_url: 'https://gitlab-tencent-sh-1.daydaymoney.com/example-user/somanyad',
          git_identity_id: 'gid-current',
        }],
      }))
      const [, opts] = apiFetch.mock.calls[0]
      const body = JSON.parse(opts.body)
      expect(body.auto_run).toBe(true)
      expect(body.repo_identities).toEqual([{
        repo_url: 'https://gitlab-tencent-sh-1.daydaymoney.com/example-user/somanyad',
        git_identity_id: 'gid-current',
        oauth_gitsite: 'gitlab-tencent-sh-1.daydaymoney.com',
      }])
    })

    it('omits repo_identities when user chooses not to auto-run', async () => {
      apiFetch.mockResolvedValue({
        ok: true,
        json: async () => ({ id: '847741372830752768' }),
      })
      await forkTask(baseForkDeps({
        autoRun: false,
        repoIdentities: [{
          repo_url: 'https://gitlab.example/a.git',
          git_identity_id: 'gid-current',
        }],
      }))
      const [, opts] = apiFetch.mock.calls[0]
      const body = JSON.parse(opts.body)
      expect(body.auto_run).toBe(false)
      expect(body.repo_identities).toBeUndefined()
    })

    it('posts first progress column instead of source task column', async () => {
      apiFetch.mockResolvedValue({
        ok: true,
        json: async () => ({ id: '847741372830752768' }),
      })
      await forkTask(baseForkDeps())
      const [, opts] = apiFetch.mock.calls[0]
      const body = JSON.parse(opts.body)
      expect(body.progress_column_id).toBe('col-todo')
      expect(body.progress_column_id).not.toBe('col-wip')
    })

    it('omits source progress column when progress options are empty', async () => {
      apiFetch.mockResolvedValue({
        ok: true,
        json: async () => ({ id: '847741372830752768' }),
      })
      await forkTask(baseForkDeps({ progressStatusOptions: ref([]) }))
      const [, opts] = apiFetch.mock.calls[0]
      const body = JSON.parse(opts.body)
      expect(body.progress_column_id).not.toBe('col-wip')
      expect(body.progress_column_id == null || body.progress_column_id === '').toBe(true)
    })

    it('fetches progress options when empty then posts first column', async () => {
      apiFetch.mockResolvedValue({
        ok: true,
        json: async () => ({ id: '847741372830752768' }),
      })
      const progressStatusOptions = ref([])
      const fetchProgressStatusOptions = vi.fn(async () => {
        progressStatusOptions.value = [{ id: 'col-todo', name: '待处理' }]
      })
      await forkTask(baseForkDeps({ progressStatusOptions, fetchProgressStatusOptions }))
      expect(fetchProgressStatusOptions).toHaveBeenCalledTimes(1)
      const [, opts] = apiFetch.mock.calls[0]
      const body = JSON.parse(opts.body)
      expect(body.progress_column_id).toBe('col-todo')
    })

    it('sets isForking before awaiting progress fetch so double-confirm cannot double-POST', async () => {
      apiFetch.mockResolvedValue({
        ok: true,
        json: async () => ({ id: '847741372830752768' }),
      })
      const progressStatusOptions = ref([])
      let releaseFetch
      const fetchGate = new Promise((resolve) => { releaseFetch = resolve })
      const fetchProgressStatusOptions = vi.fn(async () => {
        await fetchGate
        progressStatusOptions.value = [{ id: 'col-todo', name: '待处理' }]
      })
      const deps = baseForkDeps({ progressStatusOptions, fetchProgressStatusOptions, autoRun: true })
      const first = forkTask(deps)
      await Promise.resolve()
      expect(deps.isForking.value).toBe(true)
      expect(apiFetch).not.toHaveBeenCalled()
      const secondOk = await forkTask(deps)
      expect(secondOk).toBe(false)
      releaseFetch()
      const firstOk = await first
      expect(firstOk).toBe(true)
      expect(apiFetch).toHaveBeenCalledTimes(1)
      expect(deps.isForking.value).toBe(false)
    })

    it('passes Response (with traceId) to showRequestError on fork failure', async () => {
      const { showRequestError } = await import('../../utils/requestErrorDisplay.js')
      const resp = {
        ok: false,
        status: 400,
        traceId: 'fork-tid-001',
        text: async () => JSON.stringify({
          code: 'AUTO_RUN_RUNTIME_ENV_REQUIRED',
          detail: '自动运行需要镜像配置运行环境，但所选镜像暂无可用运行环境',
          error: '自动运行需要镜像配置运行环境，但所选镜像暂无可用运行环境',
        }),
      }
      apiFetch.mockResolvedValue(resp)
      const ok = await forkTask(baseForkDeps({ autoRun: true }))
      expect(ok).toBe(false)
      expect(showRequestError).toHaveBeenCalled()
      const [msg, source] = showRequestError.mock.calls[0]
      expect(String(msg)).toContain('派生任务失败')
      expect(source).toBe(resp)
      expect(source.traceId).toBe('fork-tid-001')
    })

    it('posts per-copy agent_models when auto-run agents are provided', async () => {
      apiFetch
        .mockResolvedValueOnce({ ok: true, json: async () => ({ id: 't-a' }) })
        .mockResolvedValueOnce({ ok: true, json: async () => ({ id: 't-b' }) })
      const onForked = vi.fn()
      await forkTask(baseForkDeps({
        autoRun: true,
        batchIdempotencyKey: 'batch-1',
        agentModelProvider: 'openai',
        agents: [{ model: 'gpt-4.1' }, { model: 'gpt-4.1-mini' }],
        onForked,
      }))
      expect(apiFetch).toHaveBeenCalledTimes(2)
      const bodies = apiFetch.mock.calls.map(([, opts]) => JSON.parse(opts.body))
      expect(bodies[0].fork_count).toBeUndefined()
      expect(bodies[0].agent_models[0].model).toBe('gpt-4.1')
      expect(bodies[1].agent_models[0].model).toBe('gpt-4.1-mini')
      expect(window.open).toHaveBeenCalled()
      expect(onForked).toHaveBeenCalledWith({ id: 't-a' })
    })

    it('partial later copy failure reports created count and still opens first task', async () => {
      apiFetch
        .mockResolvedValueOnce({ ok: true, json: async () => ({ id: 't-a' }) })
        .mockResolvedValueOnce({
          ok: false,
          status: 402,
          text: async () => JSON.stringify({
            code: 'INSUFFICIENT_TASK_POST_QUOTA',
            detail: '任务帖配额不足',
          }),
        })
      const onForked = vi.fn()
      const ok = await forkTask(baseForkDeps({
        autoRun: true,
        batchIdempotencyKey: 'batch-2',
        agentModelProvider: 'openai',
        agents: [{ model: 'a' }, { model: 'b' }],
        onForked,
      }))
      expect(ok).toBe(true)
      expect(apiFetch).toHaveBeenCalledTimes(2)
      expect(String(showRequestError.mock.calls[0][0])).toContain('已派生 1/2 个副本')
      expect(window.open).toHaveBeenCalled()
      expect(onForked).toHaveBeenCalledWith({ id: 't-a' })
    })

    it('first copy failure surfaces error and does not open', async () => {
      const resp = {
        ok: false,
        status: 402,
        json: async () => ({ status: 'error', error: '任务帖配额不足', code: 'INSUFFICIENT_TASK_POST_QUOTA' }),
        text: async () => '',
      }
      apiFetch.mockResolvedValue(resp)
      const ok = await forkTask(baseForkDeps({
        autoRun: true,
        batchIdempotencyKey: 'batch-3',
        agentModelProvider: 'openai',
        agents: [{ model: 'a' }, { model: 'b' }],
      }))
      expect(ok).toBe(false)
      expect(apiFetch).toHaveBeenCalledTimes(1)
      expect(JSON.parse(apiFetch.mock.calls[0][1].body).fork_count).toBeUndefined()
      expect(String(showRequestError.mock.calls[0][0])).toContain('派生任务失败')
      expect(window.open).not.toHaveBeenCalled()
    })
  })

  describe('saveEdit', () => {
    beforeEach(() => {
      vi.clearAllMocks()
    })

    it('surfaces Go API error field when save fails', async () => {
      apiFetch.mockResolvedValue({
        ok: false,
        status: 400,
        json: async () => ({ error: 'projects[0] base_branch required' }),
      })
      const editError = ref('')
      const isSaving = ref(false)
      const editingTask = ref({
        title: 'demo',
        description: '',
        priority: 1,
        owner: '827923618472235008',
        assignees: [],
        workBranchName: 'feature/x',
        mergeTargetName: 'main',
        linkedProjects: [{
          project_id: '100',
          repo_branches: { 'http://x.git': 'main' },
        }],
      })
      await saveEdit({
        effectiveTenantId: ref('t1'),
        effectiveWorkspaceId: ref('w1'),
        effectiveTaskId: ref('task-1'),
        editingTask,
        editError,
        isSaving,
        localTask: ref({}),
        cancelEdit: vi.fn(),
        getProjectRepos: () => ['http://x.git'],
        buildProjectsApiPayload: () => [{
          project_id: '100',
          repo_index: 0,
          base_branch: 'main',
          target_branch: 'feature/x',
        }],
      })
      expect(isSaving.value).toBe(false)
      expect(editError.value).toBe('请为关联项目第 1 个仓库配置基准分支')
    })

    it('surfaces forbidden error in Chinese', async () => {
      apiFetch.mockResolvedValue({
        ok: false,
        status: 403,
        json: async () => ({ error: 'forbidden' }),
      })
      const editError = ref('')
      const isSaving = ref(false)
      const editingTask = ref({
        title: 'demo',
        description: '',
        priority: 1,
        owner: '827923618472235008',
        assignees: [],
        workBranchName: 'feature/x',
        mergeTargetName: 'main',
        linkedProjects: [],
      })
      await saveEdit({
        effectiveTenantId: ref('t1'),
        effectiveWorkspaceId: ref('w1'),
        effectiveTaskId: ref('task-1'),
        editingTask,
        editError,
        isSaving,
        localTask: ref({}),
        cancelEdit: vi.fn(),
        getProjectRepos: () => [],
        buildProjectsApiPayload: () => [],
      })
      expect(editError.value).toBe('您没有权限修改此任务')
    })

    it('PATCH body includes deliverable_obj_id', async () => {
      apiFetch.mockResolvedValue({
        ok: true,
        json: async () => ({ id: 'task-1', title: 'demo' }),
      })
      const editError = ref('')
      const isSaving = ref(false)
      const cancelEdit = vi.fn()
      const editingTask = ref({
        title: 'demo',
        description: '',
        priority: 1,
        deliverable_obj_id: 'cat-42',
        parent_task: '',
        owner: '827923618472235008',
        assignees: [],
        workBranchName: 'feature/x',
        mergeTargetName: 'main',
        linkedProjects: [],
      })
      await saveEdit({
        effectiveTenantId: ref('t1'),
        effectiveWorkspaceId: ref('w1'),
        effectiveTaskId: ref('task-1'),
        editingTask,
        editError,
        isSaving,
        localTask: ref({}),
        cancelEdit,
        getProjectRepos: () => [],
        buildProjectsApiPayload: () => [],
        deliverableCategoryOptions: ref([{ id: 'cat-42', name: '顶层', order: 1 }]),
        workspaceTodos: ref([]),
      })
      expect(apiFetch).toHaveBeenCalled()
      const [, opts] = apiFetch.mock.calls[0]
      const body = JSON.parse(opts.body)
      expect(body.deliverable_obj_id).toBe('cat-42')
      expect(body.parent_task).toBe('')
      expect(cancelEdit).toHaveBeenCalled()
    })

    it('PATCH body includes parent_task for non-top category', async () => {
      apiFetch.mockResolvedValue({
        ok: true,
        json: async () => ({ id: 'task-1', title: 'demo', parent_task: 'p1' }),
      })
      const editError = ref('')
      const isSaving = ref(false)
      const cancelEdit = vi.fn()
      const cats = [
        { id: 'c1', name: '顶层', order: 1 },
        { id: 'c2', name: '子层', order: 2 },
      ]
      const editingTask = ref({
        title: 'demo',
        description: '',
        priority: 1,
        deliverable_obj_id: 'c2',
        parent_task: 'p1',
        owner: '827923618472235008',
        assignees: [],
        workBranchName: 'feature/x',
        mergeTargetName: 'main',
        linkedProjects: [],
      })
      await saveEdit({
        effectiveTenantId: ref('t1'),
        effectiveWorkspaceId: ref('w1'),
        effectiveTaskId: ref('task-1'),
        editingTask,
        editError,
        isSaving,
        localTask: ref({}),
        cancelEdit,
        getProjectRepos: () => [],
        buildProjectsApiPayload: () => [],
        deliverableCategoryOptions: ref(cats),
        workspaceTodos: ref([{ id: 'p1', title: 'root', deliverable_obj_id: 'c1' }]),
      })
      const [, opts] = apiFetch.mock.calls[0]
      const body = JSON.parse(opts.body)
      expect(body.parent_task).toBe('p1')
      expect(cancelEdit).toHaveBeenCalled()
    })

    it('blocks save when non-top category missing parent', async () => {
      const editError = ref('')
      const isSaving = ref(false)
      const editingTask = ref({
        title: 'demo',
        description: '',
        priority: 1,
        deliverable_obj_id: 'c2',
        parent_task: '',
        owner: '827923618472235008',
        assignees: [],
        workBranchName: 'feature/x',
        mergeTargetName: 'main',
        linkedProjects: [],
      })
      await saveEdit({
        effectiveTenantId: ref('t1'),
        effectiveWorkspaceId: ref('w1'),
        effectiveTaskId: ref('task-1'),
        editingTask,
        editError,
        isSaving,
        localTask: ref({}),
        cancelEdit: vi.fn(),
        getProjectRepos: () => [],
        buildProjectsApiPayload: () => [],
        deliverableCategoryOptions: ref([
          { id: 'c1', name: '顶层', order: 1 },
          { id: 'c2', name: '子层', order: 2 },
        ]),
        workspaceTodos: ref([{ id: 'p1', title: 'root', deliverable_obj_id: 'c1' }]),
      })
      expect(apiFetch).not.toHaveBeenCalled()
      expect(editError.value).toContain('上层交付物')
      expect(isSaving.value).toBe(false)
    })

    it('calls onTaskUpdated after successful save so work-panel can re-filter', async () => {
      const updated = { id: 'task-1', title: 'demo', parent_task: 'p1', deliverable_obj_id: 'c2' }
      apiFetch.mockResolvedValue({
        ok: true,
        json: async () => updated,
      })
      const onTaskUpdated = vi.fn()
      const cancelEdit = vi.fn()
      await saveEdit({
        effectiveTenantId: ref('t1'),
        effectiveWorkspaceId: ref('w1'),
        effectiveTaskId: ref('task-1'),
        editingTask: ref({
          title: 'demo',
          description: '',
          priority: 1,
          deliverable_obj_id: 'c2',
          parent_task: 'p1',
          owner: '827923618472235008',
          assignees: [],
          workBranchName: 'feature/x',
          mergeTargetName: 'main',
          linkedProjects: [],
        }),
        editError: ref(''),
        isSaving: ref(false),
        localTask: ref({}),
        cancelEdit,
        getProjectRepos: () => [],
        buildProjectsApiPayload: () => [],
        deliverableCategoryOptions: ref([
          { id: 'c1', name: '顶层', order: 1 },
          { id: 'c2', name: '子层', order: 2 },
        ]),
        workspaceTodos: ref([{ id: 'p1', title: 'root', deliverable_obj_id: 'c1' }]),
        onTaskUpdated,
      })
      expect(cancelEdit).toHaveBeenCalled()
      expect(onTaskUpdated).toHaveBeenCalledTimes(1)
      expect(onTaskUpdated.mock.calls[0][0]).toMatchObject({
        id: 'task-1',
        parent_task: 'p1',
        deliverable_obj_id: 'c2',
      })
    })

    it('does not call onTaskUpdated when save fails', async () => {
      apiFetch.mockResolvedValue({
        ok: false,
        status: 500,
        json: async () => ({ error: 'server error' }),
      })
      const onTaskUpdated = vi.fn()
      await saveEdit({
        effectiveTenantId: ref('t1'),
        effectiveWorkspaceId: ref('w1'),
        effectiveTaskId: ref('task-1'),
        editingTask: ref({
          title: 'demo',
          description: '',
          priority: 1,
          owner: '827923618472235008',
          assignees: [],
          workBranchName: 'feature/x',
          mergeTargetName: 'main',
          linkedProjects: [],
        }),
        editError: ref(''),
        isSaving: ref(false),
        localTask: ref({}),
        cancelEdit: vi.fn(),
        getProjectRepos: () => [],
        buildProjectsApiPayload: () => [],
        onTaskUpdated,
      })
      expect(onTaskUpdated).not.toHaveBeenCalled()
    })

    it('carries traceId on the thrown error when save fails (OPT-20260806-019)', async () => {
      // 回归：手动 throw new Error 必须挂 response.traceId / body._traceId，
      // 否则 showRequestError 无法挂 data-traceId（失败弹窗 trace 丢失）
      const consoleError = vi.spyOn(console, 'error').mockImplementation(() => {})
      apiFetch.mockResolvedValue({
        ok: false,
        status: 400,
        traceId: 'web-trace-abc-123',
        json: async () => ({ error: 'invalid payload', _traceId: 'web-trace-abc-123' }),
      })
      const onTaskUpdated = vi.fn()
      const editError = ref('')
      await saveEdit({
        effectiveTenantId: ref('t1'),
        effectiveWorkspaceId: ref('w1'),
        effectiveTaskId: ref('task-1'),
        editingTask: ref({
          title: 'demo',
          description: '',
          priority: 1,
          owner: '827923618472235008',
          assignees: [],
          workBranchName: 'feature/x',
          mergeTargetName: 'main',
          linkedProjects: [],
        }),
        editError,
        isSaving: ref(false),
        localTask: ref({}),
        cancelEdit: vi.fn(),
        getProjectRepos: () => [],
        buildProjectsApiPayload: () => [],
        onTaskUpdated,
      })
      expect(editError.value).toBe('invalid payload')
      const loggedError = consoleError.mock.calls.find((c) => c[0] === '保存任务失败:')
      expect(loggedError[1]).toMatchObject({ traceId: 'web-trace-abc-123' })
      expect(onTaskUpdated).not.toHaveBeenCalled()
      consoleError.mockRestore()
    })

    it('keeps GET projects instead of overwriting with send payload shape', async () => {
      const getProjects = [{
        project_id: 'proj_881195029417193472',
        repo_index: 0,
        base_branch: 'master',
        stored_repo_address: 'https://github.com/ruandao/helloworld',
        project_repo_url: 'https://github.com/test-ruandao/helloworld.git',
        repo_address_mismatch: true,
      }]
      const sendProjects = [{
        project_id: 'proj_881195029417193472',
        repo_index: 0,
        base_branch: 'master',
        target_branch: 'feature/x',
      }]
      apiFetch.mockResolvedValue({
        ok: true,
        json: async () => ({
          id: 'task_881388002226499584',
          title: 'demo',
          projects: getProjects,
        }),
      })
      const localTask = ref({})
      await saveEdit({
        effectiveTenantId: ref('t1'),
        effectiveWorkspaceId: ref('w1'),
        effectiveTaskId: ref('task-1'),
        editingTask: ref({
          title: 'demo',
          description: '',
          priority: 1,
          owner: '827923618472235008',
          assignees: [],
          workBranchName: 'feature/x',
          mergeTargetName: 'main',
          linkedProjects: [{
            project_id: 'proj_881195029417193472',
            repo_branches: { 'https://github.com/test-ruandao/helloworld.git': 'master' },
          }],
        }),
        editError: ref(''),
        isSaving: ref(false),
        localTask,
        cancelEdit: vi.fn(),
        getProjectRepos: () => ['https://github.com/test-ruandao/helloworld.git'],
        buildProjectsApiPayload: () => sendProjects,
      })
      expect(localTask.value.projects).toEqual(getProjects)
    })
  })
}

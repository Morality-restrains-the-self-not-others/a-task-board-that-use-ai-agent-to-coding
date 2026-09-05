// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] TaskDetailPageHeader.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { afterEach, beforeEach, describe, expect, it, vi } = await import('vitest')

  const hoistedMocks = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
    oauthConnected: true,
    gitIdentities: [{
      id: 'gid-1',
      git_user_name: 'Ann',
      git_user_email: 'ann@example.com',
      is_default: true,
    }],
  }))

  vi.mock('../../utils/apiUtils.js', () => ({
    apiFetch: hoistedMocks.apiFetchMock,
  }))

  vi.mock('../../utils/cookieUtils.js', () => ({
    getCookie: () => 'user-1',
  }))

  const TaskDetailPageHeader = (await import('./TaskDetailPageHeader.vue')).default

  function mountHeader(overrides = {}) {
    const forkTask = overrides.forkTask || vi.fn(async () => true)
    const wrapper = mount(TaskDetailPageHeader, {
      props: {
        hasTask: true,
        isEditing: false,
        isForking: false,
        isSaving: false,
        editError: '',
        backToWorkPanelRoute: '/work-panel',
        forkTask,
        tenantId: 't1',
        workspaceId: 'w1',
        sourceTask: { feature_params_source: 'company' },
        taskProjectsWithDetails: overrides.taskProjectsWithDetails || [{
          project_id: 'p-demo',
          project: { id: 'p-demo', git_repos: ['https://github.com/acme/demo.git'] },
        }],
        ...overrides.props,
      },
      global: {
        stubs: {
          'router-link': { template: '<a><slot /></a>' },
        },
      },
    })
    return { wrapper, forkTask }
  }

  describe('TaskDetailPageHeader Fork confirm', () => {
    beforeEach(() => {
      vi.clearAllMocks()
      hoistedMocks.oauthConnected = true
      hoistedMocks.gitIdentities = [{
        id: 'gid-1',
        git_user_name: 'Ann',
        git_user_email: 'ann@example.com',
        is_default: true,
      }]
      localStorage.setItem('currentUserId', 'user-1')
      hoistedMocks.apiFetchMock.mockImplementation(async (url) => {
        const path = String(url)
        if (path.includes('/git-identities/')) {
          return {
            ok: true,
            status: 200,
            headers: { get: () => null },
            json: async () => ({ identities: hoistedMocks.gitIdentities }),
          }
        }
        if (path.includes('/personal/feature-params-configs/')) {
          return {
            ok: true,
            status: 200,
            headers: { get: () => null },
            json: async () => ({ configs: [] }),
          }
        }
        if (path.includes('/feature-params/')) {
          return {
            ok: true,
            status: 200,
            headers: { get: () => null },
            json: async () => ({
              data: {
                providers: [{ provider: 'openai', supported_models: ['gpt-4.1'] }],
                agent_model: 'gpt-4.1',
                agent_model_provider: 'openai',
              },
            }),
          }
        }
        if (path.includes('/validate-git-repos/')) {
          return {
            ok: true,
            status: 200,
            headers: { get: () => null },
            json: async () => ({
              results: [{
                url: 'https://github.com/acme/demo.git',
                token_status: hoistedMocks.oauthConnected ? 'token_available' : 'not_bound',
              }],
            }),
          }
        }
        if (path.includes('/user-app-connection/')) {
          return {
            ok: true,
            status: 200,
            headers: { get: () => null },
            json: async () => ({ connected: hoistedMocks.oauthConnected }),
          }
        }
        return {
          ok: true,
          status: 200,
          headers: { get: () => null },
          json: async () => ({}),
        }
      })
    })

    afterEach(() => {
      localStorage.clear()
    })
    it('T1: click Fork opens confirm modal without calling forkTask', async () => {
      const { wrapper, forkTask } = mountHeader()
      expect(wrapper.find('[data-testid="fork-auto-run-confirm-modal"]').exists()).toBe(false)
      await wrapper.find('#task-fork-btn').trigger('click')
      expect(wrapper.find('[data-testid="fork-auto-run-confirm-modal"]').exists()).toBe(true)
      expect(forkTask).not.toHaveBeenCalled()
    })

    it('T2: cancel closes modal and does not fork', async () => {
      const { wrapper, forkTask } = mountHeader()
      await wrapper.find('#task-fork-btn').trigger('click')
      await wrapper.find('[data-testid="fork-confirm-cancel"]').trigger('click')
      expect(wrapper.find('[data-testid="fork-auto-run-confirm-modal"]').exists()).toBe(false)
      expect(forkTask).not.toHaveBeenCalled()
    })

    it('T3: choose no auto-run calls forkTask({ autoRun: false, copyCount: 1 })', async () => {
      const { wrapper, forkTask } = mountHeader()
      await wrapper.find('#task-fork-btn').trigger('click')
      await wrapper.find('[data-testid="fork-mode-fork-only"]').setValue(true)
      await wrapper.find('[data-testid="fork-confirm-submit"]').trigger('click')
      await flushPromises()
      expect(forkTask).toHaveBeenCalledWith(expect.objectContaining({
        autoRun: false,
        copyCount: 1,
        batchIdempotencyKey: expect.any(String),
      }))
      expect(wrapper.find('[data-testid="fork-auto-run-confirm-modal"]').exists()).toBe(false)
    })

    it('T4/T5: choose auto-run calls forkTask({ autoRun: true, repoIdentities }) and closes on success', async () => {
      const { wrapper, forkTask } = mountHeader()
      await wrapper.find('#task-fork-btn').trigger('click')
      await flushPromises()
      await flushPromises()
      await wrapper.find('[data-testid="fork-mode-auto-run"]').setValue(true)
      await flushPromises()
      await wrapper.find('[data-testid="fork-confirm-submit"]').trigger('click')
      await flushPromises()
      expect(forkTask).toHaveBeenCalledWith(expect.objectContaining({
        autoRun: true,
        copyCount: 1,
        batchIdempotencyKey: expect.any(String),
        agentModelProvider: 'openai',
        agents: [{ provider: 'openai', model: 'gpt-4.1' }],
        repoIdentities: [{
          repo_url: 'https://github.com/acme/demo.git',
          git_identity_id: 'gid-1',
        }],
        onForkProgress: expect.any(Function),
      }))
      expect(wrapper.find('[data-testid="fork-auto-run-confirm-modal"]').exists()).toBe(false)
    })

    it('T9: no copy-count input; fork-only always copyCount 1', async () => {
      const { wrapper, forkTask } = mountHeader()
      await wrapper.find('#task-fork-btn').trigger('click')
      expect(wrapper.find('[data-testid="fork-copy-count-input"]').exists()).toBe(false)
      await wrapper.find('[data-testid="fork-mode-fork-only"]').setValue(true)
      await wrapper.find('[data-testid="fork-confirm-submit"]').trigger('click')
      await flushPromises()
      expect(forkTask).toHaveBeenCalledWith(expect.objectContaining({
        autoRun: false,
        copyCount: 1,
      }))
    })

    it('keeps modal open when forkTask returns false', async () => {
      const forkTask = vi.fn(async () => false)
      const { wrapper } = mountHeader({ forkTask })
      await wrapper.find('#task-fork-btn').trigger('click')
      await flushPromises()
      await flushPromises()
      await wrapper.find('[data-testid="fork-mode-auto-run"]').setValue(true)
      await flushPromises()
      await wrapper.find('[data-testid="fork-confirm-submit"]').trigger('click')
      await flushPromises()
      expect(wrapper.find('[data-testid="fork-auto-run-confirm-modal"]').exists()).toBe(true)
    })

    it('ignores second confirm click while first forkTask is in flight', async () => {
      let release
      const gate = new Promise((resolve) => { release = resolve })
      const forkTask = vi.fn(async () => {
        await gate
        return true
      })
      const { wrapper } = mountHeader({ forkTask })
      await wrapper.find('#task-fork-btn').trigger('click')
      await flushPromises()
      await flushPromises()
      await wrapper.find('[data-testid="fork-mode-auto-run"]').setValue(true)
      await flushPromises()
      await wrapper.find('[data-testid="fork-confirm-submit"]').trigger('click')
      await wrapper.find('[data-testid="fork-confirm-submit"]').trigger('click')
      expect(forkTask).toHaveBeenCalledTimes(1)
      release()
      await flushPromises()
      expect(wrapper.find('[data-testid="fork-auto-run-confirm-modal"]').exists()).toBe(false)
    })

    it('blocks fork auto-run when github repo oauth is unbound', async () => {
      hoistedMocks.oauthConnected = false
      const { wrapper, forkTask } = mountHeader()
      await wrapper.find('#task-fork-btn').trigger('click')
      await flushPromises()
      await flushPromises()
      await wrapper.find('[data-testid="fork-mode-auto-run"]').setValue(true)
      await flushPromises()
      expect(wrapper.find('[data-testid="fork-auto-run-oauth-blocked-reason"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="fork-confirm-submit"]').attributes('disabled')).toBeDefined()
      await wrapper.find('[data-testid="fork-confirm-submit"]').trigger('click')
      await flushPromises()
      expect(forkTask).not.toHaveBeenCalled()
    })

    it('T6: blocks auto-run until current user git identity is selected', async () => {
      hoistedMocks.gitIdentities = [{
        id: 'gid-1',
        git_user_name: 'Ann',
        git_user_email: 'ann@example.com',
        is_default: false,
      }]
      const { wrapper, forkTask } = mountHeader()
      await wrapper.find('#task-fork-btn').trigger('click')
      await flushPromises()
      await flushPromises()
      await wrapper.find('[data-testid="fork-mode-auto-run"]').setValue(true)
      await flushPromises()
      expect(wrapper.find('[data-testid="fork-auto-run-git-identity-section"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="fork-auto-run-git-identity-blocked-reason"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="fork-confirm-submit"]').attributes('disabled')).toBeDefined()
      await wrapper.find('[data-testid="fork-confirm-submit"]').trigger('click')
      await flushPromises()
      expect(forkTask).not.toHaveBeenCalled()
      await wrapper.find('[data-testid="fork-mode-fork-only"]').setValue(true)
      expect(wrapper.find('[data-testid="fork-confirm-submit"]').attributes('disabled')).toBeUndefined()
    })
  })
}

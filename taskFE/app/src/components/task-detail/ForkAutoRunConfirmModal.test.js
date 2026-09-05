// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] ForkAutoRunConfirmModal.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { describe, expect, it, vi, beforeEach } = await import('vitest')

  vi.mock('../../utils/apiUtils.js', () => ({
    apiFetch: vi.fn(),
  }))

  const { apiFetch } = await import('../../utils/apiUtils.js')
  const ForkAutoRunConfirmModal = (await import('./ForkAutoRunConfirmModal.vue')).default

  function mockFeatureParams() {
    apiFetch.mockImplementation(async (url) => {
      const path = String(url)
      if (path.includes('/personal/feature-params-configs/')) {
        return { ok: true, headers: { get: () => null }, json: async () => ({ configs: [] }) }
      }
      return {
        ok: true,
        headers: { get: () => null },
        json: async () => ({
          data: {
            providers: [{ provider: 'openai', supported_models: ['gpt-4.1', 'gpt-4.1-mini'] }],
            agent_model: 'gpt-4.1',
            agent_model_provider: 'openai',
          },
        }),
      }
    })
  }

  describe('ForkAutoRunConfirmModal', () => {
    beforeEach(() => {
      vi.clearAllMocks()
      mockFeatureParams()
    })

    it('hidden when show is false', () => {
      const wrapper = mount(ForkAutoRunConfirmModal, {
        props: { show: false, forking: false },
      })
      expect(wrapper.find('[data-testid="fork-auto-run-confirm-modal"]').exists()).toBe(false)
    })

    it('shows cancel and confirm when open; no copy-count input', () => {
      const wrapper = mount(ForkAutoRunConfirmModal, {
        props: { show: true, forking: false },
      })
      expect(wrapper.find('[data-testid="fork-auto-run-confirm-modal"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="fork-confirm-cancel"]').text()).toContain('取消')
      expect(wrapper.find('[data-testid="fork-confirm-submit"]').text()).toContain('确认派生')
      expect(wrapper.find('[data-testid="fork-copy-count-input"]').exists()).toBe(false)
      expect(wrapper.find('[data-testid="fork-confirm-no-auto-run"]').exists()).toBe(false)
      expect(wrapper.find('[data-testid="fork-confirm-auto-run"]').exists()).toBe(false)
    })

    it('emits close on cancel and overlay when not forking', async () => {
      const wrapper = mount(ForkAutoRunConfirmModal, {
        props: { show: true, forking: false },
      })
      await wrapper.find('[data-testid="fork-confirm-cancel"]').trigger('click')
      expect(wrapper.emitted('close')).toHaveLength(1)

      await wrapper.find('[data-testid="fork-auto-run-confirm-modal"]').trigger('click')
      expect(wrapper.emitted('close')).toHaveLength(2)
    })

    it('does not close overlay while forking', async () => {
      const wrapper = mount(ForkAutoRunConfirmModal, {
        props: { show: true, forking: true },
      })
      await wrapper.find('[data-testid="fork-auto-run-confirm-modal"]').trigger('click')
      expect(wrapper.emitted('close')).toBeUndefined()
    })

    it('T3/T10: fork-only confirm emits autoRun false copyCount 1', async () => {
      const wrapper = mount(ForkAutoRunConfirmModal, {
        props: { show: true, forking: false },
      })
      expect(wrapper.find('[data-testid="fork-confirm-submit"]').attributes('disabled')).toBeDefined()
      await wrapper.find('[data-testid="fork-mode-fork-only"]').setValue(true)
      await wrapper.find('[data-testid="fork-mode-fork-only"]').trigger('change')
      await wrapper.vm.$nextTick()
      expect(wrapper.find('[data-testid="fork-confirm-submit"]').attributes('disabled')).toBeUndefined()
      await wrapper.find('[data-testid="fork-confirm-submit"]').trigger('click')
      expect(wrapper.emitted('confirm')).toEqual([
        [{ autoRun: false, copyCount: 1 }],
      ])
    })

    it('T14: auto-run without models keeps confirm disabled', async () => {
      apiFetch.mockImplementation(async () => ({
        ok: true,
        json: async () => ({
          data: { providers: [], agent_model: '', agent_model_provider: '' },
        }),
      }))
      const wrapper = mount(ForkAutoRunConfirmModal, {
        props: {
          show: true,
          forking: false,
          tenantId: 't1',
          workspaceId: 'w1',
          sourceTask: { feature_params_source: 'company' },
        },
      })
      await wrapper.find('[data-testid="fork-mode-auto-run"]').setValue(true)
      await wrapper.find('[data-testid="fork-mode-auto-run"]').trigger('change')
      await flushPromises()
      expect(wrapper.find('[data-testid="fork-confirm-submit"]').attributes('disabled')).toBeDefined()
      await wrapper.find('[data-testid="fork-confirm-submit"]').trigger('click')
      expect(wrapper.emitted('confirm')).toBeUndefined()
    })

    it('T15: auto-run with two models emits agents payload', async () => {
      const wrapper = mount(ForkAutoRunConfirmModal, {
        props: {
          show: true,
          forking: false,
          tenantId: 't1',
          workspaceId: 'w1',
          sourceTask: { feature_params_source: 'company' },
        },
      })
      await wrapper.find('[data-testid="fork-mode-auto-run"]').setValue(true)
      await wrapper.find('[data-testid="fork-mode-auto-run"]').trigger('change')
      await flushPromises()
      await flushPromises()
      const mini = wrapper.find('[data-testid="fork-agent-model-gpt-4.1-mini"]')
      expect(mini.exists()).toBe(true)
      await mini.trigger('change')
      await wrapper.find('[data-testid="fork-confirm-submit"]').trigger('click')
      const payload = wrapper.emitted('confirm')[0][0]
      expect(payload.autoRun).toBe(true)
      expect(payload.copyCount).toBe(2)
      expect(payload.agentModelProvider).toBe('openai')
      expect(payload.agents.map((a) => a.model).sort()).toEqual(['gpt-4.1', 'gpt-4.1-mini'])
    })

    it('T16: fork-only ignores previously checked models', async () => {
      const wrapper = mount(ForkAutoRunConfirmModal, {
        props: {
          show: true,
          forking: false,
          tenantId: 't1',
          workspaceId: 'w1',
          sourceTask: { feature_params_source: 'company' },
        },
      })
      await wrapper.find('[data-testid="fork-mode-auto-run"]').setValue(true)
      await wrapper.find('[data-testid="fork-mode-auto-run"]').trigger('change')
      await flushPromises()
      await wrapper.find('[data-testid="fork-mode-fork-only"]').setValue(true)
      await wrapper.find('[data-testid="fork-mode-fork-only"]').trigger('change')
      await wrapper.find('[data-testid="fork-confirm-submit"]').trigger('click')
      expect(wrapper.emitted('confirm')[0][0]).toEqual({ autoRun: false, copyCount: 1 })
    })

    it('disables confirm while forking', () => {
      const wrapper = mount(ForkAutoRunConfirmModal, {
        props: { show: true, forking: true },
      })
      expect(wrapper.find('[data-testid="fork-confirm-cancel"]').attributes('disabled')).toBeDefined()
      expect(wrapper.find('[data-testid="fork-confirm-submit"]').attributes('disabled')).toBeDefined()
    })

    it('blocks auto-run when oauth repos are unbound', async () => {
      const wrapper = mount(ForkAutoRunConfirmModal, {
        props: {
          show: true,
          forking: false,
          tenantId: 't1',
          workspaceId: 'w1',
          sourceTask: { feature_params_source: 'company' },
          autoRunBlockedReason: '自动运行并派生前，请为全部关联仓库完成 OAuth 绑定',
          oauthRepoRows: [{
            repoUrl: 'https://github.com/acme/demo.git',
            bound: false,
            loading: false,
            error: '',
            errorTraceId: '',
            siteLabel: 'GitHub',
            bindHref: '/api/git-oauth/github-start-from-gateway/?repo_url=https%3A%2F%2Fgithub.com%2Facme%2Fdemo.git',
          }],
        },
      })
      await wrapper.find('[data-testid="fork-mode-auto-run"]').setValue(true)
      await wrapper.find('[data-testid="fork-mode-auto-run"]').trigger('change')
      await flushPromises()
      expect(wrapper.find('[data-testid="fork-auto-run-oauth-section"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="fork-confirm-submit"]').attributes('disabled')).toBeDefined()
      await wrapper.find('[data-testid="fork-mode-fork-only"]').setValue(true)
      await wrapper.find('[data-testid="fork-mode-fork-only"]').trigger('change')
      expect(wrapper.find('[data-testid="fork-confirm-submit"]').attributes('disabled')).toBeUndefined()
    })

    it('shows current-user git identity select and blocks auto-run when missing', async () => {
      const wrapper = mount(ForkAutoRunConfirmModal, {
        props: {
          show: true,
          forking: false,
          tenantId: 't1',
          workspaceId: 'w1',
          sourceTask: { feature_params_source: 'company' },
          gitIdentityRepoUrls: ['https://gitlab.example/a.git'],
          gitIdentities: [{
            id: 'gid-1',
            git_user_name: 'Ann',
            git_user_email: 'ann@example.com',
          }],
          gitIdentitySettingsHref: '/tenant/t1/profile/git-identities/',
          gitIdentityIdForUrl: () => '',
          gitIdentityBlockedReason: '请为仓库选择 Git 提交身份：https://gitlab.example/a.git',
        },
      })
      await wrapper.find('[data-testid="fork-mode-auto-run"]').setValue(true)
      await wrapper.find('[data-testid="fork-mode-auto-run"]').trigger('change')
      await flushPromises()
      expect(wrapper.find('[data-testid="fork-auto-run-git-identity-section"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="create-task-repo-git-identity-select"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="fork-auto-run-git-identity-blocked-reason"]').text()).toContain('Git 提交身份')
      expect(wrapper.find('[data-testid="fork-confirm-submit"]').attributes('disabled')).toBeDefined()
      await wrapper.find('[data-testid="fork-confirm-submit"]').trigger('click')
      expect(wrapper.emitted('confirm')).toBeUndefined()
    })

    it('T19: auto-run 502 HTML shows retryable model error with traceId', async () => {
      apiFetch.mockImplementation(async () => ({
        ok: false,
        status: 502,
        traceId: '796a1ffa-ede1-4a2f-988d-9566d706151d',
        headers: { get: () => '796a1ffa-ede1-4a2f-988d-9566d706151d' },
        _errorData: {
          _rawErrorText:
            '<html><head><title>502 Bad Gateway</title></head><body><h1>502 Bad Gateway</h1></body></html>',
        },
        json: async () => ({}),
      }))
      const wrapper = mount(ForkAutoRunConfirmModal, {
        props: {
          show: true,
          forking: false,
          tenantId: 't1',
          workspaceId: 'w1',
          sourceTask: { feature_params_source: 'company' },
        },
      })
      await wrapper.find('[data-testid="fork-mode-auto-run"]').setValue(true)
      await wrapper.find('[data-testid="fork-mode-auto-run"]').trigger('change')
      await flushPromises()
      const err = wrapper.find('[data-testid="fork-agent-models-error"]')
      expect(err.exists()).toBe(true)
      expect(err.text()).toBe('服务暂时不可用，请稍后重试')
      expect(err.attributes('data-traceid') || err.attributes('data-traceId')).toBe(
        '796a1ffa-ede1-4a2f-988d-9566d706151d',
      )
      expect(wrapper.find('[data-testid="fork-agent-models-retry"]').exists()).toBe(true)
    })

    it('T20: auto-run 502 then 200 shows models without error', async () => {
      apiFetch
        .mockResolvedValueOnce({
          ok: false,
          status: 502,
          traceId: 't-502',
          headers: { get: () => 't-502' },
          _errorData: { _rawErrorText: '<html><h1>502 Bad Gateway</h1></html>' },
          json: async () => ({}),
        })
        .mockImplementation(async (url) => {
          const path = String(url)
          if (path.includes('/personal/feature-params-configs/')) {
            return { ok: true, headers: { get: () => null }, json: async () => ({ configs: [] }) }
          }
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
        })
      const wrapper = mount(ForkAutoRunConfirmModal, {
        props: {
          show: true,
          forking: false,
          tenantId: 't1',
          workspaceId: 'w1',
          sourceTask: { feature_params_source: 'company' },
        },
      })
      await wrapper.find('[data-testid="fork-mode-auto-run"]').setValue(true)
      await wrapper.find('[data-testid="fork-mode-auto-run"]').trigger('change')
      await flushPromises()
      expect(wrapper.find('[data-testid="fork-agent-models-error"]').exists()).toBe(false)
      expect(wrapper.find('[data-testid="fork-agent-model-gpt-4.1"]').exists()).toBe(true)
    })
  })
}

// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] TaskDetailAutoRunSkipBanner.test.js requires vitest runtime')
} else {
  const { describe, expect, it, vi, beforeEach } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')

  const {
    apiFetch,
    alertAutoRunStartSkippedIfNeeded,
    queryClientPublicIpForAutoSg,
    fetchGitOAuthUserAppConnected,
  } = vi.hoisted(() => ({
    apiFetch: vi.fn(),
    alertAutoRunStartSkippedIfNeeded: vi.fn(),
    queryClientPublicIpForAutoSg: vi.fn(async () => '1.2.3.4'),
    fetchGitOAuthUserAppConnected: vi.fn(async () => false),
  }))

  vi.mock('../../utils/apiUtils.js', () => ({ apiFetch }))
  vi.mock('../../utils/autoRunGateHints.js', () => ({ alertAutoRunStartSkippedIfNeeded }))
  vi.mock('../../utils/publicClientIp.js', () => ({ queryClientPublicIpForAutoSg }))
  vi.mock('../../utils/gitOAuthUserAppConnection.js', () => ({ fetchGitOAuthUserAppConnected }))
  vi.mock('../../utils/requestErrorDisplay.js', async (importOriginal) => {
    const actual = await importOriginal()
    return { ...actual, showRequestError: vi.fn() }
  })
  vi.mock('../../utils/modalService.js', () => ({ default: { alert: vi.fn() } }))

  const { default: TaskDetailAutoRunSkipBanner } = await import('./TaskDetailAutoRunSkipBanner.vue')

  describe('TaskDetailAutoRunSkipBanner', () => {
    beforeEach(() => {
      apiFetch.mockReset()
      alertAutoRunStartSkippedIfNeeded.mockReset()
      queryClientPublicIpForAutoSg.mockReset()
      queryClientPublicIpForAutoSg.mockResolvedValue('1.2.3.4')
      fetchGitOAuthUserAppConnected.mockReset()
      fetchGitOAuthUserAppConnected.mockResolvedValue(false)
    })

    it('auto_run skipped 时展示原因与强制重新启动', () => {
      const wrapper = mount(TaskDetailAutoRunSkipBanner, {
        props: {
          task: {
            auto_run: true,
            auto_run_start_skipped: true,
            auto_run_start_skip_reason: 'Git 授权刷新超时',
          },
          tenantId: 't1',
          workspaceId: 'ws1',
          taskId: 'task_1',
        },
      })
      expect(wrapper.get('[data-testid="auto-run-start-skipped-banner"]').exists()).toBe(true)
      expect(wrapper.get('[data-testid="auto-run-start-skipped-reason"]').text()).toContain('Git 授权刷新超时')
      expect(wrapper.get('[data-testid="auto-run-force-restart"]').exists()).toBe(true)
    })

    it('gitlab refresh http 400 展示重新绑定文案与 OAuth 链接', () => {
      const repoUrl = 'https://gitlab-tencent-sh-1.daydaymoney.com/example-user/somanyad'
      const wrapper = mount(TaskDetailAutoRunSkipBanner, {
        props: {
          task: {
            auto_run: true,
            auto_run_start_skipped: true,
            auto_run_start_skip_reason: '无法获取子 Git 仓库列表：gitlab refresh http 400',
          },
          tenantId: 't1',
          workspaceId: 'ws1',
          taskId: 'task_1',
          repoUrl,
        },
      })
      const reason = wrapper.get('[data-testid="auto-run-start-skipped-reason"]').text()
      expect(reason).not.toContain('gitlab refresh http 400')
      expect(reason).toContain('重新绑定')
      const bind = wrapper.get('[data-testid="auto-run-skip-oauth-bind"]')
      expect(bind.attributes('href')).toContain('/api/git-oauth/gitlab-start-from-gateway/')
      expect(bind.attributes('href')).toContain(encodeURIComponent(repoUrl))
    })

    it('OAuth 已绑定后隐藏「去绑定」链接，横幅与强制重启仍在', async () => {
      const repoUrl = 'https://gitlab-tencent-sh-1.daydaymoney.com/example-user/somanyad'
      fetchGitOAuthUserAppConnected.mockResolvedValue(true)
      const wrapper = mount(TaskDetailAutoRunSkipBanner, {
        props: {
          task: {
            auto_run: true,
            auto_run_start_skipped: true,
            auto_run_start_skip_reason: '无法获取子 Git 仓库列表：gitlab refresh http 400',
          },
          tenantId: 't1',
          workspaceId: 'ws1',
          taskId: 'task_1',
          repoUrl,
        },
      })
      await flushPromises()
      expect(fetchGitOAuthUserAppConnected).toHaveBeenCalledWith(repoUrl)
      expect(wrapper.get('[data-testid="auto-run-start-skipped-banner"]').exists()).toBe(true)
      expect(wrapper.get('[data-testid="auto-run-force-restart"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="auto-run-skip-oauth-bind"]').exists()).toBe(false)
      expect(wrapper.get('[data-testid="auto-run-start-skipped-reason"]').text()).toContain('已绑定')
      expect(wrapper.get('[data-testid="auto-run-start-skipped-reason"]').text()).toContain('强制重新启动')
      expect(wrapper.get('[data-testid="auto-run-start-skipped-reason"]').text()).not.toContain('重新绑定')
    })

    it('绑定查询失败时仍展示「去绑定」链接', async () => {
      const repoUrl = 'https://gitlab-tencent-sh-1.daydaymoney.com/example-user/somanyad'
      fetchGitOAuthUserAppConnected.mockRejectedValue(new Error('timeout'))
      const wrapper = mount(TaskDetailAutoRunSkipBanner, {
        props: {
          task: {
            auto_run: true,
            auto_run_start_skipped: true,
            auto_run_start_skip_reason: '无法获取子 Git 仓库列表：gitlab refresh http 400',
          },
          tenantId: 't1',
          workspaceId: 'ws1',
          taskId: 'task_1',
          repoUrl,
        },
      })
      await flushPromises()
      expect(wrapper.get('[data-testid="auto-run-skip-oauth-bind"]').exists()).toBe(true)
    })

    it('未跳过且已有评论时不渲染横幅', () => {
      const wrapper = mount(TaskDetailAutoRunSkipBanner, {
        props: {
          task: {
            auto_run: true,
            auto_run_start_skipped: false,
            auto_run_start_skip_reason: '',
            comments: [{ id: 'c1' }],
            comments_feeds_loaded: true,
          },
          tenantId: 't1',
          workspaceId: 'ws1',
          taskId: 'task_1',
        },
      })
      expect(wrapper.find('[data-testid="auto-run-start-skipped-banner"]').exists()).toBe(false)
    })

    it('工作面板列表 stub（comments=[] 且未拉 feed）不展示 Git 探测误报横幅', () => {
      const wrapper = mount(TaskDetailAutoRunSkipBanner, {
        props: {
          task: {
            auto_run: true,
            auto_run_start_skipped: false,
            auto_run_start_skip_reason: '',
            comments: [],
          },
          tenantId: 't1',
          workspaceId: 'ws1',
          taskId: 'task_1',
        },
      })
      expect(wrapper.find('[data-testid="auto-run-start-skipped-banner"]').exists()).toBe(false)
    })

    it('auto_run 且评论 feed 已加载仍为零时展示存量未启服提示', () => {
      const wrapper = mount(TaskDetailAutoRunSkipBanner, {
        props: {
          task: {
            auto_run: true,
            auto_run_start_skipped: false,
            auto_run_start_skip_reason: '',
            comments: [],
            ai_comments: [],
            container_agent_comments: [],
            comments_feeds_loaded: true,
          },
          tenantId: 't1',
          workspaceId: 'ws1',
          taskId: 'task_1',
        },
      })
      expect(wrapper.get('[data-testid="auto-run-start-skipped-banner"]').exists()).toBe(true)
      expect(wrapper.get('[data-testid="auto-run-start-skipped-reason"]').text()).toContain('尚未出现自动运行评论')
      expect(wrapper.get('[data-testid="auto-run-start-skipped-reason"]').text()).not.toContain('探测失败')
    })

    it('已有容器 Agent 评论时不把空人类评论当成软跳过', () => {
      const wrapper = mount(TaskDetailAutoRunSkipBanner, {
        props: {
          task: {
            auto_run: true,
            auto_run_start_skipped: false,
            auto_run_start_skip_reason: '',
            comments: [],
            container_agent_comments: [{ id: 'ca1' }],
            comments_feeds_loaded: true,
          },
          tenantId: 't1',
          workspaceId: 'ws1',
          taskId: 'task_1',
        },
      })
      expect(wrapper.find('[data-testid="auto-run-start-skipped-banner"]').exists()).toBe(false)
    })

    it('强制重新启动 PATCH force_auto_run 并上抛更新', async () => {
      apiFetch.mockResolvedValue({
        ok: true,
        json: async () => ({
          id: 'task_1',
          auto_run: true,
          auto_run_start_skipped: false,
          auto_run_start_skip_reason: '',
        }),
      })
      const wrapper = mount(TaskDetailAutoRunSkipBanner, {
        props: {
          task: {
            auto_run: true,
            auto_run_start_skipped: true,
            auto_run_start_skip_reason: '超时',
          },
          tenantId: 't1',
          workspaceId: 'ws1',
          taskId: 'task_1',
        },
      })
      await wrapper.get('[data-testid="auto-run-force-restart"]').trigger('click')
      await flushPromises()
      expect(apiFetch).toHaveBeenCalled()
      const [url, opts] = apiFetch.mock.calls[0]
      expect(url).toContain('/task_1/')
      expect(opts.method).toBe('PATCH')
      const body = JSON.parse(opts.body)
      expect(body.force_auto_run).toBe(true)
      expect(body.auto_run).toBe(true)
      expect(body.client_public_ip).toBe('1.2.3.4')
      expect(opts.headers['Idempotency-Key']).toMatch(/\S/)
      expect(wrapper.emitted('updated')?.[0]?.[0]?.id).toBe('task_1')
    })
  })
}

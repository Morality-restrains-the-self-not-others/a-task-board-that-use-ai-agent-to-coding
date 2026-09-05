// @vitest-environment jsdom
/**
 * TaskDetailTaskIdentityPanel：任务辅助信息折叠；排队调度入口已迁至评论区
 * pre-commit 会以 node 直跑已暂存 *.test.js，非 Vitest 环境下跳过。
 */
if (!process.env.VITEST) {
  console.log('[skip] TaskDetailTaskIdentityPanel.aux-info.test.js requires vitest runtime')
} else {
  const { describe, it, expect, vi, beforeEach } = await import('vitest')
  const { mount } = await import('@vue/test-utils')
  const { createRouter, createMemoryHistory } = await import('vue-router')

  vi.mock('../../utils/apiUtils.js', () => ({
    apiFetch: vi.fn(async () => ({
      ok: true,
      json: async () => ({ current_deliverable_objs: [] }),
    })),
  }))

  const { default: TaskDetailTaskIdentityPanel } = await import('./TaskDetailTaskIdentityPanel.vue')

  // useTaskAuxInfoExpanded persists to localStorage — reset between tests
  beforeEach(() => {
    localStorage.removeItem('task-detail-aux-info-expanded')
  })

  async function mountPanel(opts = {}) {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/', component: { template: '<div />' } }],
    })
    await router.push('/')
    await router.isReady()
    return mount(TaskDetailTaskIdentityPanel, {
      props: {
        task: {
          id: 'task_1',
          title: 't',
          description: '',
          priority: 1,
          created_at: '2026-07-13T00:00:00Z',
          auto_run: false,
          parent_task: null,
          ...(opts.taskOverrides || {}),
        },
        forkSourceTaskRoute: '/',
        progressStatusOptions: [],
        tenantId: 'ten1',
        workspaceId: 'ws1',
        taskId: 'task_1',
      },
      global: {
        plugins: [router],
        stubs: {
          MarkdownContent: true,
          TaskDetailLlmBudgetPanel: true,
          TaskDetailSubtreeStatusPanel: true,
          TagMemberInput: true,
          AutoRunStepsPreview: true,
        },
      },
    })
  }

  describe('TaskDetailTaskIdentityPanel aux info collapse', () => {
    it('默认展开任务辅助信息，排队调度入口已迁出至评论区', async () => {
      const wrapper = await mountPanel()
      const panel = wrapper.find('[data-testid="task-aux-info-panel"]')
      expect(panel.exists()).toBe(true)
      expect(panel.text()).toContain('任务辅助信息')

      const toggle = wrapper.find('[data-testid="task-aux-info-toggle"]')
      expect(toggle.exists()).toBe(true)
      expect(toggle.attributes('aria-expanded')).toBe('true')
      expect(toggle.text()).toContain('收起')

      const body = wrapper.find('[data-testid="task-aux-info-body"]')
      expect(body.isVisible()).toBe(true)
      expect(body.find('[data-testid="task-queued-auto-run-toggle"]').exists()).toBe(false)
      expect(panel.find('[data-testid="task-queued-auto-run-toggle"]').exists()).toBe(false)
      wrapper.unmount()
    })

    it('点击收起后隐藏辅助信息正文', async () => {
      const wrapper = await mountPanel()
      const toggle = wrapper.find('[data-testid="task-aux-info-toggle"]')
      await toggle.trigger('click')
      expect(toggle.attributes('aria-expanded')).toBe('false')
      expect(toggle.text()).toContain('展开')

      const body = wrapper.find('[data-testid="task-aux-info-body"]')
      expect(body.isVisible()).toBe(false)
      wrapper.unmount()
    })

    it('收起后再展开仍不含排队调度入口', async () => {
      const wrapper = await mountPanel()
      const toggle = wrapper.find('[data-testid="task-aux-info-toggle"]')
      await toggle.trigger('click')
      await toggle.trigger('click')
      expect(toggle.attributes('aria-expanded')).toBe('true')
      const body = wrapper.find('[data-testid="task-aux-info-body"]')
      expect(body.isVisible()).toBe(true)
      expect(body.find('[data-testid="task-queued-auto-run-toggle"]').exists()).toBe(false)
      wrapper.unmount()
    })

    it('任务辅助信息展示人读编号 #N 与技术任务 ID', async () => {
      const wrapper = await mountPanel({
        taskOverrides: {
          id: 'task_877131046670331904',
          workspace_seq: 3,
        },
      })
      const displayNo = wrapper.find('[data-testid="task-aux-info-display-no"]')
      expect(displayNo.exists()).toBe(true)
      expect(displayNo.text()).toBe('#3')
      const techId = wrapper.find('[data-testid="task-aux-info-task-id"]')
      expect(techId.text()).toContain('task_877131046670331904')
      wrapper.unmount()
    })

    it('任务编号 #N 复用 TaskCardIdBadge：单击复制短号', async () => {
      const writeText = vi.fn().mockResolvedValue(undefined)
      Object.defineProperty(navigator, 'clipboard', {
        configurable: true,
        value: { writeText },
      })
      const { default: toastService } = await import('../../utils/toastService.js')
      const successSpy = vi.spyOn(toastService, 'success').mockImplementation(() => {})

      const wrapper = await mountPanel({
        taskOverrides: { id: 'task_877131046670331904', workspace_seq: 3 },
      })
      const badge = wrapper.find('[data-testid="task-card-id"]')
      expect(badge.exists()).toBe(true)
      expect(badge.text()).toBe('#3')
      await badge.trigger('click')
      expect(writeText).toHaveBeenCalledWith('#3')
      expect(successSpy).toHaveBeenCalledWith(expect.stringContaining('#3'), expect.any(Number))
      successSpy.mockRestore()
      wrapper.unmount()
    })

    it('无 workspace_seq 时任务编号显示占位 —', async () => {
      const wrapper = await mountPanel({
        taskOverrides: { id: 'task_1', workspace_seq: 0 },
      })
      expect(wrapper.find('[data-testid="task-aux-info-display-no"]').text()).toBe('—')
      wrapper.unmount()
    })
  })
  describe('TaskDetailTaskIdentityPanel v15 存续期', () => {
    it('无到期时间显示占位；过期帖子显示已到期徽标与续存按钮', async () => {
      const wrapper = await mountPanel()
      const placeholder = wrapper.find('[data-testid="task-post-validity"]')
      expect(placeholder.exists()).toBe(true)
      expect(placeholder.text()).toContain('—')
      expect(wrapper.find('[data-testid="task-post-renew-button"]').exists()).toBe(false)
      wrapper.unmount()
    })

    it('续存按钮 POST renew 并 emit task-updated 刷新到期时间', async () => {
      const { apiFetch } = await import('../../utils/apiUtils.js')
      const renewedAt = new Date(Date.now() + 365 * 24 * 3600 * 1000).toISOString()
      vi.mocked(apiFetch).mockResolvedValueOnce({
        ok: true,
        json: async () => ({ id: 'task_1', post_expires_at: renewedAt, post_expired: false }),
      })
      const wrapper = await mountPanel({
        taskOverrides: { post_expires_at: '2026-01-01T00:00:00Z', post_expired: true },
      })
      const badge = wrapper.find('[data-testid="task-post-expiry-badge"]')
      expect(badge.exists()).toBe(true)
      expect(badge.text()).toContain('已到期')
      const btn = wrapper.find('[data-testid="task-post-renew-button"]')
      expect(btn.exists()).toBe(true)
      const confirmSpy = vi.spyOn(window, 'confirm').mockReturnValue(true)
      await btn.trigger('click')
      await vi.waitFor(() => {
        expect(vi.mocked(apiFetch)).toHaveBeenCalledWith(
          expect.stringContaining('/renew/'),
          expect.objectContaining({ method: 'POST' }),
        )
      })
      const emitted = wrapper.emitted('task-updated')
      expect(emitted).toBeTruthy()
      expect(emitted[0][0].post_expires_at).toBe(renewedAt)
      confirmSpy.mockRestore()
      wrapper.unmount()
    })
  })
}

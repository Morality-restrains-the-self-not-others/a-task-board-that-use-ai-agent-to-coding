// @vitest-environment jsdom
/**
 * TaskDetailTaskIdentityPanel：派生自（fork_from）链接展示「#编号 任务名称」
 */
if (!process.env.VITEST) {
  console.log('[skip] TaskDetailTaskIdentityPanel.fork-from.test.js requires vitest runtime')
} else {
  const { describe, it, expect, vi } = await import('vitest')
  const { mount } = await import('@vue/test-utils')
  const { createRouter, createMemoryHistory } = await import('vue-router')

  vi.mock('../../utils/apiUtils.js', () => ({
    apiFetch: vi.fn(async () => ({
      ok: true,
      json: async () => ({ current_deliverable_objs: [] }),
    })),
  }))

  const { default: TaskDetailTaskIdentityPanel } = await import('./TaskDetailTaskIdentityPanel.vue')

  async function mountPanel(props = {}) {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/', component: { template: '<div />' } },
        {
          path: '/tenant/:tenant/workspace/:workspaceId/task-detail/:taskId/',
          name: 'task_detail',
          component: { template: '<div />' },
        },
      ],
    })
    await router.push('/')
    await router.isReady()
    return mount(TaskDetailTaskIdentityPanel, {
      props: {
        task: {
          id: 'task_1',
          title: '当前任务',
          description: '',
          priority: 1,
          created_at: '2026-07-13T00:00:00Z',
          auto_run: false,
          parent_task: null,
          fork_from: 'task_13646028863068037877',
        },
        forkSourceTaskRoute: {
          name: 'task_detail',
          params: {
            tenant: 'ten1',
            workspaceId: 'ws1',
            taskId: 'task_13646028863068037877',
          },
        },
        forkSourceTitle: '源任务名称',
        isForkSourceTitleLoading: false,
        workspaceTodos: [
          { id: 'task_13646028863068037877', title: '源任务名称', workspace_seq: 12 },
        ],
        progressStatusOptions: [],
        tenantId: 'ten1',
        workspaceId: 'ws1',
        taskId: 'task_1',
        ...props,
      },
      global: {
        plugins: [router],
        stubs: {
          MarkdownContent: true,
          TaskDetailLlmBudgetPanel: true,
          TaskDetailSubtreeStatusPanel: true,
          TagMemberInput: true,
          AutoRunStepsPreview: true,
          CreateTaskParentDeliverableField: true,
        },
      },
    })
  }

  describe('TaskDetailTaskIdentityPanel fork_from link', () => {
    it('展示 #序号 + 任务名称，且为可点击链接', async () => {
      const wrapper = await mountPanel()
      const block = wrapper.find('[data-testid="task-fork-from"]')
      expect(block.exists()).toBe(true)
      expect(block.text()).toContain('派生自')
      const link = wrapper.find('[data-testid="task-fork-from-link"]')
      expect(link.exists()).toBe(true)
      expect(link.text().trim()).toBe('#12 源任务名称')
      expect(link.element.tagName.toLowerCase()).toBe('a')
      expect(link.text()).not.toContain('任务 task_')
      wrapper.unmount()
    })

    it('标题加载中显示加载态，不渲染裸任务 id 文案', async () => {
      const wrapper = await mountPanel({
        forkSourceTitle: '',
        isForkSourceTitleLoading: true,
      })
      const block = wrapper.find('[data-testid="task-fork-from"]')
      expect(block.text()).toContain('加载中')
      expect(wrapper.find('[data-testid="task-fork-from-link"]').exists()).toBe(false)
      expect(block.text()).not.toContain('任务 task_')
      wrapper.unmount()
    })

    it('列表未加载时仍用详情接口序号展示 #N', async () => {
      const wrapper = await mountPanel({
        workspaceTodos: [],
        forkSourceSeq: 12,
      })
      const link = wrapper.find('[data-testid="task-fork-from-link"]')
      expect(link.text().trim()).toBe('#12 源任务名称')
      wrapper.unmount()
    })

    it('无标题时回退为 #序号', async () => {
      const wrapper = await mountPanel({
        forkSourceTitle: '',
        isForkSourceTitleLoading: false,
      })
      const link = wrapper.find('[data-testid="task-fork-from-link"]')
      expect(link.text().trim()).toBe('#12')
      wrapper.unmount()
    })

    it('无 fork_from 时不展示区块', async () => {
      const wrapper = await mountPanel({
        task: {
          id: 'task_1',
          title: '当前任务',
          description: '',
          priority: 1,
          created_at: '2026-07-13T00:00:00Z',
          auto_run: false,
          parent_task: null,
          fork_from: null,
        },
      })
      expect(wrapper.find('[data-testid="task-fork-from"]').exists()).toBe(false)
      wrapper.unmount()
    })
  })
}

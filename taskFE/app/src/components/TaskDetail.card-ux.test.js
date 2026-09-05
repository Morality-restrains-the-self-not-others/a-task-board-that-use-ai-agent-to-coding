// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] TaskDetail.card-ux.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { describe, expect, it, vi } = await import('vitest')

  vi.mock('vue-router', () => ({
    useRoute: () => ({ query: {} }),
  }))

  const { apiFetch } = vi.hoisted(() => ({
    apiFetch: vi.fn(async () => ({
      ok: true,
      json: async () => ({ container_image_at_mode_enabled: false }),
    })),
  }))

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch,
  }))

  const { default: TaskDetail } = await import('./TaskDetail.vue')
  const { default: toastService } = await import('../utils/toastService')

  const baseTask = {
    id: 'task-1001',
    title: '示例任务',
    priority: 1,
    description: 'desc',
    created_by: { username: 'alice' },
    workspace_seq: 12,
    created_at: '2026-07-13T10:00:00Z',
    comments: [],
    owner: 'm-owner',
    operator: 'm-op',
    assignees: ['m-c1', 'm-c2'],
  }

  const nameById = {
    'm-owner': '王五',
    'm-op': '赵六',
    'm-c1': '张三',
    'm-c2': '李四',
  }

  const statuses = [
    { id: 'col-a', name: '待处理' },
    { id: 'col-b', name: '进行中' },
  ]

  describe('TaskDetail 看板卡片 UX', () => {
    it('编号、进度、优先级同一行，标题单独一行', () => {
      const wrapper = mount(TaskDetail, {
        props: {
          task: { ...baseTask, progress_column_id: 'col-a' },
          tenantId: 't1',
          workspaceId: 'ws1',
          taskStatuses: statuses,
          collaboratorNameById: nameById,
        },
      })
      const meta = wrapper.get('[data-testid="task-card-meta-row"]')
      expect(meta.find('[data-testid="task-card-id"]').exists()).toBe(true)
      expect(meta.find('select.task-progress-select').exists()).toBe(true)
      expect(meta.find('[data-testid="task-card-priority"]').exists()).toBe(true)
      expect(meta.find('h4').exists()).toBe(false)
      const titleEl = wrapper.get('[data-testid="task-card-title"]')
      expect(titleEl.text()).toBe('示例任务')
      expect(meta.element.contains(titleEl.element)).toBe(false)
    })

    it('进度状态下拉无 label 文案，仅 select + aria-label', () => {
      const wrapper = mount(TaskDetail, {
        props: {
          task: { ...baseTask, progress_column_id: 'col-a' },
          tenantId: 't1',
          workspaceId: 'ws1',
          taskStatuses: statuses,
        },
      })
      expect(wrapper.find('label').exists()).toBe(false)
      expect(wrapper.text()).not.toContain('进度状态')
      const select = wrapper.get('select.task-progress-select')
      expect(select.attributes('aria-label')).toBe('进度状态')
    })

    it('展示操作员(单人)、负责人(单人)、协作者(可多人)', () => {
      const wrapper = mount(TaskDetail, {
        props: {
          task: baseTask,
          collaboratorNameById: nameById,
        },
      })
      const operator = wrapper.get('[data-testid="task-card-operator"]')
      expect(operator.text()).toContain('操作员')
      expect(wrapper.get('[data-testid="task-card-operator-name"]').text()).toBe('赵六')
      expect(operator.attributes('title')).toBe('操作员：赵六')
      const owner = wrapper.get('[data-testid="task-card-owner"]')
      expect(owner.text()).toContain('负责人')
      expect(wrapper.get('[data-testid="task-card-owner-name"]').text()).toBe('王五')
      expect(owner.attributes('title')).toBe('负责人：王五')
      const collaborators = wrapper.get('[data-testid="task-card-collaborators"]')
      expect(collaborators.text()).toContain('协作者')
      expect(wrapper.get('[data-testid="task-card-collaborators-name"]').text()).toBe('张三、李四')
      const createdAt = wrapper.get('[data-testid="task-card-created-at"]')
      expect(createdAt.text()).toContain('创建于')
      expect(createdAt.text()).toMatch(/7月13日/)
    })

    it('操作员/负责人/协作者缺失时显示「未指派」', () => {
      const wrapper = mount(TaskDetail, {
        props: {
          task: { ...baseTask, owner: '', operator: '', assignees: [] },
        },
      })
      expect(wrapper.get('[data-testid="task-card-operator-name"]').text()).toBe('未指派')
      expect(wrapper.get('[data-testid="task-card-owner-name"]').text()).toBe('未指派')
      expect(wrapper.get('[data-testid="task-card-collaborators-name"]').text()).toBe('未指派')
    })

    it('点击「评论」展开下方评论区与输入框；再次点击收起', async () => {
      const wrapper = mount(TaskDetail, {
        props: { task: baseTask },
      })
      expect(wrapper.find('[data-testid="task-card-comments-panel"]').exists()).toBe(false)

      await wrapper.get('[data-testid="task-card-toggle-comments"]').trigger('click')
      await flushPromises()
      expect(wrapper.find('[data-testid="task-card-comments-panel"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="task-card-comment-input"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="task-card-comment-composer"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="task-detail-comment-submit"]').exists()).toBe(true)
      expect(wrapper.get('[contenteditable="true"]').attributes('aria-label')).toContain('写下你的评论')
      expect(wrapper.text()).toContain('暂无评论')
      expect(wrapper.text()).not.toContain('添加评论')

      await wrapper.get('[data-testid="task-card-toggle-comments"]').trigger('click')
      expect(wrapper.find('[data-testid="task-card-comments-panel"]').exists()).toBe(false)
    })

    it('运行态圆点悬停提示：仅容器时文案为「容器运行中」', async () => {
      const wrapper = mount(TaskDetail, {
        props: {
          task: baseTask,
          runtimeIndicator: { machineRunning: false, containerRunning: true },
        },
      })
      const dot = wrapper.get('[data-testid="task-card-container-dot"]')
      expect(dot.attributes('title')).toBe('容器运行中')
      expect(dot.attributes('aria-label')).toBe('容器运行中')
      expect(wrapper.find('[data-testid="task-card-machine-ring"]').exists()).toBe(false)

      await dot.trigger('mouseenter')
      expect(wrapper.get('[data-testid="task-card-runtime-tooltip"]').text()).toBe('容器运行中')
    })

    it('运行态圆点悬停提示：机器+容器时各自独立文案', async () => {
      const wrapper = mount(TaskDetail, {
        props: {
          task: baseTask,
          runtimeIndicator: { machineRunning: true, containerRunning: true },
        },
      })
      const ring = wrapper.get('[data-testid="task-card-machine-ring"]')
      const dot = wrapper.get('[data-testid="task-card-container-dot"]')
      expect(ring.attributes('title')).toBe('机器节点运行中')
      expect(dot.attributes('title')).toBe('容器运行中')

      await ring.trigger('mouseenter')
      expect(wrapper.get('[data-testid="task-card-runtime-tooltip"]').text()).toBe('机器节点运行中')

      await dot.trigger('mouseenter')
      expect(wrapper.get('[data-testid="task-card-runtime-tooltip"]').text()).toBe('容器运行中')

      await dot.trigger('mouseleave')
      expect(wrapper.get('[data-testid="task-card-runtime-tooltip"]').text()).toBe('机器节点运行中')
    })

    it('任务编号可点击复制 #序号且不触发 task-clicked', async () => {
      const writeText = vi.fn().mockResolvedValue(undefined)
      Object.defineProperty(navigator, 'clipboard', {
        configurable: true,
        value: { writeText },
      })
      const successSpy = vi.spyOn(toastService, 'success').mockImplementation(() => {})

      const wrapper = mount(TaskDetail, {
        props: {
          task: { ...baseTask, id: 'task_850256677331014872', workspace_seq: 12 },
        },
      })
      const idEl = wrapper.get('[data-testid="task-card-id"]')
      expect(idEl.text()).toBe('#12')
      expect(idEl.classes()).toContain('cursor-pointer')

      await idEl.trigger('click')
      expect(writeText).toHaveBeenCalledWith('#12')
      expect(wrapper.emitted('task-clicked')).toBeUndefined()
      expect(successSpy).toHaveBeenCalledWith(expect.stringContaining('#12'), expect.any(Number))
      successSpy.mockRestore()
    })
  })
}

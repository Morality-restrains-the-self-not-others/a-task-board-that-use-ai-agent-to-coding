// @vitest-environment jsdom
// OPT-20260808-021 回归测试：Sortable 仅在看板结构真实变化（任务增删/跨列移动/
// 列、类别、过滤栏变更）时重建；内容更新（标题/评论/运行态）不触发重建；
// 重建经 100ms 防抖合并。
if (!process.env.VITEST) {
  console.log('[skip] TaskPanel.structureSignature.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { describe, expect, it, vi, beforeEach } = await import('vitest')

  const mocks = vi.hoisted(() => ({
    sortable: vi.fn(),
  }))

  vi.mock('sortablejs', () => ({ default: mocks.sortable }))

  const { default: TaskPanel } = await import('./TaskPanel.vue')

  const STATUSES = [
    { id: 0, name: '待处理', color: '#ff6b6b' },
    { id: 1, name: '进行中', color: '#4dabf7' },
    { id: 2, name: '已完成', color: '#51cf66' },
  ]
  const TASK_TYPES = [{ id: 'cat1', name: '交付物' }]

  const todo = (id, col, title = '任务') => ({
    id,
    title,
    progress_column_id: String(col),
    content: '',
    category: TASK_TYPES[0],
  })

  /** 注入看板容器 DOM（initDragDrop 只按选择器查找，不依赖 Vue 渲染结果） */
  function installBoardDom(columnIds) {
    const html = `<div id="task-panel-container">${columnIds
      .map((c) => `<div class="task-cards-container" data-progress-column-id="${c}"></div>`)
      .join('')}</div>`
    document.body.innerHTML = html
  }

  function mountPanel(todos) {
    return mount(TaskPanel, {
      props: {
        tenantId: 't1',
        workspaceId: 'ws1',
        taskStatuses: STATUSES,
        taskTypes: TASK_TYPES,
        todos,
        runtimeIndicators: {},
        collaboratorNameById: {},
      },
      global: {
        stubs: {
          DeliverableBreadcrumb: { template: '<div class="stub-breadcrumb" />' },
          DeliverableBoardSection: { template: '<div class="stub-section"><slot /></div>' },
          DeliverableKanbanBoard: { template: '<div class="stub-kanban" />' },
        },
      },
    })
  }

  const waitDragInit = () => new Promise((r) => setTimeout(r, 120))

  beforeEach(() => {
    mocks.sortable.mockClear()
    mocks.sortable.mockImplementation(() => ({ destroy: () => {} }))
  })

  describe('TaskPanel Sortable 重建策略 OPT-20260808-021', () => {
    it('挂载后调度一次重建（每进度列容器构造一个 Sortable 实例）', async () => {
      installBoardDom([0, 1, 2])
      const wrapper = mountPanel([todo('a', 0)])
      await flushPromises()
      await waitDragInit()
      expect(mocks.sortable).toHaveBeenCalledTimes(3)
      wrapper.unmount()
    })

    it('内容更新（标题变化，结构不变）不触发 Sortable 重建', async () => {
      installBoardDom([0, 1, 2])
      const wrapper = mountPanel([todo('a', 0), todo('b', 1)])
      await flushPromises()
      await waitDragInit()
      expect(mocks.sortable).toHaveBeenCalledTimes(3)

      // 同结构内容更新：改标题（不换列、不增删任务）
      const next = [todo('a', 0, '标题已更新'), todo('b', 1)]
      await wrapper.setProps({ todos: next })
      await flushPromises()
      await waitDragInit()
      expect(mocks.sortable).toHaveBeenCalledTimes(3)
      wrapper.unmount()
    })

    it('跨列移动任务触发重建（结构签名变化）', async () => {
      installBoardDom([0, 1, 2])
      const wrapper = mountPanel([todo('a', 0), todo('b', 1)])
      await flushPromises()
      await waitDragInit()
      expect(mocks.sortable).toHaveBeenCalledTimes(3)

      // b 从列 1 移到列 2：每列任务 id 集合变化 → 签名变化 → 重建
      await wrapper.setProps({ todos: [todo('a', 0), todo('b', 2)] })
      await flushPromises()
      await waitDragInit()
      expect(mocks.sortable).toHaveBeenCalledTimes(6)
      wrapper.unmount()
    })

    it('100ms 防抖窗口内连续结构变化只重建一次', async () => {
      installBoardDom([0, 1, 2])
      const wrapper = mountPanel([todo('a', 0)])
      await flushPromises()
      await waitDragInit()
      expect(mocks.sortable).toHaveBeenCalledTimes(3)

      // 防抖窗口内两次结构变化（先加任务 c，再加任务 d），应合并为一次重建
      await wrapper.setProps({ todos: [todo('a', 0), todo('c', 1)] })
      await flushPromises()
      await wrapper.setProps({ todos: [todo('a', 0), todo('c', 1), todo('d', 2)] })
      await flushPromises()
      await waitDragInit()
      expect(mocks.sortable).toHaveBeenCalledTimes(6)
      wrapper.unmount()
    })

    it('任务增删触发重建', async () => {
      installBoardDom([0, 1, 2])
      const wrapper = mountPanel([todo('a', 0)])
      await flushPromises()
      await waitDragInit()
      expect(mocks.sortable).toHaveBeenCalledTimes(3)

      // 删除任务 a（列 0 空 → 签名变化）
      await wrapper.setProps({ todos: [] })
      await flushPromises()
      await waitDragInit()
      expect(mocks.sortable).toHaveBeenCalledTimes(6)
      wrapper.unmount()
    })
  })
}

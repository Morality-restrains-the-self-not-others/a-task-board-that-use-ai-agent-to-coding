// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] WorkPanelHeader.legend.test.js requires vitest runtime')
} else {
  const { mount } = await import('@vue/test-utils')
  const { describe, expect, it, vi } = await import('vitest')

  vi.mock('../components/WorkspaceSwitcher.vue', () => ({
    default: { name: 'WorkspaceSwitcher', template: '<div />' },
  }))

  const { default: WorkPanelHeader } = await import('./WorkPanelHeader.vue')
  const {
    RUNTIME_MACHINE_TIP,
    RUNTIME_CONTAINER_TIP,
  } = await import('../utils/workPanelRuntimeIndicators.js')

  describe('WorkPanelHeader 机器节点图例', () => {
    it('有汇总时主视图紧凑展示，图例在详情提示中', async () => {
      const wrapper = mount(WorkPanelHeader, {
        props: {
          machineSummary: {
            startedCount: 1,
            idleCount: 0,
            startingCount: 0,
            idleRecycleMinutes: 30,
          },
          machineSummaryLabel: '机器节点：已启动 1 · 闲置 0 · 闲置回收：30 分钟',
        },
      })
      const summary = wrapper.get('[data-alias="workspace-machine-summary"]')
      expect(summary.text()).not.toContain('空心=已启动机器')
      expect(summary.get('[data-testid="machine-summary-started-count"]').text()).toBe('1')

      await summary.get('[data-alias="workspace-machine-summary-info"]').trigger('mouseenter')
      const tooltip = wrapper.get('[data-testid="workspace-machine-summary-tooltip"]')
      expect(tooltip.text()).toContain('空心=已启动机器')
      expect(tooltip.text()).toContain('实心=已启动容器')
      expect(tooltip.text()).toContain(RUNTIME_MACHINE_TIP)
      expect(tooltip.text()).toContain(RUNTIME_CONTAINER_TIP)
      expect(tooltip.text()).toContain('闲置回收：30 分钟')
    })

    it('无汇总时展示占位符，图例在详情提示中', async () => {
      const wrapper = mount(WorkPanelHeader, {
        props: {
          machineSummary: null,
          machineSummaryLabel: '机器节点：—',
        },
      })
      expect(wrapper.get('[data-alias="workspace-machine-summary"]').text()).toContain('—')

      await wrapper.get('[data-alias="workspace-machine-summary-info"]').trigger('mouseenter')
      expect(wrapper.get('[data-testid="workspace-machine-summary-tooltip"]').text()).toContain('空心=已启动机器')
    })
  })

  describe('WorkPanelHeader 标题行布局', () => {
    it('工作空间切换与「工作面板」标题同一工具栏且操作区靠右对齐', () => {
      const wrapper = mount(WorkPanelHeader, {
        props: {
          tenantId: '123',
          machineSummary: null,
          machineSummaryLabel: '机器节点：—',
        },
      })
      const toolbar = wrapper.get('[data-alias="header-toolbar-row"]')
      expect(toolbar.get('h2').text()).toBe('工作面板')
      const actionsRow = toolbar.get('[data-alias="header-actions-row"]')
      expect(actionsRow.get('[data-alias="header-workspace-switcher-row"]').exists()).toBe(true)
      expect(actionsRow.classes()).toContain('ml-auto')
    })

    it('有 tenantId 时工具栏在标题与工作空间切换之间渲染任务搜索框', () => {
      const wrapper = mount(WorkPanelHeader, {
        props: {
          tenantId: '123',
          machineSummary: null,
          machineSummaryLabel: '机器节点：—',
        },
      })
      const toolbar = wrapper.get('[data-alias="header-toolbar-row"]')
      const titleRow = toolbar.get('[data-alias="header-title-row"]')
      const search = titleRow.find('[data-testid="work-panel-task-search"]')
      expect(search.exists()).toBe(true)
      expect(titleRow.find('[data-testid="work-panel-task-search-input"]').exists()).toBe(true)
      const html = toolbar.html()
      expect(html.indexOf('工作面板')).toBeLessThan(html.indexOf('work-panel-task-search'))
      expect(html.indexOf('work-panel-task-search')).toBeLessThan(html.indexOf('header-workspace-switcher-row'))
    })

    it('无 tenantId 时不渲染任务搜索框', () => {
      const wrapper = mount(WorkPanelHeader, {
        props: {
          tenantId: null,
          machineSummary: null,
          machineSummaryLabel: '机器节点：—',
        },
      })
      expect(wrapper.find('[data-testid="work-panel-task-search"]').exists()).toBe(false)
    })

    it('标题区与机器摘要区在同一工具栏行内并排，不是上下两块', () => {
      const wrapper = mount(WorkPanelHeader, {
        props: {
          tenantId: '123',
          machineSummary: {
            startedCount: 1,
            idleCount: 0,
            startingCount: 0,
            idleRecycleMinutes: 30,
          },
          machineSummaryLabel: '机器节点：已启动 1 · 闲置 0 · 闲置回收：30 分钟',
        },
      })
      const toolbar = wrapper.get('[data-alias="header-toolbar-row"]')
      expect(toolbar.classes()).toContain('flex')
      expect(toolbar.classes()).toContain('flex-wrap')
      expect(toolbar.classes()).not.toContain('flex-col')

      const titleRow = toolbar.get('[data-alias="header-title-row"]')
      const summaryRow = toolbar.get('[data-alias="header-summary-row"]')
      expect(titleRow.element.parentElement).toBe(toolbar.element)
      expect(summaryRow.element.parentElement).toBe(toolbar.element)
      expect(titleRow.classes()).not.toContain('w-full')
      expect(summaryRow.classes()).not.toContain('w-full')
      expect(summaryRow.classes()).not.toContain('mt-2')
      expect(titleRow.element.nextElementSibling).toBe(summaryRow.element)

      expect(summaryRow.get('[data-alias="workspace-machine-summary"]').exists()).toBe(true)
      expect(summaryRow.get('#create-task-btn').text()).toBe('创建任务')
      expect(titleRow.text()).not.toContain('机器节点')
    })
  })
}

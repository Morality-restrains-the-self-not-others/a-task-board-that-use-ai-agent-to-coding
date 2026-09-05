// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] WorkPanelHeader.emptyHint.test.js requires vitest runtime')
} else {
  const { mount } = await import('@vue/test-utils')
  const { describe, expect, it, vi } = await import('vitest')

  vi.mock('../components/WorkspaceSwitcher.vue', () => ({
    default: { name: 'WorkspaceSwitcher', template: '<div />' },
  }))

  const { default: WorkPanelHeader } = await import('./WorkPanelHeader.vue')

  const hintTraceId = (wrapper) => {
    const el = wrapper.get('[data-alias="work-panel-empty-task-hint"]').element
    for (const attr of el.attributes) {
      if (attr.name.toLowerCase() === 'data-traceid') return attr.value
    }
    return undefined
  }

  describe('WorkPanelHeader 空任务 / 加载失败提示', () => {
    it('任务列表加载失败时展示错误文案与 data-traceId，不展示机器不一致误报', () => {
      const wrapper = mount(WorkPanelHeader, {
        props: {
          showEmptyTaskHint: false,
          todosLoadError: 'workspace not found',
          todosLoadErrorTraceId: 'tid-todos-404',
          machineSummary: {
            startedCount: 1,
            idleCount: 0,
            startingCount: 0,
            idleRecycleMinutes: 30,
          },
        },
      })
      const hint = wrapper.get('[data-alias="work-panel-empty-task-hint"]')
      expect(hint.text()).toContain('任务列表加载失败')
      expect(hint.text()).toContain('workspace not found')
      expect(hint.text()).not.toContain('请刷新页面或检查任务是否加载失败')
      expect(hintTraceId(wrapper)).toBe('tid-todos-404')
    })

    it('加载成功且列表为空、机器已启动时仍展示不一致提示', () => {
      const wrapper = mount(WorkPanelHeader, {
        props: {
          showEmptyTaskHint: true,
          todosLoadError: '',
          machineSummary: {
            startedCount: 1,
            idleCount: 0,
            startingCount: 0,
            idleRecycleMinutes: 30,
          },
        },
      })
      expect(wrapper.get('[data-alias="work-panel-empty-task-hint"]').text()).toContain(
        '任务列表为空，但机器节点显示已启动 1 台',
      )
    })
  })
}

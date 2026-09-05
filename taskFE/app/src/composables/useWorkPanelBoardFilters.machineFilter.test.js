if (!process.env.VITEST) {
  console.log('[skip] useWorkPanelBoardFilters.machineFilter.test.js requires vitest runtime')
} else {
const { describe, expect, it, vi } = await import('vitest')
const { computed, nextTick } = await import('vue')
// vi.mock 工厂中引用的变量必须通过 vi.hoisted() 定义（Vitest 会把工厂 hoist 到模块顶部）。
const { ref } = vi.hoisted(() => ({ ref: require('vue').ref }))
const { useWorkPanelBoardFilters } = await import('./useWorkPanelBoardFilters.js')

vi.mock('../utils/workPanelMachineSummary.js', () => ({
  fetchWorkspaceMachineSummary: vi.fn(async () => ({
    startedCount: 1,
    startingCount: 0,
    idleCount: 0,
    busyCount: 1,
    idleRecycleMinutes: 30,
  })),
  formatWorkspaceMachineSummaryLabel: () => '机器节点：已启动 1 · 闲置 0 · 闲置回收：30 分钟',
}))

vi.mock('../utils/workPanelRuntimeIndicators.js', () => ({
  fetchWorkspaceRuntimeIndicators: vi.fn(async () => ({
    task_running: { machineRunning: true, containerRunning: true },
  })),
}))

vi.mock('./useWorkPanelAccessFilter.js', () => ({
  useWorkPanelAccessFilter: () => ({
    accessFilter: ref(null),
    accessPeople: ref([]),
    accessGroups: ref([]),
    accessSubjectsLoading: ref(false),
    accessFilterPanelOpen: ref(false),
    accessFilterTab: ref('person'),
    selectAccessSubject: vi.fn(),
    hydrateAccessFilterPref: vi.fn(),
    clearAccessFilter: vi.fn(),
    toggleAccessFilterPanel: vi.fn(),
    closeAccessFilterPanel: vi.fn(),
    setAccessFilterTab: vi.fn(),
    resetAccessFilter: vi.fn(),
    refreshAccessSubjects: vi.fn(async () => {}),
  }),
}))

describe('useWorkPanelBoardFilters machine filter board bars', () => {
  it('machine filter active uses root deliverable bars so matches are not hidden by runIt content filter', async () => {
    const todos = ref([
      { id: 'task_runit', title: 'runIt' },
      { id: 'task_running', title: '修改 GitLab 配置页' },
    ])
    const deliverableFilterBars = ref([
      {
        id: 'bar-0',
        path: [
          { type: 'root' },
          { type: 'category', id: 'cat1', label: '价值流' },
          { type: 'task', id: 'task_runit', label: 'runIt' },
        ],
      },
    ])
    const apiFetch = vi.fn()
    const board = useWorkPanelBoardFilters({
      apiFetch,
      tenantId: ref('t1'),
      currentWorkspace: ref({ id: 'ws1' }),
      todos,
      collaborators: ref([]),
      workspaceRefreshTrigger: ref(0),
      showEmptyTaskHint: computed(() => false),
      deliverableFilterBars,
    })

    await board.refreshBoardFilterData()
    await nextTick()

    board.handleMachineRuntimeFilter('started')
    await nextTick()

    expect(board.machineRuntimeFilter.value).toBe('started')
    expect(board.boardDeliverableFilterBars.value).toEqual([
      expect.objectContaining({
        path: [expect.objectContaining({ type: 'root' })],
      }),
    ])
    expect(board.filteredTodos.value.map((t) => t.id)).toEqual(['task_running'])
  })

  it('headerBind 透传任务列表加载失败与 traceId，避免空看板误报', () => {
    const todosError = ref('workspace not found')
    const todosErrorTraceId = ref('tid-todos-404')
    const board = useWorkPanelBoardFilters({
      apiFetch: vi.fn(),
      tenantId: ref('t1'),
      currentWorkspace: ref({ id: 'ws1' }),
      todos: ref([]),
      todosError,
      todosErrorTraceId,
      collaborators: ref([]),
      workspaceRefreshTrigger: ref(0),
      showEmptyTaskHint: computed(() => false),
    })
    expect(board.headerBind.value.todosLoadError).toBe('workspace not found')
    expect(board.headerBind.value.todosLoadErrorTraceId).toBe('tid-todos-404')
    expect(board.headerBind.value.showEmptyTaskHint).toBe(false)
  })
})

}

if (!process.env.VITEST) {
  console.log('[skip] useWorkPanelMachineSummary.pair.test.js requires vitest runtime')
} else {
const { describe, expect, it, vi, beforeEach, afterEach } = await import('vitest')
const { ref } = await import('vue')

// 头部摘要与看板卡片指示器必须始终来自同一次成功快照：
// 任一侧请求失败时不得单侧更新（否则出现「已启动 0」与卡片实心点互斥的显示不一致）。
const { summaryPayload, indicatorsPayload } = vi.hoisted(() => ({
  summaryPayload: {
    startedCount: 1,
    startingCount: 0,
    idleCount: 2,
    busyCount: 0,
    idleRecycleMinutes: 30,
  },
  indicatorsPayload: {
    task_a: { machineRunning: true, containerRunning: true },
  },
}))

vi.mock('../utils/workPanelMachineSummary.js', () => ({
  fetchWorkspaceMachineSummary: vi.fn(async () => ({ ...summaryPayload })),
  formatWorkspaceMachineSummaryLabel: () => 'label',
}))

vi.mock('../utils/workPanelRuntimeIndicators.js', () => ({
  fetchWorkspaceRuntimeIndicators: vi.fn(async () => ({ ...indicatorsPayload })),
}))

const { useWorkPanelMachineSummary } = await import('./useWorkPanelMachineSummary.js')

function createMachine() {
  return useWorkPanelMachineSummary({
    apiFetch: vi.fn(),
    tenantId: ref('t1'),
    currentWorkspace: ref({ id: 'ws1' }),
    todos: ref([]),
  })
}

describe('useWorkPanelMachineSummary 成对原子刷新（摘要 vs 卡片指示器一致性）', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.clearAllMocks()
  })
  afterEach(() => {
    vi.unstubAllGlobals()
    vi.useRealTimers()
  })

  it('summary 请求失败而 indicators 成功：两者均保持旧快照，不单侧更新', async () => {
    const machine = createMachine()
    // 基准轮询：两侧都成功
    await machine.refreshMachineSummary()
    const summaryBefore = machine.machineSummary.value
    const indicatorsBefore = machine.runtimeIndicators.value
    expect(summaryBefore.startedCount).toBe(1)

    const { fetchWorkspaceMachineSummary } = await import('../utils/workPanelMachineSummary.js')
    const { fetchWorkspaceRuntimeIndicators } = await import('../utils/workPanelRuntimeIndicators.js')

    // 下一轮：summary 失败（瞬时 502），indicators 返回新数据
    fetchWorkspaceMachineSummary.mockRejectedValueOnce(new Error('HTTP 502'))
    fetchWorkspaceRuntimeIndicators.mockResolvedValueOnce({
      task_b: { machineRunning: false, containerRunning: true },
    })
    await machine.refreshMachineSummary()

    // 两侧都不允许被更新 → 头部与卡片永远展示同一快照
    expect(machine.machineSummary.value).toBe(summaryBefore)
    expect(machine.runtimeIndicators.value).toBe(indicatorsBefore)
    expect(machine.runtimeIndicators.value.task_b).toBeUndefined()
  })

  it('indicators 请求失败而 summary 成功：两者均保持旧快照，不单侧更新', async () => {
    const machine = createMachine()
    await machine.refreshMachineSummary()
    const summaryBefore = machine.machineSummary.value
    const indicatorsBefore = machine.runtimeIndicators.value

    const { fetchWorkspaceMachineSummary } = await import('../utils/workPanelMachineSummary.js')
    const { fetchWorkspaceRuntimeIndicators } = await import('../utils/workPanelRuntimeIndicators.js')

    // 下一轮：indicators 失败（瞬时网络错误），summary 返回新计数
    fetchWorkspaceMachineSummary.mockResolvedValueOnce({
      startedCount: 9,
      startingCount: 0,
      idleCount: 0,
      busyCount: 0,
      idleRecycleMinutes: 10,
    })
    fetchWorkspaceRuntimeIndicators.mockRejectedValueOnce(new Error('network down'))
    await machine.refreshMachineSummary()

    expect(machine.machineSummary.value).toBe(summaryBefore)
    expect(machine.runtimeIndicators.value).toBe(indicatorsBefore)
    expect(machine.machineSummary.value.startedCount).toBe(1)
  })

  it('两侧同时失败：保持旧快照且不抛错', async () => {
    const machine = createMachine()
    await machine.refreshMachineSummary()
    const summaryBefore = machine.machineSummary.value
    const indicatorsBefore = machine.runtimeIndicators.value

    const { fetchWorkspaceMachineSummary } = await import('../utils/workPanelMachineSummary.js')
    const { fetchWorkspaceRuntimeIndicators } = await import('../utils/workPanelRuntimeIndicators.js')
    fetchWorkspaceMachineSummary.mockRejectedValueOnce(new Error('HTTP 500'))
    fetchWorkspaceRuntimeIndicators.mockRejectedValueOnce(new Error('HTTP 500'))

    await expect(machine.refreshMachineSummary()).resolves.toBeUndefined()
    expect(machine.machineSummary.value).toBe(summaryBefore)
    expect(machine.runtimeIndicators.value).toBe(indicatorsBefore)
  })

  it('两侧都成功：同轮新快照成对生效', async () => {
    const machine = createMachine()
    await machine.refreshMachineSummary()

    const { fetchWorkspaceMachineSummary } = await import('../utils/workPanelMachineSummary.js')
    const { fetchWorkspaceRuntimeIndicators } = await import('../utils/workPanelRuntimeIndicators.js')
    fetchWorkspaceMachineSummary.mockResolvedValueOnce({
      startedCount: 7,
      startingCount: 1,
      idleCount: 0,
      busyCount: 0,
      idleRecycleMinutes: 30,
    })
    fetchWorkspaceRuntimeIndicators.mockResolvedValueOnce({
      task_z: { machineRunning: true, containerRunning: false },
    })
    await machine.refreshMachineSummary()

    expect(machine.machineSummary.value.startedCount).toBe(7)
    expect(machine.runtimeIndicators.value.task_z).toEqual({
      machineRunning: true,
      containerRunning: false,
    })
  })

  it('首次轮询任一侧失败：保持初始空态（头部「—」与卡片不亮一致）', async () => {
    const machine = createMachine()

    const { fetchWorkspaceMachineSummary } = await import('../utils/workPanelMachineSummary.js')
    fetchWorkspaceMachineSummary.mockRejectedValueOnce(new Error('HTTP 502'))
    await machine.refreshMachineSummary()

    expect(machine.machineSummary.value).toBeNull()
    expect(Object.keys(machine.runtimeIndicators.value)).toHaveLength(0)
    expect(machine.runtimeIndicatorsReady.value).toBe(true)
  })

  it('OPT-012: 摘要轮询失败时 machineSummaryErrorTraceId 记录失败请求 traceId（header-summary-row 数据源）', async () => {
    const machine = createMachine()
    await machine.refreshMachineSummary() // 基准：两侧成功 → traceId 为空

    const { fetchWorkspaceMachineSummary } = await import('../utils/workPanelMachineSummary.js')
    const { fetchWorkspaceRuntimeIndicators } = await import('../utils/workPanelRuntimeIndicators.js')
    const failErr = new Error('HTTP 502')
    failErr.traceId = 'trace-summary-001'
    fetchWorkspaceMachineSummary.mockRejectedValueOnce(failErr)
    fetchWorkspaceRuntimeIndicators.mockResolvedValueOnce({})

    await machine.refreshMachineSummary()

    expect(machine.machineSummaryErrorTraceId.value).toBe('trace-summary-001')
  })

  it('OPT-012: indicators 轮询失败而 summary 成功：回退记录 indicators 侧 traceId', async () => {
    const machine = createMachine()
    await machine.refreshMachineSummary()

    const { fetchWorkspaceMachineSummary } = await import('../utils/workPanelMachineSummary.js')
    const { fetchWorkspaceRuntimeIndicators } = await import('../utils/workPanelRuntimeIndicators.js')
    const failErr = new Error('network down')
    failErr.traceId = 'trace-indicators-002'
    fetchWorkspaceMachineSummary.mockResolvedValueOnce({
      startedCount: 2,
      startingCount: 0,
      idleCount: 1,
      busyCount: 0,
      idleRecycleMinutes: 30,
    })
    fetchWorkspaceRuntimeIndicators.mockRejectedValueOnce(failErr)

    await machine.refreshMachineSummary()

    expect(machine.machineSummaryErrorTraceId.value).toBe('trace-indicators-002')
  })

  it('OPT-012: 成对成功后清除失败 traceId（恢复路径）', async () => {
    const machine = createMachine()
    await machine.refreshMachineSummary()

    const { fetchWorkspaceMachineSummary } = await import('../utils/workPanelMachineSummary.js')
    const { fetchWorkspaceRuntimeIndicators } = await import('../utils/workPanelRuntimeIndicators.js')

    // 先失败 → traceId 记录
    const failErr = new Error('HTTP 502')
    failErr.traceId = 'trace-summary-003'
    fetchWorkspaceMachineSummary.mockRejectedValueOnce(failErr)
    await machine.refreshMachineSummary()
    expect(machine.machineSummaryErrorTraceId.value).toBe('trace-summary-003')

    // 再成功 → 清除
    fetchWorkspaceMachineSummary.mockResolvedValueOnce({
      startedCount: 1,
      startingCount: 0,
      idleCount: 2,
      busyCount: 0,
      idleRecycleMinutes: 30,
    })
    fetchWorkspaceRuntimeIndicators.mockResolvedValueOnce({})
    await machine.refreshMachineSummary()
    expect(machine.machineSummaryErrorTraceId.value).toBe('')
  })

  it('OPT-012: 失败错误无 traceId 时不误写（保持空串）', async () => {
    const machine = createMachine()
    await machine.refreshMachineSummary()

    const { fetchWorkspaceMachineSummary } = await import('../utils/workPanelMachineSummary.js')
    fetchWorkspaceMachineSummary.mockRejectedValueOnce(new Error('HTTP 500'))
    await machine.refreshMachineSummary()

    expect(machine.machineSummaryErrorTraceId.value).toBe('')
  })

  // 注：vi.clearAllMocks 不清除 mockRejectedValue/mockResolvedValue 的持久实现，
  // 上一用例遗留的拒绝会污染本用例的「基准成对成功」。因此每个新用例显式重设
  // 两路 fetch 的基准实现后再 refresh，保证 lastSuccessAt 置位（存在可陈旧快照）。
  const setBaselineSuccess = async () => {
    const { fetchWorkspaceMachineSummary } = await import('../utils/workPanelMachineSummary.js')
    const { fetchWorkspaceRuntimeIndicators } = await import('../utils/workPanelRuntimeIndicators.js')
    fetchWorkspaceMachineSummary.mockResolvedValue({ ...summaryPayload })
    fetchWorkspaceRuntimeIndicators.mockResolvedValue({ ...indicatorsPayload })
  }
  const setSummaryFailure = async () => {
    const { fetchWorkspaceMachineSummary } = await import('../utils/workPanelMachineSummary.js')
    const { fetchWorkspaceRuntimeIndicators } = await import('../utils/workPanelRuntimeIndicators.js')
    fetchWorkspaceMachineSummary.mockRejectedValue(new Error('HTTP 502'))
    fetchWorkspaceRuntimeIndicators.mockResolvedValue({})
  }

  it('OPT-029: 连续失败未达阈值（2/3）时不标记陈旧', async () => {
    const machine = createMachine()
    await setBaselineSuccess()
    await machine.refreshMachineSummary() // 基准：成对成功 → lastSuccessAt 置位

    await setSummaryFailure()
    await machine.refreshMachineSummary() // 失败 1
    await machine.refreshMachineSummary() // 失败 2

    expect(machine.machineSummaryStaleSince.value).toBeNull()
  })

  it('OPT-029: 达阈值翻转——此前成功过 + 连续 3 轮失败 → staleSince 置位', async () => {
    const machine = createMachine()
    await setBaselineSuccess()
    await machine.refreshMachineSummary() // 基准：成对成功 → 存在可陈旧的新鲜快照

    await setSummaryFailure()
    await machine.refreshMachineSummary() // 失败 1
    await machine.refreshMachineSummary() // 失败 2
    expect(machine.machineSummaryStaleSince.value).toBeNull()

    await machine.refreshMachineSummary() // 失败 3 → 阈值翻转
    expect(machine.machineSummaryStaleSince.value).not.toBeNull()
    expect(typeof machine.machineSummaryStaleSince.value).toBe('number')
    // 旧快照仍保留（一致性不破坏）
    expect(machine.machineSummary.value.startedCount).toBe(1)
  })

  it('OPT-029: 恢复路径——连续失败标记陈旧后成对成功 → staleSince 清除', async () => {
    const machine = createMachine()
    await setBaselineSuccess()
    await machine.refreshMachineSummary() // 基准成功

    await setSummaryFailure()
    await machine.refreshMachineSummary()
    await machine.refreshMachineSummary()
    await machine.refreshMachineSummary() // 失败 3 → stale
    expect(machine.machineSummaryStaleSince.value).not.toBeNull()

    // 成对成功恢复
    await setBaselineSuccess()
    await machine.refreshMachineSummary()
    expect(machine.machineSummaryStaleSince.value).toBeNull()
    // 内容恢复为最新成功快照
    expect(machine.machineSummary.value.startedCount).toBe(1)
  })

  it('OPT-029: 从未成功过（首轮即连续失败）不标记陈旧（无可陈旧快照）', async () => {
    const machine = createMachine()

    await setSummaryFailure()
    await machine.refreshMachineSummary()
    await machine.refreshMachineSummary()
    await machine.refreshMachineSummary()

    expect(machine.machineSummaryStaleSince.value).toBeNull()
  })

  it('OPT-029: indicators 侧连续失败同样驱动陈旧标记（任一侧失败即计一轮）', async () => {
    const machine = createMachine()
    await setBaselineSuccess()
    await machine.refreshMachineSummary() // 基准成功

    const { fetchWorkspaceRuntimeIndicators } = await import('../utils/workPanelRuntimeIndicators.js')
    const { fetchWorkspaceMachineSummary } = await import('../utils/workPanelMachineSummary.js')
    fetchWorkspaceMachineSummary.mockResolvedValue({ ...summaryPayload })
    fetchWorkspaceRuntimeIndicators.mockRejectedValue(new Error('network down'))

    await machine.refreshMachineSummary()
    await machine.refreshMachineSummary()
    await machine.refreshMachineSummary()

    expect(machine.machineSummaryStaleSince.value).not.toBeNull()
  })
})
}

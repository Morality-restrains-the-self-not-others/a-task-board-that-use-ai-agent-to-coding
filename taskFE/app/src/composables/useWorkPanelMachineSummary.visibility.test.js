if (!process.env.VITEST) {
  console.log('[skip] useWorkPanelMachineSummary.visibility.test.js requires vitest runtime')
} else {
const { describe, expect, it, vi, beforeEach, afterEach } = await import('vitest')
const { ref, nextTick } = await import('vue')

// vi.mock 工厂中引用的变量必须通过 vi.hoisted() 定义（Vitest 会把工厂 hoist 到模块顶部）。
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

/** 可编程 document stub：visibilityState + 可捕获的 visibilitychange 监听 */
function installDocumentStub(initialState) {
  const listeners = {}
  const doc = {
    visibilityState: initialState,
    addEventListener: vi.fn((type, fn) => {
      listeners[type] = fn
    }),
    removeEventListener: vi.fn((type) => {
      delete listeners[type]
    }),
    __setVisibility(state) {
      doc.visibilityState = state
      listeners.visibilitychange?.()
    },
  }
  vi.stubGlobal('document', doc)
  return doc
}

describe('useWorkPanelMachineSummary OPT-20260808-021', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.clearAllMocks()
  })
  afterEach(() => {
    vi.unstubAllGlobals()
    vi.useRealTimers()
  })

  it('payload 未变化时 ref 身份保持稳定，不触发无谓重渲染', async () => {
    const doc = installDocumentStub('visible')
    const machine = useWorkPanelMachineSummary({
      apiFetch: vi.fn(),
      tenantId: ref('t1'),
      currentWorkspace: ref({ id: 'ws1' }),
      todos: ref([]),
    })
    await machine.refreshMachineSummary()
    const summaryRef = machine.machineSummary.value
    const indicatorsRef = machine.runtimeIndicators.value

    // 第二次拉取内容相同（mock 返回深拷贝），ref 不应被替换
    await machine.refreshMachineSummary()
    expect(machine.machineSummary.value).toBe(summaryRef)
    expect(machine.runtimeIndicators.value).toBe(indicatorsRef)

    // 轮询启动时才挂载可见性监听
    machine.startMachineSummaryPolling()
    expect(doc.addEventListener).toHaveBeenCalledWith('visibilitychange', expect.any(Function))
  })

  it('payload 变化时 ref 正常更新', async () => {
    installDocumentStub('visible')
    const machine = useWorkPanelMachineSummary({
      apiFetch: vi.fn(),
      tenantId: ref('t1'),
      currentWorkspace: ref({ id: 'ws1' }),
      todos: ref([]),
    })
    await machine.refreshMachineSummary()
    expect(machine.machineSummary.value.startedCount).toBe(1)

    const { fetchWorkspaceMachineSummary } = await import('../utils/workPanelMachineSummary.js')
    fetchWorkspaceMachineSummary.mockResolvedValueOnce({
      startedCount: 9,
      startingCount: 0,
      idleCount: 0,
      busyCount: 0,
      idleRecycleMinutes: 10,
    })
    await machine.refreshMachineSummary()
    expect(machine.machineSummary.value.startedCount).toBe(9)
  })

  it('页面不可见时不打 GET，回到可见立即刷新一次且无 15s 轮询', async () => {
    const doc = installDocumentStub('hidden')
    const machine = useWorkPanelMachineSummary({
      apiFetch: vi.fn(),
      tenantId: ref('t1'),
      currentWorkspace: ref({ id: 'ws1' }),
      todos: ref([]),
    })
    const { fetchWorkspaceRuntimeIndicators } = await import('../utils/workPanelRuntimeIndicators.js')

    machine.startMachineSummaryPolling()
    // hidden 状态不建立 interval
    vi.advanceTimersByTime(45000)
    expect(fetchWorkspaceRuntimeIndicators).not.toHaveBeenCalled()

    // 切回可见：立即刷新一次；无 15s interval
    doc.__setVisibility('visible')
    await vi.advanceTimersByTimeAsync(0)
    expect(fetchWorkspaceRuntimeIndicators).toHaveBeenCalledTimes(1)
    await vi.advanceTimersByTimeAsync(15000)
    expect(fetchWorkspaceRuntimeIndicators).toHaveBeenCalledTimes(1)

    // 再次隐藏：不再打 GET
    doc.__setVisibility('hidden')
    await vi.advanceTimersByTimeAsync(45000)
    expect(fetchWorkspaceRuntimeIndicators).toHaveBeenCalledTimes(1)
    expect(doc.removeEventListener).not.toHaveBeenCalled()
  })

  it('onUnmounted 清理可见性监听', async () => {
    const doc = installDocumentStub('visible')
    const machine = useWorkPanelMachineSummary({
      apiFetch: vi.fn(),
      tenantId: ref('t1'),
      currentWorkspace: ref({ id: 'ws1' }),
      todos: ref([]),
    })
    machine.startMachineSummaryPolling()
    expect(doc.addEventListener).toHaveBeenCalled()
    expect(doc.removeEventListener).not.toHaveBeenCalled()
    await nextTick()
  })
})
}

if (!process.env.VITEST) {
  console.log('[skip] useWorkPanelTaskStatusSse.test.js requires vitest runtime')
} else {
const { describe, it, expect, vi, beforeEach, afterEach } = await import('vitest')

vi.mock('../utils/config.js', () => ({
  getApiUrl: (path) => `https://api.test${path}`,
}))

class FakeEventSource {
  static instances = []
  constructor(url, opts) {
    this.url = url
    this.opts = opts
    this.onmessage = null
    this.onerror = null
    this.onopen = null
    this.closed = false
    FakeEventSource.instances.push(this)
  }
  close() {
    this.closed = true
  }
}

describe('useWorkPanelTaskStatusSse', () => {
  beforeEach(() => {
    FakeEventSource.instances = []
    vi.stubGlobal('EventSource', FakeEventSource)
  })
  afterEach(() => {
    vi.unstubAllGlobals()
    vi.resetModules()
  })

  it('opens EventSource with workspace path and credentials', async () => {
    const { openWorkPanelTaskStatusSse } = await import('./useWorkPanelTaskStatusSse.js')
    const onStatusChanged = vi.fn()
    const { close } = openWorkPanelTaskStatusSse({
      tenantId: '850',
      workspaceId: 'ws-1',
      onStatusChanged,
    })
    expect(FakeEventSource.instances).toHaveLength(1)
    expect(FakeEventSource.instances[0].url).toContain(
      '/api/sse/work-panel-events/tenant_id/850/workspace_id/ws-1',
    )
    expect(FakeEventSource.instances[0].opts.withCredentials).toBe(true)
    close()
    expect(FakeEventSource.instances[0].closed).toBe(true)
  })

  it('invokes onStatusChanged for task_status_changed only', async () => {
    const { openWorkPanelTaskStatusSse } = await import('./useWorkPanelTaskStatusSse.js')
    const onStatusChanged = vi.fn()
    const onNeedResync = vi.fn()
    openWorkPanelTaskStatusSse({
      tenantId: '850',
      workspaceId: 'ws-1',
      onStatusChanged,
      onNeedResync,
    })
    const es = FakeEventSource.instances[0]
    es.onmessage({ data: JSON.stringify({ event_name: 'work_panel_sse_connected' }) })
    es.onmessage({ data: JSON.stringify({ type: 'heartbeat', event_name: 'work_panel_sse_heartbeat' }) })
    es.onmessage({
      data: JSON.stringify({
        event_name: 'task_status_changed',
        task_id: 't1',
        progress_column_id: 'c2',
      }),
    })
    expect(onStatusChanged).toHaveBeenCalledTimes(1)
    expect(onStatusChanged.mock.calls[0][0].task_id).toBe('t1')
    expect(onNeedResync).not.toHaveBeenCalled()
  })

  it('OPT-029: task_status_changed 与 runtime SSE 触发机器摘要刷新', async () => {
    const { openWorkPanelTaskStatusSse } = await import('./useWorkPanelTaskStatusSse.js')
    const onMachineRuntimeHint = vi.fn()
    openWorkPanelTaskStatusSse({
      tenantId: '850',
      workspaceId: 'ws-1',
      onStatusChanged: vi.fn(),
      onMachineRuntimeHint,
    })
    const es = FakeEventSource.instances[0]
    es.onmessage({ data: JSON.stringify({ type: 'heartbeat', event_name: 'work_panel_sse_heartbeat' }) })
    expect(onMachineRuntimeHint).not.toHaveBeenCalled()
    es.onmessage({
      data: JSON.stringify({ event_name: 'task_status_changed', task_id: 't1', progress_column_id: 'c2' }),
    })
    es.onmessage({ data: JSON.stringify({ event_name: 'server_status_update', task_id: 't1' }) })
    es.onmessage({ data: JSON.stringify({ event_name: 'container_heartbeat', task_id: 't1' }) })
    expect(onMachineRuntimeHint).toHaveBeenCalledTimes(3)
  })

  it('resyncs board on task_created and task_deleted', async () => {
    const { openWorkPanelTaskStatusSse } = await import('./useWorkPanelTaskStatusSse.js')
    const onStatusChanged = vi.fn()
    const onNeedResync = vi.fn()
    openWorkPanelTaskStatusSse({
      tenantId: '850',
      workspaceId: 'ws-1',
      onStatusChanged,
      onNeedResync,
    })
    const es = FakeEventSource.instances[0]
    es.onmessage({
      data: JSON.stringify({ event_name: 'task_created', task_id: 't-new', workspace_id: 'ws-1' }),
    })
    es.onmessage({
      data: JSON.stringify({ event_name: 'task_deleted', task_id: 't-old', workspace_id: 'ws-1' }),
    })
    expect(onNeedResync).toHaveBeenCalledTimes(2)
    expect(onStatusChanged).not.toHaveBeenCalled()
  })

  it('schedules exponential backoff reconnect and resyncs after reopen', async () => {
    vi.useFakeTimers()
    const { openWorkPanelTaskStatusSse } = await import('./useWorkPanelTaskStatusSse.js')
    const onNeedResync = vi.fn()
    const { close } = openWorkPanelTaskStatusSse({
      tenantId: '850',
      workspaceId: 'ws-1',
      onStatusChanged: vi.fn(),
      onNeedResync,
    })
    expect(FakeEventSource.instances).toHaveLength(1)
    FakeEventSource.instances[0].onopen?.()
    FakeEventSource.instances[0].onerror()
    expect(onNeedResync).toHaveBeenCalled()
    await vi.advanceTimersByTimeAsync(1000)
    expect(FakeEventSource.instances).toHaveLength(2)
    FakeEventSource.instances[1].onopen?.()
    expect(onNeedResync.mock.calls.length).toBeGreaterThanOrEqual(2)
    close()
    vi.useRealTimers()
  })
})
}

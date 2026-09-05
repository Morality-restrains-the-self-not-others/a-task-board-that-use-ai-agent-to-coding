// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] useWorkspaceQueueSchedule.test.js requires vitest runtime')
} else {
const { describe, expect, it, vi, beforeEach } = await import('vitest')

const hoisted = vi.hoisted(() => ({
  apiFetch: vi.fn(),
}))

const { useWorkspaceQueueSchedule } = await import('./useWorkspaceQueueSchedule.js')

function jsonResponse(body, { ok = true, status = 200, headers = {} } = {}) {
  return {
    ok,
    status,
    headers: {
      get: (k) => headers[k] || headers[String(k).toLowerCase()] || null,
    },
    json: async () => body,
  }
}

describe('useWorkspaceQueueSchedule', () => {
  beforeEach(() => {
    hoisted.apiFetch.mockReset()
    window.apiFetch = hoisted.apiFetch
  })

  const SNAP = {
    workspace_id: 'ws1',
    tenant_id: 't1',
    schedule_rhythm: {
      enabled: true,
      timezone: 'Asia/Shanghai',
      windows: [{ id: 'w0', daily_start: '09:00', daily_end: '11:00', max_queued_machines: 2 }],
      max_queued_machines: 2,
      auto_close: false,
      in_window: true,
    },
    in_window: true,
    window_message: '当前处于允许运行时段',
    queued_slots_used: 1,
    max_queued_machines: 2,
    members: [{ task_id: 't1', title: 'top', depth: 0, status: 'queued' }],
    recent_history: [
      { id: 'qsh_1', event_type: 'member_enqueued', message: '加入自动调度队列：top', created_at: '2026-08-28T00:00:00Z', task_id: 't1' },
    ],
  }

  it('GET 加载快照并暴露成员与窗口状态', async () => {
    hoisted.apiFetch.mockResolvedValue(jsonResponse(SNAP))
    const c = useWorkspaceQueueSchedule({ tenantId: 't1', workspaceId: 'ws1' })
    await c.loadSnapshot()
    expect(hoisted.apiFetch).toHaveBeenCalledWith(
      '/api/tenant/t1/workspace/ws1/queue-schedule/',
      expect.objectContaining({ method: 'GET', credentials: 'include' }),
    )
    expect(c.snapshot.value.members).toHaveLength(1)
    expect(c.snapshot.value.schedule_rhythm.enabled).toBe(true)
    expect(c.error.value).toBe('')
    expect(c.historyItems.value).toHaveLength(1)
    expect(c.historyItems.value[0].id).toBe('qsh_1')
  })

  it('快照 7 条且 has_more=false → historyHasMore=false（无按钮）', async () => {
    const snap7 = {
      ...SNAP,
      recent_history: Array.from({ length: 7 }, (_, i) => ({
        id: 'qsh_' + i,
        event_type: 'member_enqueued',
        message: 'row' + i,
        created_at: '2026-08-28T00:00:0' + i + 'Z',
        task_id: 't' + i,
      })),
      recent_history_has_more: false,
    }
    hoisted.apiFetch.mockResolvedValue(jsonResponse(snap7))
    const c = useWorkspaceQueueSchedule({ tenantId: 't1', workspaceId: 'ws1' })
    await c.loadSnapshot()
    expect(c.historyItems.value).toHaveLength(7)
    expect(c.historyHasMore.value).toBe(false)
  })

  it('快照 8 条且 has_more=true → historyHasMore=true（显示按钮）', async () => {
    const snap8 = {
      ...SNAP,
      recent_history: Array.from({ length: 8 }, (_, i) => ({
        id: 'qsh_' + i,
        event_type: 'member_enqueued',
        message: 'row' + i,
        created_at: '2026-08-28T00:00:0' + i + 'Z',
        task_id: 't' + i,
      })),
      recent_history_has_more: true,
    }
    hoisted.apiFetch.mockResolvedValue(jsonResponse(snap8))
    const c = useWorkspaceQueueSchedule({ tenantId: 't1', workspaceId: 'ws1' })
    await c.loadSnapshot()
    expect(c.historyItems.value).toHaveLength(8)
    expect(c.historyHasMore.value).toBe(true)
  })

  it('PUT 保存节奏 body 为 JSON 且成功返回快照', async () => {
    hoisted.apiFetch.mockResolvedValue(jsonResponse(SNAP))
    const c = useWorkspaceQueueSchedule({ tenantId: 't1', workspaceId: 'ws1' })
    const ok = await c.saveSchedule({
      enabled: true,
      timezone: 'UTC',
      windows: [{ id: '', daily_start: '09:00', daily_end: '11:00', max_queued_machines: 1 }],
    })
    expect(ok).toBe(true)
    const [url, opts] = hoisted.apiFetch.mock.calls[0]
    expect(url).toBe('/api/tenant/t1/workspace/ws1/queue-schedule/')
    expect(opts.method).toBe('PUT')
    expect(JSON.parse(opts.body).timezone).toBe('UTC')
    expect(c.saving.value).toBe(false)
  })

  it('HTTP 错误 → error + traceId 填充且 saveSchedule 返回 false', async () => {
    hoisted.apiFetch.mockResolvedValue(jsonResponse(
      { status: 'error', error: 'windows[0].daily_start 格式须为 HH:MM', message: 'windows[0].daily_start 格式须为 HH:MM', trace_id: 'tr-abc' },
      { ok: false, status: 400, headers: { 'X-Trace-Id': 'tr-abc' } },
    ))
    const c = useWorkspaceQueueSchedule({ tenantId: 't1', workspaceId: 'ws1' })
    const ok = await c.saveSchedule({ enabled: true, timezone: 'UTC', windows: [] })
    expect(ok).toBe(false)
    expect(c.error.value).toContain('HH:MM')
    expect(c.errorTraceId.value).toBe('tr-abc')
  })

  it('网络异常 → error 兜底文案', async () => {
    hoisted.apiFetch.mockRejectedValue(new Error('network down'))
    const c = useWorkspaceQueueSchedule({ tenantId: 't1', workspaceId: 'ws1' })
    await c.loadSnapshot()
    expect(c.error.value).toBe('network down')
    expect(c.snapshot.value).toBeNull()
  })

  it('patchQueuedAutoRun：PATCH todos queued_auto_run 后刷新 GET 快照', async () => {
    hoisted.apiFetch.mockResolvedValue(jsonResponse(SNAP))
    const c = useWorkspaceQueueSchedule({ tenantId: 't1', workspaceId: 'ws1' })
    const ok = await c.patchQueuedAutoRun('task_9', true, 'ik-join-1')
    expect(ok).toBe(true)
    const patchCall = hoisted.apiFetch.mock.calls.find(([, opts]) => opts?.method === 'PATCH')
    expect(patchCall[0]).toBe('/api/tenant/t1/workspace/ws1/todos/task_9/')
    expect(JSON.parse(patchCall[1].body).queued_auto_run).toBe(true)
    expect(patchCall[1].headers['Idempotency-Key']).toBe('ik-join-1')
    expect(patchCall[1].headers.Accept).toBe('application/json')
    const getAfter = hoisted.apiFetch.mock.calls.filter(([, opts]) => (opts?.method || 'GET') === 'GET')
    expect(getAfter.length).toBeGreaterThanOrEqual(1)
    expect(getAfter[getAfter.length - 1][0]).toBe('/api/tenant/t1/workspace/ws1/queue-schedule/')
  })

  it('patchQueuedAutoRun 失败 → false 且填充 errorTraceId', async () => {
    hoisted.apiFetch.mockResolvedValue(jsonResponse(
      { error: 'forbidden', trace_id: 'tr-patch' },
      { ok: false, status: 403, headers: { 'X-Trace-Id': 'tr-patch' } },
    ))
    const c = useWorkspaceQueueSchedule({ tenantId: 't1', workspaceId: 'ws1' })
    const ok = await c.patchQueuedAutoRun('task_9', false, 'ik-leave-1')
    expect(ok).toBe(false)
    expect(c.errorTraceId.value).toBe('tr-patch')
    expect(c.error.value).toContain('forbidden')
  })

  it('tenantId/workspaceId 传函数时按调用时求值（页面切换工作空间复用实例）', async () => {
    hoisted.apiFetch.mockResolvedValue(jsonResponse(SNAP))
    const wid = { current: 'ws1' }
    const c = useWorkspaceQueueSchedule({ tenantId: () => 't1', workspaceId: () => wid.current })
    await c.loadSnapshot()
    expect(hoisted.apiFetch).toHaveBeenCalledWith(
      '/api/tenant/t1/workspace/ws1/queue-schedule/',
      expect.anything(),
    )
    // 切换工作空间后同一实例指向新路径
    hoisted.apiFetch.mockClear()
    wid.current = 'ws2'
    await c.loadSnapshot()
    expect(hoisted.apiFetch).toHaveBeenCalledWith(
      '/api/tenant/t1/workspace/ws2/queue-schedule/',
      expect.anything(),
    )
  })

  it('loadMoreHistory 带 cursor 请求 history 并追加', async () => {
    hoisted.apiFetch.mockResolvedValue(jsonResponse(SNAP))
    const c = useWorkspaceQueueSchedule({ tenantId: 't1', workspaceId: 'ws1' })
    await c.loadSnapshot()
    hoisted.apiFetch.mockClear()
    hoisted.apiFetch.mockResolvedValue(jsonResponse({
      items: [{ id: 'qsh_2', event_type: 'rhythm_saved', message: '已保存调度设置', created_at: '2026-08-27T00:00:00Z' }],
      next_cursor: 'c2',
      has_more: false,
    }))
    const ok = await c.loadMoreHistory()
    expect(ok).toBe(true)
    const [url, opts] = hoisted.apiFetch.mock.calls[0]
    expect(url).toContain('/api/tenant/t1/workspace/ws1/queue-schedule/history/')
    expect(url).toContain('cursor=')
    expect(opts.method).toBe('GET')
    expect(c.historyItems.value.map((r) => r.id)).toEqual(['qsh_1', 'qsh_2'])
    expect(c.historyHasMore.value).toBe(false)
  })

  it('loadMoreHistory 失败填充 historyErrorTraceId', async () => {
    hoisted.apiFetch.mockResolvedValue(jsonResponse(SNAP))
    const c = useWorkspaceQueueSchedule({ tenantId: 't1', workspaceId: 'ws1' })
    await c.loadSnapshot()
    hoisted.apiFetch.mockResolvedValue(jsonResponse(
      { error: 'nope', trace_id: 'tr-hist' },
      { ok: false, status: 500, headers: { 'X-Trace-Id': 'tr-hist' } },
    ))
    const ok = await c.loadMoreHistory()
    expect(ok).toBe(false)
    expect(c.historyErrorTraceId.value).toBe('tr-hist')
  })
})
}

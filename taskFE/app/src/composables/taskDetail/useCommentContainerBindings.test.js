if (!process.env.VITEST) {
  console.log('[skip] useCommentContainerBindings.test.js requires vitest runtime')
} else {
const { describe, expect, it, vi, beforeEach } = await import('vitest')
const { ref, nextTick } = await import('vue')

vi.mock('../../utils/commentExecutionApi.js', () => ({
  normalizeExecutionMode: (raw) => (raw === 'independent' ? 'independent' : 'wait_previous'),
  patchHumanCommentExecutionMode: vi.fn(async () => ({ execution_mode: 'independent' })),
  patchAICommentExecutionMode: vi.fn(async () => ({ execution_mode: 'independent' })),
  fetchCommentContainerBindings: vi.fn(async () => ([
    { comment_id: 'C1', status: 'running', container_name: 'task_task1_C1' },
  ])),
  ensureCommentContainerBinding: vi.fn(async () => ({ status: 'success' })),
  advanceCommentContainerBindings: vi.fn(async () => ({ status: 'success' })),
}))

const {
  patchHumanCommentExecutionMode,
  fetchCommentContainerBindings,
  ensureCommentContainerBinding,
  advanceCommentContainerBindings,
} = await import('../../utils/commentExecutionApi.js')
const { useCommentContainerBindings } = await import('./useCommentContainerBindings.js')

describe('useCommentContainerBindings', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('syncs ensure+advance when comments present', async () => {
    const comments = ref([
      { id: 'C1', commentKind: 'user', execution_mode: 'wait_previous' },
      { id: 'C2', commentKind: 'ai', execution_mode: 'independent' },
    ])
    const api = useCommentContainerBindings({
      tenantId: ref('t1'),
      workspaceId: ref('w1'),
      taskId: ref('task1'),
      displayComments: comments,
    })
    await vi.waitFor(() => {
      expect(ensureCommentContainerBinding).toHaveBeenCalled()
      expect(advanceCommentContainerBindings).toHaveBeenCalled()
      expect(fetchCommentContainerBindings).toHaveBeenCalled()
    })
    expect(api.bindingStatusFor('C1')).toBe('running')
    expect(api.commentHasLiveBinding('C1')).toBe(true)
    expect(api.bindingContainerNameFor('C1')).toBe('task_task1_C1')
    expect(api.bindingContainerNameFor('C2')).toBe('task_task1_C2')
  })

  it('does not treat waiting_previous as starting lifecycle', async () => {
    vi.mocked(fetchCommentContainerBindings).mockResolvedValue([
      { comment_id: 'C1', status: 'waiting_previous', container_name: 'task_task1_C1' },
      { comment_id: 'C2', status: 'starting', csc_id: 'csc_2', container_name: 'task_task1_C2' },
    ])
    const comments = ref([
      { id: 'C1', commentKind: 'user', execution_mode: 'wait_previous' },
      { id: 'C2', commentKind: 'user', execution_mode: 'independent' },
    ])
    const api = useCommentContainerBindings({
      tenantId: 't1',
      workspaceId: 'w1',
      taskId: 'task1',
      displayComments: comments,
    })
    await vi.waitFor(() => expect(fetchCommentContainerBindings).toHaveBeenCalled())
    expect(api.bindingStatusFor('C1')).toBe('waiting_previous')
    expect(api.bindingIsStarting('C1')).toBe(false)
    expect(api.commentHasLiveBinding('C1')).toBe(false)
    expect(api.bindingServerLifecycle('C1')).toBe('等待前序')
    expect(api.bindingIsStarting('C2')).toBe(true)
  })

  it('patches human execution mode and updates local comment', async () => {
    const comments = ref([
      { id: 'C1', commentKind: 'user', execution_mode: 'wait_previous' },
    ])
    const api = useCommentContainerBindings({
      tenantId: 't1',
      workspaceId: 'w1',
      taskId: 'task1',
      displayComments: comments,
    })
    await nextTick()
    await api.changeDependencyMode({
      commentId: 'C1',
      executionMode: 'independent',
      commentKind: 'user',
    })
    expect(patchHumanCommentExecutionMode).toHaveBeenCalledWith(
      expect.objectContaining({ commentId: 'C1', executionMode: 'independent' }),
    )
    expect(comments.value[0].execution_mode).toBe('independent')
  })

  it('seeds csc allocation stage log when starting with csc_id (refresh/cold-open)', async () => {
    vi.mocked(fetchCommentContainerBindings).mockResolvedValue([
      { comment_id: 'C1', status: 'starting', csc_id: 'csc_1', container_name: 'task_task1_C1' },
    ])
    const comments = ref([{ id: 'C1', commentKind: 'user', execution_mode: 'independent' }])
    const api = useCommentContainerBindings({
      tenantId: 't1',
      workspaceId: 'w1',
      taskId: 'task1',
      displayComments: comments,
    })
    await vi.waitFor(() => expect(fetchCommentContainerBindings).toHaveBeenCalled())
    const logs = api.buildPerBindingServerStatusProps('C1').statusLogs
    // 状态日志 + 阶段日志均出现，且阶段日志位于状态日志之后（时间线顺序）
    expect(logs.some((l) => l.includes('正在启动容器实例'))).toBe(true)
    expect(logs.some((l) => l.includes('容器实例已分配，等待服务就绪'))).toBe(true)
    const startIdx = logs.findIndex((l) => l.includes('正在启动容器实例'))
    const allocIdx = logs.findIndex((l) => l.includes('容器实例已分配'))
    expect(allocIdx).toBeGreaterThan(startIdx)
    // 幂等：重复 refresh 不重复追加
    const before = logs.length
    await api.refreshBindings()
    await nextTick()
    expect(api.buildPerBindingServerStatusProps('C1').statusLogs.length).toBe(before)
  })

  it('appends heartbeat stage logs from SSE once and dedups repeated heartbeats', async () => {
    const { latestPerContainerHeartbeat } = await import('./perContainerHeartbeatBus.js')
    const comments = ref([{ id: 'C1', commentKind: 'user', execution_mode: 'independent' }])
    const api = useCommentContainerBindings({
      tenantId: 't1',
      workspaceId: 'w1',
      taskId: 'task1',
      displayComments: comments,
    })
    // 默认 mock 返回 C1 running → live binding，可接收心跳阶段日志
    await vi.waitFor(() => expect(fetchCommentContainerBindings).toHaveBeenCalled())

    latestPerContainerHeartbeat.value = {
      comment_id: 'C1',
      status: 'ok',
      container_seq: 3,
      container_ack: 2,
      saas_seq: 1,
      saas_ack: 1,
      uplink_ok: true,
      downlink_ok: true,
      probe_ok: true,
      bidirectional_ok: true,
    }
    await nextTick()
    let logs = api.buildPerBindingServerStatusProps('C1').statusLogs
    expect(logs.some((l) => l.includes('容器心跳已建立，服务启动中…'))).toBe(true)
    expect(logs.some((l) => l.includes('服务健康探测通过'))).toBe(true)
    expect(logs.some((l) => l.includes('双向通信通道已建立'))).toBe(true)
    const afterFirst = logs.length

    // 重复心跳（同阶段信号）不重复追加
    latestPerContainerHeartbeat.value = {
      comment_id: 'C1',
      status: 'ok',
      container_seq: 4,
      container_ack: 3,
      saas_seq: 2,
      saas_ack: 2,
      probe_ok: true,
      bidirectional_ok: true,
    }
    await nextTick()
    logs = api.buildPerBindingServerStatusProps('C1').statusLogs
    expect(logs.length).toBe(afterFirst)
    // 心跳信号 → 连接状态
    expect(api.bindingHeartbeatStateFor('C1').status).toBe('connected')
  })

  it('appends csc allocation stage log via binding_advanced SSE event', async () => {
    const { latestBindingAdvanced } = await import('./perContainerHeartbeatBus.js')
    const comments = ref([{ id: 'C1', commentKind: 'user', execution_mode: 'independent' }])
    const api = useCommentContainerBindings({
      tenantId: 't1',
      workspaceId: 'w1',
      taskId: 'task1',
      displayComments: comments,
    })
    await vi.waitFor(() => expect(fetchCommentContainerBindings).toHaveBeenCalled())

    latestBindingAdvanced.value = {
      comment_id: 'C1',
      status: 'starting',
      csc_id: 'csc_2',
      container_name: 'task_task1_C1',
    }
    await nextTick()
    const logs = api.buildPerBindingServerStatusProps('C1').statusLogs
    expect(logs.some((l) => l.includes('正在启动容器实例'))).toBe(true)
    expect(logs.some((l) => l.includes('容器实例已分配，等待服务就绪'))).toBe(true)
    // SSE 路径同步更新 binding 状态（starting + csc）
    expect(api.bindingStatusFor('C1')).toBe('starting')
    expect(api.bindingCscIdFor('C1')).toBe('csc_2')
  })

  it('merges backend authoritative startup logs on refresh (OPT-20260809-011)', async () => {
    vi.mocked(fetchCommentContainerBindings).mockResolvedValue([
      {
        comment_id: 'C1',
        status: 'running',
        csc_id: 'csc_1',
        container_name: 'task_task1_C1',
        logs: [
          { stage: 'pending', message: '容器调度排队中', created_at: '2026-08-10T00:00:01Z' },
          { stage: 'starting', message: '正在启动容器实例', created_at: '2026-08-10T00:00:02Z' },
          { stage: 'csc_allocated', message: '容器实例已分配，等待服务就绪', created_at: '2026-08-10T00:00:03Z' },
          { stage: 'running', message: '容器已就绪，服务可用', created_at: '2026-08-10T00:00:04Z' },
        ],
      },
    ])
    const comments = ref([{ id: 'C1', commentKind: 'user', execution_mode: 'independent' }])
    const api = useCommentContainerBindings({
      tenantId: 't1',
      workspaceId: 'w1',
      taskId: 'task1',
      displayComments: comments,
    })
    await vi.waitFor(() => expect(fetchCommentContainerBindings).toHaveBeenCalled())
    const logs = api.buildPerBindingServerStatusProps('C1').statusLogs
    // 冷打开完整历史：后端权威时间线全部可见
    expect(logs.some((l) => l.includes('容器调度排队中'))).toBe(true)
    expect(logs.some((l) => l.includes('正在启动容器实例'))).toBe(true)
    expect(logs.some((l) => l.includes('容器实例已分配，等待服务就绪'))).toBe(true)
    expect(logs.some((l) => l.includes('容器已就绪，服务可用'))).toBe(true)
    // 幂等：重复 refresh 不重复追加后端行
    const before = logs.length
    await api.refreshBindings()
    await nextTick()
    expect(api.buildPerBindingServerStatusProps('C1').statusLogs.length).toBe(before)
  })

  it('appends server scheduling progress from SSE bus into binding startup logs', async () => {
    vi.mocked(fetchCommentContainerBindings).mockResolvedValue([
      { comment_id: 'C1', status: 'starting', csc_id: 'csc_1', container_name: 'task_task1_C1' },
    ])
    const { latestServerStartupStatusForBinding } = await import('./perContainerHeartbeatBus.js')
    const comments = ref([{ id: 'C1', commentKind: 'user', execution_mode: 'independent' }])
    const api = useCommentContainerBindings({
      tenantId: 't1',
      workspaceId: 'w1',
      taskId: 'task1',
      displayComments: comments,
    })
    await vi.waitFor(() => expect(fetchCommentContainerBindings).toHaveBeenCalled())

    latestServerStartupStatusForBinding.value = {
      comment_id: 'C1',
      messages: [
        '[-、task_task1_C1] 检测到自动创建资源，正在准备前置资源创建任务...',
        '[-、task_task1_C1] 正在自动创建前置资源并启动服务器...',
      ],
    }
    await nextTick()
    const logs = api.buildPerBindingServerStatusProps('C1').statusLogs
    expect(logs.some((l) => l.includes('准备前置资源创建任务'))).toBe(true)
    expect(logs.some((l) => l.includes('正在自动创建前置资源并启动服务器'))).toBe(true)
    // 容器阶段日志仍保留
    expect(logs.some((l) => l.includes('正在启动容器实例'))).toBe(true)
    // 幂等：重复写入同消息不追加
    const before = logs.length
    latestServerStartupStatusForBinding.value = {
      comment_id: 'C1',
      messages: ['[-、task_task1_C1] 检测到自动创建资源，正在准备前置资源创建任务...'],
    }
    await nextTick()
    expect(api.buildPerBindingServerStatusProps('C1').statusLogs.length).toBe(before)
  })

  it('caches per-binding start TraceId from scheduling bus (OPT-20260812-013 parallel race)', async () => {
    vi.mocked(fetchCommentContainerBindings).mockResolvedValue([
      { comment_id: 'C1', status: 'starting', csc_id: 'csc_1', container_name: 'task_task1_C1' },
      { comment_id: 'C2', status: 'starting', csc_id: 'csc_2', container_name: 'task_task1_C2' },
    ])
    const { latestServerStartupStatusForBinding } = await import('./perContainerHeartbeatBus.js')
    const comments = ref([
      { id: 'C1', commentKind: 'user', execution_mode: 'independent' },
      { id: 'C2', commentKind: 'user', execution_mode: 'independent' },
    ])
    const api = useCommentContainerBindings({
      tenantId: 't1',
      workspaceId: 'w1',
      taskId: 'task1',
      displayComments: comments,
    })
    await vi.waitFor(() => expect(fetchCommentContainerBindings).toHaveBeenCalled())

    // 并行独立 CSC：C1 先启动写入 trace-c1，C2 随后启动写入 trace-c2
    latestServerStartupStatusForBinding.value = {
      comment_id: 'C1',
      messages: ['[-、task_task1_C1] 正在自动创建前置资源...'],
      trace_id: 'trace-c1',
    }
    await nextTick()
    latestServerStartupStatusForBinding.value = {
      comment_id: 'C2',
      messages: ['[-、task_task1_C2] 正在自动创建前置资源...'],
      trace_id: 'trace-c2',
    }
    await nextTick()

    // 各自 TraceId 独立缓存，后一次启动不覆盖前一次
    expect(api.bindingStartTraceIdFor('C1')).toBe('trace-c1')
    expect(api.bindingStartTraceIdFor('C2')).toBe('trace-c2')
    expect(api.buildPerBindingServerStatusProps('C1').startTraceId).toBe('trace-c1')
    expect(api.buildPerBindingServerStatusProps('C2').startTraceId).toBe('trace-c2')
    // 无记录评论回退 ''
    expect(api.bindingStartTraceIdFor('C3')).toBe('')

    // 任务切换清理
    comments.value = []
    await nextTick()
    latestServerStartupStatusForBinding.value = null
    await api.refreshBindings()
  })

  it('recordBindingStartTraceId ignores empty commentId or traceId', async () => {
    const comments = ref([{ id: 'C1', commentKind: 'user', execution_mode: 'independent' }])
    const api = useCommentContainerBindings({
      tenantId: 't1',
      workspaceId: 'w1',
      taskId: 'task1',
      displayComments: comments,
    })
    api.recordBindingStartTraceId('', 'trace-x')
    api.recordBindingStartTraceId('C1', '')
    expect(api.bindingStartTraceIdFor('C1')).toBe('')
    api.recordBindingStartTraceId('C1', 'task1')
    expect(api.bindingStartTraceIdFor('C1')).toBe('')
    api.recordBindingStartTraceId('C1', 'trace-c1')
    expect(api.bindingStartTraceIdFor('C1')).toBe('trace-c1')
  })

  it('buildPerBindingServerStatusProps does not fall back to task-level statusTraceId', async () => {
    vi.mocked(fetchCommentContainerBindings).mockResolvedValue([
      { comment_id: 'C1', status: 'failed', csc_id: 'csc_1' },
    ])
    const comments = ref([{ id: 'C1', commentKind: 'user', execution_mode: 'independent' }])
    const statusTraceId = ref('task_15652393783064603866')
    const api = useCommentContainerBindings({
      tenantId: 't1',
      workspaceId: 'w1',
      taskId: 'task_15652393783064603866',
      displayComments: comments,
      statusTraceId,
    })
    await vi.waitFor(() => expect(fetchCommentContainerBindings).toHaveBeenCalled())
    expect(api.bindingStartTraceIdFor('C1')).toBe('')
    expect(api.buildPerBindingServerStatusProps('C1').startTraceId).toBe('')
  })

  it('hydrates startTraceId from persisted binding logs on refresh', async () => {
    vi.mocked(fetchCommentContainerBindings).mockResolvedValue([
      {
        comment_id: 'C1',
        status: 'failed',
        csc_id: 'csc_1',
        logs: [
          { stage: 'pending', message: '容器调度排队中' },
          {
            stage: 'server_failed',
            message: '未找到匹配地域的运行环境 trace_id=task_cold_open_1',
          },
        ],
      },
    ])
    const comments = ref([{ id: 'C1', commentKind: 'user', execution_mode: 'independent' }])
    const api = useCommentContainerBindings({
      tenantId: 't1',
      workspaceId: 'w1',
      taskId: 'task1',
      displayComments: comments,
    })
    await vi.waitFor(() => expect(api.bindingStartTraceIdFor('C1')).toBe('task_cold_open_1'))
    expect(api.buildPerBindingServerStatusProps('C1').startTraceId).toBe('task_cold_open_1')
  })

  it('hydrates startTraceId from start_trace_id column ahead of log suffix', async () => {
    vi.mocked(fetchCommentContainerBindings).mockResolvedValue([
      {
        comment_id: 'C1',
        status: 'failed',
        csc_id: 'csc_1',
        start_trace_id: 'cmt-start-aaa',
        logs: [
          {
            stage: 'server_failed',
            message: '未找到匹配地域的运行环境 trace_id=task_cold_open_1',
          },
        ],
      },
    ])
    const comments = ref([{ id: 'C1', commentKind: 'user', execution_mode: 'independent' }])
    const api = useCommentContainerBindings({
      tenantId: 't1',
      workspaceId: 'w1',
      taskId: 'task1',
      displayComments: comments,
    })
    await vi.waitFor(() => expect(api.bindingStartTraceIdFor('C1')).toBe('cmt-start-aaa'))
    expect(api.buildPerBindingServerStatusProps('C1').startTraceId).toBe('cmt-start-aaa')
  })

  it('ignores persisted start_trace_id equal to the current taskId', async () => {
    vi.mocked(fetchCommentContainerBindings).mockResolvedValue([
      { comment_id: 'C1', status: 'failed', csc_id: 'csc_1', start_trace_id: 'task1' },
    ])
    const comments = ref([{ id: 'C1', commentKind: 'user', execution_mode: 'independent' }])
    const api = useCommentContainerBindings({
      tenantId: 't1',
      workspaceId: 'w1',
      taskId: 'task1',
      displayComments: comments,
    })
    await vi.waitFor(() => expect(fetchCommentContainerBindings).toHaveBeenCalled())
    expect(api.bindingStartTraceIdFor('C1')).toBe('')
  })

  it('merges task-level statusLogs matching container into per-binding panel', async () => {
    vi.mocked(fetchCommentContainerBindings).mockResolvedValue([
      { comment_id: 'C1', status: 'starting', csc_id: 'csc_1', container_name: 'task_task1_C1' },
    ])
    const comments = ref([{ id: 'C1', commentKind: 'user', execution_mode: 'independent' }])
    const taskStatusLogs = ref([
      '[13:29:50] [-、task_task1_C1] 检测到自动创建资源，正在准备前置资源创建任务...',
      '[13:29:51] [-、task_task1_C2] 其他评论服务器调度',
    ])
    const api = useCommentContainerBindings({
      tenantId: 't1',
      workspaceId: 'w1',
      taskId: 'task1',
      displayComments: comments,
      taskStatusLogs,
    })
    await vi.waitFor(() => expect(fetchCommentContainerBindings).toHaveBeenCalled())
    const logs = api.buildPerBindingServerStatusProps('C1').statusLogs
    expect(logs.some((l) => l.includes('准备前置资源创建任务'))).toBe(true)
    expect(logs.some((l) => l.includes('其他评论'))).toBe(false)
  })

  it('keeps local SSE-derived lines coexisting with backend logs (OPT-20260809-011)', async () => {
    vi.mocked(fetchCommentContainerBindings).mockResolvedValue([
      {
        comment_id: 'C1',
        status: 'running',
        csc_id: 'csc_1',
        container_name: 'task_task1_C1',
        logs: [
          { stage: 'pending', message: '容器调度排队中', created_at: '2026-08-10T00:00:01Z' },
          { stage: 'starting', message: '正在启动容器实例', created_at: '2026-08-10T00:00:02Z' },
          { stage: 'csc_allocated', message: '容器实例已分配，等待服务就绪', created_at: '2026-08-10T00:00:03Z' },
          { stage: 'running', message: '容器已就绪，服务可用', created_at: '2026-08-10T00:00:04Z' },
        ],
      },
    ])
    const comments = ref([{ id: 'C1', commentKind: 'user', execution_mode: 'independent' }])
    const api = useCommentContainerBindings({
      tenantId: 't1',
      workspaceId: 'w1',
      taskId: 'task1',
      displayComments: comments,
    })
    await vi.waitFor(() => expect(fetchCommentContainerBindings).toHaveBeenCalled())
    const { latestPerContainerHeartbeat } = await import('./perContainerHeartbeatBus.js')
    latestPerContainerHeartbeat.value = {
      comment_id: 'C1',
      status: 'ok',
      container_seq: 3,
      container_ack: 2,
      probe_ok: true,
      bidirectional_ok: true,
    }
    await nextTick()
    const logs = api.buildPerBindingServerStatusProps('C1').statusLogs
    // 后端权威行仍保留，本地 SSE 派生行共存
    expect(logs.some((l) => l.includes('容器调度排队中'))).toBe(true)
    expect(logs.some((l) => l.includes('服务健康探测通过'))).toBe(true)
  })

  it('does not duplicate UserData step when backend adds trace_id and UTC clock', async () => {
    vi.mocked(fetchCommentContainerBindings).mockResolvedValue([
      {
        comment_id: 'C1',
        status: 'starting',
        csc_id: 'csc_1',
        container_name: 'task_task1_C1',
        logs: [
          {
            stage: 'server_scheduling',
            message: '[i-abc、task_task1_C1] 容器运行时服务已就绪 trace_id=bb1157e1b936e3d8eb7e7b05',
            created_at: '2026-08-13T15:30:59Z',
          },
        ],
      },
    ])
    const comments = ref([{ id: 'C1', commentKind: 'user', execution_mode: 'independent' }])
    const taskStatusLogs = ref([
      '[23:30:59] [i-abc、task_task1_C1] 容器运行时服务已就绪',
    ])
    const api = useCommentContainerBindings({
      tenantId: 't1',
      workspaceId: 'w1',
      taskId: 'task1',
      displayComments: comments,
      taskStatusLogs,
    })
    await vi.waitFor(() => expect(fetchCommentContainerBindings).toHaveBeenCalled())
    const logs = api.buildPerBindingServerStatusProps('C1').statusLogs
    expect(logs.filter((l) => l.includes('容器运行时服务已就绪'))).toHaveLength(1)
    expect(logs.some((l) => l.includes('trace_id=bb1157e1b936e3d8eb7e7b05'))).toBe(true)
  })

  it('refreshes bindings when bindingRefreshRequested bus fires (OPT-20260822-062)', async () => {
    const { bindingRefreshRequested } = await import('./perContainerHeartbeatBus.js')
    const comments = ref([{ id: 'C1', commentKind: 'user', execution_mode: 'independent' }])
    const api = useCommentContainerBindings({
      tenantId: 't1',
      workspaceId: 'w1',
      taskId: 'task1',
      displayComments: comments,
    })
    await vi.waitFor(() => expect(fetchCommentContainerBindings).toHaveBeenCalled())
    const initialCalls = vi.mocked(fetchCommentContainerBindings).mock.calls.length

    bindingRefreshRequested.value = { commentId: 'C1', ts: 1 }
    await vi.waitFor(() => {
      expect(vi.mocked(fetchCommentContainerBindings).mock.calls.length).toBeGreaterThan(initialCalls)
    })
    // 服务端 binding 已进入终态时，refresh 后列表对齐为 released
    vi.mocked(fetchCommentContainerBindings).mockResolvedValue([
      { comment_id: 'C1', status: 'released', container_name: 'task_task1_C1' },
    ])
    bindingRefreshRequested.value = { commentId: 'C1', ts: 2 }
    await vi.waitFor(() => expect(api.bindingStatusFor('C1')).toBe('released'))
  })

  it('released binding still exposes persisted startup logs after refresh', async () => {
    vi.mocked(fetchCommentContainerBindings).mockResolvedValue([
      {
        comment_id: 'C1',
        status: 'released',
        csc_id: 'csc_1',
        logs: [
          { stage: 'pending', message: '容器调度排队中', created_at: '2026-08-22T22:00:14Z' },
          { stage: 'server_started', message: 'aliyun服务器启动成功！', created_at: '2026-08-22T22:00:17Z' },
          {
            stage: 'server_scheduling',
            message: '正在调用aliyunAPI停止服务器...（触发：容器指令空闲超时回收）',
            created_at: '2026-08-22T22:05:02Z',
          },
        ],
      },
    ])
    const comments = ref([{ id: 'C1', commentKind: 'user', execution_mode: 'independent' }])
    const api = useCommentContainerBindings({
      tenantId: 't1',
      workspaceId: 'w1',
      taskId: 'task1',
      displayComments: comments,
    })
    await vi.waitFor(() => expect(fetchCommentContainerBindings).toHaveBeenCalled())
    const props = api.buildPerBindingServerStatusProps('C1')
    expect(props.serverStatus).toBe('stopped')
    expect(props.statusLogs.some((l) => l.includes('aliyun服务器启动成功'))).toBe(true)
    expect(props.statusLogs.some((l) => l.includes('触发：容器指令空闲超时回收'))).toBe(true)
  })
})

}

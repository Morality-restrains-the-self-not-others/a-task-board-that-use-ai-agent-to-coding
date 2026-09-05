// @vitest-environment node
/**
 * fetchServerStartHistoryApi：空成功不写 message（由面板空态单条展示，OPT-20260811-054）。
 */
if (!process.env.VITEST) {
  console.log('[skip] useServerConfigRuntimeFetch.test.js requires vitest runtime')
} else {
const { describe, expect, it, vi, beforeEach } = await import('vitest')
const { ref } = await import('vue')

const hoisted = vi.hoisted(() => ({ apiFetch: vi.fn() }))

vi.mock('../../utils/apiUtils.js', () => ({
  apiFetch: (...args) => hoisted.apiFetch(...args),
}))

const { fetchServerContentApi, fetchServerRuntimeStatusApi, fetchServerStartHistoryApi, openWorkbenchLinkApi, stopServerApi } = await import('./useServerConfigRuntimeFetch.js')

function jsonResponse(body, { ok = true, status = 200, headers = {} } = {}) {
  const headerMap = Object.fromEntries(
    Object.entries(headers).map(([k, v]) => [String(k).toLowerCase(), v]),
  )
  return {
    ok,
    status,
    json: async () => body,
    headers: {
      get: (name) => headerMap[String(name).toLowerCase()] || null,
    },
  }
}

function makeDeps() {
  const serverStartHistoryMessage = ref('')
  const serverStartHistoryMessageTraceId = ref('')
  const serverStartHistoryRecords = ref([])
  const isServerStartHistoryLoading = ref(false)
  return {
    props: { task: { id: 'task-1' } },
    route: { params: { tenant: 't1' } },
    resolveRuntimeApiWorkspaceId: () => 'ws-1',
    serverStartHistoryMessage,
    serverStartHistoryMessageTraceId,
    serverStartHistoryRecords,
    isServerStartHistoryLoading,
  }
}

describe('fetchServerStartHistoryApi 空态单条（OPT-20260811-054）', () => {
  beforeEach(() => {
    hoisted.apiFetch.mockReset()
  })

  it('空成功 records=[] 不写 message（面板空态单条展示）', async () => {
    hoisted.apiFetch.mockResolvedValue(jsonResponse({ status: 'success', records: [] }))
    const deps = makeDeps()
    await fetchServerStartHistoryApi(deps)
    expect(deps.serverStartHistoryRecords.value).toEqual([])
    expect(deps.serverStartHistoryMessage.value).toBe('')
  })

  it('有记录时 message 为共 N 条历史记录', async () => {
    hoisted.apiFetch.mockResolvedValue(
      jsonResponse({ status: 'success', records: [{ id: 'r1' }, { id: 'r2' }] }),
    )
    const deps = makeDeps()
    await fetchServerStartHistoryApi(deps)
    expect(deps.serverStartHistoryRecords.value.length).toBe(2)
    expect(deps.serverStartHistoryMessage.value).toBe('共 2 条历史记录')
  })

  it('error 响应写错误 message 并清空 records', async () => {
    hoisted.apiFetch.mockResolvedValue(
      jsonResponse({ status: 'error', message: '服务不可用' }, { ok: false, status: 500 }),
    )
    const deps = makeDeps()
    await fetchServerStartHistoryApi(deps)
    expect(deps.serverStartHistoryRecords.value).toEqual([])
    expect(deps.serverStartHistoryMessage.value).toBe('服务不可用')
  })
})

describe('fetchServerStartHistoryApi 任务ID解析（URL/pk 回退）', () => {
  beforeEach(() => {
    hoisted.apiFetch.mockReset()
  })

  it('task 无 id 但 route.params.taskId 存在时发起请求，不展示缺少任务ID', async () => {
    hoisted.apiFetch.mockResolvedValue(jsonResponse({ status: 'success', records: [] }))
    const deps = makeDeps()
    deps.props.task = {}
    deps.route.params.taskId = 'task_881388002226499584'
    await fetchServerStartHistoryApi(deps)
    expect(hoisted.apiFetch).toHaveBeenCalledTimes(1)
    expect(String(hoisted.apiFetch.mock.calls[0][0])).toContain('task_id=task_881388002226499584')
    expect(deps.serverStartHistoryMessage.value).toBe('')
  })

  it('task 仅有 pk 时用 pk 发起请求', async () => {
    hoisted.apiFetch.mockResolvedValue(jsonResponse({ status: 'success', records: [] }))
    const deps = makeDeps()
    deps.props.task = { pk: 'task_pk_1' }
    await fetchServerStartHistoryApi(deps)
    expect(hoisted.apiFetch).toHaveBeenCalledTimes(1)
    expect(String(hoisted.apiFetch.mock.calls[0][0])).toContain('task_id=task_pk_1')
  })

  it('props.taskId 优先于 route.params.taskId', async () => {
    hoisted.apiFetch.mockResolvedValue(jsonResponse({ status: 'success', records: [] }))
    const deps = makeDeps()
    deps.props.task = null
    deps.props.taskId = 'prop-task'
    deps.route.params.taskId = 'route-task'
    await fetchServerStartHistoryApi(deps)
    expect(String(hoisted.apiFetch.mock.calls[0][0])).toContain('task_id=prop-task')
  })

  it('无任何任务ID时写缺少任务ID且不请求', async () => {
    const deps = makeDeps()
    deps.props.task = {}
    deps.props.taskId = ''
    deps.route.params = { tenant: 't1' }
    await fetchServerStartHistoryApi(deps)
    expect(hoisted.apiFetch).not.toHaveBeenCalled()
    expect(deps.serverStartHistoryMessage.value).toBe('缺少任务ID')
    expect(deps.serverStartHistoryMessageTraceId.value).toBe('')
  })
})

describe('fetchServerContentApi 启动中降级（OPT-20260812-042）', () => {
  beforeEach(() => {
    hoisted.apiFetch.mockReset()
  })

  function makeContentDeps() {
    const serverContentMessage = ref('')
    const serverContentMessageTraceId = ref('')
    const serverContentText = ref('')
    const serverContentTargetUrl = ref('')
    const isServerContentLoading = ref(false)
    return {
      props: { task: { id: 'task-1' } },
      route: { params: { tenant: 't1' } },
      resolveRuntimeApiWorkspaceId: () => 'ws-1',
      commentId: 'cmt-test',
      serverContentMessage,
      serverContentMessageTraceId,
      serverContentText,
      serverContentTargetUrl,
      isServerContentLoading,
    }
  }

  it('status=starting 时降级展示「启动中」而非成功/硬错误', async () => {
    hoisted.apiFetch.mockResolvedValue(
      jsonResponse({
        status: 'starting',
        message: '容器业务端口尚未就绪，正在启动中',
        target_url: 'http://198.51.100.7:8080',
      }),
    )
    const deps = makeContentDeps()
    await fetchServerContentApi(deps)
    expect(deps.serverContentMessage.value).toBe('容器业务端口尚未就绪，正在启动中')
    expect(deps.serverContentText.value).toBe('')
    expect(deps.serverContentTargetUrl.value).toBe('http://198.51.100.7:8080')
    expect(deps.isServerContentLoading.value).toBe(false)
  })

  it('status=error 时展示错误 message（回归）', async () => {
    hoisted.apiFetch.mockResolvedValue(
      jsonResponse({ status: 'error', message: '拉取服务器内容失败' }, { ok: false, status: 502 }),
    )
    const deps = makeContentDeps()
    await fetchServerContentApi(deps)
    expect(deps.serverContentMessage.value).toBe('拉取服务器内容失败')
    expect(deps.serverContentText.value).toBe('')
  })

  it('status=success 时展示内容（回归）', async () => {
    hoisted.apiFetch.mockResolvedValue(
      jsonResponse({ status: 'success', http_status: 200, content: '<h1>ok</h1>', target_url: 'http://x:8080' }),
    )
    const deps = makeContentDeps()
    await fetchServerContentApi(deps)
    expect(deps.serverContentMessage.value).toContain('拉取成功')
    expect(deps.serverContentText.value).toBe('<h1>ok</h1>')
    expect(String(hoisted.apiFetch.mock.calls[0][0])).toContain('comment_id')
  })

  it('无评论 ID 时不发请求并提示缺少评论ID', async () => {
    const deps = makeContentDeps()
    deps.commentId = ''
    await fetchServerContentApi(deps)
    expect(hoisted.apiFetch).not.toHaveBeenCalled()
    expect(deps.serverContentMessage.value).toBe('缺少评论ID')
    expect(deps.serverContentText.value).toBe('')
  })
})

describe('openWorkbenchLinkApi 失败挂 data-traceId', () => {
  beforeEach(() => {
    hoisted.apiFetch.mockReset()
  })

  function makeWorkbenchDeps() {
    return {
      props: { task: { id: 'task-1' } },
      route: { params: { tenant: 't1' } },
      resolveRuntimeApiWorkspaceId: () => 'ws-1',
      serverRuntimeStatusMessage: ref(''),
      serverRuntimeStatusTraceId: ref(''),
      isWorkbenchLinkLoading: ref(false),
      openWindow: vi.fn(),
      commentId: 'cmt-test',
    }
  }

  it('400 缺少实例ID或地域时写入 message 与 traceId', async () => {
    hoisted.apiFetch.mockResolvedValue(
      jsonResponse(
        { status: 'error', message: '服务器配置缺少实例ID或地域', trace_id: 'trace-wb-1' },
        { ok: false, status: 400, headers: { 'X-Trace-Id': 'trace-wb-1' } },
      ),
    )
    const deps = makeWorkbenchDeps()
    await openWorkbenchLinkApi(deps)
    expect(deps.serverRuntimeStatusMessage.value).toBe('服务器配置缺少实例ID或地域')
    expect(deps.serverRuntimeStatusTraceId.value).toBe('trace-wb-1')
    expect(deps.openWindow).not.toHaveBeenCalled()
  })

  it('成功时打开 workbench_url', async () => {
    hoisted.apiFetch.mockResolvedValue(
      jsonResponse({
        status: 'success',
        workbench_url: 'https://ecs-workbench.aliyun.com/?instanceId=i-1&regionId=cn-hangzhou',
      }),
    )
    const deps = makeWorkbenchDeps()
    await openWorkbenchLinkApi(deps)
    expect(deps.openWindow).toHaveBeenCalledWith(
      'https://ecs-workbench.aliyun.com/?instanceId=i-1&regionId=cn-hangzhou',
    )
  })

  it('appends comment_id for comment card Workbench', async () => {
    hoisted.apiFetch.mockResolvedValue(
      jsonResponse({
        status: 'success',
        workbench_url: 'https://ecs-workbench.aliyun.com/?instanceId=i-cmt',
      }),
    )
    const deps = makeWorkbenchDeps()
    await openWorkbenchLinkApi({ ...deps, commentId: 'cmt_99' })
    expect(hoisted.apiFetch.mock.calls[0][0]).toContain('/comment_id/cmt_99/')
  })
})

describe('stopServerApi 评论级 comment_id', () => {
  beforeEach(() => {
    hoisted.apiFetch.mockReset()
  })

  it('POST body includes comment_id from comment card', async () => {
    hoisted.apiFetch.mockResolvedValue({ ok: true, json: async () => ({}) })
    await stopServerApi({
      props: { task: { id: 'task-1' } },
      route: { params: { tenant: 't1' } },
      resolveRuntimeApiWorkspaceId: () => 'ws-1',
      fetchServerRuntimeStatus: vi.fn(),
      fetchServerStartHistory: vi.fn(),
      commentId: 'cmt_stop',
    })
    const body = JSON.parse(hoisted.apiFetch.mock.calls[0][1].body)
    // stopVmBodyWithCommentId（cloudComputeCommentQuery）随 task_id 附带 stop_reason: 'user_stop'
    expect(body).toEqual({ task_id: 'task-1', comment_id: 'cmt_stop', stop_reason: 'user_stop' })
    expect(hoisted.apiFetch.mock.calls[0][0]).toContain('/comment_id/cmt_stop/')
    expect(hoisted.apiFetch.mock.calls[0][0]).not.toMatch(/[?&]comment_id=/)
  })

  it('does not write 缺少评论ID into task statusLogs when comment_id is missing', async () => {
    const updateServerStatus = vi.fn()
    await stopServerApi({
      props: { task: { id: 'task-1' }, updateServerStatus },
      route: { params: { tenant: 't1' } },
      resolveRuntimeApiWorkspaceId: () => 'ws-1',
      fetchServerRuntimeStatus: vi.fn(),
      fetchServerStartHistory: vi.fn(),
      commentId: '',
    })
    expect(hoisted.apiFetch).not.toHaveBeenCalled()
    expect(updateServerStatus).not.toHaveBeenCalled()
  })
})

describe('fetchServerRuntimeStatusApi 评论级 comment_id', () => {
  beforeEach(() => {
    hoisted.apiFetch.mockReset()
  })

  it('appends comment_id for comment card refresh', async () => {
    hoisted.apiFetch.mockResolvedValue(
      jsonResponse({ status: 'success', runtime_status: 'Running', instance_id: 'i-cmt' }),
    )
    await fetchServerRuntimeStatusApi({
      props: { task: { id: 'task-1' } },
      route: { params: { tenant: 't1' } },
      resolveRuntimeApiWorkspaceId: () => 'ws-1',
      commentId: 'cmt_rt',
      serverRuntimeStatus: ref(''),
      serverRuntimeStatusMessage: ref(''),
      serverRuntimeStatusTraceId: ref(''),
      serverRuntimeStatusResponse: ref(null),
      isServerRuntimeStatusLoading: ref(false),
      applyDefaultServerTabByRuntime: () => {},
    })
    expect(hoisted.apiFetch.mock.calls[0][0]).toContain('/comment_id/cmt_rt/')
  })

  it('does not request without comment_id', async () => {
    const serverRuntimeStatusMessage = ref('')
    await fetchServerRuntimeStatusApi({
      props: { task: { id: 'task-1' } },
      route: { params: { tenant: 't1' } },
      resolveRuntimeApiWorkspaceId: () => 'ws-1',
      commentId: '',
      serverRuntimeStatus: ref(''),
      serverRuntimeStatusMessage,
      serverRuntimeStatusTraceId: ref(''),
      serverRuntimeStatusResponse: ref(null),
      isServerRuntimeStatusLoading: ref(false),
      applyDefaultServerTabByRuntime: () => {},
    })
    expect(hoisted.apiFetch).not.toHaveBeenCalled()
    expect(serverRuntimeStatusMessage.value).toBe('缺少评论ID')
  })
})
}

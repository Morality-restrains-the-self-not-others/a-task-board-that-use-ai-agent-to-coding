// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] taskDetailExecLog.test.js requires vitest runtime')
} else {
const { beforeEach, describe, expect, it, vi } = await import('vitest')
const { ref } = await import('vue')

vi.mock('../../utils/apiUtils.js', () => ({
  apiFetch: vi.fn(),
}))

const { apiFetch } = await import('../../utils/apiUtils.js')
const { refreshZTreeExecutionLog, shouldSkipZTreeExecLogFetch } = await import('./taskDetailExecLog.js')

function makeDeps(overrides = {}) {
  return {
    selectedLayerGraphNode: ref({
      nodeKind: 'layer',
      id: '__layer__:layer-1',
      layerId: 'layer-1',
    }),
    layerGraphSnapshot: ref({ jobs: [] }),
    containerEndpointRegistered: ref(true),
    effectiveTenantId: ref('t1'),
    effectiveWorkspaceId: ref('w1'),
    effectiveTaskId: ref('task_1'),
    layerExecLogLoading: ref(false),
    layerExecLogTopError: ref(''),
    layerCloneLogText: ref(''),
    layerCloneLogFetchError: ref(''),
    layerCloneLogFetchErrorTraceId: ref(''),
    layerJobLogFetchError: ref(''),
    layerJobLogFetchErrorTraceId: ref(''),
    layerJobExecutionPayload: ref(null),
    layerChangesByLayerId: ref({}),
    ingestLayerChangesFromExecutionPayload: vi.fn(),
    markContainerTransportUnreachableIfForwardingFailed: vi.fn(),
    markContainerTransportOk: vi.fn(),
    resolveZTreeLogTargets: () => ({ layerId: 'layer-1', jobId: '' }),
    layerLogAbortController: null,
    commentId: ref('cmt-exec'),
    displayComments: ref([{ id: 'cmt-exec', commentKind: 'ai' }]),
    ...overrides,
  }
}

describe('refreshZTreeExecutionLog data-traceId', () => {
  beforeEach(() => {
    vi.mocked(apiFetch).mockReset()
  })

  it('stores clone-log fetch error text and response.traceId', async () => {
    vi.mocked(apiFetch).mockResolvedValue({
      ok: false,
      status: 401,
      traceId: 'clone-trace-abc',
      json: async () => ({ detail: 'Invalid or missing access token' }),
    })
    const deps = makeDeps()
    await refreshZTreeExecutionLog({ reset: true }, deps)
    expect(deps.layerCloneLogFetchError.value).toBe('Invalid or missing access token')
    expect(deps.layerCloneLogFetchErrorTraceId.value).toBe('clone-trace-abc')
  })

  it('stores job-log fetch error text and response.traceId', async () => {
    vi.mocked(apiFetch).mockResolvedValue({
      ok: false,
      status: 401,
      traceId: 'job-trace-xyz',
      json: async () => ({ detail: 'Invalid or missing access token' }),
    })
    const deps = makeDeps({
      resolveZTreeLogTargets: () => ({ layerId: '', jobId: 'job-1' }),
    })
    await refreshZTreeExecutionLog({ reset: true }, deps)
    expect(deps.layerJobLogFetchError.value).toBe('Invalid or missing access token')
    expect(deps.layerJobLogFetchErrorTraceId.value).toBe('job-trace-xyz')
  })

  it('does not surface a bare 404 when clone-log has no JSON detail', async () => {
    vi.mocked(apiFetch).mockResolvedValue({
      ok: false,
      status: 404,
      json: async () => ({}),
    })
    const deps = makeDeps()
    await refreshZTreeExecutionLog({ reset: true }, deps)
    expect(deps.layerCloneLogFetchError.value).not.toBe('404')
    expect(deps.layerCloneLogFetchError.value).not.toBe(404)
    expect(deps.layerCloneLogFetchError.value).toMatch(/HTTP 404/)
    expect(deps.layerCloneLogFetchError.value).toMatch(/克隆日志/)
  })

  it('does not surface a bare 404 when job-log has no JSON detail', async () => {
    vi.mocked(apiFetch).mockResolvedValue({
      ok: false,
      status: 404,
      json: async () => ({}),
    })
    const deps = makeDeps({
      resolveZTreeLogTargets: () => ({ layerId: '', jobId: 'job-1' }),
    })
    await refreshZTreeExecutionLog({ reset: true }, deps)
    expect(deps.layerJobLogFetchError.value).not.toBe('404')
    expect(deps.layerJobLogFetchError.value).not.toBe(404)
    expect(deps.layerJobLogFetchError.value).toMatch(/HTTP 404/)
    expect(deps.layerJobLogFetchError.value).toMatch(/任务日志/)
  })

  it('does not fetch clone-log without comment_id (comment-scoped CSC)', async () => {
    const deps = makeDeps({
      commentId: ref(''),
      displayComments: ref([]),
    })
    await refreshZTreeExecutionLog({ reset: true }, deps)
    expect(apiFetch).not.toHaveBeenCalled()
    expect(deps.layerExecLogTopError.value).toBe('缺少评论ID')
  })

  it('fetches clone-log with funcName-first path and comment_id', async () => {
    vi.mocked(apiFetch).mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({ text: 'clone ok' }),
    })
    await refreshZTreeExecutionLog({ reset: true }, makeDeps())
    const url = String(apiFetch.mock.calls[0][0])
    expect(url).toContain('/api/cloud/compute/container-clone-log/tenant_id/t1/')
    expect(url).toContain('/task_id/task_1/')
    expect(url).toContain('layer_id=layer-1')
    expect(url).toContain('/comment_id/cmt-exec/')
    expect(url).not.toMatch(/[?&]comment_id=/)
    expect(url).not.toMatch(/\/api\/cloud\/compute\/tenant_id\//)
  })

  it('fetches job-log with comment_id path', async () => {
    vi.mocked(apiFetch).mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({
        job: { id: 'job-1', status: 'completed' },
        steps: { steps: [], has_more: false },
      }),
    })
    await refreshZTreeExecutionLog({ reset: true }, makeDeps({
      resolveZTreeLogTargets: () => ({ layerId: '', jobId: 'job-1' }),
    }))
    const url = String(apiFetch.mock.calls[0][0])
    expect(url).toContain('container-job-execution-log')
    expect(url).toContain('/comment_id/cmt-exec/')
    expect(url).toContain('/task_id/task_1/')
  })

  it('clears clone-log error traceId on successful fetch', async () => {
    vi.mocked(apiFetch).mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({ text: 'clone ok' }),
    })
    const deps = makeDeps({
      layerCloneLogFetchError: ref('old'),
      layerCloneLogFetchErrorTraceId: ref('old-tid'),
    })
    await refreshZTreeExecutionLog({ reset: true }, deps)
    expect(deps.layerCloneLogText.value).toBe('clone ok')
    expect(deps.layerCloneLogFetchError.value).toBe('')
    expect(deps.layerCloneLogFetchErrorTraceId.value).toBe('')
  })

  it('still fetches when containerEndpointRegistered is false if comment_id is present', async () => {
    vi.mocked(apiFetch).mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({ text: 'clone ok' }),
    })
    const deps = makeDeps({
      containerEndpointRegistered: ref(false),
    })
    await refreshZTreeExecutionLog({ reset: true }, deps)
    expect(apiFetch).toHaveBeenCalled()
    expect(deps.layerExecLogTopError.value).toBe('')
    expect(deps.layerExecLogTopError.value).not.toBe('容器 server_url 未就绪，无法拉取执行日志')
    expect(deps.layerCloneLogText.value).toBe('clone ok')
  })

  it('maps 409 missing business address to waiting hint instead of a blocking top error', async () => {
    vi.mocked(apiFetch).mockResolvedValue({
      ok: false,
      status: 409,
      json: async () => ({ detail: '容器尚未注册可用业务地址，请先完成启动与 exchange-refresh' }),
    })
    const deps = makeDeps({
      containerEndpointRegistered: ref(false),
    })
    await refreshZTreeExecutionLog({ reset: true }, deps)
    expect(deps.layerExecLogTopError.value).not.toBe('容器 server_url 未就绪，无法拉取执行日志')
    expect(deps.layerCloneLogFetchError.value).toMatch(/正在注册业务地址/)
    expect(deps.layerCloneLogFetchError.value).not.toMatch(/layer not found/)
    expect(deps.markContainerTransportUnreachableIfForwardingFailed).not.toHaveBeenCalled()
  })

  it('patches layer graph job status when execution log infers completed', async () => {
    vi.mocked(apiFetch).mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({
        job: { id: 'job-1', layer_id: 'layer-1', status: '' },
        steps: {
          steps: [
            { step_number: 1, state: 'completed' },
            { step_number: 2, state: 'completed' },
          ],
          has_more: false,
          total_steps: 2,
        },
      }),
    })
    const layerGraphSnapshot = ref({
      layers: [{ layer_id: 'layer-1', job_status: 'running', mind_state: 'running' }],
      jobs: [{ id: 'job-1', layer_id: 'layer-1', status: 'running', command: 'fix' }],
      layers_root: '',
      bootstrap_layer_id: '',
    })
    const deps = makeDeps({
      layerGraphSnapshot,
      resolveZTreeLogTargets: () => ({ layerId: 'layer-1', jobId: 'job-1' }),
    })
    await refreshZTreeExecutionLog({ reset: true }, deps)
    expect(deps.layerJobExecutionPayload.value?.job?.status).toBe('completed')
    expect(layerGraphSnapshot.value.jobs[0].status).toBe('completed')
    expect(layerGraphSnapshot.value.layers[0].job_status).toBe('completed')
  })

  it('does not fetch when containerReleased is true', async () => {
    const deps = makeDeps({ containerReleased: true })
    await refreshZTreeExecutionLog({ reset: true }, deps)
    expect(apiFetch).not.toHaveBeenCalled()
    expect(deps.layerExecLogLoading.value).toBe(false)
  })

  it('does not fetch when bindingStatusFor reports released', async () => {
    const deps = makeDeps({
      commentId: ref('cmt-exec'),
      bindingStatusFor: () => 'released',
    })
    expect(shouldSkipZTreeExecLogFetch(deps)).toBe(true)
    await refreshZTreeExecutionLog({ reset: true }, deps)
    expect(apiFetch).not.toHaveBeenCalled()
  })

  it('does not fetch clone-log when runtimeStatusFor reports Released while binding is still running（OPT-20260823-049）', async () => {
    const deps = makeDeps({
      commentId: ref('cmt-exec'),
      bindingStatusFor: () => 'running',
      runtimeStatusFor: () => 'Released',
    })
    expect(shouldSkipZTreeExecLogFetch(deps)).toBe(true)
    await refreshZTreeExecutionLog({ reset: true }, deps)
    expect(apiFetch).not.toHaveBeenCalled()
    expect(deps.layerExecLogLoading.value).toBe(false)
  })

  it('still fetches when runtimeStatusFor is absent and binding is running', async () => {
    vi.mocked(apiFetch).mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({ text: 'clone ok' }),
    })
    const deps = makeDeps({
      commentId: ref('cmt-exec'),
      bindingStatusFor: () => 'running',
    })
    expect(shouldSkipZTreeExecLogFetch(deps)).toBe(false)
    await refreshZTreeExecutionLog({ reset: true }, deps)
    expect(apiFetch).toHaveBeenCalled()
  })

  it('does not fetch when serverRuntimeNotServing is true', async () => {
    const deps = makeDeps({ serverRuntimeNotServing: true })
    await refreshZTreeExecutionLog({ reset: true }, deps)
    expect(apiFetch).not.toHaveBeenCalled()
  })

  it('fetches SaaS job-execution-log but skips clone-log when containerReleased（OPT-20260823-038）', async () => {
    vi.mocked(apiFetch).mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({
        job: { id: 'job-1', status: 'completed' },
        steps: { steps: [{ step_number: 1, state: 'completed' }], has_more: false },
      }),
    })
    const deps = makeDeps({
      containerReleased: true,
      resolveZTreeLogTargets: () => ({ layerId: 'layer-1', jobId: 'job-1' }),
    })
    expect(shouldSkipZTreeExecLogFetch(deps)).toBe(true)
    await refreshZTreeExecutionLog({ reset: true }, deps)
    const urls = apiFetch.mock.calls.map((c) => String(c[0]))
    expect(urls.some((u) => u.includes('container-job-execution-log'))).toBe(true)
    expect(urls.some((u) => u.includes('container-clone-log'))).toBe(false)
    expect(deps.layerJobExecutionPayload.value?.steps?.steps).toHaveLength(1)
    expect(deps.layerCloneLogFetchError.value).toBe('')
  })

  it('clears leftover clone-log error state when containerReleased（OPT-20260823-038）', async () => {
    vi.mocked(apiFetch).mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({
        job: { id: 'job-1', status: 'completed' },
        steps: { steps: [{ step_number: 1, state: 'completed' }], has_more: false },
      }),
    })
    const deps = makeDeps({
      containerReleased: true,
      resolveZTreeLogTargets: () => ({ layerId: 'layer-1', jobId: 'job-1' }),
    })
    deps.layerCloneLogFetchError.value = '容器正在注册业务地址，就绪后将自动加载'
    deps.layerCloneLogFetchErrorTraceId.value = 'clone-409'
    await refreshZTreeExecutionLog({ reset: false }, deps)
    expect(deps.layerCloneLogFetchError.value).toBe('')
    expect(deps.layerCloneLogFetchErrorTraceId.value).toBe('')
    expect(apiFetch.mock.calls.some((c) => String(c[0]).includes('container-clone-log'))).toBe(false)
  })
})

}

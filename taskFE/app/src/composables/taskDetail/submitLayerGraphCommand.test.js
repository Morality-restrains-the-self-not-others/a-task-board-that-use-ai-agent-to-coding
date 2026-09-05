// @vitest-environment node
/**
 * 「发送给AI」失败须展示业务错误而非裸 HTTP 403，并写入 data-traceId。
 */
if (!process.env.VITEST) {
  console.log('[skip] submitLayerGraphCommand.test.js requires vitest runtime')
} else {
  const { describe, expect, it, vi, beforeEach } = await import('vitest')
  const { ref } = await import('vue')

  vi.mock('../../utils/apiUtils.js', async (importOriginal) => {
    const actual = await importOriginal()
    return { ...actual, apiFetch: vi.fn() }
  })

  const { apiFetch } = await import('../../utils/apiUtils.js')
  const { submitLayerGraphCommand } = await import('./submitLayerGraphCommand.js')
  const { createContainerHeartbeatState } = await import('./taskDetailContainerHeartbeat.js')

  function makeHeartbeat() {
    return createContainerHeartbeatState({
      effectiveTenantId: ref('t1'),
      effectiveWorkspaceId: ref('w1'),
      effectiveTaskId: ref('task_1'),
      containerEndpointRegistered: ref(true),
      containerPageUrl: ref(''),
      containerVscodeUrl: ref(''),
      containerPageLinkPendingReveal: ref(false),
      layerGraphSnapshot: ref(null),
      serverUrl: ref(''),
      isServerRunning: ref(true),
      isServerStarting: ref(false),
      resetLayerGraphFetchBackoff: vi.fn(),
      ensureServerRuntimeAllowsContainerLayerGraph: vi.fn(async () => true),
      refreshLayerGraphFromServer: vi.fn(),
      applyLayerGraphFromPayload: vi.fn(),
      establishSSEConnection: vi.fn(),
    })
  }

  function makeDeps(overrides = {}) {
    const hb = makeHeartbeat()
    return {
      effectiveTenantId: ref('t1'),
      effectiveWorkspaceId: ref('w1'),
      effectiveTaskId: ref('task_1'),
      selectedLayerGraphNode: ref({ nodeKind: 'job', id: 'job-1' }),
      layerGraphSnapshot: ref({ jobs: [{ id: 'job-1', layer_id: 'L1' }] }),
      layerGraphCommandText: ref('写一个 hello world'),
      layerGraphCommandKind: ref('trae'),
      layerGraphCmdError: ref(''),
      layerGraphCmdErrorTraceId: ref(''),
      layerGraphCmdSending: ref(false),
      layerGraphAutoIterationCount: ref(''),
      layerGraphSelectedModel: ref(''),
      layerGraphModelProvider: ref(''),
      layerGraphEditRunTargetJobId: ref(''),
      validateLinkedProjectReposBeforeSendToAi: vi.fn(async () => ({ ok: true })),
      normalizeSelectedAgentModels: (v) => v,
      formatLayerGraphCommandErrorForUser: hb.formatLayerGraphCommandErrorForUser,
      markContainerTransportUnreachableIfForwardingFailed: vi.fn(),
      markContainerTransportOk: vi.fn(),
      refreshLayerGraphFromServer: vi.fn(),
      bumpProjectFileTreeRefresh: vi.fn(),
      displayComments: ref([]),
      activeContainerAgentId: ref(''),
      localTask: ref({}),
      commentId: 'cmt-exec',
      ...overrides,
    }
  }

  describe('submitLayerGraphCommand 403', () => {
    beforeEach(() => {
      apiFetch.mockReset()
    })

    it('shows message body instead of HTTP 403 and sets traceId', async () => {
      apiFetch.mockResolvedValue({
        ok: false,
        status: 403,
        traceId: 'tid-layer-cmd-403',
        _errorData: { message: 'consumer not found' },
        json: async () => ({ message: 'consumer not found' }),
        text: async () => {
          throw new Error('body already consumed')
        },
      })
      const deps = makeDeps()
      await submitLayerGraphCommand(deps)
      expect(deps.layerGraphCmdError.value).toBe('consumer not found')
      expect(deps.layerGraphCmdError.value).not.toBe('HTTP 403')
      expect(deps.layerGraphCmdErrorTraceId.value).toBe('tid-layer-cmd-403')
    })

    it('maps forbidden scope 403 to container-config wording', async () => {
      apiFetch.mockResolvedValue({
        ok: false,
        status: 403,
        traceId: 'tid-forbidden-scope',
        _errorData: { detail: 'forbidden scope' },
        json: async () => ({ detail: 'forbidden scope' }),
        text: async () => {
          throw new Error('body already consumed')
        },
      })
      const deps = makeDeps()
      await submitLayerGraphCommand(deps)
      expect(deps.layerGraphCmdError.value).toMatch(/容器配置/)
      expect(deps.layerGraphCmdErrorTraceId.value).toBe('tid-forbidden-scope')
    })

    it('maps HTML 403 (no detail) away from bare HTTP 403', async () => {
      apiFetch.mockResolvedValue({
        ok: false,
        status: 403,
        traceId: 'tid-html-403',
        _errorData: {
          _rawErrorText: '<html><title>403 Forbidden</title><h1>403 Forbidden</h1></html>',
        },
        json: async () => {
          throw new Error('not json')
        },
        text: async () => {
          throw new Error('body already consumed')
        },
      })
      const deps = makeDeps()
      await submitLayerGraphCommand(deps)
      expect(deps.layerGraphCmdError.value).not.toBe('HTTP 403')
      expect(deps.layerGraphCmdError.value).toMatch(/权限|授权|登录|刷新/)
      expect(deps.layerGraphCmdErrorTraceId.value).toBe('tid-html-403')
    })

    it('skips sending when containerReleased is true（OPT-20260823-038）', async () => {
      const deps = makeDeps({ containerReleased: ref(true) })
      await submitLayerGraphCommand(deps)
      expect(apiFetch).not.toHaveBeenCalled()
      expect(deps.layerGraphCmdError.value).toMatch(/已释放/)
      expect(deps.layerGraphCmdSending.value).toBe(false)
    })
  })
}

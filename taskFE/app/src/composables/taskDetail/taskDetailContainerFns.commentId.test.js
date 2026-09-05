// @vitest-environment node
/**
 * 页面级 container-task-ui-context 必须带 comment_id；无评论 id 时不发请求。
 */
if (!process.env.VITEST) {
  console.log('[skip] taskDetailContainerFns.commentId.test.js requires vitest runtime')
} else {
  const { describe, expect, it, vi, beforeEach } = await import('vitest')
  const { ref } = await import('vue')

  vi.mock('../../utils/apiUtils.js', () => ({
    apiFetch: vi.fn(),
  }))

  const { apiFetch } = await import('../../utils/apiUtils.js')
  const {
    fetchContainerTaskUiContext,
    refreshLayerGraphFromServer,
    resolveContainerUiContextCommentId,
  } = await import('./taskDetailContainerFns.js')
  const { createCommentLayerPanelStore } = await import('./commentLayerPanelStore.js')

  function makeDeps(overrides = {}) {
    return {
      effectiveTenantId: ref('t1'),
      effectiveWorkspaceId: ref('ws1'),
      effectiveTaskId: ref('task1'),
      displayComments: ref([]),
      activeContainerAgentId: ref(''),
      containerEndpointRegistered: ref(true),
      containerPageUrl: ref('https://old.example/page'),
      containerVscodeUrl: ref('vscode://old'),
      containerPageLinkPendingReveal: ref(false),
      markContainerTransportOk: vi.fn(),
      refreshLayerGraphFromServer: vi.fn().mockResolvedValue(undefined),
      ensureServerRuntimeAllowsContainerLayerGraph: vi.fn().mockResolvedValue(true),
      containerHeartbeatSseBuffer: ref([]),
      containerHeartbeatStatus: ref('idle'),
      containerHeartbeatPaused: ref(false),
      isServerRunning: ref(false),
      isServerStarting: ref(false),
      serverRuntimeNotServing: ref(false),
      ...overrides,
    }
  }

  describe('resolveContainerUiContextCommentId', () => {
    it('优先使用显式 commentId', () => {
      expect(resolveContainerUiContextCommentId({
        commentId: ref('c-explicit'),
        displayComments: ref([{ id: 'c-other', commentKind: 'ai' }]),
      })).toBe('c-explicit')
    })

    it('无显式 id 时取当前执行评论', () => {
      expect(resolveContainerUiContextCommentId({
        displayComments: ref([
          { id: 'c1', commentKind: 'user' },
          { id: 'c2', commentKind: 'ai' },
        ]),
        activeContainerAgentId: ref(''),
      })).toBe('c2')
    })

    it('有绑定 CSC 的 running 评论优先于无绑定评论', () => {
      expect(resolveContainerUiContextCommentId({
        displayComments: ref([
          { id: 'c-live', commentKind: 'user' },
          { id: 'c-bound', commentKind: 'user' },
        ]),
        bindingStatusFor: (id) => (id === 'c-live' || id === 'c-bound' ? 'running' : ''),
        bindingCscIdFor: (id) => (id === 'c-bound' ? 'csc-1' : ''),
      })).toBe('c-bound')
    })

    it('有 CSC 的 running 评论优先于最后一条 AI 评论（OPT-20260816-002）', () => {
      expect(resolveContainerUiContextCommentId({
        displayComments: ref([
          { id: 'c-bound', commentKind: 'user' },
          { id: 'c-last-ai', commentKind: 'ai' },
        ]),
        bindingStatusFor: (id) => (id === 'c-bound' ? 'running' : ''),
        bindingCscIdFor: (id) => (id === 'c-bound' ? 'csc-1' : ''),
      })).toBe('c-bound')
    })
  })

  describe('fetchContainerTaskUiContext comment_id', () => {
    beforeEach(() => {
      apiFetch.mockReset()
    })

    it('无 comment_id 时不发请求并保持 unregistered', async () => {
      const deps = makeDeps()
      await fetchContainerTaskUiContext(deps)
      expect(apiFetch).not.toHaveBeenCalled()
      expect(deps.containerEndpointRegistered.value).toBe(false)
      expect(deps.containerPageUrl.value).toBe('')
      expect(deps.containerVscodeUrl.value).toBe('')
    })

    it('带 comment_id 才请求 container-task-ui-context', async () => {
      apiFetch.mockResolvedValue({
        ok: true,
        json: async () => ({
          status: 'success',
          container_endpoint_registered: true,
          container_page_url: 'https://c.example/page',
          container_vscode_url: 'vscode://c',
        }),
      })
      const deps = makeDeps({
        displayComments: ref([{ id: 'cmt-9', commentKind: 'ai' }]),
      })
      await fetchContainerTaskUiContext(deps)
      expect(apiFetch).toHaveBeenCalledTimes(1)
      const url = String(apiFetch.mock.calls[0][0])
      expect(url).toContain('/comment_id/cmt-9/')
      expect(url).toContain('task_id=task1')
      expect(url).not.toMatch(/[?&]comment_id=/)
      expect(deps.containerEndpointRegistered.value).toBe(true)
      expect(deps.containerPageUrl.value).toBe('https://c.example/page')
    })

    it('override A endpoint does not overwrite B slot or non-active page-level flag', async () => {
      apiFetch.mockResolvedValue({
        ok: true,
        json: async () => ({
          status: 'success',
          container_endpoint_registered: false,
          container_page_url: '',
          container_vscode_url: '',
        }),
      })
      const store = createCommentLayerPanelStore()
      store.patch('cmt_b', {
        containerEndpointRegistered: true,
        snapshot: { layers: [{ layer_id: 'layer-b' }], jobs: [] },
      })
      const deps = makeDeps({
        displayComments: ref([{ id: 'cmt_b', commentKind: 'ai' }]),
        layerPanelStore: store,
        refreshLayerGraphFromServer: vi.fn().mockResolvedValue(undefined),
      })
      await fetchContainerTaskUiContext(deps, 'cmt_a')
      expect(store.get('cmt_a').containerEndpointRegistered).toBe(false)
      expect(store.get('cmt_b').containerEndpointRegistered).toBe(true)
      expect(store.get('cmt_b').snapshot.layers[0].layer_id).toBe('layer-b')
      expect(deps.containerEndpointRegistered.value).toBe(true)
    })
  })

  describe('refreshLayerGraphFromServer comment_id', () => {
    beforeEach(() => {
      apiFetch.mockReset()
    })

    it('GET container-layer-graph 带当前执行评论 comment_id', async () => {
      apiFetch.mockResolvedValue({
        ok: true,
        json: async () => ({ layers: [] }),
      })
      const deps = makeDeps({
        displayComments: ref([{ id: 'cmt-layer', commentKind: 'ai' }]),
        containerLayerGraphAuthInvalid: ref(false),
        applyLayerGraphFromPayload: vi.fn(),
        registerLayerGraphFetchFailure: vi.fn(),
        resetLayerGraphFetchBackoff: vi.fn(),
        selectedLayerGraphNode: ref(null),
        layerGraphZNodes: ref([]),
        onLayerGraphNodeSelect: vi.fn(),
        layerGraphSnapshot: ref({ layers: [] }),
        layerGraphSnapshotHasContent: () => false,
        layerGraphSnapshotHasActiveJob: () => false,
        layerGraphSelectionDismissedByUser: ref(false),
        layerGraphFetchBackoffUntil: 0,
        layerGraphHydrateInFlight: false,
        nextTick: async () => {},
      })
      await refreshLayerGraphFromServer(true, { bypassBackoff: true }, deps)
      expect(apiFetch).toHaveBeenCalled()
      const url = String(apiFetch.mock.calls[0][0])
      expect(url).toContain('container-layer-graph')
      expect(url).toContain('/comment_id/cmt-layer/')
    })

    it('endpoint 已注册但 runtime gate 失败时仍 GET 层图快照', async () => {
      apiFetch.mockResolvedValue({
        ok: true,
        json: async () => ({ layers: [{ layer_id: 'L-db' }], jobs: [], source: 'saas_db' }),
      })
      const ensureGate = vi.fn().mockResolvedValue(false)
      const apply = vi.fn()
      const deps = makeDeps({
        containerEndpointRegistered: ref(true),
        displayComments: ref([{ id: 'cmt-layer', commentKind: 'ai' }]),
        ensureServerRuntimeAllowsContainerLayerGraph: ensureGate,
        containerLayerGraphAuthInvalid: ref(false),
        applyLayerGraphFromPayload: apply,
        registerLayerGraphFetchFailure: vi.fn(),
        resetLayerGraphFetchBackoff: vi.fn(),
        markContainerTransportOk: vi.fn(),
        markContainerTransportUnreachableIfForwardingFailed: vi.fn(),
        selectedLayerGraphNode: ref(null),
        layerGraphZNodes: ref([]),
        onLayerGraphNodeSelect: vi.fn(),
        layerGraphSnapshot: ref({ layers: [] }),
        layerGraphSnapshotHasContent: () => false,
        layerGraphSnapshotHasActiveJob: () => false,
        layerGraphSelectionDismissedByUser: ref(false),
        layerGraphFetchBackoffUntil: 0,
        layerGraphHydrateInFlight: false,
        nextTick: async () => {},
      })
      await refreshLayerGraphFromServer(true, { bypassBackoff: true }, deps)
      expect(apiFetch).toHaveBeenCalled()
      expect(String(apiFetch.mock.calls[0][0])).toContain('container-layer-graph')
      expect(apply).toHaveBeenCalled()
    })

    it('endpoint 未注册且 runtime 停机时仍 GET 层图快照', async () => {
      apiFetch.mockResolvedValue({
        ok: true,
        json: async () => ({ layers: [{ layer_id: 'L-snap' }], jobs: [], source: 'saas_db' }),
      })
      const ensureGate = vi.fn().mockResolvedValue(false)
      const apply = vi.fn()
      const deps = makeDeps({
        containerEndpointRegistered: ref(false),
        displayComments: ref([{ id: 'cmt-layer', commentKind: 'ai' }]),
        ensureServerRuntimeAllowsContainerLayerGraph: ensureGate,
        containerLayerGraphAuthInvalid: ref(false),
        applyLayerGraphFromPayload: apply,
        registerLayerGraphFetchFailure: vi.fn(),
        resetLayerGraphFetchBackoff: vi.fn(),
        markContainerTransportOk: vi.fn(),
        markContainerTransportUnreachableIfForwardingFailed: vi.fn(),
        selectedLayerGraphNode: ref(null),
        layerGraphZNodes: ref([]),
        onLayerGraphNodeSelect: vi.fn(),
        layerGraphSnapshot: ref({ layers: [] }),
        layerGraphSnapshotHasContent: () => false,
        layerGraphSnapshotHasActiveJob: () => false,
        layerGraphSelectionDismissedByUser: ref(false),
        layerGraphFetchBackoffUntil: 0,
        layerGraphHydrateInFlight: false,
        nextTick: async () => {},
      })
      await refreshLayerGraphFromServer(true, { bypassBackoff: true }, deps)
      expect(ensureGate).not.toHaveBeenCalled()
      expect(apiFetch).toHaveBeenCalled()
      expect(String(apiFetch.mock.calls[0][0])).toContain('container-layer-graph')
      expect(apply).toHaveBeenCalled()
    })
  })
}

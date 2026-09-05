// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] taskDetailZTreeExecLogLayerChanges.test.js requires vitest runtime')
} else {
  const { describe, expect, it, vi, beforeEach } = await import('vitest')
  const { ref } = await import('vue')

  vi.mock('../../utils/apiUtils.js', () => ({
    apiFetch: vi.fn(),
  }))

  const { apiFetch } = await import('../../utils/apiUtils.js')
  const { resolveContainerUiContextCommentId } = await import('./resolveContainerUiContextCommentId.js')
  const { createLayerChangesController } = await import('./taskDetailZTreeExecLogLayerChanges.js')

  function makeCtx(overrides = {}) {
    const deps = {
      commentId: ref('cmt-exec'),
      displayComments: ref([]),
      ...(overrides.deps || {}),
    }
    return {
      layerChangesByLayerId: ref({}),
      layerChangesRefreshBusy: ref(false),
      layerChangesRefreshError: ref(''),
      layerChangesRefreshErrorTraceId: ref(''),
      layerChangesLoadMoreBusy: ref(false),
      zTreeLogTargets: ref({ layerId: 'layer-1', jobId: 'job-1' }),
      selectedLayerGraphFileTreeLayerId: ref('layer-1'),
      layerGraphSnapshot: ref({
        jobs: [{ id: 'job-1', layer_id: 'layer-1', status: 'completed', command_kind: 'trae' }],
        layers: [{ layer_id: 'layer-1' }],
      }),
      effectiveTenantId: ref('t1'),
      effectiveWorkspaceId: ref('w1'),
      effectiveTaskId: ref('task_1'),
      containerEndpointRegistered: ref(true),
      bumpProjectFileTreeRefresh: vi.fn(),
      resolveContainerUiContextCommentId,
      ...overrides,
      deps,
    }
  }

  describe('createLayerChangesController comment_id', () => {
    beforeEach(() => {
      apiFetch.mockReset()
      apiFetch.mockResolvedValue({
        ok: true,
        json: async () => ({
          layer_changes: { layer_id: 'layer-1', changes: [], change_count: 0 },
        }),
      })
    })

    it('does not refresh without comment_id', async () => {
      const ctx = makeCtx({ deps: { commentId: ref(''), displayComments: ref([]) } })
      const ctl = createLayerChangesController(ctx)
      await ctl.refreshSelectedLayerChanges()
      expect(apiFetch).not.toHaveBeenCalled()
      expect(ctx.layerChangesRefreshError.value).toBe('缺少评论ID')
    })

    it('refresh URL keeps path task_id and path comment_id', async () => {
      const ctl = createLayerChangesController(makeCtx())
      await ctl.refreshSelectedLayerChanges()
      const url = String(apiFetch.mock.calls[0][0])
      expect(url).toContain('/api/cloud/compute/container-job-execution-log/tenant_id/t1/')
      expect(url).toContain('/task_id/task_1/')
      expect(url).toContain('/comment_id/cmt-exec/')
      expect(url).not.toMatch(/[?&]comment_id=/)
    })

    it('does not load-more without comment_id', async () => {
      const ctx = makeCtx({
        deps: { commentId: ref(''), displayComments: ref([]) },
        layerChangesByLayerId: ref({
          'layer-1': {
            layer_id: 'layer-1',
            changes: [{ path: 'a.js', kind: 'added' }],
            change_count: 20,
            has_more: true,
            next_offset: 1,
          },
        }),
      })
      const ctl = createLayerChangesController(ctx)
      await ctl.loadMoreSelectedLayerChanges()
      expect(apiFetch).not.toHaveBeenCalled()
      expect(ctx.layerChangesRefreshError.value).toBe('缺少评论ID')
    })
  })
}

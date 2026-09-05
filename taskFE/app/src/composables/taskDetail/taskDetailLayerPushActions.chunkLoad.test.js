// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] taskDetailLayerPushActions.chunkLoad.test.js requires vitest runtime')
} else {
  const { beforeEach, describe, expect, it, vi } = await import('vitest')
  const { ref } = await import('vue')

  vi.mock('../../utils/requestErrorDisplay.js', () => ({
    showRequestError: vi.fn(),
  }))
  vi.mock('../../utils/gitOauthPushPrecheck.js', () => ({
    gitOauthUnboundReasonForRepoUrls: vi.fn(async () => null),
    repoUrlsFromTaskRepoRows: vi.fn(() => []),
    blockCommentRunIfGitOauthUnbound: vi.fn(async () => false),
  }))
  vi.mock('./containerComputeRequest.js', () => ({
    postContainerCompute: vi.fn(async () => {
      throw new TypeError(
        'Failed to fetch dynamically imported module: https://www.daydaymoney.com/static/assets/layerZtreeNodes-abc.js',
      )
    }),
  }))

  const { showRequestError } = await import('../../utils/requestErrorDisplay.js')
  const { CHUNK_STALE_RELOAD_MESSAGE } = await import('../../utils/chunkLoadGuard.js')
  const { onLayerGraphLayerSubmitAndPush } = await import('./taskDetailLayerPushActions.js')
  const { postContainerCompute } = await import('./containerComputeRequest.js')

  const makeDeps = (overrides = {}) => ({
    containerEndpointRegistered: ref(true),
    containerHttpUnreachable: ref(false),
    containerPageUrl: ref('http://container/ui/dev-local-token'),
    taskRepoRows: ref([]),
    repoCloneIdentityIdForUrl: () => '',
    firstTaskRepoCloneIdentityId: () => '',
    layerGraphPushTargetBranch: ref('main'),
    layerGraphBusyActionKey: ref(''),
    effectiveTenantId: ref('t1'),
    effectiveWorkspaceId: ref('w1'),
    effectiveTaskId: ref('task_1'),
    markContainerTransportUnreachableIfForwardingFailed: vi.fn(),
    markContainerTransportOk: vi.fn(),
    refreshLayerGraphFromServer: vi.fn(),
    refreshZTreeExecutionLog: vi.fn(),
    fetchTaskDetail: vi.fn(),
    layerGraphSnapshot: ref({ layers: [{ layer_id: 'layer_1', git_worktree_dirty: true }] }),
    layerChangesByLayerId: ref({}),
    ...overrides,
  })

  describe('onLayerGraphLayerSubmitAndPush chunk load failure', () => {
    beforeEach(() => {
      vi.clearAllMocks()
      vi.spyOn(console, 'error').mockImplementation(() => {})
    })

    it('shows refresh message when a nested chunk fetch fails during submit-and-push', async () => {
      const deps = makeDeps()
      await onLayerGraphLayerSubmitAndPush({ layerId: 'layer_1' }, deps)
      expect(showRequestError).toHaveBeenCalledWith(CHUNK_STALE_RELOAD_MESSAGE)
      expect(deps.layerGraphBusyActionKey.value).toBe('')
    })

    it('shows generic error for non-chunk failures instead of swallowing', async () => {
      postContainerCompute.mockImplementation(async () => {
        throw new Error('网络超时')
      })
      const deps = makeDeps()
      await onLayerGraphLayerSubmitAndPush({ layerId: 'layer_1' }, deps)
      expect(showRequestError).toHaveBeenCalledWith('网络错误，请稍后重试', expect.any(Error))
      expect(deps.layerGraphBusyActionKey.value).toBe('')
    })
  })
}

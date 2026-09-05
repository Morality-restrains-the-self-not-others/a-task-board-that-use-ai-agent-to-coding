// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] submitAIComment.chunkLoad.test.js requires vitest runtime')
} else {
  const { beforeEach, describe, expect, it, vi } = await import('vitest')
  const { ref } = await import('vue')

  vi.mock('../../utils/requestErrorDisplay.js', () => ({
    showRequestError: vi.fn(),
  }))
  vi.mock('../../domain/auth/services/saved_accounts_store.js', () => ({
    getActiveToken: vi.fn(async () => 'token'),
  }))

  const { showRequestError } = await import('../../utils/requestErrorDisplay.js')
  const { CHUNK_STALE_RELOAD_MESSAGE } = await import('../../utils/chunkLoadGuard.js')
  const { submitAIComment } = await import('./submitAIComment.js')

  const makeDeps = (overrides = {}) => ({
    effectiveTenantId: ref('t1'),
    effectiveWorkspaceId: ref('w1'),
    effectiveTaskId: ref('task_1'),
    selectedLayerGraphNode: ref({ nodeKind: 'job', id: 'job_1' }),
    layerGraphSnapshot: ref({ jobs: [], layers: [] }),
    layerGraphAutoIterationCount: ref(''),
    layerGraphSelectedModel: ref(''),
    layerGraphCommandKind: ref('shell'),
    layerGraphModelProvider: ref(''),
    newComment: ref(''),
    commentComposerChips: ref([]),
    normalizeSelectedAgentModels: (arr) => arr,
    buildCommentBodyWithChips: () => 'hello',
    aiStreamBuffer: ref(''),
    aiStreamBusy: ref(false),
    activeAiInstructId: ref(null),
    localTask: ref({ id: 'task_1' }),
    defaultExecutionMode: 'wait_previous',
    markContainerTransportUnreachableIfForwardingFailed: vi.fn(),
    fetchTaskDetail: vi.fn(),
    ...overrides,
  })

  describe('submitAIComment chunk load failure', () => {
    beforeEach(() => {
      vi.clearAllMocks()
      vi.spyOn(console, 'error').mockImplementation(() => {})
      vi.spyOn(console, 'warn').mockImplementation(() => {})
    })

    it('shows refresh message when a nested chunk fetch fails', async () => {
      const deps = makeDeps({
        buildCommentBodyWithChips: () => {
          throw new TypeError(
            'Failed to fetch dynamically imported module: https://www.daydaymoney.com/static/assets/saved_accounts_store-abc.js',
          )
        },
      })
      await submitAIComment(deps)
      expect(showRequestError).toHaveBeenCalledWith(CHUNK_STALE_RELOAD_MESSAGE)
      expect(deps.aiStreamBusy.value).toBe(false)
      expect(deps.activeAiInstructId.value).toBe(null)
    })

    it('shows generic error for non-chunk failures instead of swallowing', async () => {
      const deps = makeDeps({
        buildCommentBodyWithChips: () => {
          throw new Error('网络超时')
        },
      })
      await submitAIComment(deps)
      expect(showRequestError).toHaveBeenCalledWith(
        expect.stringContaining('发送给 AI 失败'),
        expect.any(Error),
      )
      expect(deps.aiStreamBusy.value).toBe(false)
    })
  })
}

// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] taskDetailFetchFns.chunkLoad.test.js requires vitest runtime')
} else {
  const { beforeEach, describe, expect, it, vi } = await import('vitest')
  const { ref } = await import('vue')

  vi.mock('../../utils/apiUtils.js', () => ({
    apiFetch: vi.fn(),
  }))
  vi.mock('../../utils/requestErrorDisplay.js', () => ({
    showRequestError: vi.fn(),
  }))
  vi.mock('../../utils/gitOauthPushPrecheck.js', async (importOriginal) => {
    const actual = await importOriginal()
    return {
      ...actual,
      blockCommentRunIfGitOauthUnbound: vi.fn(async () => false),
    }
  })

  const { apiFetch } = await import('../../utils/apiUtils.js')
  const { showRequestError } = await import('../../utils/requestErrorDisplay.js')
  const { CHUNK_STALE_RELOAD_MESSAGE } = await import('../../utils/chunkLoadGuard.js')
  const { submitComment } = await import('./taskDetailFetchFns.js')

  describe('submitComment chunk load failure', () => {
    beforeEach(() => {
      vi.clearAllMocks()
      vi.spyOn(console, 'error').mockImplementation(() => {})
    })

    it('shows refresh message when gitOauthPushPrecheck chunk fetch fails', async () => {
      await submitComment({
        effectiveTenantId: ref('t1'),
        effectiveTaskId: ref('task_1'),
        newComment: ref('hello'),
        commentComposerChips: ref([]),
        buildCommentBodyWithChips: () => {
          throw new TypeError(
            'Failed to fetch dynamically imported module: https://www.daydaymoney.com/static/assets/gitOauthPushPrecheck-C15U_Qut.js',
          )
        },
        fetchTaskDetail: vi.fn(),
      })
      expect(apiFetch).not.toHaveBeenCalled()
      expect(showRequestError).toHaveBeenCalledWith(CHUNK_STALE_RELOAD_MESSAGE)
    })

    it('shows request error for non-chunk failures instead of swallowing', async () => {
      await submitComment({
        effectiveTenantId: ref('t1'),
        effectiveTaskId: ref('task_1'),
        newComment: ref('hello'),
        commentComposerChips: ref([]),
        buildCommentBodyWithChips: () => {
          throw new Error('网络超时')
        },
        fetchTaskDetail: vi.fn(),
      })
      expect(apiFetch).not.toHaveBeenCalled()
      expect(showRequestError).toHaveBeenCalledWith('网络超时', expect.any(Error))
    })
  })
}

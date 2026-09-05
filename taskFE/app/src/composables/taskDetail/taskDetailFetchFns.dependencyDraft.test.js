// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] taskDetailFetchFns.dependencyDraft.test.js requires vitest runtime')
} else {
  const { beforeEach, describe, expect, it, vi } = await import('vitest')
  const { ref } = await import('vue')

  vi.mock('../../utils/apiUtils.js', () => ({
    apiFetch: vi.fn(),
  }))

  const { apiFetch } = await import('../../utils/apiUtils.js')
  const { submitComment } = await import('./taskDetailFetchFns.js')
  const {
    setCommentDependencyDraft,
    resetCommentDependencyDraft,
    readCommentDependencyDraft,
  } = await import('./commentDependencyDraft.js')
  const { clearPendingImageMention } = await import('./commentImageMentionState.js')

  describe('submitComment dependency draft', () => {
    beforeEach(() => {
      vi.clearAllMocks()
      resetCommentDependencyDraft()
      clearPendingImageMention()
    })

    it('posts depends_on_comment_ids from draft and resets after success', async () => {
      apiFetch.mockResolvedValue({ ok: true })
      setCommentDependencyDraft({
        executionMode: 'wait_previous',
        dependsOnCommentIds: ['c-prev-1', 'c-prev-2'],
      })
      await submitComment({
        effectiveTenantId: ref('t1'),
        effectiveTaskId: ref('task_1'),
        newComment: ref('next'),
        commentComposerChips: ref([]),
        buildCommentBodyWithChips: () => 'next',
        fetchTaskDetail: vi.fn().mockResolvedValue(undefined),
      })
      const body = JSON.parse(apiFetch.mock.calls[0][1].body)
      expect(body.execution_mode).toBe('wait_previous')
      expect(body.depends_on_comment_ids).toEqual(['c-prev-1', 'c-prev-2'])
      expect(readCommentDependencyDraft()).toEqual({
        executionMode: 'wait_previous',
        dependsOnCommentIds: [],
        autoCommit: false,
      })
    })
  })
}

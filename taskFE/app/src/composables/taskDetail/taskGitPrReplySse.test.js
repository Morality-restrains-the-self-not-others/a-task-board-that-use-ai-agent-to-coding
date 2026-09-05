// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] taskGitPrReplySse.test.js requires vitest runtime')
} else {
const { describe, expect, it } = await import('vitest')
const {
  isTaskGitPrReplyCreatedSse,
  planGitPrReplyCreatedSseRefetch,
} = await import('./taskGitPrReplySse.js')

describe('taskGitPrReplySse', () => {
  it('recognizes task_git_pr_reply_created', () => {
    expect(isTaskGitPrReplyCreatedSse({ event_name: 'task_git_pr_reply_created' })).toBe(true)
    expect(isTaskGitPrReplyCreatedSse({ event_name: 'container_heartbeat' })).toBe(false)
  })

  it('refetches once per comment_id', () => {
    const seen = new Set()
    const first = planGitPrReplyCreatedSseRefetch(
      { event_name: 'task_git_pr_reply_created', comment_id: 'cmt_1' },
      seen,
    )
    expect(first.shouldRefetch).toBe(true)
    const second = planGitPrReplyCreatedSseRefetch(
      { event_name: 'task_git_pr_reply_created', comment_id: 'cmt_1' },
      seen,
    )
    expect(second.shouldRefetch).toBe(false)
    const other = planGitPrReplyCreatedSseRefetch(
      { event_name: 'task_git_pr_reply_created', comment_id: 'cmt_2' },
      seen,
    )
    expect(other.shouldRefetch).toBe(true)
  })
})
}

// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] taskDetailGitPrReply.merge-timeout.test.js requires vitest runtime')
} else {
const { describe, expect, it, vi, beforeEach } = await import('vitest')

const hoisted = vi.hoisted(() => ({
  apiFetch: vi.fn(),
}))

vi.mock('../../utils/apiUtils.js', () => ({
  apiFetch: (...args) => hoisted.apiFetch(...args),
}))

const {
  mergeGitPullRequest,
  MERGE_GIT_PR_TIMEOUT_MS,
  MERGE_GIT_PR_TIMEOUT_MESSAGE,
} = await import('./taskDetailGitPrReply.js')

describe('mergeGitPullRequest timeout contract', () => {
  beforeEach(() => {
    hoisted.apiFetch.mockReset()
  })

  it('uses a 90s apiFetch timeout so GitLab merge can finish before the browser abort', async () => {
    hoisted.apiFetch.mockResolvedValue({
      ok: true,
      json: async () => ({ ok: true, merged: true, noop: true }),
    })
    const out = await mergeGitPullRequest({
      tenantId: '877397588196749312',
      htmlUrl: 'https://gitlab-tencent-sh-1.daydaymoney.com/g/r/-/merge_requests/4',
      taskId: 'task_1',
      commentId: 'cmt_1',
      headers: { 'Idempotency-Key': 'idem-1' },
    })
    expect(out.noop).toBe(true)
    expect(MERGE_GIT_PR_TIMEOUT_MS).toBe(90000)
    const [, opts] = hoisted.apiFetch.mock.calls[0]
    expect(opts.timeout).toBe(90000)
  })

  it('rewrites client TimeoutError away from「请检查网络」', async () => {
    const timeoutErr = new Error('请求超时（30 秒），请检查网络后重试')
    timeoutErr.name = 'TimeoutError'
    timeoutErr.traceId = 'ab3e126c-2618-47c4-ba4f-675948d61ad9'
    hoisted.apiFetch.mockRejectedValue(timeoutErr)
    await expect(
      mergeGitPullRequest({
        tenantId: 't1',
        htmlUrl: 'https://gitlab.example/g/r/-/merge_requests/1',
        taskId: 'task_1',
        commentId: 'cmt_1',
      }),
    ).rejects.toMatchObject({
      name: 'TimeoutError',
      message: MERGE_GIT_PR_TIMEOUT_MESSAGE,
      traceId: 'ab3e126c-2618-47c4-ba4f-675948d61ad9',
    })
    expect(MERGE_GIT_PR_TIMEOUT_MESSAGE).not.toContain('请检查网络')
  })
})
}

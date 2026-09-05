// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] taskDetailGitPrReply.test.js requires vitest runtime')
} else {
const { describe, expect, it, vi, beforeEach } = await import('vitest')
const { ref } = await import('vue')

vi.mock('../../utils/apiUtils.js', () => ({
  apiFetch: vi.fn(),
}))

const { apiFetch } = await import('../../utils/apiUtils.js')
const {
  collectGitPrHtmlUrls,
  gitPrHtmlUrlOf,
  gitPrProviderOf,
  gitPrReplyCommentPath,
  recordGitPrReplyComment,
  fetchGitPrStatuses,
} = await import('./taskDetailGitPrReply.js')

describe('taskDetailGitPrReply', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('collects nested git_pr urls', () => {
    const urls = collectGitPrHtmlUrls([
      {
        id: 'p',
        children: [
          { id: 'c', git_pr: { html_url: 'https://github.com/acme/repo/pull/1' } },
        ],
      },
    ])
    expect(urls).toEqual(['https://github.com/acme/repo/pull/1'])
    expect(gitPrHtmlUrlOf({ git_pr_html_url: 'https://x' })).toBe('https://x')
    expect(gitPrProviderOf('https://gitlab.example/a/b/-/merge_requests/2')).toBe('gitlab')
  })

  // OPT-20260827-040：卡片链接必须原样继承存储的 scheme（http:// IP GitLab 不得被改成 https://host:443）。
  it('gitPrHtmlUrlOf preserves http scheme for IP GitLab', () => {
    expect(gitPrHtmlUrlOf({ git_pr: { html_url: 'http://115.29.110.74/acme/repo/-/merge_requests/1' } }))
      .toBe('http://115.29.110.74/acme/repo/-/merge_requests/1')
    expect(gitPrHtmlUrlOf({ git_pr_html_url: 'http://127.0.0.1:8012/a/b' }))
      .toBe('http://127.0.0.1:8012/a/b')
  })

  it('posts independent reply with parent in path', async () => {
    apiFetch.mockResolvedValue({ ok: true, json: async () => ({ id: 'cmt_pr' }) })
    await recordGitPrReplyComment(
      {
        effectiveTenantId: ref('t1'),
        effectiveWorkspaceId: ref('ws1'),
        effectiveTaskId: ref('task_1'),
        displayComments: ref([{ id: 'cmt-exec', commentKind: 'user' }]),
        activeContainerAgentId: ref(''),
      },
      'https://github.com/acme/repo/pull/9',
    )
    expect(apiFetch).toHaveBeenCalledTimes(1)
    const [url, opts] = apiFetch.mock.calls[0]
    expect(url).toBe(gitPrReplyCommentPath({
      tenantId: 't1',
      workspaceId: 'ws1',
      taskId: 'task_1',
      parentCommentId: 'cmt-exec',
    }))
    expect(String(url)).toContain('/api/tenant_id/t1/workspaceId/ws1/tasks/task_1/comments/cmt-exec/')
    const body = JSON.parse(opts.body)
    expect(body).toMatchObject({
      execution_mode: 'independent',
      git_pr: { html_url: 'https://github.com/acme/repo/pull/9', provider: 'github' },
    })
    expect(body.parent_comment_id).toBeUndefined()
  })

  it('skips when parent or workspace is missing', async () => {
    await recordGitPrReplyComment(
      {
        effectiveTenantId: ref('t1'),
        effectiveWorkspaceId: ref(''),
        effectiveTaskId: ref('task_1'),
        displayComments: ref([{ id: 'cmt-exec', commentKind: 'user' }]),
        activeContainerAgentId: ref(''),
      },
      'https://github.com/acme/repo/pull/9',
    )
    expect(apiFetch).not.toHaveBeenCalled()
    await recordGitPrReplyComment(
      {
        effectiveTenantId: ref('t1'),
        effectiveWorkspaceId: ref('ws1'),
        effectiveTaskId: ref('task_1'),
        displayComments: ref([]),
        activeContainerAgentId: ref(''),
      },
      'https://github.com/acme/repo/pull/9',
    )
    expect(apiFetch).not.toHaveBeenCalled()
  })

  it('maps merge-request-status results including merged audit by', async () => {
    apiFetch.mockResolvedValue({
      ok: true,
      json: async () => ({
        results: [
          {
            html_url: 'https://gitlab-tencent-sh-1.daydaymoney.com/example-user/somanyad/-/merge_requests/4',
            state: 'merged',
            title: 'done',
            merged_by: '张三',
            merged_at: '2026-08-24T02:14:29Z',
          },
        ],
      }),
    })
    const out = await fetchGitPrStatuses('877397588196749312', [
      'https://gitlab-tencent-sh-1.daydaymoney.com/example-user/somanyad/-/merge_requests/4',
    ])
    expect(out['https://gitlab-tencent-sh-1.daydaymoney.com/example-user/somanyad/-/merge_requests/4']).toEqual({
      state: 'merged',
      title: 'done',
      error: '',
      merged_by: '张三',
      merged_at: '2026-08-24T02:14:29Z',
    })
  })

  it('defaults audit by fields empty when absent', async () => {
    apiFetch.mockResolvedValue({
      ok: true,
      json: async () => ({
        results: [
          {
            html_url: 'https://gitlab-tencent-sh-1.daydaymoney.com/example-user/somanyad/-/merge_requests/4',
            state: 'open',
          },
        ],
      }),
    })
    const out = await fetchGitPrStatuses('877397588196749312', [
      'https://gitlab-tencent-sh-1.daydaymoney.com/example-user/somanyad/-/merge_requests/4',
    ])
    expect(out['https://gitlab-tencent-sh-1.daydaymoney.com/example-user/somanyad/-/merge_requests/4']).toEqual({
      state: 'open',
      title: '',
      error: '',
      merged_by: '',
      merged_at: '',
    })
  })
})
}

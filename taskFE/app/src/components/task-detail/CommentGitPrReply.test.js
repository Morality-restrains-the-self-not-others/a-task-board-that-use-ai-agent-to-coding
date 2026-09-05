// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] CommentGitPrReply.test.js requires vitest runtime')
} else {
const { describe, expect, it, vi, beforeEach } = await import('vitest')
const { mount, flushPromises } = await import('@vue/test-utils')

vi.mock('../../utils/requestErrorDisplay.js', async (importOriginal) => {
  const actual = await importOriginal()
  return {
    ...actual,
    showRequestError: vi.fn(),
  }
})
vi.mock('../../composables/taskDetail/taskDetailGitPrReply.js', async (importOriginal) => {
  const actual = await importOriginal()
  return {
    ...actual,
    mergeGitPullRequest: vi.fn(),
  }
})

const { default: CommentGitPrReply } = await import('./CommentGitPrReply.vue')
const { mergeGitPullRequest } = await import('../../composables/taskDetail/taskDetailGitPrReply.js')
const { showRequestError } = await import('../../utils/requestErrorDisplay.js')

describe('CommentGitPrReply', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  const comment = {
    id: 'cmt_pr',
    git_pr: { html_url: 'https://github.com/acme/repo/pull/42', provider: 'github' },
  }

  it('shows open state and merge button', () => {
    const wrapper = mount(CommentGitPrReply, {
      props: { comment, tenantId: 't1', taskId: 'task_1', status: { state: 'open' } },
    })
    expect(wrapper.get('[data-testid="comment-git-pr-link"]').attributes('href')).toBe(
      'https://github.com/acme/repo/pull/42',
    )
    expect(wrapper.get('[data-testid="comment-git-pr-state"]').text()).toBe('未合并')
    expect(wrapper.get('[data-testid="comment-git-pr-merge-btn"]').exists()).toBe(true)
  })

  it('hides merge when already merged', () => {
    const wrapper = mount(CommentGitPrReply, {
      props: { comment, tenantId: 't1', taskId: 'task_1', status: { state: 'merged' } },
    })
    expect(wrapper.get('[data-testid="comment-git-pr-state"]').text()).toBe('已合并')
    expect(wrapper.find('[data-testid="comment-git-pr-merge-btn"]').exists()).toBe(false)
  })

  it('shows merged-by with hover time from audit when available', () => {
    const wrapper = mount(CommentGitPrReply, {
      props: {
        comment,
        tenantId: 't1',
        taskId: 'task_1',
        status: { state: 'merged', merged_by: '张三', merged_at: '2026-08-24T02:14:29Z' },
      },
    })
    const badge = wrapper.get('[data-testid="comment-git-pr-state"]')
    expect(badge.text()).toBe('已由 张三 合并')
    // 悬停显示点击时间（本地时区格式化）
    expect(badge.attributes('title')).toContain('由 张三 于 ')
    expect(badge.attributes('title')).toContain('合并')
    expect(badge.attributes('data-merged-by')).toBe('张三')
    expect(badge.attributes('data-merged-at')).toBe('2026-08-24T02:14:29Z')
    expect(wrapper.find('[data-testid="comment-git-pr-merge-btn"]').exists()).toBe(false)
  })

  it('shows merge time only when merged_by is absent', () => {
    const wrapper = mount(CommentGitPrReply, {
      props: {
        comment,
        tenantId: 't1',
        taskId: 'task_1',
        status: { state: 'merged', merged_at: '2026-08-24T02:14:29Z' },
      },
    })
    const badge = wrapper.get('[data-testid="comment-git-pr-state"]')
    expect(badge.text()).toBe('已合并')
    expect(badge.attributes('title')).toContain('合并时间：')
  })

  it('hides merge while status is unknown so already-merged MRs are not clickable before poll', () => {
    const wrapper = mount(CommentGitPrReply, {
      props: { comment, tenantId: 't1', taskId: 'task_1', status: { state: 'unknown' } },
    })
    expect(wrapper.get('[data-testid="comment-git-pr-state"]').text()).toBe('状态未知')
    expect(wrapper.find('[data-testid="comment-git-pr-merge-btn"]').exists()).toBe(false)
  })

  it('shows status error and bind link when git oauth is not connected', () => {
    const wrapper = mount(CommentGitPrReply, {
      props: {
        comment,
        tenantId: 't1',
        taskId: 'task_1',
        status: { state: 'unknown', error: 'git oauth not connected' },
      },
    })
    expect(wrapper.get('[data-testid="comment-git-pr-error"]').text()).toBe(
      '尚未绑定 Git 网站 OAuth，请先在账号中心完成授权后再试',
    )
    const bind = wrapper.get('[data-testid="comment-git-pr-oauth-bind"]')
    expect(bind.attributes('href')).toContain('/api/git-oauth/github-start-from-gateway/')
    expect(bind.text()).toContain('去绑定 Git OAuth')
  })

  it('hides bind link and oauth-expired error when shared oauth readiness is bound', () => {
    const wrapper = mount(CommentGitPrReply, {
      props: {
        comment,
        tenantId: 't1',
        taskId: 'task_1',
        status: { state: 'open', error: 'Git OAuth 授权已失效，请重新绑定后再合并' },
        oauthReadiness: { loading: false, allBound: true, unboundRepoUrls: [] },
      },
    })
    expect(wrapper.find('[data-testid="comment-git-pr-oauth-bind"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="comment-git-pr-error"]').exists()).toBe(false)
    expect(wrapper.get('[data-testid="comment-git-pr-merge-btn"]').exists()).toBe(true)
  })

  it('shows bind link when GitLab refresh token is rejected', () => {
    const wrapper = mount(CommentGitPrReply, {
      props: {
        comment,
        tenantId: 't1',
        taskId: 'task_1',
        status: { state: 'unknown', error: 'gitlab refresh http 400' },
      },
    })
    expect(wrapper.get('[data-testid="comment-git-pr-error"]').text()).toBe(
      'Git OAuth 授权已失效，请重新绑定后再试',
    )
    expect(wrapper.get('[data-testid="comment-git-pr-oauth-bind"]').text()).toContain('去绑定 Git OAuth')
  })

  it('one-click merge emits merged with audit by meta', async () => {
    mergeGitPullRequest.mockResolvedValue({
      ok: true,
      merged: true,
      merged_by: '李四',
      merged_at: '2026-08-24T02:20:00Z',
    })
    const wrapper = mount(CommentGitPrReply, {
      props: { comment, tenantId: 't1', taskId: 'task_1', status: { state: 'open' } },
    })
    await wrapper.get('[data-testid="comment-git-pr-merge-btn"]').trigger('click')
    await flushPromises()
    expect(mergeGitPullRequest).toHaveBeenCalled()
    const arg = mergeGitPullRequest.mock.calls[0][0]
    expect(arg.htmlUrl).toBe('https://github.com/acme/repo/pull/42')
    expect(arg.commentId).toBe('cmt_pr')
    expect(arg.headers['Idempotency-Key']).toBeTruthy()
    const emitted = wrapper.emitted('merged')
    expect(emitted).toBeTruthy()
    // 第二个参数携带审计回显（merged_by/merged_at），父组件据此立即改徽章
    expect(emitted[0][1]).toEqual({ merged_by: '李四', merged_at: '2026-08-24T02:20:00Z' })
  })

  it('merge failure with oauth-unbound passes bind link in popup and stamps error trace', async () => {
    const mergeErr = new Error('尚未绑定 Git 网站 OAuth，请先在账号中心完成授权后再试')
    mergeErr.traceId = '4d3f9c2a1b2e4f8a9c6d1a2b3c4d5e6f'
    mergeGitPullRequest.mockRejectedValue(mergeErr)
    const wrapper = mount(CommentGitPrReply, {
      props: {
        comment,
        tenantId: 't1',
        taskId: 'task_1',
        status: { state: 'open', trace_id: '8b7a6c5d4e3f2a1b0c9d8e7f6a5b4c3d' },
      },
    })
    await wrapper.get('[data-testid="comment-git-pr-merge-btn"]').trigger('click')
    await flushPromises()
    expect(showRequestError).toHaveBeenCalledTimes(1)
    const [, , options] = showRequestError.mock.calls[0]
    expect(options.additionalActions).toHaveLength(1)
    const action = options.additionalActions[0]
    expect(action.text).toContain('去绑定 Git OAuth')
    expect(action.href).toContain('/api/git-oauth/github-start-from-gateway/')
    // error 行带本次合并失败的 trace（优先 merge trace，而非 status 载荷）
    const errSpan = wrapper.get('[data-testid="comment-git-pr-error"]')
    expect(errSpan.attributes('data-traceid') || errSpan.attributes('data-traceId')).toBe(
      '4d3f9c2a1b2e4f8a9c6d1a2b3c4d5e6f',
    )
  })
})
}

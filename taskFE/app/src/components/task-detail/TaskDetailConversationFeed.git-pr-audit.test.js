// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] TaskDetailConversationFeed.git-pr-audit.test.js requires vitest runtime')
} else {
  /**
   * 审计回显回归：一键合并成功后徽章显示「已由 xxx 合并」，
   * 悬停显示点击时间；服务端 status 审计字段亦回显到徽章。
   */
  const { describe, expect, it, vi, beforeEach } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')

  vi.mock('../../utils/apiUtils.js', () => ({ apiFetch: vi.fn() }))

  const { apiFetch } = await import('../../utils/apiUtils.js')
  const { default: TaskDetailConversationFeed } = await import('./TaskDetailConversationFeed.vue')

  const prUrl = 'https://gitlab-tencent-sh-1.daydaymoney.com/example-user/somanyad/-/merge_requests/4'
  const gitPrComment = (extra = {}) => ({
    id: 'c-pr',
    commentKind: 'user',
    content: prUrl,
    created_at: '2026-08-15T13:32:31Z',
    created_by: { username: '软刀' },
    git_pr: { html_url: prUrl, provider: 'gitlab' },
    ...extra,
  })

  describe('TaskDetailConversationFeed git-pr audit by', () => {
    beforeEach(() => {
      vi.clearAllMocks()
    })

    it('shows merged-by from server status audit with hover time', async () => {
      apiFetch.mockResolvedValue({
        ok: true,
        json: async () => ({
          results: [
            { html_url: prUrl, state: 'merged', merged_by: '张三', merged_at: '2026-08-24T02:14:29Z' },
          ],
        }),
      })
      const wrapper = mount(TaskDetailConversationFeed, {
        props: { tenantId: '877397588196749312', taskId: 'task_1', comments: [gitPrComment()] },
      })
      await flushPromises()
      const badge = wrapper.get('[data-testid="comment-git-pr-state"]')
      expect(badge.text()).toBe('已由 张三 合并')
      expect(badge.attributes('title')).toContain('由 张三 于 ')
      expect(badge.attributes('title')).toContain('合并')
    })

    it('shows plain merged when status lacks audit by', async () => {
      apiFetch.mockResolvedValue({
        ok: true,
        json: async () => ({ results: [{ html_url: prUrl, state: 'merged' }] }),
      })
      const wrapper = mount(TaskDetailConversationFeed, {
        props: { tenantId: '877397588196749312', taskId: 'task_1', comments: [gitPrComment()] },
      })
      await flushPromises()
      expect(wrapper.get('[data-testid="comment-git-pr-state"]').text()).toBe('已合并')
    })

    it('updates badge immediately after one-click merge with audit meta', async () => {
      // 首次状态查询返回 open（展示合并按钮）；点击合并后 merge 接口返回审计回显；
      // 合并成功触发的状态轮询再返回 merged 审计数据
      let statusCalls = 0
      apiFetch.mockImplementation(async (url) => {
        const u = String(url || '')
        if (u.includes('/merge-request-merge/')) {
          return {
            ok: true,
            json: async () => ({
              ok: true,
              merged: true,
              state: 'merged',
              merged_by: '李四',
              merged_at: '2026-08-24T02:20:00Z',
            }),
          }
        }
        statusCalls += 1
        if (statusCalls === 1) {
          return { ok: true, json: async () => ({ results: [{ html_url: prUrl, state: 'open' }] }) }
        }
        return {
          ok: true,
          json: async () => ({
            results: [
              { html_url: prUrl, state: 'merged', merged_by: '李四', merged_at: '2026-08-24T02:20:00Z' },
            ],
          }),
        }
      })
      const wrapper = mount(TaskDetailConversationFeed, {
        props: {
          tenantId: '877397588196749312',
          taskId: 'task_1',
          comments: [gitPrComment()],
        },
      })
      await flushPromises()
      const btn = wrapper.get('[data-testid="comment-git-pr-merge-btn"]')
      await btn.trigger('click')
      await flushPromises()
      const badge = wrapper.get('[data-testid="comment-git-pr-state"]')
      expect(badge.text()).toBe('已由 李四 合并')
      expect(wrapper.find('[data-testid="comment-git-pr-merge-btn"]').exists()).toBe(false)
    })
  })
}

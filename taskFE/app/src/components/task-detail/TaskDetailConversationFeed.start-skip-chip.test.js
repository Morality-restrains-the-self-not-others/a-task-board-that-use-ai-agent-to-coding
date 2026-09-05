// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] TaskDetailConversationFeed.start-skip-chip.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const { mount } = await import('@vue/test-utils')
  const { default: TaskDetailConversationFeed } = await import('./TaskDetailConversationFeed.vue')

  const skipNotice =
    '未启动服务器：无法获取子 Git 仓库列表：未检测到可用授权。请先在个人资料完成 Git 网站绑定，或先登录本地 GitLab 后重试。'

  describe('TaskDetailConversationFeed start-skip chip', () => {
    it('renders nested-git skip notice in the author row, not in the body paragraph', () => {
      const wrapper = mount(TaskDetailConversationFeed, {
        props: {
          comments: [
            {
              id: 'c-auto-1',
              commentKind: 'user',
              content: `【自动运行】\n将当前时间，硬编码进 now.md 文件中\n$trae-agent-skill /general-coding\n\n${skipNotice}`,
              created_at: '2026-08-27T05:00:00Z',
              created_by: { username: '软刀' },
            },
          ],
        },
      })

      const row = wrapper.get('[data-testid="comment-author-row"]')
      const chip = row.get('[data-testid="comment-start-skip-notice"]')
      expect(chip.text()).toContain('未启动服务器：')
      expect(chip.text()).toContain('未检测到可用授权')

      const body = wrapper.get('p.mt-1.text-sm')
      expect(body.text()).toContain('将当前时间')
      expect(body.text()).not.toContain('未启动服务器：')
      expect(body.text()).not.toContain('未检测到可用授权')
    })
  })
}

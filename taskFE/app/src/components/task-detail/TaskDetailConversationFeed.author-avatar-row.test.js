// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] TaskDetailConversationFeed.author-avatar-row.test.js requires vitest runtime')
} else {
  /**
   * 回归：评论气泡头像须与昵称同一行，不能作为整卡左侧独立列
   * （全局 .flex.gap-3 { align-items:center } 会把头像垂直居中到执行细节中部）。
   */
  const { describe, expect, it } = await import('vitest')
  const { mount } = await import('@vue/test-utils')
  const { default: TaskDetailConversationFeed } = await import('./TaskDetailConversationFeed.vue')

  describe('TaskDetailConversationFeed author avatar row', () => {
    it('places the avatar in the same row as the nickname, not as a left column of the bubble', () => {
      const wrapper = mount(TaskDetailConversationFeed, {
        props: {
          comments: [
            {
              id: 'c-user-1',
              commentKind: 'user',
              content: '写一个 hello world程序 用 js',
              created_at: '2026-08-15T13:32:31Z',
              created_by: { username: '软刀' },
            },
          ],
        },
      })

      const row = wrapper.get('[data-testid="comment-author-row"]')
      const avatar = row.get('[data-testid="comment-author-avatar"]')
      const name = row.get('[data-testid="comment-author-name"]')
      expect(name.text()).toBe('软刀')
      expect(avatar.element.tagName).toBe('IMG')
      expect(avatar.element.nextElementSibling).toBe(name.element)

      const bubble = wrapper.get('[data-testid="comment-bubble"]')
      const directImgs = [...bubble.element.children].filter((el) => el.tagName === 'IMG')
      expect(directImgs).toHaveLength(0)
    })

    it('places child comment avatars next to the nickname as well', () => {
      const wrapper = mount(TaskDetailConversationFeed, {
        props: {
          comments: [
            {
              id: 'c-user-1',
              commentKind: 'user',
              content: 'parent',
              created_at: '2026-08-15T13:32:31Z',
              created_by: { username: '软刀' },
              children: [
                {
                  id: 'c-agent-1',
                  commentKind: 'container_agent',
                  content: 'agent reply',
                  created_at: '2026-08-15T13:33:00Z',
                  created_by: { username: 'Agent' },
                },
              ],
            },
          ],
        },
      })

      const rows = wrapper.findAll('[data-testid="comment-author-row"]')
      expect(rows).toHaveLength(2)
      const childRow = rows[1]
      const childAvatar = childRow.get('[data-testid="comment-author-avatar"]')
      const childName = childRow.get('[data-testid="comment-author-name"]')
      expect(childName.text()).toBe('Agent')
      expect(childAvatar.element.nextElementSibling).toBe(childName.element)
    })
  })
}

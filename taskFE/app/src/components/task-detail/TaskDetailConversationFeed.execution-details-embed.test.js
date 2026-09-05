// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] TaskDetailConversationFeed.execution-details-embed.test.js requires vitest runtime')
} else {
const { describe, expect, it } = await import('vitest')
const { mount } = await import('@vue/test-utils')
const { default: TaskDetailConversationFeed } = await import('./TaskDetailConversationFeed.vue')

describe('TaskDetailConversationFeed execution-details embed', () => {
  it('renders execution-details slot inside the comment bubble, not as a sibling block', () => {
    const wrapper = mount(TaskDetailConversationFeed, {
      props: {
        comments: [
          {
            id: 'c-user-1',
            commentKind: 'user',
            content: '写一个 hello world 用 lisp',
            created_at: '2026-07-22T18:41:09Z',
            created_by: { username: '公司创建者' },
          },
        ],
      },
      slots: {
        'execution-details': `
          <details data-testid="comment-execution-details" class="mt-2">
            <summary data-testid="comment-execution-details-summary">执行细节</summary>
          </details>
        `,
      },
    })

    const bubble = wrapper.get('[data-testid="comment-bubble"]')
    expect(bubble.find('[data-testid="comment-execution-details"]').exists()).toBe(true)

    const itemRoot = wrapper.get('div.space-y-2')
    const directDetails = itemRoot.element.querySelector(':scope > [data-testid="comment-execution-details"]')
    expect(directDetails).toBeNull()
  })

  it('does not render execution-details slot when comment id is empty', () => {
    const wrapper = mount(TaskDetailConversationFeed, {
      props: {
        comments: [
          {
            id: '',
            commentKind: 'user',
            content: '无 id 的脏数据',
            created_at: '2026-07-22T18:41:09Z',
            created_by: { username: '公司创建者' },
          },
        ],
      },
      slots: {
        'execution-details': `
          <details data-testid="comment-execution-details" class="mt-2">
            <summary>执行细节</summary>
          </details>
        `,
      },
    })

    expect(wrapper.find('[data-testid="comment-bubble"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="comment-execution-details"]').exists()).toBe(false)
  })
})

}

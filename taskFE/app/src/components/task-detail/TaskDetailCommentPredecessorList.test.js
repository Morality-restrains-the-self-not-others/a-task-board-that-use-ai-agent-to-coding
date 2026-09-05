// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] TaskDetailCommentPredecessorList.test.js requires vitest runtime')
} else {
const { describe, expect, it } = await import('vitest')
const { mount } = await import('@vue/test-utils')
const { default: TaskDetailCommentPredecessorList } = await import('./TaskDetailCommentPredecessorList.vue')

const SAMPLE = [
  {
    id: 'C1',
    summary: '先跑测试套件',
    status: 'running',
    statusLabel: '运行中',
    finished: false,
    missing: false,
  },
  {
    id: 'C2',
    summary: '再部署',
    status: 'completed',
    statusLabel: '已完成',
    finished: true,
    missing: false,
  },
]

describe('TaskDetailCommentPredecessorList', () => {
  it('lists each predecessor with status label', () => {
    const wrapper = mount(TaskDetailCommentPredecessorList, {
      props: { predecessors: SAMPLE },
    })
    const root = wrapper.get('[data-testid="comment-execution-predecessor-list"]')
    expect(root.element.tagName).toBe('DIV')
    expect(root.text()).toContain('1 个未完成')
    const rows = wrapper.findAll('[data-testid="comment-execution-predecessor-row"]')
    expect(rows).toHaveLength(2)
    expect(rows[0].text()).toContain('先跑测试套件')
    expect(rows[0].get('[data-testid="comment-execution-predecessor-status"]').text()).toBe('运行中')
    expect(rows[1].get('[data-testid="comment-execution-predecessor-status"]').text()).toBe('已完成')
  })

  it('shows empty hint when waiting with no identified predecessors', () => {
    const wrapper = mount(TaskDetailCommentPredecessorList, {
      props: { predecessors: [] },
    })
    expect(wrapper.get('[data-testid="comment-execution-predecessor-empty"]').text()).toContain('前序不在当前页，可加载更早评论')
  })

  it('renders unloaded predecessors with 未加载 label and load-earlier hint', () => {
    const wrapper = mount(TaskDetailCommentPredecessorList, {
      props: {
        predecessors: [
          {
            id: 'ghost_7',
            summary: '前序评论 ghost_7（不在当前页）',
            status: '',
            statusLabel: '未加载',
            finished: false,
            missing: true,
          },
        ],
      },
    })
    const row = wrapper.get('[data-testid="comment-execution-predecessor-row"]')
    expect(row.get('[data-testid="comment-execution-predecessor-status"]').text()).toBe('未加载')
    expect(row.text()).toContain('不在当前页')
    expect(wrapper.get('[data-testid="comment-execution-predecessor-unloaded-hint"]').text()).toContain('加载更早评论后可查看前序')
  })

  it('emits focus-predecessor on row click', async () => {
    const wrapper = mount(TaskDetailCommentPredecessorList, {
      props: { predecessors: SAMPLE },
    })
    await wrapper.get('[data-testid="comment-execution-predecessor-row"]').trigger('click')
    expect(wrapper.emitted('focus-predecessor')[0]).toEqual(['C1'])
  })
})
}

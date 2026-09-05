// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] TaskDetailCommentExecutionDetails.server-content-tab.test.js requires vitest runtime')
} else {
const { describe, expect, it } = await import('vitest')
const { mount } = await import('@vue/test-utils')
const { default: TaskDetailCommentExecutionDetails } = await import('./TaskDetailCommentExecutionDetails.vue')

describe('TaskDetailCommentExecutionDetails 服务器内容 Tab', () => {
  it('shows 服务器内容 tab beside 服务器运行状态 when runtime tab is enabled', () => {
    const wrapper = mount(TaskDetailCommentExecutionDetails, {
      props: {
        commentId: 'C1',
        dependencyMode: 'wait_previous',
        isActive: true,
        defaultOpen: true,
        serverRuntimeStatusTab: true,
      },
      slots: {
        default: '<div data-testid="slot-body">panel</div>',
        'server-runtime-status': '<div data-testid="slot-runtime">runtime</div>',
        'server-content': '<div data-testid="slot-content">content</div>',
      },
    })
    const tabs = wrapper.get('[data-testid="comment-execution-tablist"]')
    expect(tabs.get('[data-testid="comment-execution-tab-details"]').text()).toContain('执行细节')
    expect(tabs.get('[data-testid="comment-execution-tab-server-runtime"]').text()).toContain('服务器运行状态')
    expect(tabs.get('[data-testid="comment-execution-tab-server-content"]').text()).toContain('服务器内容')
    expect(wrapper.find('[data-testid="comment-execution-panel-server-content"]').exists()).toBe(false)
  })

  it('switches to comment-level server content panel on tab click', async () => {
    const wrapper = mount(TaskDetailCommentExecutionDetails, {
      props: {
        commentId: 'C1',
        dependencyMode: 'wait_previous',
        isActive: true,
        defaultOpen: true,
        serverRuntimeStatusTab: true,
      },
      slots: {
        default: '<div data-testid="slot-body">panel</div>',
        'server-runtime-status': '<div data-testid="slot-runtime">runtime</div>',
        'server-content': '<div data-testid="slot-content">content-C1</div>',
      },
    })
    await wrapper.get('[data-testid="comment-execution-tab-server-content"]').trigger('click')
    expect(wrapper.find('[data-testid="comment-execution-panel-details"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="comment-execution-panel-server-runtime"]').exists()).toBe(false)
    expect(
      wrapper.get('[data-testid="comment-execution-panel-server-content"]').get('[data-testid="slot-content"]').text(),
    ).toBe('content-C1')
  })

  it('hides 服务器内容 tab when runtime tab is disabled', () => {
    const wrapper = mount(TaskDetailCommentExecutionDetails, {
      props: {
        commentId: 'C1',
        dependencyMode: 'wait_previous',
        isActive: true,
        defaultOpen: true,
      },
      slots: { default: '<div data-testid="slot-body">panel</div>' },
    })
    expect(wrapper.find('[data-testid="comment-execution-tab-server-content"]').exists()).toBe(false)
  })
})
}

// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] TaskDetailCommentExecutionDetails.ztree-tab.test.js requires vitest runtime')
} else {
const { describe, expect, it } = await import('vitest')
const { mount } = await import('@vue/test-utils')
const { default: TaskDetailCommentExecutionDetails } = await import('./TaskDetailCommentExecutionDetails.vue')

describe('TaskDetailCommentExecutionDetails ztree tab', () => {
  function mountZtreeTabs(extraProps = {}) {
    return mount(TaskDetailCommentExecutionDetails, {
      props: {
        commentId: 'cmt99',
        taskId: 'task42',
        dependencyMode: 'independent',
        isActive: true,
        bindingStatus: 'running',
        containerName: 'task_task42_cmt99',
        cscId: 'csc_abc',
        startTraceId: 'web-start-trace-001',
        defaultOpen: true,
        layerZtreeTab: true,
        serverRuntimeStatusTab: true,
        ...extraProps,
      },
      slots: {
        default: '<div data-testid="slot-body">panel</div>',
        'layer-ztree': '<div data-testid="slot-ztree">ztree panel</div>',
        'server-runtime-status': '<div data-testid="slot-runtime">runtime panel</div>',
      },
    })
  }

  function expectContainerMeta(wrapper) {
    const meta = wrapper.get('[data-testid="comment-execution-container-meta"]')
    expect(meta.get('[data-testid="comment-execution-container-name-full"]').text()).toBe('task_task42_cmt99')
    expect(meta.get('[data-testid="comment-execution-csc-id"]').text()).toBe('csc_abc')
    const row = meta.get('[data-testid="comment-execution-start-trace-id"]')
    expect(row.text()).toContain('启动 TraceId')
    expect(row.get('[data-testid="comment-execution-start-trace-id-value"]').text()).toBe('web-start-trace-001')
  }

  it('shows 任务关联 as the first tab and defaults to ztree panel', () => {
    const wrapper = mountZtreeTabs()
    const tabs = wrapper.get('[data-testid="comment-execution-tablist"]').findAll('[role="tab"]')
    expect(tabs).toHaveLength(4)
    expect(tabs[0].attributes('data-testid')).toBe('comment-execution-tab-ztree')
    expect(tabs[0].text()).toContain('任务关联')
    expect(tabs[1].attributes('data-testid')).toBe('comment-execution-tab-details')
    expect(tabs[2].attributes('data-testid')).toBe('comment-execution-tab-server-runtime')
    expect(tabs[3].attributes('data-testid')).toBe('comment-execution-tab-server-content')
    expect(wrapper.get('[data-testid="comment-execution-panel-ztree"]').get('[data-testid="slot-ztree"]').text()).toBe('ztree panel')
    expect(wrapper.find('[data-testid="comment-execution-panel-details"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="comment-execution-panel-server-runtime"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="comment-execution-panel-server-content"]').exists()).toBe(false)
    expectContainerMeta(wrapper)
  })

  it('switches from ztree tab to details and back', async () => {
    const wrapper = mountZtreeTabs()
    await wrapper.get('[data-testid="comment-execution-tab-details"]').trigger('click')
    // OPT-20260816-036：ztree 面板用 v-show 保留挂载（树展开/滚动不丢失），只是隐藏。
    // jsdom 无布局引擎，isVisible() 恒 false，改断言内联 style.display + 槽位仍挂载。
    const ztreePanel = wrapper.get('[data-testid="comment-execution-panel-ztree"]')
    expect(ztreePanel.element.style.display).toBe('none')
    expect(ztreePanel.get('[data-testid="slot-ztree"]').text()).toBe('ztree panel')
    expect(wrapper.get('[data-testid="comment-execution-panel-details"]').get('[data-testid="slot-body"]').text()).toBe('panel')
    await wrapper.get('[data-testid="comment-execution-tab-ztree"]').trigger('click')
    expect(wrapper.get('[data-testid="comment-execution-panel-ztree"]').element.style.display).not.toBe('none')
    expect(wrapper.get('[data-testid="comment-execution-panel-ztree"]').get('[data-testid="slot-ztree"]').text()).toBe('ztree panel')
    expect(wrapper.find('[data-testid="comment-execution-panel-details"]').exists()).toBe(false)
  })

  it('hides ztree on server runtime tab and keeps container meta', async () => {
    const wrapper = mountZtreeTabs()
    await wrapper.get('[data-testid="comment-execution-tab-server-runtime"]').trigger('click')
    const ztreePanel = wrapper.get('[data-testid="comment-execution-panel-ztree"]')
    expect(ztreePanel.element.style.display).toBe('none')
    expect(ztreePanel.get('[data-testid="slot-ztree"]').text()).toBe('ztree panel')
    expect(wrapper.get('[data-testid="comment-execution-panel-server-runtime"]').get('[data-testid="slot-runtime"]').text()).toBe('runtime panel')
    expectContainerMeta(wrapper)
  })

  it('shows tab bar with only 任务关联 and 执行细节 when runtime tab disabled', () => {
    const wrapper = mountZtreeTabs({ serverRuntimeStatusTab: false })
    const tabs = wrapper.get('[data-testid="comment-execution-tablist"]').findAll('[role="tab"]')
    expect(tabs).toHaveLength(2)
    expect(tabs[0].text()).toContain('任务关联')
    expect(tabs[1].text()).toContain('执行细节')
    expect(wrapper.find('[data-testid="comment-execution-tab-server-runtime"]').exists()).toBe(false)
    expect(wrapper.get('[data-testid="comment-execution-panel-ztree"]').exists()).toBe(true)
  })

  it('keeps layer-ztree slot inside details panel when ztree tab disabled', () => {
    const wrapper = mount(TaskDetailCommentExecutionDetails, {
      props: {
        commentId: 'C1',
        dependencyMode: 'wait_previous',
        isActive: true,
        defaultOpen: true,
      },
      slots: {
        default: '<div data-testid="slot-body">panel</div>',
        'layer-ztree': '<div data-testid="slot-ztree">ztree inline</div>',
      },
    })
    expect(wrapper.find('[data-testid="comment-execution-tablist"]').exists()).toBe(false)
    expect(wrapper.get('[data-testid="comment-execution-panel-details"]').get('[data-testid="slot-ztree"]').text()).toBe('ztree inline')
    expect(wrapper.find('[data-testid="comment-execution-panel-ztree"]').exists()).toBe(false)
  })

  it('auto-switches to ztree when layerZtreeTab arrives late and user has not picked a tab', async () => {
    // OPT-20260820-003：bindings 晚到 → layerZtreeTab 晚变 true → 自动切「任务关联」。
    const wrapper = mountZtreeTabs({ layerZtreeTab: false, serverRuntimeStatusTab: false })
    expect(wrapper.find('[data-testid="comment-execution-tablist"]').exists()).toBe(false)
    expect(wrapper.get('[data-testid="comment-execution-panel-details"]').get('[data-testid="slot-body"]').text()).toBe('panel')
    await wrapper.setProps({ layerZtreeTab: true })
    expect(wrapper.get('[data-testid="comment-execution-tablist"]').exists()).toBe(true)
    const ztreePanel = wrapper.get('[data-testid="comment-execution-panel-ztree"]')
    expect(ztreePanel.element.style.display).not.toBe('none')
    expect(ztreePanel.get('[data-testid="slot-ztree"]').text()).toBe('ztree panel')
    expect(wrapper.find('[data-testid="comment-execution-panel-details"]').exists()).toBe(false)
  })

  it('does not override user-picked tab when layerZtreeTab arrives late', async () => {
    const wrapper = mountZtreeTabs({ layerZtreeTab: false, serverRuntimeStatusTab: true })
    expect(wrapper.get('[data-testid="comment-execution-tablist"]').exists()).toBe(true)
    await wrapper.get('[data-testid="comment-execution-tab-server-runtime"]').trigger('click')
    await wrapper.setProps({ layerZtreeTab: true })
    const ztreePanel = wrapper.get('[data-testid="comment-execution-panel-ztree"]')
    expect(ztreePanel.element.style.display).toBe('none')
    expect(wrapper.get('[data-testid="comment-execution-panel-server-runtime"]').get('[data-testid="slot-runtime"]').text()).toBe('runtime panel')
  })
})
}

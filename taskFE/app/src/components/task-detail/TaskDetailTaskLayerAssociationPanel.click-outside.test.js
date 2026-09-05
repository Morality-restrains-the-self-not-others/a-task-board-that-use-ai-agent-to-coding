// @vitest-environment jsdom
if (!process.env.VITEST) {
  // pre-commit 会以 node 直跑已暂存 *.test.js，非 Vitest 环境下跳过。
  console.log('[skip] TaskDetailTaskLayerAssociationPanel.click-outside.test.js requires vitest runtime')
} else {
  const { describe, expect, it, vi } = await import('vitest')
  const { nextTick } = await import('vue')
  const { mount } = await import('@vue/test-utils')
  const { default: TaskDetailTaskLayerAssociationPanel } = await import('./TaskDetailTaskLayerAssociationPanel.vue')

const selectedNode = {
  id: 'layer-1',
  name: 'L1',
  nodeKind: 'layer',
  layerId: 'layer-1',
}

function mountPanel(overrides = {}) {
  return mount(TaskDetailTaskLayerAssociationPanel, {
    props: {
      layerGraphCommandKind: 'trae',
      layerGraphSelectedModel: '',
      layerGraphAutoIterationCount: '',
      layerGraphCommandText: '',
      'onUpdate:layerGraphCommandKind': vi.fn(),
      'onUpdate:layerGraphSelectedModel': vi.fn(),
      'onUpdate:layerGraphAutoIterationCount': vi.fn(),
      'onUpdate:layerGraphCommandText': vi.fn(),
      layerGraphRefreshing: false,
      layerGraphZNodes: [],
      layerGraphMetaLine: '',
      layerGraphBusyActionKey: '',
      selectedLayerGraphNode: selectedNode,
      selectedZTreeLayerChangesPanel: null,
      selectedLayerGraphFileTreeLayerId: '',
      layerChangesRefreshBusy: false,
      layerChangesRefreshEnabled: false,
      layerChangesRefreshError: '',
      layerChangesRefreshErrorTraceId: '',
      layerChangesGitStagedActionsBlocked: false,
      layerChangesGitCommitIdentityBlocked: false,
      containerPageUrl: '',
      containerPageLinkPendingReveal: false,
      containerEndpointRegistered: true,
      containerHttpUnreachable: false,
      projectFileTreeRefreshNonce: 0,
      layerGraphModelSelectDisabled: false,
      layerGraphModelOptions: [],
      layerGraphDefaultModel: '',
      layerGraphEditRunTargetJobId: '',
      layerGraphModelLoadError: '',
      layerGraphModelLoadErrorTraceId: '',
      layerGraphCmdError: '',
      layerGraphCmdErrorTraceId: '',
      layerGraphCmdSending: false,
      containerActionsBlocked: false,
      layerExecLogLoading: false,
      layerExecLogCopyable: false,
      layerExecLogCopyFeedback: '',
      layerExecLogTopError: '',
      zTreeLogTargets: {},
      layerCloneLogFetchError: '',
      layerCloneLogFetchErrorTraceId: '',
      layerCloneLogText: '',
      layerLiveOutputDisplay: '',
      layerJobLogFetchError: '',
      layerJobLogFetchErrorTraceId: '',
      layerJobExecutionPayload: null,
      layerJobCommandHead: '',
      layerJobOutputDisplay: '',
      layerAgentStepCopyFeedbackKey: '',
      layerAgentStepCards: [],
      ...overrides,
    },
    global: {
      stubs: {
        LayerGraphZtree: { template: '<div data-testid="stub-ztree"></div>' },
        TaskDetailExecLayerChanges: true,
        TaskDetailExecLogPanel: true,
        TaskDetailProjectFileTree: true,
      },
    },
    attachTo: document.body,
  })
}

describe('TaskDetailTaskLayerAssociationPanel click-outside', () => {
  it('点击面板外不取消选中，指令面板保持可见', async () => {
    const wrapper = mountPanel()
    await nextTick()
    expect(wrapper.find('#layer-graph-command-input').exists()).toBe(true)
    expect(wrapper.find('[data-testid="comment-layer-ztree-command-panel"]').exists()).toBe(true)

    document.body.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }))
    await nextTick()

    expect(wrapper.emitted('layer-graph-node-select')).toBeUndefined()
    expect(wrapper.find('[data-testid="comment-layer-ztree-command-panel"]').exists()).toBe(true)
    wrapper.unmount()
  })

  it('点击面板内（含指令输入框）不关闭选中', async () => {
    const wrapper = mountPanel()
    await nextTick()
    const input = wrapper.find('#layer-graph-command-input')
    expect(input.exists()).toBe(true)
    await input.trigger('click')
    await nextTick()
    expect(wrapper.emitted('layer-graph-node-select')).toBeUndefined()
    wrapper.unmount()
  })

  it('无选中节点时点击页面不发出 select 事件', async () => {
    const wrapper = mountPanel({ selectedLayerGraphNode: null })
    await nextTick()
    document.body.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }))
    await nextTick()
    expect(wrapper.emitted('layer-graph-node-select')).toBeUndefined()
    wrapper.unmount()
  })

  it('点击面板外仍关闭模型下拉，但不取消选中', async () => {
    const wrapper = mountPanel()
    await nextTick()
    const details = wrapper.find('details')
    expect(details.exists()).toBe(true)
    details.element.open = true
    await nextTick()
    expect(details.element.open).toBe(true)

    document.body.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }))
    await nextTick()

    expect(details.element.open).toBe(false)
    expect(wrapper.emitted('layer-graph-node-select')).toBeUndefined()
    expect(wrapper.find('[data-testid="comment-layer-ztree-command-panel"]').exists()).toBe(true)
    wrapper.unmount()
  })
})
}

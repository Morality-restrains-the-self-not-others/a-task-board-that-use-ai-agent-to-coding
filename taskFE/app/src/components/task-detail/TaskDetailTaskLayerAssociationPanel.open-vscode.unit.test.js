// @vitest-environment jsdom
/**
 * 回归：「打开容器开发页面」挂在任务关联区域（与「打开容器页面」同排），
 * 而非评论输入区。
 */
if (!process.env.VITEST) {
  console.log('[skip] TaskDetailTaskLayerAssociationPanel.open-vscode.unit.test.js requires vitest runtime')
} else {
  const { describe, expect, it, vi } = await import('vitest')
  const { nextTick } = await import('vue')
  const { mount } = await import('@vue/test-utils')
  const { default: TaskDetailTaskLayerAssociationPanel } = await import('./TaskDetailTaskLayerAssociationPanel.vue')

  const VSCODE_URL = 'http://203.0.113.10:18888/'

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
        layerGraphZNodes: [{ id: 'layer-1', name: 'L1', nodeKind: 'layer', layerId: 'layer-1' }],
        layerGraphMetaLine: '',
        layerGraphBusyActionKey: '',
        selectedLayerGraphNode: null,
        selectedZTreeLayerChangesPanel: null,
        selectedLayerGraphFileTreeLayerId: '',
        layerChangesRefreshBusy: false,
        layerChangesRefreshEnabled: false,
        layerChangesRefreshError: '',
        layerChangesRefreshErrorTraceId: '',
        layerChangesGitStagedActionsBlocked: false,
        layerChangesGitCommitIdentityBlocked: false,
        tenantId: 't1',
        workspaceId: 'w1',
        taskId: 'task1',
        containerPageUrl: 'http://203.0.113.10:8765/ui/tok',
        containerPageLinkPendingReveal: false,
        containerEndpointRegistered: true,
        containerHttpUnreachable: false,
        displayContainerVscodeUrl: VSCODE_URL,
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
    })
  }

  describe('TaskDetailTaskLayerAssociationPanel open-container-vscode', () => {
    it('在任务关联标题行渲染「打开容器开发页面」且带 vscode URL', async () => {
      const wrapper = mountPanel()
      await nextTick()

      const panel = wrapper.get('[data-testid="comment-layer-ztree-panel"]')
      const vscodeBtn = panel.find('#open-container-vscode-btn')
      expect(vscodeBtn.exists()).toBe(true)
      expect(vscodeBtn.text()).toContain('打开容器开发页面')
      expect(vscodeBtn.attributes('href')).toBe(VSCODE_URL)
      expect(vscodeBtn.attributes('target')).toBe('_blank')

      // 与「打开容器页面」同排（同 flex 容器）
      const headerRow = panel.find('.flex.flex-wrap.items-center.gap-2')
      expect(headerRow.find('#open-container-page-btn').exists()).toBe(true)
      expect(headerRow.find('#open-container-vscode-btn').exists()).toBe(true)

      wrapper.unmount()
    })

    it('无 vscode URL 时不渲染该按钮', async () => {
      const wrapper = mountPanel({ displayContainerVscodeUrl: '' })
      await nextTick()
      expect(wrapper.find('#open-container-vscode-btn').exists()).toBe(false)
      wrapper.unmount()
    })

    it('容器不可达时不渲染该按钮', async () => {
      const wrapper = mountPanel({ containerHttpUnreachable: true })
      await nextTick()
      expect(wrapper.find('#open-container-vscode-btn').exists()).toBe(false)
      wrapper.unmount()
    })
  })
}

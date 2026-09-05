// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] TaskDetailTaskLayerAssociationPanel.bootstrap-error.test.js requires vitest runtime')
} else {
  const { describe, expect, it, vi } = await import('vitest')
  const { nextTick } = await import('vue')
  const { mount } = await import('@vue/test-utils')
  const { default: TaskDetailTaskLayerAssociationPanel } = await import(
    './TaskDetailTaskLayerAssociationPanel.vue'
  )

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
        layerGraphZNodes: [
          { id: '__layer_graph_root__', nodeKind: 'virtual', name: 'root' },
          { id: '__layer__:boot', nodeKind: 'layer', bootstrapAnchor: true, name: '引导克隆失败' },
        ],
        layerGraphMetaLine: '可写层 1 个（串行 · 按创建时间旧→新）',
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
        containerPageUrl: 'https://container.example/app',
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
        containerReleased: false,
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
          LayerGraphZtree: { template: '<div class="layer-graph-ztree"></div>' },
          TaskDetailExecLayerChanges: true,
          TaskDetailExecLogPanel: true,
          TaskDetailProjectFileTree: true,
          TaskDetailLayerFilesTabs: true,
        },
      },
    })
  }

  describe('TaskDetailTaskLayerAssociationPanel bootstrap credentials error', () => {
    it('renders Git 授权未齐 inside comment-layer-ztree-panel', async () => {
      const wrapper = mountPanel({
        commentLayerZtreeLoadingIsError: true,
        commentLayerZtreeLoadingHint:
          '引导克隆失败：仓库 Git 授权未齐（http://115.29.110.74/example-user/somanyad.git）。请由任务创建者在创建或编辑任务时为全部仓库完成 OAuth 绑定后重试克隆。',
        commentLayerZtreeLoadingErrorTraceId: 'cdc9471ebc27e3aef6092497',
      })
      await nextTick()
      const panel = wrapper.get('[data-testid="comment-layer-ztree-panel"]')
      const err = panel.get('[data-testid="comment-layer-ztree-loading-error"]')
      expect(err.text()).toContain('Git 授权未齐')
      expect(err.text()).toContain('115.29.110.74')
      expect(err.attributes('data-traceid') || err.attributes('data-traceId')).toBe(
        'cdc9471ebc27e3aef6092497',
      )
    })

    it('does not show credentials banner when not in error', async () => {
      const wrapper = mountPanel()
      await nextTick()
      expect(wrapper.find('[data-testid="comment-layer-ztree-loading-error"]').exists()).toBe(false)
    })
  })
}

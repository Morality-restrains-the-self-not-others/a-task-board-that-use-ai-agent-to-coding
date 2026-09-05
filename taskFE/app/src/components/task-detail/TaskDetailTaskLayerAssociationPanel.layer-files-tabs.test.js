// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] TaskDetailTaskLayerAssociationPanel.layer-files-tabs.test.js requires vitest runtime')
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

  const changesPanel = {
    layer_id: 'layer-1',
    changes: [{ path: 'a.txt' }],
    displayCount: 4,
    truncated: false,
    has_more: false,
    detail: '',
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
        selectedZTreeLayerChangesPanel: changesPanel,
        selectedLayerGraphFileTreeLayerId: 'layer-1',
        layerChangesRefreshBusy: false,
        layerChangesRefreshEnabled: true,
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
          TaskDetailExecLayerChanges: {
            template: '<div data-testid="task-detail-layer-changes">layer changes</div>',
          },
          TaskDetailExecLogPanel: true,
          TaskDetailProjectFileTree: {
            template: '<div data-testid="comment-layer-ztree-project-file-tree">file tree</div>',
          },
        },
      },
    })
  }

  describe('TaskDetailTaskLayerAssociationPanel layer files tabs', () => {
    it('shows tabs when a layer is selected and defaults to file changes', async () => {
      const wrapper = mountPanel()
      await nextTick()
      expect(wrapper.find('[data-testid="layer-files-tablist"]').exists()).toBe(true)
      expect(wrapper.get('[data-testid="layer-files-tab-changes"]').attributes('aria-selected')).toBe('true')
      expect(wrapper.get('[data-testid="layer-files-tab-changes"]').text()).toContain('· 4')
      expect(wrapper.get('[data-testid="layer-files-panel-changes"]').element.style.display).not.toBe('none')
      expect(wrapper.get('[data-testid="task-detail-layer-changes"]').text()).toBe('layer changes')
      expect(wrapper.get('[data-testid="layer-files-panel-tree"]').element.style.display).toBe('none')
      expect(wrapper.get('[data-testid="comment-layer-ztree-project-file-tree"]').text()).toBe('file tree')
    })

    it('reveals the file tree after picking the tree tab', async () => {
      const wrapper = mountPanel()
      await nextTick()
      await wrapper.get('[data-testid="layer-files-tab-tree"]').trigger('click')
      await nextTick()
      expect(wrapper.get('[data-testid="layer-files-panel-tree"]').element.style.display).not.toBe('none')
      expect(wrapper.get('[data-testid="comment-layer-ztree-project-file-tree"]').text()).toBe('file tree')
      expect(wrapper.get('[data-testid="layer-files-panel-changes"]').element.style.display).toBe('none')
      expect(wrapper.get('[data-testid="task-detail-layer-changes"]').exists()).toBe(true)
    })

    it('does not render tabs when no file-tree layer is selected', async () => {
      const wrapper = mountPanel({
        selectedLayerGraphFileTreeLayerId: '',
        selectedZTreeLayerChangesPanel: null,
      })
      await nextTick()
      expect(wrapper.find('[data-testid="layer-files-tablist"]').exists()).toBe(false)
    })
  })
}

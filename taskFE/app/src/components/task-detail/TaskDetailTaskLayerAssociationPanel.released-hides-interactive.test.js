// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] TaskDetailTaskLayerAssociationPanel.released-hides-interactive.test.js requires vitest runtime')
} else {
  const { describe, expect, it, vi } = await import('vitest')
  const { nextTick } = await import('vue')
  const { mount } = await import('@vue/test-utils')
  const { default: TaskDetailTaskLayerAssociationPanel } = await import(
    './TaskDetailTaskLayerAssociationPanel.vue'
  )

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
        layerGraphZNodes: [{ id: 'layer-1', name: 'L1' }],
        layerGraphMetaLine: '',
        layerGraphBusyActionKey: '',
        selectedLayerGraphNode: selectedNode,
        selectedZTreeLayerChangesPanel: {
          layer_id: 'layer-1',
          changes: [{ path: 'a.txt' }],
          displayCount: 1,
        },
        selectedLayerGraphFileTreeLayerId: 'layer-1',
        layerChangesRefreshBusy: false,
        layerChangesRefreshEnabled: true,
        layerChangesRefreshError: '',
        layerChangesRefreshErrorTraceId: '',
        layerChangesGitStagedActionsBlocked: false,
        layerChangesGitCommitIdentityBlocked: false,
        containerPageUrl: 'https://container.example/app',
        containerPageLinkPendingReveal: false,
        containerEndpointRegistered: false,
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
        containerActionsBlocked: true,
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
          LayerGraphZtree: { template: '<div data-testid="stub-ztree"></div>' },
          TaskDetailExecLayerChanges: true,
          TaskDetailExecLogPanel: {
            template: '<div data-testid="comment-layer-ztree-exec-log-panel">exec log</div>',
          },
          TaskDetailProjectFileTree: {
            props: ['containerReleased'],
            template:
              '<div data-testid="comment-layer-ztree-project-file-tree">file tree {{ containerReleased }}</div>',
          },
        },
      },
    })
  }

  describe('TaskDetailTaskLayerAssociationPanel released hides interactive chrome', () => {
    it('keeps command, logs, file tabs and send-to-AI when container is not released', async () => {
      const wrapper = mountPanel()
      await nextTick()
      expect(wrapper.get('[data-testid="comment-layer-ztree-panel"]').exists()).toBe(true)
      expect(wrapper.get('[data-testid="comment-layer-ztree-command-panel"]').exists()).toBe(true)
      expect(wrapper.get('#layer-graph-command-input').exists()).toBe(true)
      expect(wrapper.get('#layer-graph-command-send-btn').exists()).toBe(true)
      expect(wrapper.get('[data-testid="comment-layer-ztree-exec-log-panel"]').exists()).toBe(true)
      expect(wrapper.get('[data-testid="layer-files-tablist"]').exists()).toBe(true)
      expect(wrapper.text()).toContain('发送给AI')
    })

    it('passes containerReleased to the file tree when not released（OPT-20260823-038）', async () => {
      const wrapper = mountPanel()
      await nextTick()
      const ft = wrapper.find('[data-testid="comment-layer-ztree-project-file-tree"]')
      expect(ft.exists()).toBe(true)
      expect(ft.text()).toContain('file tree false')
      // 释放态时文件树整个不渲染，容器文件树请求被短路
      const releasedWrapper = mountPanel({ containerReleased: true })
      await nextTick()
      expect(releasedWrapper.find('[data-testid="comment-layer-ztree-project-file-tree"]').exists()).toBe(false)
    })

    it('hides command/file tabs but keeps archived exec log when containerReleased', async () => {
      const wrapper = mountPanel({
        containerReleased: true,
        layerAgentStepCards: [{ key: 's1', title: '步骤 1 · ✅' }],
      })
      await nextTick()
      expect(wrapper.get('[data-testid="comment-layer-ztree-panel"]').exists()).toBe(true)
      expect(wrapper.get('[data-testid="stub-ztree"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="comment-layer-ztree-command-panel"]').exists()).toBe(false)
      expect(wrapper.find('#layer-graph-command-input').exists()).toBe(false)
      expect(wrapper.find('#layer-graph-command-send-btn').exists()).toBe(false)
      expect(wrapper.get('[data-testid="comment-layer-ztree-exec-log-panel"]').exists()).toBe(true)
      expect(wrapper.get('[data-testid="comment-layer-ztree-released-selected-node"]').exists()).toBe(true)
      expect(wrapper.text()).toContain('步骤来自归档')
      expect(wrapper.find('[data-testid="layer-files-tablist"]').exists()).toBe(false)
      expect(wrapper.text()).not.toContain('发送给AI')
      expect(wrapper.text()).not.toContain('容器业务端点尚未就绪')
      expect(wrapper.text()).not.toContain('层级推送')
      expect(wrapper.text()).not.toContain('刷新')
      const box = wrapper.get('[data-testid="comment-layer-ztree-released-selected-node"]').element
        .parentElement
      expect(box.className).toContain('p-3')
      expect(box.className).not.toContain('py-1.5')
    })
  })
}

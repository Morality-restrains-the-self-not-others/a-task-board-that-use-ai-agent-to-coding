// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] TaskDetailCommentLayerAssociationBody.released-hides-nodes.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const { mount } = await import('@vue/test-utils')
  const { default: TaskDetailCommentLayerAssociationBody } = await import(
    './TaskDetailCommentLayerAssociationBody.vue'
  )

  const selectedNode = {
    id: 'job-1',
    name: 'completed 2026-08-22T13:23:40.981Z',
    nodeKind: 'job',
    layerId: 'layer-1',
  }

  function mountBody(overrides = {}) {
    return mount(TaskDetailCommentLayerAssociationBody, {
      props: {
        layerGraphCommandKind: 'trae',
        layerGraphSelectedModel: '',
        layerGraphAutoIterationCount: '',
        layerGraphCommandText: '',
        showCommentLayerZtreeLoading: false,
        showCommentLayerZtreeReleased: true,
        commentLayerZtreeReleasedTitle: '服务器已释放，暂无已保存的层图',
        commentLayerZtreeReleasedBody: '历史评论仍可查看。',
        tenantId: 't1',
        workspaceId: 'ws1',
        taskId: 'task_1',
        commentId: 'cmt_1',
        containerPageUrl: '',
        containerPageLinkPendingReveal: false,
        containerHttpUnreachable: false,
        displayContainerVscodeUrl: '',
        layerGraphZNodes: [
          { id: 'layer-1', name: 'L1' },
          { id: 'job-1', name: selectedNode.name },
        ],
        layerGraphRefreshing: false,
        selectedLayerGraphNode: selectedNode,
        selectedZTreeLayerChangesPanel: { displayCount: 2, changes: [] },
        selectedLayerGraphFileTreeLayerId: 'layer-1',
        ...overrides,
      },
      global: {
        stubs: {
          TaskDetailTaskLayerAssociationPanel: {
            props: ['containerReleased'],
            template:
              '<div data-testid="comment-layer-ztree-panel" :data-container-released="containerReleased ? \'true\' : \'false\'" />',
          },
          TaskDetailProjectFileTree: true,
          TaskDetailExecLayerChanges: true,
          TaskDetailLayerFilesTabs: {
            template: '<div data-testid="layer-files-tablist" />',
          },
        },
      },
    })
  }

  describe('TaskDetailCommentLayerAssociationBody released hides nodes', () => {
    it('shows released empty state instead of ztree, file tabs, and selected node panel', () => {
      const wrapper = mountBody()
      expect(wrapper.get('[data-testid="comment-layer-ztree-released"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="comment-layer-ztree-panel"]').exists()).toBe(false)
      expect(wrapper.find('[data-testid="layer-files-tablist"]').exists()).toBe(false)
      expect(wrapper.find('[data-testid="comment-layer-ztree-command-panel"]').exists()).toBe(false)
    })

    it('keeps association panel when serving and not released', () => {
      const wrapper = mountBody({
        showCommentLayerZtreeReleased: false,
        showCommentLayerZtreeLoading: false,
      })
      expect(wrapper.find('[data-testid="comment-layer-ztree-released"]').exists()).toBe(false)
      expect(wrapper.get('[data-testid="comment-layer-ztree-panel"]').exists()).toBe(true)
      expect(wrapper.get('[data-testid="comment-layer-ztree-panel"]').attributes('data-container-released')).toBe('false')
    })

    it('forwards containerReleased so persisted ztree can hide interactive chrome', () => {
      const wrapper = mountBody({
        showCommentLayerZtreeReleased: false,
        showCommentLayerZtreeLoading: false,
        containerReleased: true,
      })
      expect(wrapper.find('[data-testid="comment-layer-ztree-released"]').exists()).toBe(false)
      expect(wrapper.get('[data-testid="comment-layer-ztree-panel"]').exists()).toBe(true)
      expect(wrapper.get('[data-testid="comment-layer-ztree-panel"]').attributes('data-container-released')).toBe('true')
    })
  })
}

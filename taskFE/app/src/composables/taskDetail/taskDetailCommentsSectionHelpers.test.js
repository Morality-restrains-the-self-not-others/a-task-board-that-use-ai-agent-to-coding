// @vitest-environment node
import { describe, expect, it } from 'vitest'
import {
  buildLayerBodyBindForComment,
  lastPushErrorForComment,
  resolveCommentLayerZtreeLoadingErrorTraceId,
  resolvePerCommentLayerZtreeUi,
} from './taskDetailCommentsSectionHelpers.js'

const pageLevelLoadingProps = {
  showCommentLayerZtreeLoading: true,
  showCommentLayerZtreeReleased: false,
  commentLayerZtreeLoadingHint: 'page-level hint',
  commentLayerZtreeLoadingIsError: false,
  commentLayerZtreeReleasedTitle: 'page title',
  commentLayerZtreeReleasedBody: 'page body',
  containerHttpUnreachable: false,
  serverRuntimeNotServing: false,
  containerHeartbeatPaused: false,
  containerBootstrapFailureMessage: '',
  containerBootstrapFailureTraceId: '',
  containerLayerGraphAuthInvalid: false,
  buildPerBindingServerStatusProps: () => ({
    isServerRunning: true,
    isServerStarting: false,
    heartbeatStatus: 'connected',
    heartbeatSeqInfo: { probeOk: true, bidirectionalOk: true },
  }),
}

describe('resolvePerCommentLayerZtreeUi', () => {
  it('uses per-comment layer nodes + endpoint instead of page-level loading', () => {
    const ui = resolvePerCommentLayerZtreeUi(pageLevelLoadingProps, 'cmt_1', {
      layerGraphZNodes: [{ id: 'L1' }, { id: 'L2' }],
      containerEndpointRegistered: true,
      layerGraphRefreshing: false,
    })
    expect(ui.showCommentLayerZtreeLoading).toBe(false)
    expect(ui.showCommentLayerZtreeReleased).toBe(false)
  })

  it('shows loading when comment slot empty but binding is running and heartbeat connecting', () => {
    const ui = resolvePerCommentLayerZtreeUi(
      {
        ...pageLevelLoadingProps,
        showCommentLayerZtreeLoading: false,
        buildPerBindingServerStatusProps: () => ({
          isServerRunning: true,
          isServerStarting: false,
          heartbeatStatus: 'connecting',
          heartbeatSeqInfo: null,
        }),
      },
      'cmt_1',
      {
        layerGraphZNodes: [],
        containerEndpointRegistered: true,
        layerGraphRefreshing: false,
      },
    )
    expect(ui.showCommentLayerZtreeLoading).toBe(true)
    expect(ui.commentLayerZtreeLoadingHint).toContain('连接确认中')
  })

  it('uses per-binding heartbeat when endpoint is not registered', () => {
    const ui = resolvePerCommentLayerZtreeUi(
      {
        ...pageLevelLoadingProps,
        buildPerBindingServerStatusProps: () => ({
          isServerRunning: true,
          isServerStarting: false,
          heartbeatStatus: 'connected',
          heartbeatSeqInfo: { probeOk: true },
        }),
      },
      'cmt_2',
      {
        layerGraphZNodes: [],
        containerEndpointRegistered: false,
        layerGraphRefreshing: false,
      },
    )
    expect(ui.showCommentLayerZtreeLoading).toBe(true)
    expect(ui.commentLayerZtreeLoadingHint).toContain('register-reachability')
  })

  it('shows loading when serving with empty tree even if heartbeat is idle', () => {
    const ui = resolvePerCommentLayerZtreeUi(
      {
        ...pageLevelLoadingProps,
        buildPerBindingServerStatusProps: () => ({
          isServerRunning: true,
          isServerStarting: false,
          heartbeatStatus: 'idle',
          heartbeatSeqInfo: null,
        }),
      },
      'cmt_1',
      {
        layerGraphZNodes: [],
        containerEndpointRegistered: true,
        layerGraphRefreshing: false,
      },
    )
    expect(ui.showCommentLayerZtreeLoading).toBe(true)
    expect(ui.showCommentLayerZtreeReleased).toBe(false)
  })

  it('does not show loading when binding is not serving', () => {
    const ui = resolvePerCommentLayerZtreeUi(
      {
        ...pageLevelLoadingProps,
        isServerRunning: true,
        isServerStarting: true,
        buildPerBindingServerStatusProps: () => ({
          isServerRunning: false,
          isServerStarting: false,
          heartbeatStatus: 'connecting',
          heartbeatSeqInfo: null,
        }),
      },
      'cmt_idle',
      {
        layerGraphZNodes: [],
        containerEndpointRegistered: true,
        layerGraphRefreshing: false,
      },
    )
    expect(ui.showCommentLayerZtreeLoading).toBe(false)
  })

  it('passes bootstrap SSE trace_id only when showing clone-failure error', () => {
    const ui = resolvePerCommentLayerZtreeUi(
      {
        ...pageLevelLoadingProps,
        containerBootstrapFailureMessage:
          'repo-clone-credentials 未返回完整；缺失仓库(1): https://github.com/example/repo',
        containerBootstrapFailureTraceId: 'boot-sse-trace-1',
        buildPerBindingServerStatusProps: () => ({
          isServerRunning: true,
          isServerStarting: false,
          heartbeatStatus: 'connected',
          heartbeatSeqInfo: { probeOk: true },
        }),
      },
      'cmt_1',
      {
        layerGraphZNodes: [],
        containerEndpointRegistered: true,
        layerGraphRefreshing: false,
      },
    )
    expect(ui.commentLayerZtreeLoadingIsError).toBe(true)
    expect(ui.commentLayerZtreeLoadingErrorTraceId).toBe('boot-sse-trace-1')
  })

  it('keeps credentials-failure in panel when ztree only has empty pending anchor', () => {
    const ui = resolvePerCommentLayerZtreeUi(
      {
        ...pageLevelLoadingProps,
        containerBootstrapFailureMessage:
          'repo-clone-credentials 未返回完整；缺失仓库(1): http://115.29.110.74/example-user/somanyad.git',
        buildPerBindingServerStatusProps: () => ({
          isServerRunning: true,
          isServerStarting: false,
          heartbeatStatus: 'connected',
          heartbeatSeqInfo: { probeOk: true },
        }),
      },
      'cmt_1',
      {
        layerGraphZNodes: [
          { id: '__layer_graph_root__', nodeKind: 'virtual' },
          { id: '__layer__:boot', nodeKind: 'layer', bootstrapAnchor: true, name: '正在准备可写层' },
        ],
        containerEndpointRegistered: true,
        layerGraphRefreshing: false,
        layerGraphHasRealWritableLayer: false,
      },
    )
    expect(ui.showCommentLayerZtreeLoading).toBe(false)
    expect(ui.commentLayerZtreeLoadingIsError).toBe(true)
    expect(ui.commentLayerZtreeLoadingHint).toContain('Git 授权未齐')
    expect(ui.commentLayerZtreeLoadingHint).toContain('115.29.110.74')
  })

  it('treats bootstrap clone-log credentials failure as error hint', () => {
    const ui = resolvePerCommentLayerZtreeUi(
      {
        ...pageLevelLoadingProps,
        bootstrapCloneLogText: [
          '【项目克隆】引导失败（phase=task_detail_or_credentials code=REPO_CLONE_CREDENTIALS_INCOMPLETE）。',
          'repo-clone-credentials 未返回完整 repo_clone_credentials；缺失仓库(1): https://github.com/ruandao/somanyad',
        ].join('\n'),
        buildPerBindingServerStatusProps: () => ({
          isServerRunning: true,
          isServerStarting: false,
          heartbeatStatus: 'connected',
          heartbeatSeqInfo: { probeOk: true },
        }),
      },
      'cmt_1',
      {
        layerGraphZNodes: [],
        containerEndpointRegistered: true,
        layerGraphRefreshing: false,
      },
    )
    expect(ui.showCommentLayerZtreeLoading).toBe(true)
    expect(ui.commentLayerZtreeLoadingIsError).toBe(true)
    expect(ui.commentLayerZtreeLoadingHint).toContain('Git 授权未齐')
    expect(ui.commentLayerZtreeLoadingHint).not.toContain('等待可写层就绪')
  })

  it('keeps persisted layer nodes when runtime is Released even if binding still running', () => {
    const ui = resolvePerCommentLayerZtreeUi(
      {
        ...pageLevelLoadingProps,
        serverRuntimeNotServing: false,
        containerHeartbeatPaused: false,
        bindingStatusFor: () => 'running',
        serverRuntimeStatusPanel: {
          snapshotForComment: () => ({ serverRuntimeStatus: 'Released' }),
        },
        buildPerBindingServerStatusProps: () => ({
          isServerRunning: true,
          isServerStarting: false,
          heartbeatStatus: 'connected',
          heartbeatSeqInfo: { probeOk: true },
        }),
      },
      'cmt_1',
      {
        layerGraphZNodes: [
          { id: 'L1' },
          { id: 'job-1' },
        ],
        containerEndpointRegistered: true,
        layerGraphRefreshing: false,
      },
    )
    expect(ui.showCommentLayerZtreeLoading).toBe(false)
    expect(ui.showCommentLayerZtreeReleased).toBe(false)
  })

  it('keeps layer tree when runtime is still Running', () => {
    const ui = resolvePerCommentLayerZtreeUi(
      {
        ...pageLevelLoadingProps,
        bindingStatusFor: () => 'running',
        serverRuntimeStatusPanel: {
          snapshotForComment: () => ({ serverRuntimeStatus: 'Running' }),
        },
        buildPerBindingServerStatusProps: () => ({
          isServerRunning: true,
          isServerStarting: false,
          heartbeatStatus: 'connected',
          heartbeatSeqInfo: { probeOk: true },
        }),
      },
      'cmt_1',
      {
        layerGraphZNodes: [{ id: 'L1' }],
        containerEndpointRegistered: true,
        layerGraphRefreshing: false,
      },
    )
    expect(ui.showCommentLayerZtreeReleased).toBe(false)
    expect(ui.showCommentLayerZtreeLoading).toBe(false)
  })

  it('omits error trace_id when auth-invalid takes priority', () => {
    expect(
      resolveCommentLayerZtreeLoadingErrorTraceId({
        containerLayerGraphAuthInvalid: true,
        containerBootstrapFailureMessage: 'repo-clone-credentials 未返回完整',
        containerBootstrapFailureTraceId: 'boot-sse-trace-1',
      }),
    ).toBe('')
  })
})

describe('buildLayerBodyBindForComment', () => {
  it('overrides page-level ztree loading flags with per-comment ui', () => {
    const bind = buildLayerBodyBindForComment(
      {
        ...pageLevelLoadingProps,
        tenantId: 't1',
        workspaceId: 'ws1',
        taskId: 'task_1',
        layerPanelByCommentId: {
          cmt_1: {
            snapshot: {
              layers: [{ layer_id: 'L1', git_worktree_dirty: false }],
              jobs: [],
              layers_root: '/layers',
              bootstrap_layer_id: '',
            },
            containerEndpointRegistered: true,
          },
        },
      },
      'cmt_1',
    )
    expect(bind.layerGraphZNodes.length).toBeGreaterThan(0)
    expect(bind.showCommentLayerZtreeLoading).toBe(false)
    expect(bind.commentLayerZtreeLoadingHint).not.toBe('page-level hint')
  })
})

describe('lastPushErrorForComment', () => {
  it('reads last_push_error from the comment layer snapshot and prefers permission denied', () => {
    const permission = 'remote: Permission to ruandao/helloworld.git denied to alice.'
    expect(
      lastPushErrorForComment(
        {
          cmt_1: {
            snapshot: {
              layers: [
                { git_remote: { last_push_error: permission } },
                { git_remote: { last_push_error: 'network unreachable' } },
              ],
            },
          },
        },
        'cmt_1',
      ),
    ).toBe(permission)
  })

  it('returns empty when the comment slot has no snapshot error', () => {
    expect(lastPushErrorForComment({}, 'cmt_1')).toBe('')
    expect(lastPushErrorForComment({ cmt_1: { snapshot: { layers: [] } } }, 'cmt_1')).toBe('')
  })
})

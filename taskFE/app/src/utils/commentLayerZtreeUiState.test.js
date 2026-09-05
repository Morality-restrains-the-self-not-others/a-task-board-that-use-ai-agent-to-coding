// @vitest-environment node
import { describe, expect, it } from 'vitest'
import {
  isServerNotServingUi,
  isServerRuntimeNotServingStatus,
  shouldShowCommentLayerZtreeLoading,
  shouldShowCommentLayerZtreeReleased,
  resolveCommentLayerZtreeLoadingHint,
  formatContainerBootstrapFailureHint,
  isCommentLayerZtreeLoadingError,
  bootstrapFailureMessageFromCloneLogPayload,
} from './commentLayerZtreeUiState.js'

describe('commentLayerZtreeUiState', () => {
  it('detects runtime not-serving statuses', () => {
    expect(isServerRuntimeNotServingStatus('released')).toBe(true)
    expect(isServerRuntimeNotServingStatus('STOPPED')).toBe(true)
    expect(isServerRuntimeNotServingStatus('running')).toBe(false)
    expect(isServerRuntimeNotServingStatus('')).toBe(false)
  })

  it('treats idle machine as not-serving ui unless starting/running', () => {
    expect(isServerNotServingUi({ isServerRunning: false, isServerStarting: false })).toBe(true)
    expect(isServerNotServingUi({ isServerRunning: true })).toBe(false)
    expect(isServerNotServingUi({ isServerStarting: true })).toBe(false)
  })

  it('S1: not-serving + connecting does not show loading spinner', () => {
    expect(
      shouldShowCommentLayerZtreeLoading({
        layerGraphNodeCount: 0,
        isServerRunning: false,
        isServerStarting: false,
        containerHeartbeatStatus: 'connecting',
        containerEndpointRegistered: true,
      }),
    ).toBe(false)
  })

  it('S6: running + connecting still shows loading', () => {
    expect(
      shouldShowCommentLayerZtreeLoading({
        layerGraphNodeCount: 0,
        isServerRunning: true,
        containerHeartbeatStatus: 'connecting',
        containerEndpointRegistered: true,
      }),
    ).toBe(true)
    expect(
      resolveCommentLayerZtreeLoadingHint({
        containerHeartbeatStatus: 'connecting',
        containerEndpointRegistered: true,
      }),
    ).toBe('容器连接确认中，即将拉取可写层…')
  })

  it('server running without endpoint shows loading + registration hint', () => {
    expect(
      shouldShowCommentLayerZtreeLoading({
        layerGraphNodeCount: 0,
        isServerRunning: true,
        containerEndpointRegistered: false,
        containerHeartbeatStatus: 'idle',
      }),
    ).toBe(true)
    expect(
      resolveCommentLayerZtreeLoadingHint({
        containerEndpointRegistered: false,
        containerHeartbeatStatus: 'idle',
      }),
    ).toContain('register-reachability')
  })

  it('running + registered + idle heartbeat still shows loading (no blank ztree)', () => {
    expect(
      shouldShowCommentLayerZtreeLoading({
        layerGraphNodeCount: 0,
        isServerRunning: true,
        containerEndpointRegistered: true,
        containerHeartbeatStatus: 'idle',
      }),
    ).toBe(true)
    expect(
      resolveCommentLayerZtreeLoadingHint({
        containerEndpointRegistered: true,
        containerHeartbeatStatus: 'idle',
      }),
    ).toContain('等待可写层')
  })

  it('S2: released empty state for stuck connecting while not serving', () => {
    expect(
      shouldShowCommentLayerZtreeReleased({
        layerGraphNodeCount: 0,
        showLoading: false,
        isServerRunning: false,
        isServerStarting: false,
        containerHeartbeatStatus: 'connecting',
      }),
    ).toBe(true)
  })

  it('never-started idle does not show released banner', () => {
    expect(
      shouldShowCommentLayerZtreeReleased({
        layerGraphNodeCount: 0,
        showLoading: false,
        isServerRunning: false,
        isServerStarting: false,
        containerHeartbeatStatus: 'idle',
        containerHeartbeatPaused: false,
        serverRuntimeNotServing: false,
      }),
    ).toBe(false)
  })

  it('paused or runtime not-serving shows released banner', () => {
    expect(
      shouldShowCommentLayerZtreeReleased({
        layerGraphNodeCount: 0,
        showLoading: false,
        isServerRunning: false,
        containerHeartbeatPaused: true,
        containerHeartbeatStatus: 'idle',
      }),
    ).toBe(true)
    expect(
      shouldShowCommentLayerZtreeReleased({
        layerGraphNodeCount: 0,
        showLoading: false,
        isServerRunning: false,
        serverRuntimeNotServing: true,
        containerHeartbeatStatus: 'idle',
      }),
    ).toBe(true)
  })

  it('keeps comment-layer-ztree-panel when credentials fail and only empty pending nodes exist', () => {
    const raw =
      'repo-clone-credentials 未返回完整 repo_clone_credentials；请在任务详情为全部仓库绑定 Git 授权后重试。 缺失仓库(1): https://github.com/ruandao/somanyad detail=repo clone credentials incomplete'
    expect(
      shouldShowCommentLayerZtreeLoading({
        layerGraphNodeCount: 2,
        layerGraphHasRealWritableLayer: false,
        isServerRunning: true,
        containerEndpointRegistered: true,
        containerHeartbeatStatus: 'connected',
        containerBootstrapFailureMessage: raw,
      }),
    ).toBe(false)
    expect(isCommentLayerZtreeLoadingError({ containerBootstrapFailureMessage: raw })).toBe(true)
  })

  it('has layers while serving → neither loading nor released', () => {
    expect(
      shouldShowCommentLayerZtreeLoading({
        layerGraphNodeCount: 2,
        isServerRunning: true,
        containerHeartbeatStatus: 'connecting',
      }),
    ).toBe(false)
    expect(
      shouldShowCommentLayerZtreeReleased({
        layerGraphNodeCount: 2,
        showLoading: false,
        isServerRunning: true,
        isServerStarting: false,
        containerHeartbeatPaused: false,
        serverRuntimeNotServing: false,
      }),
    ).toBe(false)
  })

  it('persisted layers after not-serving still show ztree for review', () => {
    expect(
      shouldShowCommentLayerZtreeLoading({
        layerGraphNodeCount: 2,
        isServerRunning: false,
        isServerStarting: false,
        executionReleased: false,
      }),
    ).toBe(false)
    expect(
      shouldShowCommentLayerZtreeReleased({
        layerGraphNodeCount: 2,
        showLoading: false,
        isServerRunning: false,
        isServerStarting: false,
        containerHeartbeatPaused: true,
        containerHeartbeatStatus: 'idle',
      }),
    ).toBe(false)
  })

  it('executionReleased with persisted nodes still shows ztree', () => {
    expect(
      shouldShowCommentLayerZtreeLoading({
        layerGraphNodeCount: 3,
        isServerRunning: true,
        executionReleased: true,
      }),
    ).toBe(false)
    expect(
      shouldShowCommentLayerZtreeReleased({
        layerGraphNodeCount: 3,
        showLoading: false,
        isServerRunning: true,
        executionReleased: true,
      }),
    ).toBe(false)
  })

  it('executionReleased without nodes still shows released empty state', () => {
    expect(
      shouldShowCommentLayerZtreeReleased({
        layerGraphNodeCount: 0,
        showLoading: false,
        isServerRunning: true,
        executionReleased: true,
      }),
    ).toBe(true)
  })

  it('surfaces repo-clone-credentials bootstrap failure instead of waiting spinner copy', () => {
    const raw =
      'repo-clone-credentials 未返回完整 repo_clone_credentials；请在任务详情为全部仓库绑定 Git 授权后重试。 缺失仓库(1): https://github.com/ruandao/somanyad detail=repo clone credentials incomplete'
    expect(formatContainerBootstrapFailureHint(raw)).toContain('Git 授权未齐')
    expect(formatContainerBootstrapFailureHint(raw)).toContain('ruandao/somanyad')
    expect(
      resolveCommentLayerZtreeLoadingHint({
        containerEndpointRegistered: true,
        containerHeartbeatStatus: 'connected',
        containerBootstrapFailureMessage: raw,
      }),
    ).toContain('创建或编辑任务')
    expect(
      resolveCommentLayerZtreeLoadingHint({
        containerEndpointRegistered: true,
        containerHeartbeatStatus: 'connected',
        containerBootstrapFailureMessage: raw,
      }),
    ).not.toContain('添加评论')
    expect(
      isCommentLayerZtreeLoadingError({ containerBootstrapFailureMessage: raw }),
    ).toBe(true)
  })

  it('surfaces 409 outbound JSON and structured clone-log payload as Git 授权未齐', () => {
    const outbound =
      '2026-08-27T07:41:17.241Z | postJson POST https://api.daydaymoney.com/api/tenant/t/workspace/w/task/task_1/comment/cmt_1/cloud/server-container-token/repo-clone-credentials/ -> error HTTP 409 https://api.daydaymoney.com/api/tenant/t/workspace/w/task/task_1/comment/cmt_1/cloud/server-container-token/repo-clone-credentials/: {"detail":"repo clone credentials incomplete","error_code":"REPO_CLONE_CREDENTIALS_INCOMPLETE","missing_repo_credentials":["http://115.29.110.74/example-user/somanyad.git"]} 86ms'
    expect(formatContainerBootstrapFailureHint(outbound)).toContain('Git 授权未齐')
    expect(formatContainerBootstrapFailureHint(outbound)).toContain('115.29.110.74')
    expect(
      bootstrapFailureMessageFromCloneLogPayload({
        error_code: 'REPO_CLONE_CREDENTIALS_INCOMPLETE',
        missing_repo_credentials: ['http://115.29.110.74/example-user/somanyad.git'],
      }),
    ).toContain('115.29.110.74')
    expect(isCommentLayerZtreeLoadingError({ bootstrapCloneLogText: outbound })).toBe(true)
  })

  it('treats bootstrap-clone-log failure payload as error instead of waiting spinner', () => {
    const logText = [
      '【项目克隆】引导失败（phase=task_detail_or_credentials code=REPO_CLONE_CREDENTIALS_INCOMPLETE）。',
      'repo-clone-credentials 未返回完整 repo_clone_credentials；请在任务详情为全部仓库绑定 Git 授权后重试。',
    ].join('\n')
    expect(
      resolveCommentLayerZtreeLoadingHint({
        containerEndpointRegistered: true,
        containerHeartbeatStatus: 'connected',
        bootstrapCloneLogText: logText,
      }),
    ).toContain('Git 授权未齐')
    expect(
      isCommentLayerZtreeLoadingError({ bootstrapCloneLogText: logText }),
    ).toBe(true)
    expect(
      resolveCommentLayerZtreeLoadingHint({
        containerEndpointRegistered: true,
        containerHeartbeatStatus: 'connected',
      }),
    ).toContain('等待可写层就绪')
  })

  it('shows clone-in-progress copy from bootstrap clone log', () => {
    expect(
      resolveCommentLayerZtreeLoadingHint({
        containerEndpointRegistered: true,
        containerHeartbeatStatus: 'connected',
        bootstrapCloneLogText: '【项目克隆】正在并行克隆任务关联仓库（任务详情已拉取）…',
      }),
    ).toContain('正在克隆任务关联仓库')
  })

  it('when probe fails but bidirectional connected, hint mentions HTTP probe', () => {
    expect(
      resolveCommentLayerZtreeLoadingHint({
        containerEndpointRegistered: true,
        containerHeartbeatStatus: 'connected',
        containerHeartbeatProbeOk: false,
      }),
    ).toContain('HTTP 探测失败')
  })
})

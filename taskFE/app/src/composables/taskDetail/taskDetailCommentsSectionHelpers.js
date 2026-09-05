/**
 * TaskDetailCommentsSection 纯函数助手（削文件：从 .vue 抽出 props 派生逻辑）。
 */
import { trimCommentId } from '../../utils/cloudComputeCommentQuery.js'
import { collectLatestLayerPushError } from '../../utils/layerZtreePushError.js'
import {
  shouldShowCommentLayerZtreeLoading,
  shouldShowCommentLayerZtreeReleased,
  resolveCommentLayerZtreeLoadingHint,
  isCommentLayerZtreeLoadingError,
  formatContainerBootstrapFailureHint,
  COMMENT_LAYER_ZTREE_RELEASED_TITLE,
  COMMENT_LAYER_ZTREE_RELEASED_BODY,
} from '../../utils/commentLayerZtreeUiState.js'
import { resolveLayerGraphHasRealWritableLayer } from '../../utils/layerZtreeBootstrapAnchor.js'
import { layerPanelViewFromSlot } from './commentLayerPanelBind.js'
import { emptyCommentLayerPanelSlot } from './commentLayerPanelStore.js'
import { runtimeStatusFromCommentPanel } from './bindCommentRuntimePanel.js'
import { isCommentExecutionReleased } from './useCommentExecutionContext.js'

/** 仅引导克隆失败文案对应的 SSE trace_id；令牌无效等无请求 trace 时省略 */
export function resolveCommentLayerZtreeLoadingErrorTraceId(props = {}) {
  if (props.containerLayerGraphAuthInvalid) return ''
  if (!formatContainerBootstrapFailureHint(props.containerBootstrapFailureMessage)) return ''
  return String(props.containerBootstrapFailureTraceId || '').trim()
}

/** 该评论层图快照中最近一次 git push 失败原文（优先仓库写权限拒绝） */
export function lastPushErrorForComment(layerPanelByCommentId, commentId) {
  const cid = trimCommentId(commentId)
  const slot = cid ? layerPanelByCommentId?.[cid] : null
  return collectLatestLayerPushError(slot?.snapshot)
}

/** layer association body v-bind 快照 */
export function buildLayerBodyBind(props) {
  return {
    tenantId: props.tenantId,
    workspaceId: props.workspaceId,
    taskId: props.taskId,
    commentId: props.commentId || '',
    containerPageUrl: props.containerPageUrl,
    containerPageLinkPendingReveal: props.containerPageLinkPendingReveal,
    containerHttpUnreachable: props.containerHttpUnreachable,
    displayContainerVscodeUrl: props.displayContainerVscodeUrl,
    layerGraphZNodes: props.layerGraphZNodes,
    layerGraphRefreshing: props.layerGraphRefreshing,
    layerGraphMetaLine: props.layerGraphMetaLine,
    layerGraphBusyActionKey: props.layerGraphBusyActionKey,
    selectedLayerGraphNode: props.selectedLayerGraphNode,
    selectedZTreeLayerChangesPanel: props.selectedZTreeLayerChangesPanel,
    selectedLayerGraphFileTreeLayerId: props.selectedLayerGraphFileTreeLayerId,
    layerChangesRefreshBusy: props.layerChangesRefreshBusy,
    layerChangesRefreshEnabled: props.layerChangesRefreshEnabled,
    layerChangesRefreshError: props.layerChangesRefreshError,
    layerChangesRefreshErrorTraceId: props.layerChangesRefreshErrorTraceId,
    layerChangesGitStagedActionsBlocked: props.layerChangesGitStagedActionsBlocked,
    layerChangesGitCommitIdentityBlocked: props.layerChangesGitCommitIdentityBlocked,
    containerEndpointRegistered: props.containerEndpointRegistered,
    projectFileTreeRefreshNonce: props.projectFileTreeRefreshNonce,
    layerGraphModelSelectDisabled: props.layerGraphModelSelectDisabled,
    layerGraphModelOptions: props.layerGraphModelOptions,
    layerGraphDefaultModel: props.layerGraphDefaultModel,
    layerGraphEditRunTargetJobId: props.layerGraphEditRunTargetJobId,
    layerGraphModelLoadError: props.layerGraphModelLoadError,
    layerGraphModelLoadErrorTraceId: props.layerGraphModelLoadErrorTraceId,
    layerGraphCmdError: props.layerGraphCmdError,
    layerGraphCmdErrorTraceId: props.layerGraphCmdErrorTraceId,
    layerGraphCmdSending: props.layerGraphCmdSending,
    containerActionsBlocked: props.containerActionsBlocked,
    layerExecLogLoading: props.layerExecLogLoading,
    layerExecLogCopyable: props.layerExecLogCopyable,
    layerExecLogClearable: props.layerExecLogClearable,
    layerExecLogCopyFeedback: props.layerExecLogCopyFeedback,
    layerExecLogTopError: props.layerExecLogTopError,
    zTreeLogTargets: props.zTreeLogTargets,
    layerCloneLogFetchError: props.layerCloneLogFetchError,
    layerCloneLogFetchErrorTraceId: props.layerCloneLogFetchErrorTraceId,
    layerCloneLogText: props.layerCloneLogText,
    layerLiveOutputDisplay: props.layerLiveOutputDisplay,
    layerJobLogFetchError: props.layerJobLogFetchError,
    layerJobLogFetchErrorTraceId: props.layerJobLogFetchErrorTraceId,
    layerJobExecutionPayload: props.layerJobExecutionPayload,
    layerJobCommandHead: props.layerJobCommandHead,
    layerJobOutputDisplay: props.layerJobOutputDisplay,
    layerAgentStepCopyFeedbackKey: props.layerAgentStepCopyFeedbackKey,
    layerAgentStepCards: props.layerAgentStepCards,
  }
}

/**
 * 按评论槽 + per-binding 心跳重算 ztree 加载/释放态。
 * serving 只来自 buildPerBindingServerStatusProps，不读页面单例 isServerRunning。
 */
export function resolvePerCommentLayerZtreeUi(props, commentId, view = {}) {
  const cid = trimCommentId(commentId)
  const bindingProps =
    typeof props.buildPerBindingServerStatusProps === 'function'
      ? props.buildPerBindingServerStatusProps(cid)
      : null

  const isServerRunning = Boolean(bindingProps?.isServerRunning)
  const isServerStarting = Boolean(bindingProps?.isServerStarting)
  const heartbeatStatus = bindingProps?.heartbeatStatus || 'idle'
  const heartbeatSeqInfo = bindingProps?.heartbeatSeqInfo || null
  const layerGraphNodeCount = Array.isArray(view.layerGraphZNodes)
    ? view.layerGraphZNodes.length
    : 0
  const bindingStatus =
    typeof props.bindingStatusFor === 'function' ? props.bindingStatusFor(cid) : ''
  const runtimeStatus = runtimeStatusFromCommentPanel(props.serverRuntimeStatusPanel, cid)
  const executionReleased = isCommentExecutionReleased(bindingStatus, runtimeStatus)
  const layerGraphHasRealWritableLayer = resolveLayerGraphHasRealWritableLayer(view)
  const bootstrapCloneLogText = props.bootstrapCloneLogText || view.layerCloneLogText || ''
  const containerBootstrapFailureMessage =
    props.containerBootstrapFailureMessage || view.layerGraphBootstrapFailureMessage || ''

  const showCommentLayerZtreeLoading = shouldShowCommentLayerZtreeLoading({
    layerGraphNodeCount,
    layerGraphHasRealWritableLayer,
    isServerRunning,
    isServerStarting,
    containerHeartbeatStatus: heartbeatStatus,
    containerEndpointRegistered: Boolean(view.containerEndpointRegistered),
    containerHttpUnreachable: Boolean(props.containerHttpUnreachable),
    layerGraphRefreshing: Boolean(view.layerGraphRefreshing),
    executionReleased,
    containerLayerGraphAuthInvalid: Boolean(props.containerLayerGraphAuthInvalid),
    containerBootstrapFailureMessage,
    bootstrapCloneLogText,
  })

  const showCommentLayerZtreeReleased = shouldShowCommentLayerZtreeReleased({
    layerGraphNodeCount,
    showLoading: showCommentLayerZtreeLoading,
    isServerRunning,
    isServerStarting,
    serverRuntimeNotServing: Boolean(props.serverRuntimeNotServing),
    containerHeartbeatPaused: Boolean(props.containerHeartbeatPaused),
    containerHeartbeatStatus: heartbeatStatus,
    executionReleased,
  })

  return {
    showCommentLayerZtreeLoading,
    showCommentLayerZtreeReleased,
    commentLayerZtreeLoadingHint: resolveCommentLayerZtreeLoadingHint({
      containerLayerGraphAuthInvalid: Boolean(props.containerLayerGraphAuthInvalid),
      containerBootstrapFailureMessage,
      bootstrapCloneLogText,
      containerHttpUnreachable: Boolean(props.containerHttpUnreachable),
      containerEndpointRegistered: Boolean(view.containerEndpointRegistered),
      containerHeartbeatStatus: heartbeatStatus,
      containerHeartbeatProbeOk: heartbeatSeqInfo?.probeOk ?? null,
      layerGraphRefreshing: Boolean(view.layerGraphRefreshing),
    }),
    commentLayerZtreeLoadingIsError: isCommentLayerZtreeLoadingError({
      containerLayerGraphAuthInvalid: Boolean(props.containerLayerGraphAuthInvalid),
      containerBootstrapFailureMessage,
      bootstrapCloneLogText,
    }),
    commentLayerZtreeLoadingErrorTraceId: resolveCommentLayerZtreeLoadingErrorTraceId(props),
    commentLayerZtreeReleasedTitle: COMMENT_LAYER_ZTREE_RELEASED_TITLE,
    commentLayerZtreeReleasedBody: COMMENT_LAYER_ZTREE_RELEASED_BODY,
  }
}

/**
 * 按 comment_id 合并该评论的层图/端点/执行日志槽；无槽时回退页面级 props。
 */
export function buildLayerBodyBindForComment(props, commentId) {
  const cid = trimCommentId(commentId)
  const base = buildLayerBodyBind({ ...props, commentId: cid || props.commentId || '' })
  const raw = cid ? props.layerPanelByCommentId?.[cid] : null
  const slot = { ...emptyCommentLayerPanelSlot(), ...(raw || {}) }
  // OPT-20260822-001(3)：per-comment 容器已释放判定透传到 ztree 节点构造，
  // 使「提交并创建PR」等 git 动作按钮在无活容器时禁用并给出明确文案。
  const bindingStatus =
    typeof props.bindingStatusFor === 'function' ? props.bindingStatusFor(cid) : ''
  const runtimeStatus = runtimeStatusFromCommentPanel(props.serverRuntimeStatusPanel, cid)
  const containerReleased = isCommentExecutionReleased(bindingStatus, runtimeStatus)
  const view = layerPanelViewFromSlot(slot, {
    mergeTargetBranch: props.layerGraphMergeTargetBranch || '',
    displayContainerVscodeUrl: props.displayContainerVscodeUrl || '',
    containerReleased,
    bootstrapFailureRaw: props.containerBootstrapFailureMessage || '',
  })
  return {
    ...base,
    commentId: cid,
    ...view,
    ...resolvePerCommentLayerZtreeUi(props, cid, view),
  }
}

export function layerPanelCommandFields(props, commentId) {
  const cid = trimCommentId(commentId)
  const slot = cid ? props.layerPanelByCommentId?.[cid] : null
  const merged = { ...emptyCommentLayerPanelSlot(), ...(slot || {}) }
  return {
    layerGraphCommandKind: merged.commandKind || 'trae',
    layerGraphSelectedModel: merged.selectedModel || '',
    layerGraphAutoIterationCount: merged.autoIterationCount || '',
    layerGraphCommandText: merged.commandText || '',
  }
}

export function buildLayerPanelCommandListeners(emit, commentId) {
  const cid = trimCommentId(commentId)
  return {
    'update:layerGraphCommandKind': (v) => emit('patch-layer-panel', cid, { commandKind: v }),
    'update:layerGraphSelectedModel': (v) => emit('patch-layer-panel', cid, { selectedModel: v }),
    'update:layerGraphAutoIterationCount': (v) => emit('patch-layer-panel', cid, { autoIterationCount: v }),
    'update:layerGraphCommandText': (v) => emit('patch-layer-panel', cid, { commandText: v }),
  }
}

/** 单条 v-bind：层图槽 + 命令框。Vue 禁止同一节点写两个 v-bind。 */
export function buildPerCommentLayerBodyBind(props, commentId) {
  return {
    ...buildLayerBodyBindForComment(props, commentId),
    ...layerPanelCommandFields(props, commentId),
  }
}

/** CSS 属性值选择器转义；优先原生 CSS.escape，回退反斜杠/引号替换 */
export function cssAttrEscape(value) {
  if (typeof CSS !== 'undefined' && typeof CSS.escape === 'function') {
    return CSS.escape(value)
  }
  return String(value).replace(/\\/g, '\\\\').replace(/"/g, '\\"')
}

/** 点击前序行：展开并滚到该评论的执行细节（无新请求）；doc 参数便于单测注入假 document */
export function onFocusPredecessor(commentId, doc = typeof document !== 'undefined' ? document : null) {
  const id = String(commentId || '').trim()
  if (!id || !doc) return
  const root = doc.getElementById('comments-container') || doc
  const el = root.querySelector(
    `[data-testid="comment-execution-details"][data-comment-id="${cssAttrEscape(id)}"]`,
  )
  if (!el) return
  el.open = true
  el.scrollIntoView({ behavior: 'smooth', block: 'center' })
}

/**
 * 按 comment_id 滚到评论气泡并高亮（OPT-20260817-013 前端部分）。
 * 评论气泡由 TaskDetailConversationFeed 渲染并带 data-comment-id 属性；
 * 返回是否找到目标元素，便于调用方决定是否需要重试。doc 参数便于单测注入假 document。
 */
export function scrollToCommentById(commentId, doc = typeof document !== 'undefined' ? document : null) {
  const id = String(commentId || '').trim()
  if (!id || !doc) return false
  const root = doc.getElementById('comments-container') || doc
  const el = root.querySelector(`[data-comment-id="${cssAttrEscape(id)}"]`)
  if (!el) return false
  el.scrollIntoView({ behavior: 'smooth', block: 'center' })
  el.classList.add('comment-highlight')
  return true
}

/** 单条 v-on：动作转发 + 命令框 patch。 */
export function buildPerCommentLayerBodyListeners(emit, commentId) {
  return {
    ...buildLayerBodyListeners(emit, commentId),
    ...buildLayerPanelCommandListeners(emit, commentId),
  }
}

/** layer association body v-on 转发表 */
export function buildLayerBodyListeners(emit, commentId) {
  const cid = trimCommentId(commentId)
  return {
    'refresh-layer-graph': () => emit('refresh-layer-graph', cid),
    'layer-graph-node-select': (e) => emit('layer-graph-node-select', e, cid),
    'layer-graph-job-redo': (e) => emit('layer-graph-job-redo', e),
    'layer-graph-job-interrupt': (e) => emit('layer-graph-job-interrupt', e),
    'layer-graph-job-continue': (e) => emit('layer-graph-job-continue', e),
    'layer-graph-job-edit-run': (e) => emit('layer-graph-job-edit-run', e),
    'layer-graph-job-delete': (e) => emit('layer-graph-job-delete', e),
    'layer-graph-layer-delete': (e) => emit('layer-graph-layer-delete', e),
    'layer-graph-layer-submit': (e) => emit('layer-graph-layer-submit', e),
    'layer-graph-layer-push': (e) => emit('layer-graph-layer-push', e),
    'layer-graph-layer-submit-and-push': (e) => emit('layer-graph-layer-submit-and-push', e),
    'layer-graph-layer-submit-and-merge': (e) => emit('layer-graph-layer-submit-and-merge', e),
    'layer-graph-layer-merge': (e) => emit('layer-graph-layer-merge', e),
    'layer-changes-refresh': () => emit('layer-changes-refresh'),
    'layer-changes-staged-refresh': () => emit('layer-changes-staged-refresh'),
    'layer-changes-commit-staged': (e) => emit('layer-changes-commit-staged', e),
    'layer-changes-load-more': () => emit('layer-changes-load-more'),
    'submit-layer-graph-command': () => emit('submit-layer-graph-command', cid),
    'copy-layer-exec-log': () => emit('copy-layer-exec-log', cid),
    'clear-layer-exec-log': () => emit('clear-layer-exec-log', cid),
    'copy-agent-step-json': (e) => emit('copy-agent-step-json', e),
    'agent-step-rich-interact': (e) => emit('agent-step-rich-interact', e),
  }
}

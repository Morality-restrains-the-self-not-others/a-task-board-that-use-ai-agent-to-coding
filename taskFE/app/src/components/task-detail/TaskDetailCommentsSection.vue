<template>
  <TaskDetailCommentsPanel
    v-model:new-comment="newComment"
    :show-container-clone-progress-banner="showContainerCloneProgressBanner"
    :container-clone-progress-entries="containerCloneProgressEntries"
    :repo-clone-field-id="repoCloneFieldId"
    :short-clone-repo-label="shortCloneRepoLabel"
    :clone-progress-row-has-sub-phases="cloneProgressRowHasSubPhases"
    :clone-progress-recv-pct="cloneProgressRecvPct"
    :clone-progress-unpack-pct="cloneProgressUnpackPct"
    :clone-progress-bar-width-transition-class="cloneProgressBarWidthTransitionClass"
    :clone-progress-row-log-is-placeholder="cloneProgressRowLogIsPlaceholder"
    :clone-progress-row-log-display-text="cloneProgressRowLogDisplayText"
    :display-comments="feedDisplayComments"
    :collaborator-name-by-id="collaboratorNameById"
    :collaborator-avatar-by-id="collaboratorAvatarById"
    :comments-feed-errors="commentsFeedErrors"
    :comments-has-more="commentsHasMore"
    :comments-loading-more="commentsLoadingMore"
    :ai-stream-busy="aiStreamBusy"
    :ai-stream-buffer="aiStreamBuffer"
    :active-container-agent-id="activeContainerAgentId"
    :active-execution-comment-id="activeExecutionCommentId"
    :has-ztree-agent-steps="hasZtreeAgentSteps"
    :agent-step-count="agentStepCount"
    :tenant-id="tenantId"
    :workspace-id="workspaceId"
    :task-id="taskId"
    :task="task"
    :comment-composer-chips="commentComposerChips"
    :project-server-run-template="projectServerRunTemplate"
    :task-projects-with-details="taskProjectsWithDetails"
    :oauth-readiness="oauthReadinessForDetails"
    @remove-comment-composer-chip="emit('remove-comment-composer-chip', $event)"
    @comment-textarea-keydown="emit('comment-textarea-keydown', $event)"
    @submit-comment="emit('submit-comment')"
    @load-more-comments="emit('load-more-comments')"
    @task-updated="emit('task-updated', $event)"
  >
    <template #execution-details="{ comment, isActive, executionMode }">
      <TaskDetailCommentExecutionDetails
        v-if="String(comment?.id || '').trim()"
        :comment-id="String(comment.id)"
        :task-id="taskId"
        :dependency-mode="executionMode"
        :is-active="isActive"
        :default-open="isActive || commentHasLiveBinding(comment.id) || bindingStatusFor(comment.id) === 'waiting_previous'"
        :binding-status="bindingStatusFor(comment.id)"
        :server-runtime-status="runtimeStatusFromCommentPanel(serverRuntimeStatusPanel, comment.id)"
        :container-name="bindingContainerNameFor(comment.id)"
        :csc-id="bindingCscIdFor(comment.id)"
        :owns-shared-container="commentOwnsSharedContainer(comment.id)"
        :start-trace-id="bindingStartTraceIdFor(comment.id)"
        :clone-progress-rows="commentCloneProgressRows(comment.id)"
        :reclone-loading-by-url="recloneLoadingByUrl"
        :reclone-status-by-url="recloneStatusByUrl"
        :reclone-error-trace-id-by-url="recloneErrorTraceIdByUrl"
        :repo-reclone-global-loading="repoRecloneGlobalLoading"
        :can-edit-mode="false"
        :queued-auto-run="task?.queued_auto_run === true"
        :server-runtime-status-tab="shouldEnableCommentRuntimeTab(serverRuntimeStatusPanel)"
        :layer-ztree-tab="shouldMountCommentConnectionPanel(commentOwnsSharedContainer(comment.id))"
        :has-effective-predecessors="hasEffectivePredecessors(comment)"
        :predecessors="predecessorsFor(comment)"
        :cancel-waiting-busy="isCancelWaitingBusy(comment.id)"
        :repo-identities="Array.isArray(comment.repo_identities) ? comment.repo_identities : []"
        :git-identity-options="gitIdentityOptions"
        :fallback-repo-identities="taskRepoIdentities"
        :oauth-readiness="oauthReadinessForDetails"
        :last-push-error="lastPushErrorFor(comment.id)"
        @repo-reclone="emit('repo-reclone', $event)"
        @focus-predecessor="onFocusPredecessor"
        @cancel-waiting="onCancelWaiting"
        @retry-oauth-probe="onRetryOauthProbe"
      >
        <template
          v-if="shouldMountCommentConnectionPanel(commentOwnsSharedContainer(comment.id))"
          #layer-ztree
        >
          <TaskDetailCommentLayerAssociationBody
            :ref="(el) => onLayerBodyRef(isActive, el)"
            v-bind="buildPerCommentLayerBodyBind(props, comment.id)"
            v-on="buildPerCommentLayerBodyListeners(emit, comment.id)"
          />
        </template>
        <template #server-runtime-status>
          <!-- 启动详情与运行态同屏；授权缺失时由后端缓存/前端对齐避免「未知」 -->
          <div class="space-y-2" data-testid="comment-execution-runtime-with-start">
            <TaskDetailServerStartStatusPanel
              v-if="commentOwnsSharedContainer(comment.id) || commentHasLiveBinding(comment.id)"
              v-bind="buildPerBindingServerStatusProps(comment.id)"
              :sse-live="sseLive"
              :sse-reconnecting="sseReconnecting"
              :sse-reconnect-attempts="sseReconnectAttempts"
              :sse-platform-restart-hint="ssePlatformRestartHint"
              @sse-manual-reconnect="onPerBindingSSEReconnect(comment.id)"
            />
            <ServerConfigRuntimeStatusSection
              v-if="shouldMountCommentRuntimeStatusSection(serverRuntimeStatusPanel)"
              v-bind="bindCommentRuntimePanel(serverRuntimeStatusPanel, comment.id, bindingStatusFor(comment.id), comment.created_at)"
            />
          </div>
        </template>
        <template #server-content>
          <ServerConfigServerContentSection
            v-if="serverContentPanel"
            v-bind="bindCommentServerContentPanel(serverContentPanel, comment.id)"
          />
        </template>
        <!-- 每条评论独立连接面板；心跳取该评论 binding，禁止复用任务级单例 -->
        <template v-if="shouldMountCommentConnectionPanel(commentOwnsSharedContainer(comment.id))">
          <TaskDetailContainerConnectionStatus
            v-bind="connectionStatusBind(comment.id)"
            force-visible
          />
          <TaskDetailServerStartStatusPanel
            v-bind="buildPerBindingServerStatusProps(comment.id)"
            :sse-live="sseLive"
            :sse-reconnecting="sseReconnecting"
            :sse-reconnect-attempts="sseReconnectAttempts"
            :sse-platform-restart-hint="ssePlatformRestartHint"
            @sse-manual-reconnect="onPerBindingSSEReconnect(comment.id)"
          />
        </template>
        <template v-else-if="commentHasLiveBinding(comment.id)">
          <TaskDetailServerStartStatusPanel
            v-bind="buildPerBindingServerStatusProps(comment.id)"
            :sse-live="sseLive"
            :sse-reconnecting="sseReconnecting"
            :sse-reconnect-attempts="sseReconnectAttempts"
            :sse-platform-restart-hint="ssePlatformRestartHint"
            @sse-manual-reconnect="onPerBindingSSEReconnect(comment.id)"
          />
          <p
            class="pt-1 text-[11px] text-amber-800"
            data-testid="comment-execution-parallel-no-shared-csc"
          >
            已调度（不等待前序）；独立 CSC 正在分配中。
          </p>
        </template>
      </TaskDetailCommentExecutionDetails>
    </template>
  </TaskDetailCommentsPanel>
</template>

<script setup>
import { computed, nextTick, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useCommentGitOauthAccessTokenProbe } from '../../composables/taskDetail/useCommentGitOauthAccessTokenProbe.js'
import TaskDetailCommentsPanel from './TaskDetailCommentsPanel.vue'
import TaskDetailCommentExecutionDetails from './TaskDetailCommentExecutionDetails.vue'
import { handleCancelWaitingPrevious } from './handleCancelWaitingPrevious.js'
import { useCommentSectionCloneProgressRows } from './useCommentSectionCloneProgressRows.js'
import TaskDetailContainerConnectionStatus from './TaskDetailContainerConnectionStatus.vue'
import TaskDetailServerStartStatusPanel from './TaskDetailServerStartStatusPanel.vue'
import TaskDetailCommentLayerAssociationBody from './TaskDetailCommentLayerAssociationBody.vue'
import ServerConfigRuntimeStatusSection from '../ServerConfigRuntimeStatusSection.vue'
import ServerConfigServerContentSection from '../ServerConfigServerContentSection.vue'
import { bindCommentRuntimePanel, runtimeStatusFromCommentPanel } from '../../composables/taskDetail/bindCommentRuntimePanel.js'
import { bindCommentServerContentPanel } from '../../composables/taskDetail/bindCommentServerContentPanel.js'
import { resolveRepoCloneIdentityFromMap } from '../../utils/taskDetailBranchAndRepoUtils.js'
import {
  resolveActiveExecutionCommentId,
  commentHasEffectivePredecessors,
  listCommentPredecessors,
} from '../../composables/taskDetail/useCommentExecutionContext.js'
import {
  shouldEnableCommentRuntimeTab,
  shouldMountCommentConnectionPanel,
  shouldMountCommentRuntimeStatusSection,
} from '../../composables/taskDetail/commentExecutionPanelPolicy.js'
import {
  buildPerCommentLayerBodyBind,
  buildPerCommentLayerBodyListeners,
  lastPushErrorForComment,
  onFocusPredecessor,
  scrollToCommentById,
} from '../../composables/taskDetail/taskDetailCommentsSectionHelpers.js'
import {
  collectPrHtmlUrlsFromZNodes,
  decorateAgentRepliesWithLayerPr,
} from '../../composables/taskDetail/decorateAgentRepliesWithLayerPr.js'

const newComment = defineModel('newComment', { type: String, required: true })
const layerGraphCommandKind = defineModel('layerGraphCommandKind', { type: String, required: true })
const layerGraphSelectedModel = defineModel('layerGraphSelectedModel', { type: String, required: true })
const layerGraphAutoIterationCount = defineModel('layerGraphAutoIterationCount', { type: String, required: true })
const layerGraphCommandText = defineModel('layerGraphCommandText', { type: String, required: true })

const props = defineProps({
  showContainerCloneProgressBanner: { type: Boolean, required: true },
  containerCloneProgressEntries: { type: Array, required: true },
  cloneProgressByCommentId: { type: Object, default: () => ({}) },
  recloneLoadingByUrl: { type: Object, default: () => ({}) },
  recloneStatusByUrl: { type: Object, default: () => ({}) },
  recloneErrorTraceIdByUrl: { type: Object, default: () => ({}) },
  repoRecloneGlobalLoading: { type: Boolean, default: false },
  repoCloneFieldId: { type: Function, required: true },
  shortCloneRepoLabel: { type: Function, required: true },
  cloneProgressRowHasSubPhases: { type: Function, required: true },
  cloneProgressRecvPct: { type: Function, required: true },
  cloneProgressUnpackPct: { type: Function, required: true },
  cloneProgressBarWidthTransitionClass: { type: String, required: true },
  cloneProgressRowLogIsPlaceholder: { type: Function, required: true },
  cloneProgressRowLogDisplayText: { type: Function, required: true },
  displayComments: { type: Array, required: true },
  collaboratorNameById: { type: Object, default: () => ({}) },
  collaboratorAvatarById: { type: Object, default: () => ({}) },
  commentsFeedErrors: { type: Array, default: () => [] },
  commentsHasMore: { type: Boolean, default: false },
  commentsLoadingMore: { type: Boolean, default: false },
  aiStreamBusy: { type: Boolean, required: true },
  aiStreamBuffer: { type: String, required: true },
  activeContainerAgentId: { type: String, default: '' },
  // OPT-20260816-002: 复用 useTaskDetail 单一评论容器绑定实例（原本地重复实例已删除）
  bindingStatusFor: { type: Function, required: true },
  bindingCscIdFor: { type: Function, required: true },
  bindingContainerNameFor: { type: Function, required: true },
  commentHasLiveBinding: { type: Function, required: true },
  commentOwnsSharedContainer: { type: Function, required: true },
  bindingStartTraceIdFor: { type: Function, required: true },
  buildPerBindingServerStatusProps: { type: Function, required: true },
  markBindingReconnecting: { type: Function, required: true },
  cancelWaitingPreviousBinding: { type: Function, required: true },
  isCancelWaitingBusy: { type: Function, required: true },
  hasZtreeAgentSteps: { type: Boolean, required: true },
  agentStepCount: { type: Number, required: true },
  tenantId: { type: String, required: true },
  workspaceId: { type: String, required: true },
  taskId: { type: String, required: true },
  /** 当前任务对象，透传给评论区镜像选择 hints */
  task: { type: Object, default: null },
  containerPageLinkPendingReveal: { type: Boolean, required: true },
  containerHttpUnreachable: { type: Boolean, required: true },
  displayContainerVscodeUrl: { type: String, default: '' },
  commentComposerChips: { type: Array, required: true },
  taskProjectsWithDetails: { type: Array, default: () => [] },
  taskRepoRows: { type: Array, default: () => [] },
  /** 任务级仓库已选 Git 身份（url → git_identity_id，UI 态），供 auto_run 首评回退（OPT-20260821-028） */
  repoCloneIdentityByUrl: { type: Object, default: () => ({}) },
  serverRuntimeNotServing: { type: Boolean, default: false },
  containerHeartbeatPaused: { type: Boolean, default: false },
  containerBootstrapFailureMessage: { type: String, default: '' },
  containerBootstrapFailureTraceId: { type: String, default: '' },
  bootstrapCloneLogText: { type: String, default: '' },
  containerLayerGraphAuthInvalid: { type: Boolean, default: false },
  containerPageUrl: { type: String, default: '' },
  layerGraphZNodes: { type: Array, required: true },
  layerGraphRefreshing: { type: Boolean, required: true },
  layerGraphMetaLine: { type: String, default: '' },
  layerGraphBusyActionKey: { type: String, default: '' },
  selectedLayerGraphNode: { type: Object, default: null },
  selectedZTreeLayerChangesPanel: { type: Object, default: null },
  selectedLayerGraphFileTreeLayerId: { type: String, default: '' },
  layerChangesRefreshBusy: { type: Boolean, default: false },
  layerChangesRefreshEnabled: { type: Boolean, default: false },
  layerChangesRefreshError: { type: String, default: '' },
  layerChangesRefreshErrorTraceId: { type: String, default: '' },
  layerChangesGitStagedActionsBlocked: { type: Boolean, default: false },
  layerChangesGitCommitIdentityBlocked: { type: Boolean, default: false },
  containerEndpointRegistered: { type: Boolean, default: false },
  projectFileTreeRefreshNonce: { type: Number, default: 0 },
  layerGraphModelSelectDisabled: { type: Boolean, default: false },
  layerGraphModelOptions: { type: Array, default: () => [] },
  layerGraphDefaultModel: { type: String, default: '' },
  layerGraphEditRunTargetJobId: { type: String, default: '' },
  layerGraphModelLoadError: { type: String, default: '' },
  layerGraphModelLoadErrorTraceId: { type: String, default: '' },
  layerGraphCmdError: { type: String, default: '' },
  layerGraphCmdErrorTraceId: { type: String, default: '' },
  layerGraphCmdSending: { type: Boolean, default: false },
  containerActionsBlocked: { type: Boolean, default: false },
  layerExecLogLoading: { type: Boolean, default: false },
  layerExecLogCopyable: { type: Boolean, default: false },
  layerExecLogClearable: { type: Boolean, default: false },
  layerExecLogCopyFeedback: { type: Boolean, default: false },
  layerExecLogTopError: { type: String, default: '' },
  zTreeLogTargets: { type: Object, default: () => ({}) },
  layerCloneLogFetchError: { type: String, default: '' },
  layerCloneLogFetchErrorTraceId: { type: String, default: '' },
  layerCloneLogText: { type: String, default: '' },
  layerLiveOutputDisplay: { type: String, default: '' },
  layerJobLogFetchError: { type: String, default: '' },
  layerJobLogFetchErrorTraceId: { type: String, default: '' },
  layerJobExecutionPayload: { type: Object, default: null },
  layerJobCommandHead: { type: String, default: '' },
  layerJobOutputDisplay: { type: String, default: '' },
  layerAgentStepCopyFeedbackKey: { type: String, default: '' },
  layerAgentStepCards: { type: Array, default: () => [] },
  /** 任务级 SSE 状态（per-binding 启动面板共用；启服进度/日志走 buildPerBindingServerStatusProps） */
  sseLive: { type: Boolean, default: false },
  sseReconnecting: { type: Boolean, default: false },
  sseReconnectAttempts: { type: Number, default: 0 },
  ssePlatformRestartHint: { type: Boolean, default: false },
  projectServerRunTemplate: { type: Object, default: null },
  /**
   * 服务器运行状态面板数据（由 TaskDetail 经 ServerConfig defineExpose 桥接）。
   * 非空时，每条评论的「执行细节」各自显示 Tab 并注入该评论的运行态面板。
   */
  serverRuntimeStatusPanel: { type: Object, default: null },
  /**
   * 服务器内容面板数据（由 TaskDetail 经 ServerConfig defineExpose 桥接）。
   * 非空时注入各评论「服务器内容」Tab。
   */
  serverContentPanel: { type: Object, default: null },
  /** 按 comment_id 分片的层图 / 端点 / 执行日志 / 命令框 */
  layerPanelByCommentId: { type: Object, default: () => ({}) },
  /** 公司 Git 身份目录，供执行细节摘要解析评论 repo_identities */
  gitIdentityOptions: { type: Array, default: () => [] },
  /** 任务级 Git OAuth 绑定检查结果，供执行细节摘要展示 */
  repoOAuthReadiness: { type: Object, default: null },
})

const emit = defineEmits([
  'remove-comment-composer-chip',
  'comment-textarea-keydown',
  'submit-comment',
  'load-more-comments',
  'task-updated',
  'refresh-layer-graph',
  'layer-graph-node-select',
  'layer-graph-job-redo',
  'layer-graph-job-interrupt',
  'layer-graph-job-continue',
  'layer-graph-job-edit-run',
  'layer-graph-job-delete',
  'layer-graph-layer-delete',
  'layer-graph-layer-submit',
  'layer-graph-layer-push',
  'layer-graph-layer-merge',
  'layer-graph-layer-submit-and-push',
  'layer-graph-layer-submit-and-merge',
  'layer-changes-refresh',
  'layer-changes-staged-refresh',
  'layer-changes-commit-staged',
  'layer-changes-load-more',
  'submit-layer-graph-command',
  'copy-layer-exec-log',
  'clear-layer-exec-log',
  'copy-agent-step-json',
  'agent-step-rich-interact',
  'sse-manual-reconnect',
  'repo-reclone',
  'patch-layer-panel',
  'repo-oauth-readiness',
])

// OPT-20260816-002: 绑定实例已提升至 useTaskDetail，此处接收同一套函数 props
const {
  bindingStatusFor,
  bindingCscIdFor,
  bindingContainerNameFor,
  commentHasLiveBinding,
  commentOwnsSharedContainer,
  bindingStartTraceIdFor,
  buildPerBindingServerStatusProps,
  markBindingReconnecting,
  cancelWaitingPreviousBinding,
  isCancelWaitingBusy,
} = props

/** auto_run 首评无 composer 选择：评论 repo_identities 为空时回退任务级仓库身份（OPT-20260821-028） */
const taskRepoIdentities = computed(() => {
  const rows = Array.isArray(props.taskRepoRows) ? props.taskRepoRows : []
  const out = []
  const seen = new Set()
  const taskParams = props.task?.parameters?.repo_clone_git_identities
  for (const row of rows) {
    const url = String(row?.url || '').trim()
    if (!url || seen.has(url)) continue
    const identityId =
      String(props.repoCloneIdentityByUrl?.[url] || '').trim()
      || resolveRepoCloneIdentityFromMap(taskParams, url)
    if (!identityId) continue
    seen.add(url)
    out.push({ repo_url: url, git_identity_id: identityId })
  }
  return out
})

const { oauthReadinessForDetails, retryProbeFor } = useCommentGitOauthAccessTokenProbe({
  displayComments: computed(() => props.displayComments),
  fallbackRepoIdentities: taskRepoIdentities,
  repoOAuthReadiness: computed(() => props.repoOAuthReadiness),
  emit,
})

/** OPT-20260902-011：check_failed 徽标「重试」→ 移除 probedUrls 并强制再探测一次 */
function onRetryOauthProbe(urls) {
  retryProbeFor(urls)
}

function lastPushErrorFor(commentId) {
  return lastPushErrorForComment(props.layerPanelByCommentId, commentId)
}

const feedDisplayComments = computed(() =>
  decorateAgentRepliesWithLayerPr(
    props.displayComments,
    collectPrHtmlUrlsFromZNodes(props.layerGraphZNodes),
    {
      activeAgentId: props.activeContainerAgentId,
      streamBusy: props.aiStreamBusy,
      streamText: props.aiStreamBuffer,
    },
  ),
)

async function onCancelWaiting(payload) {
  await handleCancelWaitingPrevious(cancelWaitingPreviousBinding, payload)
}

const { commentCloneProgressRows } = useCommentSectionCloneProgressRows(props, {
  bindingStatusFor,
  buildPerBindingServerStatusProps,
})

/** wait_previous 模式下是否存在未完成的有效前序（OPT-20260811-085 badge 文案） */
function hasEffectivePredecessors(comment) {
  return commentHasEffectivePredecessors(comment, props.displayComments, { bindingStatusFor })
}

function predecessorsFor(comment) {
  return listCommentPredecessors(comment, props.displayComments, { bindingStatusFor })
}

const activeExecutionCommentId = computed(() =>
  resolveActiveExecutionCommentId(props.displayComments, props.activeContainerAgentId, {
    bindingStatusFor,
    bindingCscIdFor,
  }),
)

/** OPT-20260724-022: per-binding SSE 重连 — 标记特定 binding + 触发 task 级重连 */
function onPerBindingSSEReconnect(commentId) {
  markBindingReconnecting(commentId, () => emit('sse-manual-reconnect'))
}

function connectionStatusBind(commentId) {
  const p = buildPerBindingServerStatusProps(commentId)
  return {
    containerHeartbeatStatus: p.heartbeatStatus || 'idle',
    containerHeartbeatLastSuccess: p.heartbeatLastSuccess ?? null,
    containerHeartbeatAttempts: p.heartbeatAttempts || 0,
    containerHeartbeatError: p.heartbeatError || '',
    containerHeartbeatSeqInfo: p.heartbeatSeqInfo || {},
    containerHeartbeatLogLines: p.heartbeatLogLines || [],
  }
}

function onLayerBodyRef(active, el) {
  if (active) layerAssociationBodyRef.value = el
}

const layerAssociationBodyRef = ref(null)
defineExpose({
  taskLayerAssociationPanelRef: computed(() => layerAssociationBodyRef.value?.taskLayerAssociationPanelRef),
  focusLayerGraphCommandInput: (...args) => layerAssociationBodyRef.value?.focusLayerGraphCommandInput?.(...args),
  selectLayerGraphCommandInput: (...args) => layerAssociationBodyRef.value?.selectLayerGraphCommandInput?.(...args),
})

// OPT-20260817-013 前端部分：搜索命中评论（?comment=<id>）进入详情页后，
// 评论 feed 加载完成即滚动到该评论并高亮；同一 id 只滚一次。
const route = useRoute()
const routeCommentId = computed(() => {
  const v = route.query?.comment
  return String(Array.isArray(v) ? (v[0] ?? '') : (v ?? '')).trim()
})
let lastScrolledCommentId = ''
watch(
  () => props.displayComments,
  () => {
    const id = routeCommentId.value
    if (!id || lastScrolledCommentId === id) return
    nextTick(() => {
      if (scrollToCommentById(id)) {
        lastScrolledCommentId = id
      }
    })
  },
  { immediate: true },
)
</script>

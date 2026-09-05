<template>
  <div data-testid="comment-layer-association-body" class="mb-1">
    <TaskDetailCommentLayerZtreeStatus
      v-if="showCommentLayerZtreeLoading || showCommentLayerZtreeReleased"
      :show-loading="showCommentLayerZtreeLoading"
      :show-released="showCommentLayerZtreeReleased"
      :loading-hint="commentLayerZtreeLoadingHint"
      :loading-is-error="commentLayerZtreeLoadingIsError"
      :loading-error-trace-id="commentLayerZtreeLoadingErrorTraceId"
      :released-title="commentLayerZtreeReleasedTitle"
      :released-body="commentLayerZtreeReleasedBody"
      :tenant-id="tenantId"
      :workspace-id="workspaceId"
      :task-id="taskId"
      :container-page-url="containerPageUrl"
      :container-page-link-pending-reveal="containerPageLinkPendingReveal"
      :container-http-unreachable="containerHttpUnreachable"
      :display-container-vscode-url="displayContainerVscodeUrl"
    />
    <TaskDetailTaskLayerAssociationPanel
      v-else-if="layerGraphZNodes.length > 0"
      ref="taskLayerAssociationPanelRef"
      v-model:layer-graph-command-kind="layerGraphCommandKind"
      v-model:layer-graph-selected-model="layerGraphSelectedModel"
      v-model:layer-graph-auto-iteration-count="layerGraphAutoIterationCount"
      v-model:layer-graph-command-text="layerGraphCommandText"
      :layer-graph-refreshing="layerGraphRefreshing"
      :layer-graph-z-nodes="layerGraphZNodes"
      :layer-graph-meta-line="layerGraphMetaLine"
      :layer-graph-busy-action-key="layerGraphBusyActionKey"
      :selected-layer-graph-node="selectedLayerGraphNode"
      :selected-z-tree-layer-changes-panel="selectedZTreeLayerChangesPanel"
      :selected-layer-graph-file-tree-layer-id="selectedLayerGraphFileTreeLayerId"
      :layer-changes-refresh-busy="layerChangesRefreshBusy"
      :layer-changes-refresh-enabled="layerChangesRefreshEnabled"
      :layer-changes-refresh-error="layerChangesRefreshError"
      :layer-changes-refresh-error-trace-id="layerChangesRefreshErrorTraceId"
      :layer-changes-git-staged-actions-blocked="layerChangesGitStagedActionsBlocked"
      :layer-changes-git-commit-identity-blocked="layerChangesGitCommitIdentityBlocked"
      :tenant-id="tenantId"
      :workspace-id="workspaceId"
      :task-id="taskId"
      :comment-id="commentId"
      :container-page-url="containerPageUrl"
      :container-page-link-pending-reveal="containerPageLinkPendingReveal"
      :container-endpoint-registered="containerEndpointRegistered"
      :container-http-unreachable="containerHttpUnreachable"
      :display-container-vscode-url="displayContainerVscodeUrl"
      :project-file-tree-refresh-nonce="projectFileTreeRefreshNonce"
      :layer-graph-model-select-disabled="layerGraphModelSelectDisabled"
      :layer-graph-model-options="layerGraphModelOptions"
      :layer-graph-default-model="layerGraphDefaultModel"
      :layer-graph-edit-run-target-job-id="layerGraphEditRunTargetJobId"
      :layer-graph-model-load-error="layerGraphModelLoadError"
      :layer-graph-model-load-error-trace-id="layerGraphModelLoadErrorTraceId"
      :layer-graph-cmd-error="layerGraphCmdError"
      :layer-graph-cmd-error-trace-id="layerGraphCmdErrorTraceId"
      :layer-graph-cmd-sending="layerGraphCmdSending"
      :container-actions-blocked="containerActionsBlocked"
      :container-released="containerReleased"
      :layer-exec-log-loading="layerExecLogLoading"
      :layer-exec-log-copyable="layerExecLogCopyable"
      :layer-exec-log-clearable="layerExecLogClearable"
      :layer-exec-log-copy-feedback="layerExecLogCopyFeedback"
      :layer-exec-log-top-error="layerExecLogTopError"
      :z-tree-log-targets="zTreeLogTargets"
      :layer-clone-log-fetch-error="layerCloneLogFetchError"
      :layer-clone-log-fetch-error-trace-id="layerCloneLogFetchErrorTraceId"
      :layer-clone-log-text="layerCloneLogText"
      :layer-live-output-display="layerLiveOutputDisplay"
      :layer-job-log-fetch-error="layerJobLogFetchError"
      :layer-job-log-fetch-error-trace-id="layerJobLogFetchErrorTraceId"
      :layer-job-execution-payload="layerJobExecutionPayload"
      :layer-job-command-head="layerJobCommandHead"
      :layer-job-output-display="layerJobOutputDisplay"
      :layer-agent-step-copy-feedback-key="layerAgentStepCopyFeedbackKey"
      :layer-agent-step-cards="layerAgentStepCards"
      :comment-layer-ztree-loading-hint="commentLayerZtreeLoadingHint"
      :comment-layer-ztree-loading-is-error="commentLayerZtreeLoadingIsError"
      :comment-layer-ztree-loading-error-trace-id="commentLayerZtreeLoadingErrorTraceId"
      @refresh-layer-graph="emit('refresh-layer-graph')"
      @layer-graph-node-select="emit('layer-graph-node-select', $event)"
      @layer-graph-job-redo="emit('layer-graph-job-redo', $event)"
      @layer-graph-job-interrupt="emit('layer-graph-job-interrupt', $event)"
      @layer-graph-job-continue="emit('layer-graph-job-continue', $event)"
      @layer-graph-job-edit-run="emit('layer-graph-job-edit-run', $event)"
      @layer-graph-job-delete="emit('layer-graph-job-delete', $event)"
      @layer-graph-layer-delete="emit('layer-graph-layer-delete', $event)"
      @layer-graph-layer-submit="emit('layer-graph-layer-submit', $event)"
      @layer-graph-layer-push="emit('layer-graph-layer-push', $event)"
      @layer-graph-layer-merge="emit('layer-graph-layer-merge', $event)"
      @layer-graph-layer-submit-and-push="emit('layer-graph-layer-submit-and-push', $event)"
      @layer-graph-layer-submit-and-merge="emit('layer-graph-layer-submit-and-merge', $event)"
      @layer-changes-refresh="emit('layer-changes-refresh')"
      @layer-changes-staged-refresh="emit('layer-changes-staged-refresh')"
      @layer-changes-commit-staged="emit('layer-changes-commit-staged', $event)"
      @layer-changes-load-more="emit('layer-changes-load-more')"
      @submit-layer-graph-command="emit('submit-layer-graph-command')"
      @copy-layer-exec-log="emit('copy-layer-exec-log')"
      @clear-layer-exec-log="emit('clear-layer-exec-log')"
      @copy-agent-step-json="emit('copy-agent-step-json', $event)"
      @agent-step-rich-interact="emit('agent-step-rich-interact', $event)"
    />
  </div>
</template>

<script setup>
import { ref } from 'vue'
import TaskDetailCommentLayerZtreeStatus from './TaskDetailCommentLayerZtreeStatus.vue'
import TaskDetailTaskLayerAssociationPanel from './TaskDetailTaskLayerAssociationPanel.vue'

const layerGraphCommandKind = defineModel('layerGraphCommandKind', { type: String, required: true })
const layerGraphSelectedModel = defineModel('layerGraphSelectedModel', { type: String, required: true })
const layerGraphAutoIterationCount = defineModel('layerGraphAutoIterationCount', { type: String, required: true })
const layerGraphCommandText = defineModel('layerGraphCommandText', { type: String, required: true })

defineProps({
  showCommentLayerZtreeLoading: { type: Boolean, required: true },
  showCommentLayerZtreeReleased: { type: Boolean, required: true },
  commentLayerZtreeLoadingHint: { type: String, default: '' },
  commentLayerZtreeLoadingIsError: { type: Boolean, default: false },
  commentLayerZtreeLoadingErrorTraceId: { type: String, default: '' },
  commentLayerZtreeReleasedTitle: { type: String, default: '' },
  commentLayerZtreeReleasedBody: { type: String, default: '' },
  tenantId: { type: String, default: '' },
  workspaceId: { type: String, default: '' },
  taskId: { type: String, default: '' },
  commentId: { type: String, default: '' },
  containerPageUrl: { type: String, default: '' },
  containerPageLinkPendingReveal: { type: Boolean, required: true },
  containerHttpUnreachable: { type: Boolean, required: true },
  displayContainerVscodeUrl: { type: String, default: '' },
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
  containerReleased: { type: Boolean, default: false },
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
})

const emit = defineEmits([
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
])

const taskLayerAssociationPanelRef = ref(null)
defineExpose({
  taskLayerAssociationPanelRef,
  focusLayerGraphCommandInput: (...args) => taskLayerAssociationPanelRef.value?.focusLayerGraphCommandInput?.(...args),
  selectLayerGraphCommandInput: (...args) => taskLayerAssociationPanelRef.value?.selectLayerGraphCommandInput?.(...args),
})
</script>

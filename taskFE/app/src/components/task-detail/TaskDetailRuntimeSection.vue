<template>
  <div>
    <ServerConfig
      ref="serverConfigRef"
      :task="task"
      :task-id="taskId"
      :tenant-id="tenantId"
      :workspaceId="workspaceId"
      :sseConnection="sseConnection"
      :updateServerStatus="updateServerStatus"
      :statusMessage="statusMessage"
      :statusCommentId="statusCommentId"
      :statusRuntimeStatus="statusRuntimeStatus"
      :statusProgress="statusProgress"
      :statusLogs="statusLogs"
      :serverStatus="serverStatus"
      :isServerRunning="isServerRunning"
      :isServerStarting="isServerStarting"
      :serverUrl="serverUrl"
      :container-vscode-url="containerVscodeUrl"
      :container-page-url="containerPageUrl"
      :container-heartbeat-status="containerHeartbeatStatus"
      :pause-container-heartbeat-for-relay-stop="pauseContainerHeartbeatForRelayStop"
      :resume-container-heartbeat-for-relay-start="resumeContainerHeartbeatForRelayStart"
      :append-container-heartbeat-log-lines="appendContainerHeartbeatLogLines"
      :refresh-task-detail="refreshTaskDetail"
      :start-blocked-by-unbound-oauth="startBlockedByUnboundOAuth"
      :start-blocked-by-oauth-check-loading="startBlockedByOAuthCheckLoading"
      :project-server-run-template="projectServerRunTemplate"
      :display-comments="displayComments"
      :active-container-agent-id="activeContainerAgentId"
      @task-updated="emit('task-updated', $event)"
      @start-request-accepted="emit('start-request-accepted', $event)"
    >
      <template #after-mirror="{ defaultImageId, installedImages }">
        <TaskDetailLinkedProjectsPanel
          ref="linkedProjectsPanelRef"
          :tenant-id="tenantId"
          :workspace-id="workspaceId"
          :task-id="taskId"
          :is-editing="isEditing"
          :task-projects-with-details="taskProjectsWithDetails"
          :fallback-api-projects="Array.isArray(task && task.projects) ? task.projects : []"
          :editing-task="editingTask"
          :default-image-id="linkedProjectsDefaultImageId(defaultImageId)"
          :default-image-label="linkedProjectsDefaultImageLabel(defaultImageId, installedImages)"
          :default-image-removed="linkedProjectsDefaultImageRemoved()"
          :container-endpoint-registered="containerEndpointRegistered"
          :relay-to-trae-enabled="relayToTraeEnabled"
          :repo-clone-identity-save-error="repoCloneIdentitySaveError"
          :container-heartbeat-status="containerHeartbeatStatus"
          :reclone-loading-by-url="recloneLoadingByUrl"
          :repo-reclone-global-loading="repoRecloneGlobalLoading"
          :reclone-status-by-url="recloneStatusByUrl"
          :repo-clone-identity-by-url="repoCloneIdentityByUrl"
          :layer-git-identity-options="layerGitIdentityOptions"
          :repo-clone-identity-saving="repoCloneIdentitySaving"
          :layer-git-identity-loading="layerGitIdentityLoading"
          :clone-progress-entry-by-repo-match-key="cloneProgressEntryByRepoMatchKey"
          :git-clone-ref-match-key="gitCloneRefMatchKey"
          :repo-clone-field-id="repoCloneFieldId"
          :clone-progress-row-has-sub-phases="cloneProgressRowHasSubPhases"
          :clone-progress-recv-pct="cloneProgressRecvPct"
          :clone-progress-unpack-pct="cloneProgressUnpackPct"
          :clone-progress-bar-width-transition-class="cloneProgressBarWidthTransitionClass"
          :bootstrap-clone-done="bootstrapCloneDone"
          :bootstrap-clone-log-full="bootstrapCloneLogFull"
          :bootstrap-clone-log-segments="bootstrapCloneLogSegments"
          :workspace-projects="workspaceProjects"
          :workspace-projects-loading="workspaceProjectsLoading"
          :get-project-repos="getProjectRepos"
          :get-repo-branch-value="getRepoBranchValue"
          :get-repo-branches="getRepoBranches"
          :get-repo-branch-error="getRepoBranchError"
          :is-loading-repo-branches="isLoadingRepoBranches"
          :selected-layer-graph-file-tree-layer-id="selectedLayerGraphFileTreeLayerId"
          :layer-repo-git-identity-loading="layerRepoGitIdentityLoading"
          :layer-repo-git-identity-fetch-error="layerRepoGitIdentityFetchError"
          :per-repo-git-identity-syncing="perRepoGitIdentitySyncing"
          :per-repo-git-identity-sync-error="perRepoGitIdentitySyncError"
          :stale-repo-sync-loading="staleRepoSyncLoading"
          :stale-repo-sync-error="staleRepoSyncError"
          :stale-repo-sync-needs-reclone="staleRepoSyncNeedsReclone"
          :container-git-identity-line-for-repo-url="containerGitIdentityLineForRepoUrl"
          :git-oauth-catalog-version="gitOAuthCatalogVersion"
          :on-readiness-change="onReadinessChange"
          @repo-reclone="emit('repo-reclone', $event)"
          @fetch-layer-repo-git-identities="emit('fetch-layer-repo-git-identities')"
          @sync-per-repo-git-identities-to-container="emit('sync-per-repo-git-identities-to-container')"
          @sync-repo-address="emit('sync-repo-address')"
          @repo-clone-identity-change="emit('repo-clone-identity-change', $event)"
          @project-change="emit('project-change', $event)"
          @remove-project-association="emit('remove-project-association', $event)"
          @set-repo-branch="emit('set-repo-branch', $event)"
          @fetch-repo-branches="emit('fetch-repo-branches', $event)"
          @add-project-association="emit('add-project-association')"
          @git-identity-created="emit('git-identity-created')"
          @repo-oauth-readiness="emit('repo-oauth-readiness', $event)"
        />
      </template>
    </ServerConfig>

    <TaskDetailContainerHttpUnreachableBanner
      :visible="showContainerHttpUnreachableBanner"
    />
  </div>
</template>

<script setup>
import { ref } from 'vue'
import ServerConfig from '../ServerConfig.logic.vue'
import TaskDetailLinkedProjectsPanel from './TaskDetailLinkedProjectsPanel.vue'
import TaskDetailContainerHttpUnreachableBanner from './TaskDetailContainerHttpUnreachableBanner.vue'
import { resolveTaskDefaultImageLabel } from '../../utils/installedImageLabel.js'

const props = defineProps({
  task: { type: Object, required: true },
  workspaceId: { type: [String, Number], default: null },
  sseConnection: { type: Object, default: null },
  updateServerStatus: { type: Function, required: true },
  statusMessage: { type: String, default: '' },
  statusCommentId: { type: String, default: '' },
  statusRuntimeStatus: { type: String, default: '' },
  statusProgress: { type: Number, default: 0 },
  statusLogs: { type: Array, default: () => [] },
  serverStatus: { type: String, default: '' },
  isServerRunning: { type: Boolean, default: false },
  isServerStarting: { type: Boolean, default: false },
  runtimeStatus: { type: String, default: '' },
  serverUrl: { type: String, default: '' },
  containerVscodeUrl: { type: String, default: '' },
  containerPageUrl: { type: String, default: '' },
  containerHeartbeatStatus: { type: String, default: '' },
  pauseContainerHeartbeatForRelayStop: { type: Function, required: true },
  resumeContainerHeartbeatForRelayStart: { type: Function, required: true },
  appendContainerHeartbeatLogLines: { type: Function, required: true },
  refreshTaskDetail: { type: Function, required: true },
  startBlockedByUnboundOAuth: { type: Boolean, default: false },
  startBlockedByOAuthCheckLoading: { type: Boolean, default: false },
  projectServerRunTemplate: { type: Object, default: null },
  displayComments: { type: Array, default: () => [] },
  activeContainerAgentId: { type: String, default: '' },
  tenantId: { type: String, required: true },
  taskId: { type: String, required: true },
  isEditing: { type: Boolean, default: false },
  taskProjectsWithDetails: { type: Array, default: () => [] },
  editingTask: { type: Object, default: null },
  containerEndpointRegistered: { type: Boolean, default: false },
  relayToTraeEnabled: { type: Boolean, default: false },
  repoCloneIdentitySaveError: { type: String, default: '' },
  recloneLoadingByUrl: { type: Object, default: () => ({}) },
  repoRecloneGlobalLoading: { type: Boolean, default: false },
  recloneStatusByUrl: { type: Object, default: () => ({}) },
  repoCloneIdentityByUrl: { type: Object, default: () => ({}) },
  layerGitIdentityOptions: { type: Array, default: () => [] },
  repoCloneIdentitySaving: { type: Boolean, default: false },
  layerGitIdentityLoading: { type: Boolean, default: false },
  cloneProgressEntryByRepoMatchKey: { type: Object, default: () => ({}) },
  gitCloneRefMatchKey: { type: Function, required: true },
  repoCloneFieldId: { type: Function, required: true },
  cloneProgressRowHasSubPhases: { type: Function, required: true },
  cloneProgressRecvPct: { type: Function, required: true },
  cloneProgressUnpackPct: { type: Function, required: true },
  cloneProgressBarWidthTransitionClass: { type: String, default: '' },
  bootstrapCloneDone: { type: Boolean, default: false },
  bootstrapCloneLogFull: { type: String, default: '' },
  bootstrapCloneLogSegments: { type: Array, default: null },
  workspaceProjects: { type: Array, default: () => [] },
  workspaceProjectsLoading: { type: Boolean, default: false },
  getProjectRepos: { type: Function, required: true },
  getRepoBranchValue: { type: Function, required: true },
  getRepoBranches: { type: Function, required: true },
  getRepoBranchError: { type: Function, required: true },
  isLoadingRepoBranches: { type: Function, required: true },
  selectedLayerGraphFileTreeLayerId: { type: String, default: '' },
  layerRepoGitIdentityLoading: { type: Boolean, default: false },
  layerRepoGitIdentityFetchError: { type: String, default: '' },
  perRepoGitIdentitySyncing: { type: Boolean, default: false },
  perRepoGitIdentitySyncError: { type: String, default: '' },
  staleRepoSyncLoading: { type: Boolean, default: false },
  staleRepoSyncError: { type: String, default: '' },
  staleRepoSyncNeedsReclone: { type: Boolean, default: false },
  containerGitIdentityLineForRepoUrl: { type: Function, required: true },
  gitOAuthCatalogVersion: { type: Number, default: 0 },
  onReadinessChange: { type: Function, default: null },
  showContainerHttpUnreachableBanner: { type: Boolean, default: false },
  sseLive: { type: Boolean, default: false },
  sseReconnecting: { type: Boolean, default: false },
  sseReconnectAttempts: { type: Number, default: 0 },
  ssePlatformRestartHint: { type: Boolean, default: false },
  containerHeartbeatLastSuccess: { type: [Date, String, null], default: null },
  containerHeartbeatAttempts: { type: Number, default: 0 },
  containerHeartbeatError: { type: String, default: '' },
  containerHeartbeatSeqInfo: {
    type: Object,
    default: () => ({
      containerSeq: null,
      containerAck: null,
      saasSeq: null,
      saasAck: null,
      uplinkOk: null,
      downlinkOk: null,
      probeOk: null,
      bidirectionalOk: null,
    }),
  },
  containerHeartbeatLogLines: { type: Array, default: () => [] },
})

function linkedProjectsDefaultImageId(slotImageId) {
  return String(
    slotImageId || props.task?.container_image_id || props.task?.container_image?.id || '',
  ).trim()
}

function linkedProjectsDefaultImageLabel(imageId, installedImages) {
  return resolveTaskDefaultImageLabel({
    imageId: linkedProjectsDefaultImageId(imageId),
    installedImages,
    nestedImage: props.task?.container_image,
    comments: props.displayComments,
  })
}

function linkedProjectsDefaultImageRemoved() {
  return Boolean(props.task?.container_image_removed)
}

const emit = defineEmits([
  'task-updated',
  'start-request-accepted',
  'repo-reclone',
  'fetch-layer-repo-git-identities',
  'sync-per-repo-git-identities-to-container',
  'sync-repo-address',
  'repo-clone-identity-change',
  'project-change',
  'remove-project-association',
  'set-repo-branch',
  'fetch-repo-branches',
  'add-project-association',
  'git-identity-created',
  'repo-oauth-readiness',
  'sse-manual-reconnect',
])

const serverConfigRef = ref(null)
const linkedProjectsPanelRef = ref(null)
defineExpose({
  get serverConfigRef() { return serverConfigRef.value },
  get linkedProjectsPanelRef() { return linkedProjectsPanelRef.value },
})
</script>

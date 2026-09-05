<template>
  <div class="space-y-6">
    <slot
      name="after-mirror"
      :default-image-id="selectedImageId"
      :installed-images="installedImages"
    />

    <!-- 服务器信息 Tab（硬件已迁入评论区 composer，见 CommentComposerHardwareCard） -->
    <ServerConfigSectionTabs
      :active-server-section="activeServerSection"
      :is-relay-to-trae-enabled="isRelayToTraeEnabled"
      :body-expanded="serverSectionBodyExpanded"
      @select="selectServerSection"
      @select-content="onServerContentTabClick"
      @select-history="onServerHistoryTabClick"
      @toggle-body="toggleServerSectionBody"
    />

    <div v-show="serverSectionBodyExpanded" data-testid="server-section-body">
      <!-- OPT-20260809-025: 服务器运行状态已移入各评论「执行细节」Tab（每评论独立面板） -->
      <div
        v-show="activeServerSection === 'runtime'"
        class="p-4 border border-blue-200 rounded-md bg-blue-50"
        data-testid="server-runtime-moved-hint"
      >
        <p class="text-xs text-blue-800 m-0">
          服务器运行状态已移至各评论「执行细节」Tab 中查看（Workbench / VS Code / 停止服务器 / 跳转等操作不变）。
        </p>
      </div>

      <div
        v-show="activeServerSection === 'content'"
        class="p-4 border border-blue-200 rounded-md bg-blue-50"
        data-testid="server-content-moved-hint"
      >
        <p class="text-xs text-blue-800 m-0">
          服务器内容已移至各评论「执行细节」中的「服务器内容」Tab 查看（按评论拉取，须带评论 ID）。
        </p>
      </div>

      <ServerConfigServerStartHistoryPanel
        v-show="activeServerSection === 'history'"
        :records="serverStartHistoryRecords"
        :loading="isServerStartHistoryLoading"
        :message="serverStartHistoryMessage"
        :message-trace-id="serverStartHistoryMessageTraceId"
        @refresh="fetchServerStartHistory"
      />

      <ServerConfigRelayDirectPanel
        :visible="activeServerSection === 'relayDirect'"
        :service-online="relayToTraeServiceOnline"
        :is-online-service-up="isRelayToTraeOnlineServiceUp"
        :online-service-status-label="relayToTraeOnlineServiceStatusLabel"
        :is-status-loading="isRelayToTraeStatusLoading"
        :env-items="relayToTraeEnvItems"
        :access-token-masked-label="relayAccessTokenMaskedLabel"
        :is-starting="isRelayToTraeStarting"
        :is-stopping="isRelayToTraeStopping"
        :show-stop-button="showRelayToTraeStopButton"
        :has-task-id="Boolean(resolvedTaskId)"
        :has-image="!!selectedImageId"
        :ui-url="effectiveRelayToTraeUiUrl"
        :message="relayToTraeMessage"
        :repo-credential-guide-visible="relayToTraeRepoCredentialGuideVisible"
        :stale-repo-mismatches="staleRepoMismatches"
        :stale-repo-sync-loading="staleRepoSyncLoading"
        :start-blocked-by-stale-repo="startBlockedByStaleRepo"
        :start-blocked-by-unbound-oauth="effectiveStartBlockedByUnboundOAuth"
        :start-blocked-by-oauth-check-loading="effectiveStartBlockedByOAuthCheckLoading"
        :env-params-source-required-hint="envParamsSourceRequiredHint"
        :logs="relayToTraeLogs"
        :logs-text="relayToTraeLogsText"
        :log-copy-state="relayToTraeLogCopyState"
        :logs-expanded="relayToTraeLogsExpanded"
        @refresh-status="fetchRelayToTraeServiceStatus"
        @start="startRelayToTrae"
        @stop="stopRelayToTrae"
        @toggle-logs="relayToTraeLogsExpanded = !relayToTraeLogsExpanded"
        @clear-logs="clearRelayToTraeLogOutput"
        @copy-logs="copyRelayToTraeLogs"
        @sync-stale-repo-addresses="syncStaleRepoAddresses"
        @acknowledge-stale-repo="acknowledgeStaleRepo"
      />
    </div>

  </div>
</template>

<script setup>
import { ref, computed, inject, onUnmounted, unref } from 'vue'
import { useRoute } from 'vue-router'
import { useServerSectionBodyExpanded } from '../composables/useServerSectionBodyExpanded.js'
import { useServerConfigFeatureParams } from '../composables/taskDetail/useServerConfigFeatureParams.js'
import { useServerConfigImages } from '../composables/taskDetail/useServerConfigImages.js'
import { useServerConfigRuntime } from '../composables/taskDetail/useServerConfigRuntime.js'
import { useServerConfigRelayToTrae } from '../composables/taskDetail/useServerConfigRelayToTrae.js'
import { useServerConfigSections, createServerConfigSectionHandlers, useServerConfigRouteFlags } from '../composables/taskDetail/useServerConfigSections.js'
import ServerConfigSectionTabs from './ServerConfigSectionTabs.vue'
import ServerConfigServerStartHistoryPanel from './ServerConfigServerStartHistoryPanel.vue'
import ServerConfigRelayDirectPanel from './ServerConfigRelayDirectPanel.vue'
import { useServerConfigLifecycle } from '../composables/taskDetail/useServerConfigLifecycle.js'
import { bindCommentComposerFeatureParams } from '../composables/taskDetail/taskDetailFeatureParamsBridge.js'
import { expandHardwareForComment as doExpandHardware, restoreHardwareToProjectDefaults as doRestoreHardware } from '../composables/taskDetail/useServerConfigHardwareContext.js'
import { resolveServerConfigTaskId } from '../utils/serverConfigRouteHelpers.js'

const props = defineProps({
  task: { type: Object, default: null },
  taskId: { type: [String, Number], default: '' },
  tenantId: { type: [String, Number], default: '' },
  workspaceId: { type: String, default: null },
  sseConnection: { type: Object, default: null },
  updateServerStatus: { type: Function, default: null },
  statusMessage: { type: String, default: '等待启动...' },
  statusCommentId: { type: String, default: '' },
  statusRuntimeStatus: { type: String, default: '' },
  statusProgress: { type: Number, default: 0 },
  statusLogs: { type: Array, default: () => [] },
  serverStatus: { type: String, default: 'idle' },
  isServerRunning: { type: Boolean, default: false },
  isServerStarting: { type: Boolean, default: false },
  serverUrl: { type: String, default: '' },
  containerVscodeUrl: { type: String, default: '' },
  containerPageUrl: { type: String, default: '' },
  containerHeartbeatStatus: { type: String, default: 'idle' },
  pauseContainerHeartbeatForRelayStop: { type: Function, default: null },
  resumeContainerHeartbeatForRelayStart: { type: Function, default: null },
  appendContainerHeartbeatLogLines: { type: Function, default: null },
  startBlockedByUnboundOAuth: { type: Boolean, default: false },
  startBlockedByOAuthCheckLoading: { type: Boolean, default: false },
  projectServerRunTemplate: { type: Object, default: null },
  /** 评论 Feed（用于运行状态按评论 Tab） */
  displayComments: { type: Array, default: () => [] },
  activeContainerAgentId: { type: String, default: '' },
})

const injectedStartBlockedByUnboundOAuth = inject('taskDetailRelayStartBlockedByUnboundOAuth', null)
const injectedStartBlockedByOAuthCheckLoading = inject('taskDetailRelayStartBlockedByOAuthCheckLoading', null)

const effectiveStartBlockedByUnboundOAuth = computed(
  () => Boolean(props.startBlockedByUnboundOAuth || unref(injectedStartBlockedByUnboundOAuth)),
)
const effectiveStartBlockedByOAuthCheckLoading = computed(
  () => Boolean(props.startBlockedByOAuthCheckLoading || unref(injectedStartBlockedByOAuthCheckLoading)),
)

const emit = defineEmits(['task-updated', 'start-request-accepted'])

const serverHardwarePanelRef = ref(null)
const route = useRoute()
const resolvedTaskId = computed(() => resolveServerConfigTaskId({
  route,
  task: props.task,
  taskId: props.taskId,
  tenantId: props.tenantId,
  workspaceId: props.workspaceId,
}))
const { isRelayToTraeEnabled } = useServerConfigRouteFlags(route)
const featureParamsTenantId = computed(() => String(route.params.tenant || ''))

const {
  serverSectionBodyExpanded,
  expandServerSectionBody,
  toggleServerSectionBody,
} = useServerSectionBodyExpanded()

const { activeServerSection } = useServerConfigSections({ route, isRelayToTraeEnabled })

const {
  installedImages,
  selectedImageId,
  isInitialized,
  fetchInstalledImages,
} = useServerConfigImages({ props, route })

const {
  featureParamsSource,
  personalConfigs,
  selectedPersonalConfigId,
  resolvedEnvPreview,
  isEnvPreviewLoading,
  envPreviewExpanded,
  featureParamsPersistError,
  featureParamsSourcesAvailable,
  envParamsSourceRequiredHint,
  initFeatureParamsSource,
  fetchEnvPreview,
  onFeatureParamsSourceUpdate,
  onPersonalConfigIdUpdate,
  onSourceChange,
} = useServerConfigFeatureParams({ props, emit, route, workspaceId: props.workspaceId })

const stopFeatureParamsBridge = bindCommentComposerFeatureParams({
  tenantId: featureParamsTenantId,
  featureParamsSource,
  personalConfigs,
  selectedPersonalConfigId,
  resolvedEnvPreview,
  isEnvPreviewLoading,
  envPreviewExpanded,
  featureParamsPersistError,
  featureParamsSourcesAvailable,
  onSourceChange,
  fetchEnvPreview,
  onFeatureParamsSourceUpdate,
  onPersonalConfigIdUpdate,
})
onUnmounted(stopFeatureParamsBridge)

let relayToTraeDefaultAppliedRef = null

const runtime = useServerConfigRuntime({
  props,
  route,
  installedImages,
  activeServerSection,
  isRelayToTraeEnabled,
  relayToTraeDefaultAppliedGetter: () => relayToTraeDefaultAppliedRef,
})

const relay = useServerConfigRelayToTrae({
  props,
  route,
  emit,
  selectedImageId,
  isRelayToTraeEnabled,
  featureParamsSource,
  selectedPersonalConfigId,
  envParamsSourceRequiredHint,
  effectiveStartBlockedByUnboundOAuth,
  effectiveStartBlockedByOAuthCheckLoading,
  runtimePublicIp: runtime.runtimePublicIp,
  resumeContainerHeartbeatForRelayStart: props.resumeContainerHeartbeatForRelayStart,
  pauseContainerHeartbeatForRelayStop: props.pauseContainerHeartbeatForRelayStop,
  appendContainerHeartbeatLogLines: props.appendContainerHeartbeatLogLines,
})

relayToTraeDefaultAppliedRef = relay.relayToTraeDefaultApplied

useServerConfigLifecycle({
  props,
  route,
  serverHardwarePanelRef,
  isInitialized,
  selectedImageId,
  installedImages,
  fetchInstalledImages,
  initFeatureParamsSource,
  isRelayToTraeEnabled,
  activeServerSection,
  relayToTraeDefaultApplied: relay.relayToTraeDefaultApplied,
  relayToTraeEnvItems: relay.relayToTraeEnvItems,
  fetchServerStartHistory: runtime.fetchServerStartHistory,
  fetchRelayToTraeStatusOnMount: relay.fetchRelayToTraeStatusOnMount,
  fetchRelayToTraeServiceStatus: relay.fetchRelayToTraeServiceStatus,
  loadRelayToTraeEnvDefaults: relay.loadRelayToTraeEnvDefaults,
  resetRuntimeForNewTask: runtime.resetRuntimeForNewTask,
  resetRelayForNewTask: relay.resetRelayForNewTask,
  resetStaleRepoAck: relay.resetStaleRepoAck,
  stopRelayToTraePoll: relay.stopRelayToTraePoll,
  watchStatusMessageForRefresh: runtime.watchStatusMessageForRefresh,
  disposeRelayTimers: relay.disposeRelayTimers,
  disposeRuntimeTimers: runtime.disposeRuntimeTimers,
})

const {
  selectServerSection,
  onServerContentTabClick,
  onServerHistoryTabClick,
} = createServerConfigSectionHandlers({
  activeServerSection,
  expandServerSectionBody,
  fetchServerContent: runtime.fetchServerContent,
  fetchServerStartHistory: runtime.fetchServerStartHistory,
})

const onStartRequestAccepted = (result) => {
  emit('start-request-accepted', result)
}

// 展开硬件配置面板，供评论区「临时调节」按钮调用 (OPT-039)
const expandHardwareForComment = () => doExpandHardware(expandServerSectionBody, activeServerSection, serverHardwarePanelRef)

// 恢复硬件配置为项目模版默认值
const restoreHardwareToProjectDefaults = () => doRestoreHardware(serverHardwarePanelRef)

const {
  serverRuntimeStatus,
  serverRuntimeStatusResponse,
  isServerStartHistoryLoading,
  serverStartHistoryRecords,
  serverStartHistoryMessage,
  serverStartHistoryMessageTraceId,
  serverJumpUrl,
  serverJumpDefaultPort,
  fetchServerStartHistory,
  stopServer,
  serverRuntimeStatusPanel,
  serverContentPanel,
} = runtime

const {
  relayToTraeEnvItems,
  relayAccessTokenMaskedLabel,
  relayToTraeMessage,
  relayToTraeLogs,
  relayToTraeLogsExpanded,
  isRelayToTraeStarting,
  isRelayToTraeStopping,
  relayToTraeServiceOnline,
  isRelayToTraeStatusLoading,
  relayToTraeLogCopyState,
  relayToTraeRepoCredentialGuideVisible,
  staleRepoSyncLoading,
  relayToTraeLogsText,
  isRelayToTraeOnlineServiceUp,
  showRelayToTraeStopButton,
  staleRepoMismatches,
  startBlockedByStaleRepo,
  relayToTraeOnlineServiceStatusLabel,
  effectiveRelayToTraeUiUrl,
  acknowledgeStaleRepo,
  syncStaleRepoAddresses,
  clearRelayToTraeLogOutput,
  copyRelayToTraeLogs,
  fetchRelayToTraeServiceStatus,
  startRelayToTrae,
  stopRelayToTrae,
} = relay

defineExpose({
  applyRelayToTraeStatusFromSse: relay.applyRelayToTraeStatusFromSse,
  expandHardwareForComment,
  restoreHardwareToProjectDefaults,
  stopServer,
  /** 服务器运行状态面板数据源（评论区「执行细节」Tab 消费） */
  serverRuntimeStatusPanel,
  /** 服务器内容面板数据源（评论区「服务器内容」Tab 消费） */
  serverContentPanel,
})
</script>

<style scoped>
.server-section-tabs {
  position: sticky;
  top: 12px;
  z-index: 30;
  background-color: #ffffff;
}
</style>

import { ref, computed } from 'vue'
import { apiFetch } from '../../utils/apiUtils.js'
import {
  buildDefaultRelayToTraeEnvItems,
  mergeRelayToTraeLogLines,
  parseRelayToTraeEnv,
  resolveRelayConsoleOpenUrl,
} from '../../utils/relayToTraeUtils.js'
import {
  collectTaskRepoAddressMismatches,
  syncTaskRepoAddressesFromProjects,
} from '../../utils/taskRepoAddressMismatch.js'
import {
  resolveRelayContextIds,
  relayToTraeApiUrl as buildRelayToTraeApiUrl,
} from '../../utils/serverConfigRouteHelpers.js'
import { createServerConfigRelayToTraeStatusApi } from './useServerConfigRelayToTraeStatus.js'
import { createServerConfigRelayToTraeActions } from './useServerConfigRelayToTraeActions.js'

const RELAY_TO_TRAE_BOOTSTRAP_POLL_MS = 2000
const RELAY_TO_TRAE_BOOTSTRAP_MAX = 5

/**
 * ServerConfig relay-to-trae 面板 orchestrator。
 */
export function useServerConfigRelayToTrae({
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
  runtimePublicIp,
  resumeContainerHeartbeatForRelayStart,
  pauseContainerHeartbeatForRelayStop,
  appendContainerHeartbeatLogLines,
}) {
  const relayToTraeEnvItems = ref(buildDefaultRelayToTraeEnvItems(route.query))
  const relayAccessTokenMaskedLabel = '由服务端签发（不展示）'
  const relayToTraeMessage = ref('')
  const relayToTraeLogs = ref([])
  const relayToTraeLogsExpanded = ref(false)
  const relayToTraeUiUrl = ref('')
  const relayRuntimeMode = ref('')
  const isRelayToTraeStarting = ref(false)
  const isRelayToTraeStopping = ref(false)
  const isRelayToTraeRunning = ref(false)
  const relayToTraeServiceOnline = ref(false)
  const isRelayToTraeStatusLoading = ref(false)
  const relayToTraeOrphanPort = ref(false)
  const relayToTraeLogCopyState = ref('idle')
  const relayToTraeOnlineServiceUp = ref(false)
  const relayToTraeDefaultApplied = ref(isRelayToTraeEnabled.value)
  const relayToTraeRepoCredentialGuideVisible = ref(false)
  const staleRepoAcknowledged = ref(false)
  const staleRepoSyncLoading = ref(false)

  const internals = {
    relayToTraeLogCopyTimer: null,
    relayToTraeLogSnapshot: [],
    relayToTraeLogSuppressedAfterClear: false,
    relayToTraeLogAwaitingFirstStatusAfterStart: false,
    relayToTraeStatusLogCursor: 0,
    relayStatusFetchGeneration: 0,
    relayToTraePollTimer: null,
    relayToTraeBootstrapCount: 0,
  }

  const relayToTraeLogsText = computed(() => {
    if (relayToTraeLogs.value.length === 0) {
      return '暂无日志'
    }
    return relayToTraeLogs.value.join('\n')
  })

  const isRelayToTraeOnlineServiceUp = computed(() => relayToTraeOnlineServiceUp.value)
  const showRelayToTraeStopButton = computed(() => relayToTraeOnlineServiceUp.value || isRelayToTraeStarting.value)

  const staleRepoMismatches = computed(() =>
    collectTaskRepoAddressMismatches(props.task?.projects))
  const startBlockedByStaleRepo = computed(
    () => staleRepoMismatches.value.length > 0 && !staleRepoAcknowledged.value,
  )

  const relayToTraeOnlineServiceStatusLabel = computed(() => {
    if (isRelayToTraeStarting.value) {
      return '启动中'
    }
    if (relayToTraeOrphanPort.value) {
      return '端口占用（relay 未跟踪）'
    }
    if (relayToTraeOnlineServiceUp.value) {
      return '运行中'
    }
    return '未启动'
  })

  const relayRuntimeDeps = computed(() => ({
    relayToTraeEnvItems: relayToTraeEnvItems.value,
    relayRuntimeMode: relayRuntimeMode.value,
    selectedImageId: selectedImageId.value,
    isRelayToTraeRunning: isRelayToTraeRunning.value,
    isRelayToTraeStarting: isRelayToTraeStarting.value,
    relayToTraeOnlineServiceUp: relayToTraeOnlineServiceUp.value,
    relayToTraeUiUrl: relayToTraeUiUrl.value,
  }))

  const effectiveRelayToTraeUiUrl = computed(() => {
    const env = parseRelayToTraeEnv(relayToTraeEnvItems.value)
    const mode =
      relayRuntimeMode.value ||
      (selectedImageId.value && (isRelayToTraeRunning.value || isRelayToTraeStarting.value || relayToTraeOnlineServiceUp.value)
        ? 'selected_image'
        : '')
    return resolveRelayConsoleOpenUrl({
      mode,
      relayUiUrl: relayToTraeUiUrl.value,
      serverUrl: props.serverUrl,
      containerPageUrl: props.containerPageUrl,
      businessApiOrigin: env.BUSINESS_API_ENDPOINT_ORIGIN,
      taskApiOrigin: env.TASK_API_ENDPOINT_ORIGIN,
      publicIp: runtimePublicIp?.value ?? '',
    })
  })

  const ctxIds = () => resolveRelayContextIds({
    route,
    task: props.task,
    workspaceId: props.workspaceId,
    taskId: props.taskId,
    tenantId: props.tenantId,
  })
  const relayApi = (suffix) => buildRelayToTraeApiUrl({
    route,
    task: props.task,
    workspaceId: props.workspaceId,
    taskId: props.taskId,
    tenantId: props.tenantId,
    suffix,
  })

  const stopRelayToTraePoll = () => {
    if (internals.relayToTraePollTimer !== null) {
      clearInterval(internals.relayToTraePollTimer)
      internals.relayToTraePollTimer = null
    }
  }

  const resetRelayToTraeLogSnapshot = () => {
    internals.relayToTraeLogSnapshot = []
  }

  const appendRelayToTraeLogs = (lines) => {
    const merged = mergeRelayToTraeLogLines(
      {
        logs: relayToTraeLogs.value,
        snapshot: internals.relayToTraeLogSnapshot,
        suppressedAfterClear: internals.relayToTraeLogSuppressedAfterClear,
        awaitingFirstStatusAfterStart: internals.relayToTraeLogAwaitingFirstStatusAfterStart,
      },
      lines,
    )
    relayToTraeLogs.value = merged.logs
    internals.relayToTraeLogSnapshot = merged.snapshot
    internals.relayToTraeLogSuppressedAfterClear = merged.suppressedAfterClear
    internals.relayToTraeLogAwaitingFirstStatusAfterStart = merged.awaitingFirstStatusAfterStart
    if (Array.isArray(merged.heartbeatLines) && merged.heartbeatLines.length > 0) {
      appendContainerHeartbeatLogLines?.(merged.heartbeatLines)
    }
  }

  const statusApi = createServerConfigRelayToTraeStatusApi({
    isRelayToTraeEnabled,
    relayToTraeEnvItems,
    relayToTraeMessage,
    relayToTraeUiUrl,
    relayRuntimeMode,
    isRelayToTraeStarting,
    isRelayToTraeRunning,
    relayToTraeServiceOnline,
    isRelayToTraeStatusLoading,
    relayToTraeOrphanPort,
    relayToTraeOnlineServiceUp,
    effectiveRelayToTraeUiUrl,
    ctxIds,
    relayApi,
    appendRelayToTraeLogs,
    stopRelayToTraePoll,
    internals,
  })

  const startRelayToTraePoll = () => {
    stopRelayToTraePoll()
    internals.relayToTraeBootstrapCount = 0
    internals.relayToTraePollTimer = setInterval(async () => {
      internals.relayToTraeBootstrapCount += 1
      await statusApi.fetchRelayToTraeServiceStatus()
      if (relayToTraeOnlineServiceUp.value || internals.relayToTraeBootstrapCount >= RELAY_TO_TRAE_BOOTSTRAP_MAX) {
        stopRelayToTraePoll()
      }
    }, RELAY_TO_TRAE_BOOTSTRAP_POLL_MS)
  }

  const actionsApi = createServerConfigRelayToTraeActions({
    props,
    route,
    selectedImageId,
    isRelayToTraeEnabled,
    featureParamsSource,
    selectedPersonalConfigId,
    envParamsSourceRequiredHint,
    effectiveStartBlockedByUnboundOAuth,
    effectiveStartBlockedByOAuthCheckLoading,
    relayToTraeEnvItems,
    relayToTraeMessage,
    relayToTraeLogs,
    relayToTraeUiUrl,
    relayRuntimeMode,
    isRelayToTraeStarting,
    isRelayToTraeStopping,
    isRelayToTraeRunning,
    relayToTraeServiceOnline,
    relayToTraeOrphanPort,
    relayToTraeOnlineServiceUp,
    relayToTraeRepoCredentialGuideVisible,
    ctxIds,
    relayApi,
    resetRelayToTraeLogSnapshot,
    startRelayToTraePoll,
    stopRelayToTraePoll,
    fetchRelayToTraeServiceStatus: statusApi.fetchRelayToTraeServiceStatus,
    resumeContainerHeartbeatForRelayStart,
    pauseContainerHeartbeatForRelayStop,
    internals,
  })

  const clearRelayToTraeLogOutput = async () => {
    if (internals.relayToTraeLogSnapshot.length === 0 && relayToTraeLogs.value.length > 0) {
      internals.relayToTraeLogSnapshot = [...relayToTraeLogs.value]
    }
    relayToTraeLogs.value = []
    internals.relayToTraeLogSuppressedAfterClear = true
    internals.relayToTraeLogAwaitingFirstStatusAfterStart = false
    relayToTraeLogCopyState.value = 'idle'
    if (internals.relayToTraeLogCopyTimer) {
      clearTimeout(internals.relayToTraeLogCopyTimer)
      internals.relayToTraeLogCopyTimer = null
    }
    try {
      const response = await apiFetch(relayApi('clear-logs/'), {
        method: 'POST',
        credentials: 'include',
        headers: { Accept: 'application/json' },
      })
      if (!response.ok) {
        const data = await response.json().catch(() => ({}))
        const detail = String(data?.message || data?.detail || '').trim()
        relayToTraeMessage.value = detail
          ? `服务端日志清理失败：${detail}`
          : '服务端日志清理失败'
      }
    } catch (error) {
      console.error('relay clear-logs failed:', error)
      relayToTraeMessage.value = '服务端日志清理失败'
    }
  }

  const copyRelayToTraeLogs = async () => {
    const text = relayToTraeLogsText.value === '暂无日志' ? '' : relayToTraeLogsText.value
    if (!text) {
      return
    }
    try {
      if (navigator?.clipboard?.writeText) {
        await navigator.clipboard.writeText(text)
      } else {
        const ta = document.createElement('textarea')
        ta.value = text
        document.body.appendChild(ta)
        ta.select()
        document.execCommand('copy')
        document.body.removeChild(ta)
      }
      relayToTraeLogCopyState.value = 'copied'
      if (internals.relayToTraeLogCopyTimer) {
        clearTimeout(internals.relayToTraeLogCopyTimer)
      }
      internals.relayToTraeLogCopyTimer = window.setTimeout(() => {
        relayToTraeLogCopyState.value = 'idle'
        internals.relayToTraeLogCopyTimer = null
      }, 1500)
    } catch (error) {
      console.error('复制日志失败:', error)
    }
  }

  const acknowledgeStaleRepo = () => {
    staleRepoAcknowledged.value = true
  }

  const syncStaleRepoAddresses = async () => {
    const ctx = ctxIds()
    if (!ctx.tenant_id || !ctx.workspace_id || !ctx.task_id) {
      relayToTraeMessage.value = '缺少任务上下文，无法同步仓库地址'
      return
    }
    staleRepoSyncLoading.value = true
    relayToTraeMessage.value = ''
    try {
      const data = await syncTaskRepoAddressesFromProjects({
        apiFetch,
        tenantId: ctx.tenant_id,
        workspaceId: ctx.workspace_id,
        taskId: ctx.task_id,
        projects: props.task?.projects,
      })
      staleRepoAcknowledged.value = false
      emit('task-updated', data)
      relayToTraeMessage.value = '任务仓库地址已更新'
    } catch (error) {
      relayToTraeMessage.value = error?.message || '同步仓库地址失败'
    } finally {
      staleRepoSyncLoading.value = false
    }
  }

  const resetRelayForNewTask = () => {
    resetRelayToTraeLogSnapshot()
    relayToTraeLogs.value = []
    internals.relayToTraeLogSuppressedAfterClear = true
    internals.relayToTraeLogAwaitingFirstStatusAfterStart = true
    relayToTraeEnvItems.value = buildDefaultRelayToTraeEnvItems(route.query)
    isRelayToTraeRunning.value = false
    relayToTraeOrphanPort.value = false
    relayToTraeOnlineServiceUp.value = false
    relayToTraeUiUrl.value = ''
    stopRelayToTraePoll()
  }

  const resetStaleRepoAck = () => {
    staleRepoAcknowledged.value = false
  }

  const disposeRelayTimers = () => {
    stopRelayToTraePoll()
    if (internals.relayToTraeLogCopyTimer) {
      clearTimeout(internals.relayToTraeLogCopyTimer)
      internals.relayToTraeLogCopyTimer = null
    }
  }

  return {
    relayToTraeEnvItems,
    relayAccessTokenMaskedLabel,
    relayToTraeMessage,
    relayToTraeLogs,
    relayToTraeLogsExpanded,
    relayToTraeUiUrl,
    relayRuntimeMode,
    isRelayToTraeStarting,
    isRelayToTraeStopping,
    isRelayToTraeRunning,
    relayToTraeServiceOnline,
    isRelayToTraeStatusLoading,
    relayToTraeLogCopyState,
    relayToTraeDefaultApplied,
    relayToTraeRepoCredentialGuideVisible,
    staleRepoSyncLoading,
    relayToTraeLogsText,
    isRelayToTraeOnlineServiceUp,
    showRelayToTraeStopButton,
    staleRepoMismatches,
    startBlockedByStaleRepo,
    relayToTraeOnlineServiceStatusLabel,
    relayRuntimeDeps,
    effectiveRelayToTraeUiUrl,
    acknowledgeStaleRepo,
    syncStaleRepoAddresses,
    clearRelayToTraeLogOutput,
    copyRelayToTraeLogs,
    fetchRelayToTraeServiceStatus: statusApi.fetchRelayToTraeServiceStatus,
    startRelayToTrae: actionsApi.startRelayToTrae,
    stopRelayToTrae: actionsApi.stopRelayToTrae,
    fetchRelayToTraeStatusOnMount: actionsApi.fetchRelayToTraeStatusOnMount,
    loadRelayToTraeEnvDefaults: actionsApi.loadRelayToTraeEnvDefaults,
    applyRelayToTraeStatusFromSse: statusApi.applyRelayToTraeStatusFromSse,
    resetRelayForNewTask,
    resetStaleRepoAck,
    disposeRelayTimers,
    stopRelayToTraePoll,
  }
}

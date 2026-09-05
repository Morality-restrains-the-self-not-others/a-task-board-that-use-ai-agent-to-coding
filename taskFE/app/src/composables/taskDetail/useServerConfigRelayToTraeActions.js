import { apiFetch } from '../../utils/apiUtils.js'
import {
  buildDefaultRelayToTraeEnvItems,
  buildRelayToTraeStartEnvPayload,
  buildRelayToTraeTokenInitPayload,
  TASK2APP_ACCESS_TOKEN_PLACEHOLDER,
} from '../../utils/relayToTraeUtils.js'
import {
  repoCredentialGuideMessage,
  resolveRelayToTraeErrorMessage,
  summarizeMissingRepoCredentialsForUi,
  summarizeTokenRefreshFailuresForUi,
} from '../../utils/serverConfigRelayToTraeErrors.js'
import { resolveServerConfigTaskId } from '../../utils/serverConfigRouteHelpers.js'

/**
 * relay-to-trae 启动/停止与环境变量预加载。
 */
export function createServerConfigRelayToTraeActions(ctx) {
  const {
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
    fetchRelayToTraeServiceStatus,
    resumeContainerHeartbeatForRelayStart,
    pauseContainerHeartbeatForRelayStop,
    internals,
  } = ctx

  const loadRelayToTraeEnvDefaults = async () => {
    const taskId = resolveServerConfigTaskId({
      route,
      task: props.task,
      taskId: props.taskId,
      tenantId: props.tenantId,
      workspaceId: props.workspaceId,
    })
    if (!taskId || !isRelayToTraeEnabled.value) {
      return
    }
    try {
      const params = new URLSearchParams({ task_id: String(taskId) })
      const response = await apiFetch(`${relayApi('env-prepare/')}?${params.toString()}`, {
        credentials: 'include',
        headers: { Accept: 'application/json' },
      })
      const data = await response.json().catch(() => ({}))
      if (!response.ok || data.status === 'error') {
        return
      }
      const envObj = data.env && typeof data.env === 'object' ? data.env : {}
      const merged = buildDefaultRelayToTraeEnvItems(route.query)
      for (const item of merged) {
        if (item.key === 'ACCESS_TOKEN') {
          item.value = TASK2APP_ACCESS_TOKEN_PLACEHOLDER
          continue
        }
        const fromApi = String(envObj[item.key] ?? '').trim()
        if (fromApi) {
          item.value = fromApi
        }
      }
      relayToTraeEnvItems.value = merged
    } catch (error) {
      console.error('加载 relayToTrae 默认环境变量失败:', error)
    }
  }

  const startRelayToTrae = async () => {
    if (effectiveStartBlockedByOAuthCheckLoading.value) {
      relayToTraeMessage.value = '正在检测仓库 OAuth 绑定状态，请稍候…'
      return
    }
    if (effectiveStartBlockedByUnboundOAuth.value) {
      relayToTraeRepoCredentialGuideVisible.value = true
      relayToTraeMessage.value = repoCredentialGuideMessage
      return
    }
    if (!selectedImageId.value) {
      relayToTraeMessage.value = '请先选择镜像'
      return
    }
    if (envParamsSourceRequiredHint.value) {
      relayToTraeMessage.value = envParamsSourceRequiredHint.value
      return
    }
    try {
      isRelayToTraeStarting.value = true
      relayRuntimeMode.value = 'selected_image'
      relayToTraeRepoCredentialGuideVisible.value = false
      relayToTraeMessage.value = '正在初始化启动凭证…'
      resetRelayToTraeLogSnapshot()
      relayToTraeLogs.value = []
      internals.relayToTraeLogSuppressedAfterClear = true
      internals.relayToTraeLogAwaitingFirstStatusAfterStart = true
      relayToTraeUiUrl.value = ''
      resumeContainerHeartbeatForRelayStart?.()
      const tokenInitBuilt = buildRelayToTraeTokenInitPayload(
        relayToTraeEnvItems.value,
        TASK2APP_ACCESS_TOKEN_PLACEHOLDER,
      )
      if (!tokenInitBuilt.ok) {
        relayToTraeRepoCredentialGuideVisible.value = false
        relayToTraeMessage.value = `请填写 ${tokenInitBuilt.missingKey}`
        isRelayToTraeStarting.value = false
        return
      }
      const tokenInitResponse = await apiFetch(relayApi('token-init/'), {
        method: 'POST',
        credentials: 'include',
        headers: {
          Accept: 'application/json',
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          ...tokenInitBuilt.payload,
          ...ctxIds(),
        }),
      })
      const tokenInitData = await tokenInitResponse.json().catch(() => ({}))
      if (!tokenInitResponse.ok || tokenInitData.status === 'error') {
        relayToTraeRepoCredentialGuideVisible.value = false
        relayToTraeMessage.value = resolveRelayToTraeErrorMessage(
          tokenInitData,
          'relayToTrae token 初始化失败',
        )
        isRelayToTraeStarting.value = false
        return
      }
      relayToTraeMessage.value = '正在预检仓库克隆凭证…'
      const precheckResponse = await apiFetch(relayApi('repo-credentials-precheck/'), {
        method: 'POST',
        credentials: 'include',
        headers: {
          Accept: 'application/json',
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          env: tokenInitBuilt.payload.env,
          ...ctxIds(),
        }),
      })
      const precheckData = await precheckResponse.json().catch(() => ({}))
      if (!precheckResponse.ok || precheckData.status === 'error') {
        const tokenRefreshFailed =
          precheckResponse.status === 502 &&
          String(precheckData?.error_code || '').trim() === 'REPO_CLONE_TOKEN_REFRESH_FAILED'
        const hasMissingRepoCredentials =
          precheckResponse.status === 409 &&
          Array.isArray(precheckData?.missing_repo_credentials) &&
          precheckData.missing_repo_credentials.length > 0
        relayToTraeRepoCredentialGuideVisible.value = hasMissingRepoCredentials && !tokenRefreshFailed
        const missingBrief = summarizeMissingRepoCredentialsForUi(precheckData)
        const tokenRefreshBrief = summarizeTokenRefreshFailuresForUi(precheckData)
        const baseMessage = resolveRelayToTraeErrorMessage(
          precheckData,
          '仓库克隆凭证预检失败',
        )
        if (tokenRefreshFailed) {
          relayToTraeMessage.value = tokenRefreshBrief
            ? `${baseMessage}：${tokenRefreshBrief}`
            : baseMessage
        } else if (hasMissingRepoCredentials) {
          relayToTraeMessage.value = missingBrief
            ? `${baseMessage}（${missingBrief}）。${repoCredentialGuideMessage}`
            : `${baseMessage}。${repoCredentialGuideMessage}`
        } else if (precheckResponse.status >= 500) {
          relayToTraeMessage.value =
            `${baseMessage}。任务 API 不可达或网关异常；请确认 Django 服务（如 8001）已运行。`
        } else {
          relayToTraeMessage.value = baseMessage
        }
        isRelayToTraeStarting.value = false
        return
      }
      relayToTraeRepoCredentialGuideVisible.value = false
      relayToTraeMessage.value = '启动请求已受理，等待状态回传…'
      const built = buildRelayToTraeStartEnvPayload(relayToTraeEnvItems.value, TASK2APP_ACCESS_TOKEN_PLACEHOLDER)
      if (!built.ok) {
        relayToTraeRepoCredentialGuideVisible.value = false
        relayToTraeMessage.value = `请填写 ${built.missingKey}`
        isRelayToTraeStarting.value = false
        return
      }
      const response = await apiFetch(relayApi('start/'), {
        method: 'POST',
        credentials: 'include',
        headers: {
          Accept: 'application/json',
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          env: built.env,
          installed_image_id: selectedImageId.value,
          ...ctxIds(),
          feature_params_source: featureParamsSource.value,
          personal_feature_params_config_id:
            featureParamsSource.value === 'personal' ? selectedPersonalConfigId.value : '',
        }),
      })
      const data = await response.json().catch(() => ({}))
      if (!response.ok || data.status === 'error') {
        relayToTraeRepoCredentialGuideVisible.value = false
        relayToTraeMessage.value = resolveRelayToTraeErrorMessage(data, 'relayToTrae 启动失败')
        isRelayToTraeStarting.value = false
        return
      }
      if (String(data.status || '').toLowerCase() !== 'accepted') {
        relayToTraeRepoCredentialGuideVisible.value = false
        relayToTraeMessage.value = '启动请求未被受理'
        isRelayToTraeStarting.value = false
        return
      }
      isRelayToTraeRunning.value = false
      relayToTraeOnlineServiceUp.value = false
      relayToTraeServiceOnline.value = true
      relayToTraeUiUrl.value = ''
      relayRuntimeMode.value = 'selected_image'
      relayToTraeMessage.value = data.request_id
        ? `启动请求已受理（request_id: ${data.request_id}），等待状态回传`
        : '启动请求已受理，等待状态回传'
      resumeContainerHeartbeatForRelayStart?.()
      startRelayToTraePoll()
    } catch (error) {
      console.error('relayToTrae 启动失败:', error)
      relayToTraeRepoCredentialGuideVisible.value = false
      relayToTraeMessage.value = 'relayToTrae 启动失败，请确认本机 relayToTrae 服务已运行'
      isRelayToTraeStarting.value = false
    }
  }

  const stopRelayToTrae = async () => {
    try {
      isRelayToTraeStopping.value = true
      relayToTraeMessage.value = '正在停止 onlineServiceJS…'
      const response = await apiFetch(relayApi('stop/'), {
        method: 'POST',
        credentials: 'include',
        headers: {
          Accept: 'application/json',
          'Content-Type': 'application/json',
        },
      })
      const data = await response.json().catch(() => ({}))
      if (!response.ok || data.status === 'error') {
        relayToTraeMessage.value = resolveRelayToTraeErrorMessage(data, '停止失败')
        return
      }
      isRelayToTraeRunning.value = false
      isRelayToTraeStarting.value = false
      relayToTraeOrphanPort.value = false
      relayToTraeOnlineServiceUp.value = false
      relayToTraeUiUrl.value = ''
      relayRuntimeMode.value = ''
      relayToTraeServiceOnline.value = true
      relayToTraeMessage.value = '已停止 onlineServiceJS'
      stopRelayToTraePoll()
      pauseContainerHeartbeatForRelayStop?.()
      await fetchRelayToTraeServiceStatus({ preserveRelayOnline: true })
    } catch (error) {
      console.error('relayToTrae 停止失败:', error)
      relayToTraeMessage.value = '停止失败，请稍后重试'
    } finally {
      isRelayToTraeStopping.value = false
    }
  }

  const fetchRelayToTraeStatusOnMount = async () => {
    if (!isRelayToTraeEnabled.value) {
      return
    }
    resetRelayToTraeLogSnapshot()
    relayToTraeLogs.value = []
    internals.relayToTraeLogSuppressedAfterClear = false
    internals.relayToTraeLogAwaitingFirstStatusAfterStart = false
    relayToTraeEnvItems.value = buildDefaultRelayToTraeEnvItems(route.query)
    await loadRelayToTraeEnvDefaults()
    await fetchRelayToTraeServiceStatus()
  }

  return {
    startRelayToTrae,
    stopRelayToTrae,
    loadRelayToTraeEnvDefaults,
    fetchRelayToTraeStatusOnMount,
  }
}

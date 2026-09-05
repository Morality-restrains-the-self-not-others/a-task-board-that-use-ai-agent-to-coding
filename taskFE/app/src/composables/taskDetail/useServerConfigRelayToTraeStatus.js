import { apiFetch } from '../../utils/apiUtils.js'
import {
  TASK2APP_ACCESS_TOKEN_PLACEHOLDER,
  shouldApplyRelayToTraeStatusLogs,
  parseRelayToTraeEnv,
  mapRelayToTraeStatusErrorMessage,
  buildRelayToTraeStatusSearchParams,
  advanceRelayStatusLogCursor,
} from '../../utils/relayToTraeUtils.js'
import { resolveRelayToTraeErrorMessage } from '../../utils/serverConfigRelayToTraeErrors.js'

/**
 * relay-to-trae /status 拉取、payload 应用与 register。
 */
export function createServerConfigRelayToTraeStatusApi(ctx) {
  const {
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
  } = ctx

  const checkRelayToTraeHealth = async () => {
    try {
      const response = await apiFetch(relayApi('health/'), {
        credentials: 'include',
        headers: { Accept: 'application/json' },
      })
      return response.ok
    } catch {
      return false
    }
  }

  const applyRelayToTraeOfflineState = () => {
    isRelayToTraeRunning.value = false
    isRelayToTraeStarting.value = false
    relayToTraeOrphanPort.value = false
    relayToTraeOnlineServiceUp.value = false
    relayToTraeUiUrl.value = ''
    relayToTraeServiceOnline.value = false
    relayToTraeMessage.value =
      '无法连接本机 relayToTrae，请先启动 go-relay（runAll 或 go_relayToTrae/bin/go_relayToTrae，默认 http://127.0.0.1:8797）'
  }

  const isRelayStatusPayloadLike = (data) => {
    if (!data || typeof data !== 'object' || Array.isArray(data)) {
      return false
    }
    const candidateKeys = [
      'running',
      'online_service_up',
      'orphan_port',
      'ui_url',
      'error',
      'logs',
      'next_cursor',
    ]
    return candidateKeys.some((key) => Object.prototype.hasOwnProperty.call(data, key))
  }

  const applyRelayToTraeStatusPayload = (data, options = {}) => {
    if (!data || typeof data !== 'object') {
      return { trackedRunning: false, onlineUp: false, orphan: false, shouldRefetchFromZero: false }
    }
    if (
      Array.isArray(data.logs)
      && data.logs.length > 0
      && shouldApplyRelayToTraeStatusLogs(ctxIds().task_id, data)
    ) {
      appendRelayToTraeLogs(data.logs)
    }
    const requestedCursor =
      options.requestedCursor != null ? options.requestedCursor : internals.relayToTraeStatusLogCursor
    const advanced = advanceRelayStatusLogCursor(
      internals.relayToTraeStatusLogCursor,
      data,
      requestedCursor,
    )
    internals.relayToTraeStatusLogCursor = advanced.cursor
    const trackedRunning = Boolean(data.running)
    const portListening = Boolean(data.port_listening)
    const orphan = Boolean(data.orphan_port)
    if (typeof data.mode === 'string' && data.mode.trim()) {
      relayRuntimeMode.value = data.mode.trim()
    } else if (!trackedRunning && !portListening) {
      relayRuntimeMode.value = ''
    }
    const onlineUp = portListening
    isRelayToTraeRunning.value = trackedRunning && onlineUp
    relayToTraeOnlineServiceUp.value = onlineUp
    relayToTraeOrphanPort.value = orphan
    if (portListening || data.error || !trackedRunning) {
      isRelayToTraeStarting.value = false
    }
    if (data.ui_url && portListening) {
      relayToTraeUiUrl.value = String(data.ui_url)
    } else if (!portListening) {
      relayToTraeUiUrl.value = ''
    }
    if (data.error) {
      isRelayToTraeStarting.value = false
      relayToTraeMessage.value = mapRelayToTraeStatusErrorMessage(data.error)
    } else if (relayToTraeOrphanPort.value) {
      relayToTraeMessage.value = '检测到 onlineServiceJS 端口仍被占用，可点击「停止」释放'
    } else if (trackedRunning && effectiveRelayToTraeUiUrl.value) {
      isRelayToTraeStarting.value = false
      relayToTraeMessage.value = `容器页面：${effectiveRelayToTraeUiUrl.value}`
    } else if (trackedRunning && relayRuntimeMode.value === 'selected_image') {
      isRelayToTraeStarting.value = false
      relayToTraeMessage.value = '容器已启动，等待 register-reachability 登记 server_url…'
    } else if (trackedRunning && data.ui_url) {
      isRelayToTraeStarting.value = false
      relayToTraeMessage.value = `onlineServiceJS 运行中：${effectiveRelayToTraeUiUrl.value || data.ui_url}`
    } else if (trackedRunning) {
      isRelayToTraeStarting.value = false
      relayToTraeMessage.value = 'onlineServiceJS 运行中'
    } else if (!orphan) {
      const msg = String(relayToTraeMessage.value || '')
      if (!msg || msg === '已停止 onlineServiceJS' || msg.includes('onlineServiceJS 已停止')) {
        relayToTraeMessage.value = 'onlineServiceJS 已停止，relay 服务在线'
      }
    }
    if (
      portListening
      || (
        Array.isArray(data.logs)
        && data.logs.length > 0
        && shouldApplyRelayToTraeStatusLogs(ctxIds().task_id, data)
      )
    ) {
      stopRelayToTraePoll()
    }
    return {
      trackedRunning,
      onlineUp,
      orphan,
      shouldRefetchFromZero: advanced.shouldRefetchFromZero,
    }
  }

  const applyRelayToTraeStatusFromSse = (statusData) => {
    const payload =
      statusData?.relay_payload && typeof statusData.relay_payload === 'object'
        ? statusData.relay_payload
        : statusData
    relayToTraeServiceOnline.value = true
    return applyRelayToTraeStatusPayload(payload)
  }

  const registerRelayToTraeTask = async ({ issueTokenOnServer = false, registerOnly = false } = {}) => {
    const ids = ctxIds()
    if (!ids.task_id) {
      return false
    }
    const env = parseRelayToTraeEnv(relayToTraeEnvItems.value)
    const taskOrigin = String(env.TASK_API_ENDPOINT_ORIGIN || '').trim()
    const accessToken = String(env.ACCESS_TOKEN || '').trim()
    if (!taskOrigin) {
      return false
    }
    if (!issueTokenOnServer && !registerOnly) {
      if (!accessToken || accessToken === TASK2APP_ACCESS_TOKEN_PLACEHOLDER) {
        return false
      }
    }
    const tokenForRegister = issueTokenOnServer
      ? accessToken || TASK2APP_ACCESS_TOKEN_PLACEHOLDER
      : accessToken
    const body = {
      tenant_id: ids.tenant_id,
      workspace_id: ids.workspace_id,
      task_id: ids.task_id,
      task_api_endpoint_origin: taskOrigin,
    }
    if (registerOnly) {
      body.register_only = true
    } else {
      body.access_token = tokenForRegister
    }
    try {
      const response = await apiFetch(relayApi('register/'), {
        method: 'POST',
        credentials: 'include',
        headers: {
          Accept: 'application/json',
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(body),
      })
      if (!response.ok) {
        if (response.status === 401 || response.status === 502) {
          const errBody = await response.json().catch(() => ({}))
          relayToTraeMessage.value = resolveRelayToTraeErrorMessage(
            errBody,
            'relayToTrae 鉴权或转发失败，请检查服务端 RELAY_TO_TRAE 配置',
          )
        }
        return false
      }
      return true
    } catch (error) {
      console.error('登记 relayToTrae 任务失败:', error)
      return false
    }
  }

  const shouldPreserveRelayOnlineOnProbeFailure = (explicitPreserve = false) => {
    if (explicitPreserve && relayToTraeServiceOnline.value) {
      return true
    }
    return (
      relayToTraeServiceOnline.value &&
      !relayToTraeOnlineServiceUp.value &&
      !isRelayToTraeRunning.value &&
      !isRelayToTraeStarting.value
    )
  }

  const fetchRelayToTraeServiceStatus = async (options = {}) => {
    const { preserveRelayOnline = false, forceCursorZero = false } = options
    if (!isRelayToTraeEnabled.value) {
      return
    }
    if (forceCursorZero) {
      internals.relayToTraeStatusLogCursor = 0
    }
    const viewerTaskId = String(ctxIds().task_id || '').trim()
    const requestedCursor = forceCursorZero ? 0 : internals.relayToTraeStatusLogCursor
    const statusQuery = buildRelayToTraeStatusSearchParams({
      taskId: viewerTaskId,
      cursor: requestedCursor,
    })
    const statusSuffix = statusQuery.toString() ? `status/?${statusQuery.toString()}` : 'status/'
    internals.relayStatusFetchGeneration += 1
    const generation = internals.relayStatusFetchGeneration
    const isCurrentFetch = () => generation === internals.relayStatusFetchGeneration
    isRelayToTraeStatusLoading.value = true
    try {
      const healthy = await checkRelayToTraeHealth()
      if (!isCurrentFetch()) {
        return
      }
      if (!healthy) {
        try {
          const fallbackStatusResponse = await apiFetch(relayApi(statusSuffix), {
            credentials: 'include',
            headers: { Accept: 'application/json' },
          })
          if (!isCurrentFetch()) {
            return
          }
          if (fallbackStatusResponse.ok) {
            relayToTraeServiceOnline.value = true
            const fallbackStatusData = await fallbackStatusResponse.json().catch(() => ({}))
            if (isRelayStatusPayloadLike(fallbackStatusData)) {
              const applied = applyRelayToTraeStatusPayload(fallbackStatusData, { requestedCursor })
              if (applied.shouldRefetchFromZero && !forceCursorZero && isCurrentFetch()) {
                await fetchRelayToTraeServiceStatus({
                  preserveRelayOnline,
                  forceCursorZero: true,
                })
              }
            }
            return
          }
        } catch {
          // ignore fallback failures
        }
        if (!isCurrentFetch()) {
          return
        }
        if (shouldPreserveRelayOnlineOnProbeFailure(preserveRelayOnline)) {
          return
        }
        applyRelayToTraeOfflineState()
        return
      }
      relayToTraeServiceOnline.value = true
      await registerRelayToTraeTask({ registerOnly: true })
      if (!isCurrentFetch()) {
        return
      }
      const statusResponse = await apiFetch(relayApi(statusSuffix), {
        credentials: 'include',
        headers: { Accept: 'application/json' },
      })
      if (!isCurrentFetch()) {
        return
      }
      if (statusResponse.ok) {
        const statusData = await statusResponse.json().catch(() => ({}))
        const applied = applyRelayToTraeStatusPayload(statusData, { requestedCursor })
        if (applied.shouldRefetchFromZero && !forceCursorZero && isCurrentFetch()) {
          await fetchRelayToTraeServiceStatus({
            preserveRelayOnline,
            forceCursorZero: true,
          })
        }
      }
    } finally {
      if (isCurrentFetch()) {
        isRelayToTraeStatusLoading.value = false
      }
    }
  }

  return {
    applyRelayToTraeStatusFromSse,
    fetchRelayToTraeServiceStatus,
  }
}

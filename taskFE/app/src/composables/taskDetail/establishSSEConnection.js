/**
 * Pure function: establishes SSE connection for TaskDetail server startup status.
 * All closure dependencies passed explicitly via `deps`.
 */
import { getApiUrl } from '../../utils/config.js'
import { nextTick } from 'vue'
import {
  applyContainerHeartbeatSsePayload,
  isContainerHeartbeatServing,
} from './applyContainerHeartbeatSse.js'
import { latestPerContainerHeartbeat, latestBindingAdvanced, latestContainerRunning } from './perContainerHeartbeatBus.js'
import { planGitPrReplyCreatedSseRefetch } from './taskGitPrReplySse.js'

/**
 * nginx HTML 502（或其它 HTML 5xx）才视为「平台推送侧不可用」，
 * 与云服务器生命周期（已停止/已启动）无关。
 * @param {{ status?: number, headers?: { get?: (name: string) => string | null } }} resp
 */
export function isPlatformRestartSseProbeResponse(resp) {
  const status = Number(resp?.status)
  if (!Number.isFinite(status)) return false
  const ct = String(resp?.headers?.get?.('content-type') || '').toLowerCase()
  return status === 502 || (status >= 500 && ct.includes('text/html'))
}

export function establishSSEConnection(taskId, deps) {
  const {
    sseConnection, sseLive, sseReconnecting, sseReconnectAttempts,
    closeSSEConnection, scheduleSSEReconnect,
    updateServerStatus,
    effectiveTenantId, effectiveWorkspaceId,
    relayAccessCode,
  } = deps

  if (!taskId) return
  const normalizedTaskId = String(taskId)

  // 防止同一任务重复建立连接，避免服务端出现重复订阅
  if (
    sseConnection.value &&
    deps.currentSseTaskId === normalizedTaskId &&
    (sseConnection.value.readyState === EventSource.CONNECTING || sseConnection.value.readyState === EventSource.OPEN)
  ) {
    return
  }

  // 建立新连接前，确保旧连接已关闭（但不重置重连计数器，保持重连状态）
  if (sseConnection.value) {
    sseConnection.value.close()
    sseConnection.value = null
  }
  if (deps.sseReconnectTimer) {
    clearTimeout(deps.sseReconnectTimer)
    deps.sseReconnectTimer = null
  }
  deps.currentSseTaskId = normalizedTaskId

  // 获取租户ID和工作空间ID
  const tenantId = effectiveTenantId.value
  const workspaceId = effectiveWorkspaceId.value
  if (!tenantId || !workspaceId) return

  const sseBaseUrl = getApiUrl(
    `/api/sse/server-startup-status/tenant_id/${tenantId}/workspace_id/${workspaceId}/task_id/${taskId}/`,
  )
  const accessCode = String(relayAccessCode?.value || "").trim()
  const sseUrl = accessCode
    ? `${sseBaseUrl}${sseBaseUrl.includes("?") ? "&" : "?"}accessCode=${encodeURIComponent(accessCode)}`
    : sseBaseUrl

  // 创建SSE连接
  const connectStartedAt = Date.now()
  let sseEverOpened = false
  /** @type {Set<string>} 本连接内已因 git_pr 回复 SSE 触发过重拉的 comment_id */
  const gitPrReplySeenCommentIds = new Set()
  sseConnection.value = new EventSource(sseUrl, { withCredentials: true })

  sseConnection.value.addEventListener('open', () => {
    sseEverOpened = true
    sseLive.value = true
    sseReconnecting.value = false
    sseReconnectAttempts.value = 0
    if (deps.ssePlatformRestartHint) deps.ssePlatformRestartHint.value = false
    if (deps.sseReconnectTimer) {
      clearTimeout(deps.sseReconnectTimer)
      deps.sseReconnectTimer = null
    }
    console.log('SSE 连接成功')
  })

  // 监听消息事件
  sseConnection.value.onmessage = (event) => {
    try {
      // OPT-20260724-023: 防御深度 — 检查当前连接 taskId 与 UI 当前 taskId 一致。
      // 当用户快速切换任务时，旧连接的事件可能在 closeSSEConnection() 关闭后
      // 仍触发（EventSource close 非同步），拒绝处理非当前任务的事件。
      if (String(deps.currentSseTaskId) !== normalizedTaskId) return
      const statusData = JSON.parse(event.data)
      if (statusData.type === 'heartbeat') {
        sseLive.value = true
        return
      }
      if (statusData.event_name === 'container_heartbeat') {
        if (deps.containerHeartbeatPaused?.value) {
          return
        }
        // OPT-20260724-022: 将携带 comment_id 的心跳事件路由到 per-binding 状态总线
        if (statusData.comment_id) {
          latestPerContainerHeartbeat.value = statusData
          // OPT-20260724-021: 容器 running 信号 — 通知 useCommentContainerBindings 触发 advance
          if (statusData.status === 'ok' && statusData.bidirectional_ok === true) {
            latestContainerRunning.value = statusData
          }
        }
        // 已释放/非服务态忽略；冷打开 endpoint 已登记时放行（勿仅依赖 isServerRunning）
        if (!isContainerHeartbeatServing(deps)) {
          deps.containerHeartbeatSseBuffer?.stash?.(statusData)
          return
        }
        applyContainerHeartbeatSsePayload(statusData, deps)
        return
      }
      // OPT-20260724-021: CommentContainerBindingAdvanced 领域事件 —
      // 后端 advanceCommentContainerBindings 完成后推送，前端本地更新 binding 状态
      if (statusData.event_name === 'comment_container_binding_advanced') {
        latestBindingAdvanced.value = statusData
        return
      }
      // git_pr 子评论创建后：重拉 Feed，无需整页刷新
      {
        const plan = planGitPrReplyCreatedSseRefetch(statusData, gitPrReplySeenCommentIds)
        if (plan.shouldRefetch && typeof deps.onTaskGitPrReplyCreated === 'function') {
          void deps.onTaskGitPrReplyCreated(statusData)
          return
        }
        if (statusData.event_name === 'task_git_pr_reply_created') {
          return
        }
      }
      if (statusData.type === 'close') {
        const reopenTaskId = String(normalizedTaskId || '')
        closeSSEConnection()
        if (reopenTaskId) {
          void nextTick(() => deps.establishSSEConnectionSelf(reopenTaskId))
        }
        return
      }
      updateServerStatus(statusData)
    } catch (error) {
      console.error('解析SSE消息出错:', error)
    }
  }

  // 监听错误事件：触发断线重连；仅确认 nginx HTML 502/5xx 时提示「状态推送服务暂不可用」
  // （勿用短耗时失败直接置位——易与「服务器启动状态=已停止」混淆成假阳性）
  sseConnection.value.onerror = (error) => {
    sseLive.value = false
    console.error('SSE连接错误:', error)
    const elapsedMs = Date.now() - connectStartedAt
    const maybePlatformRestart = !sseEverOpened && elapsedMs < 2000
    if (sseConnection.value) {
      sseConnection.value.close()
      sseConnection.value = null
    }
    if (maybePlatformRestart && deps.ssePlatformRestartHint) {
      // EventSource 不暴露状态码；异步探测，仅 502/HTML 5xx 才置位，否则清除
      void (async () => {
        try {
          const resp = await fetch(sseUrl, {
            method: 'GET',
            credentials: 'include',
            headers: { Accept: 'text/event-stream' },
            signal: AbortSignal.timeout?.(1500),
          })
          if (isPlatformRestartSseProbeResponse(resp)) {
            deps.ssePlatformRestartHint.value = true
          } else {
            deps.ssePlatformRestartHint.value = false
          }
        } catch {
          // 探测失败：不标「平台重启」，避免与云服务器生命周期文案冲突
          deps.ssePlatformRestartHint.value = false
        }
      })()
    }
    scheduleSSEReconnect(normalizedTaskId)
  }

  // 监听关闭事件：触发断线重连
  sseConnection.value.onclose = () => {
    sseLive.value = false
    if (sseConnection.value) {
      sseConnection.value = null
    }
    scheduleSSEReconnect(normalizedTaskId)
  }
}

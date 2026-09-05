import { apiFetch } from '../../utils/apiUtils.js'
import { extractTraceId } from '../../utils/traceId.js'
import { stopVmBodyWithCommentId, trimCommentId, withCommentIdQuery } from '../../utils/cloudComputeCommentQuery.js'
import { resolveRuntimeStatusOnAuthError } from '../../utils/serverRuntimeAuthAlign.js'
import { isServerRuntimeAbsentPayload } from '../../utils/serverRuntimeAbsent.js'
import { resolveServerConfigTaskId } from '../../utils/serverConfigRouteHelpers.js'
import { notifyRuntimeAbsentHydrate, notifyRuntimeHydrate } from './serverRuntimeHydrate.js'

function hasComputeCommentId(commentId) {
  return Boolean(trimCommentId(commentId))
}

function resolveComputeTaskId(props, route) {
  return resolveServerConfigTaskId({
    route,
    task: props.task,
    taskId: props.taskId,
    tenantId: props.tenantId,
    workspaceId: props.workspaceId,
  })
}

export async function fetchServerContentApi({
  props,
  route,
  resolveRuntimeApiWorkspaceId,
  serverContentMessage,
  serverContentMessageTraceId,
  serverContentText,
  serverContentTargetUrl,
  isServerContentLoading,
  commentId,
}) {
  const taskId = resolveComputeTaskId(props, route)
  if (!taskId) {
    serverContentMessage.value = '缺少任务ID'
    if (serverContentMessageTraceId) serverContentMessageTraceId.value = ''
    serverContentText.value = ''
    serverContentTargetUrl.value = ''
    return
  }
  if (!hasComputeCommentId(commentId)) {
    serverContentMessage.value = '缺少评论ID'
    if (serverContentMessageTraceId) serverContentMessageTraceId.value = ''
    serverContentText.value = ''
    serverContentTargetUrl.value = ''
    return
  }
  try {
    isServerContentLoading.value = true
    const tenantId = route.params.tenant
    const workspaceId = resolveRuntimeApiWorkspaceId()
    const response = await apiFetch(
      withCommentIdQuery(
        `/api/cloud/compute/server-content/tenant_id/${tenantId}/workspace_id/${workspaceId}?task_id=${taskId}`,
        commentId,
      ),
      {
        credentials: 'include',
        headers: {
          Accept: 'application/json',
        },
      },
    )
    const data = await response.json()
    const tid = extractTraceId(response) || extractTraceId(data) || ''
    if (!response.ok || data.status === 'error') {
      serverContentMessage.value = data.message || '拉取服务器内容失败'
      if (serverContentMessageTraceId) serverContentMessageTraceId.value = tid
      serverContentText.value = ''
      serverContentTargetUrl.value = data.target_url || ''
      return
    }
    // OPT-20260812-042：公网 IP 已回填但容器业务端口尚未就绪（connection refused），
    // 后端返回 status=starting —— 降级展示「启动中」而非硬错误。
    if (data.status === 'starting') {
      serverContentMessage.value = data.message || '容器业务端口尚未就绪，正在启动中'
      if (serverContentMessageTraceId) serverContentMessageTraceId.value = tid
      serverContentText.value = ''
      serverContentTargetUrl.value = data.target_url || ''
      return
    }
    if (serverContentMessageTraceId) serverContentMessageTraceId.value = ''
    serverContentMessage.value = data.truncated
      ? `拉取成功（内容过长，已截断展示）HTTP ${data.http_status}`
      : `拉取成功 HTTP ${data.http_status}`
    serverContentText.value = data.content || ''
    serverContentTargetUrl.value = data.target_url || ''
  } catch (error) {
    console.error('拉取服务器内容出错:', error)
    serverContentMessage.value = '拉取服务器内容失败'
    if (serverContentMessageTraceId) {
      serverContentMessageTraceId.value = extractTraceId(error) || ''
    }
    serverContentText.value = ''
    serverContentTargetUrl.value = ''
  } finally {
    isServerContentLoading.value = false
  }
}

export async function fetchServerStartHistoryApi({
  props,
  route,
  resolveRuntimeApiWorkspaceId,
  serverStartHistoryMessage,
  serverStartHistoryMessageTraceId,
  serverStartHistoryRecords,
  isServerStartHistoryLoading,
}) {
  const taskId = resolveComputeTaskId(props, route)
  if (!taskId) {
    serverStartHistoryMessage.value = '缺少任务ID'
    if (serverStartHistoryMessageTraceId) serverStartHistoryMessageTraceId.value = ''
    serverStartHistoryRecords.value = []
    return
  }
  try {
    isServerStartHistoryLoading.value = true
    const tenantId = route.params.tenant
    const workspaceId = resolveRuntimeApiWorkspaceId()
    const response = await apiFetch(
      `/api/cloud/compute/server-start-history/tenant_id/${tenantId}/workspace_id/${workspaceId}?task_id=${taskId}`,
      {
        credentials: 'include',
        headers: {
          Accept: 'application/json',
        },
      },
    )
    const data = await response.json()
    const tid = extractTraceId(response) || extractTraceId(data) || ''
    if (!response.ok || data.status === 'error') {
      serverStartHistoryMessage.value = data.message || '获取历史服务器启动记录失败'
      if (serverStartHistoryMessageTraceId) serverStartHistoryMessageTraceId.value = tid
      serverStartHistoryRecords.value = []
      return
    }
    if (serverStartHistoryMessageTraceId) serverStartHistoryMessageTraceId.value = ''
    const records = Array.isArray(data.records) ? data.records : []
    serverStartHistoryRecords.value = records
    // 空成功不写 message：由面板「暂无历史服务器启动记录」空态单条展示，避免双文案（OPT-20260811-054）
    serverStartHistoryMessage.value = records.length > 0 ? `共 ${records.length} 条历史记录` : ''
  } catch (error) {
    console.error('获取历史服务器启动记录出错:', error)
    serverStartHistoryMessage.value = '获取历史服务器启动记录失败'
    if (serverStartHistoryMessageTraceId) {
      serverStartHistoryMessageTraceId.value = extractTraceId(error) || ''
    }
    serverStartHistoryRecords.value = []
  } finally {
    isServerStartHistoryLoading.value = false
  }
}

export async function openWorkbenchLinkApi({
  props,
  route,
  resolveRuntimeApiWorkspaceId,
  serverRuntimeStatusMessage,
  serverRuntimeStatusTraceId,
  isWorkbenchLinkLoading,
  openWindow,
  commentId,
}) {
  const taskId = resolveComputeTaskId(props, route)
  if (!taskId) {
    serverRuntimeStatusMessage.value = '缺少任务ID，无法打开 Workbench'
    if (serverRuntimeStatusTraceId) serverRuntimeStatusTraceId.value = ''
    return
  }
  if (!hasComputeCommentId(commentId)) {
    serverRuntimeStatusMessage.value = '缺少评论ID'
    if (serverRuntimeStatusTraceId) serverRuntimeStatusTraceId.value = ''
    return
  }
  const open = typeof openWindow === 'function'
    ? openWindow
    : (url) => {
        if (typeof window !== 'undefined' && typeof window.open === 'function') {
          window.open(url, '_blank', 'noopener,noreferrer')
        }
      }
  try {
    isWorkbenchLinkLoading.value = true
    const tenantId = route.params.tenant
    const workspaceId = resolveRuntimeApiWorkspaceId()
    const response = await apiFetch(
      withCommentIdQuery(
        `/api/cloud/compute/workbench-link/tenant_id/${tenantId}/workspace_id/${workspaceId}?task_id=${taskId}`,
        commentId,
      ),
      {
        credentials: 'include',
        headers: {
          Accept: 'application/json',
        },
      },
    )
    const data = await response.json().catch(() => ({}))
    const tid = extractTraceId(response) || extractTraceId(data) || ''
    if (!response.ok || data.status === 'error') {
      serverRuntimeStatusMessage.value = data.message || '获取 Workbench 链接失败'
      if (serverRuntimeStatusTraceId) serverRuntimeStatusTraceId.value = tid
      return
    }
    if (!data.workbench_url) {
      serverRuntimeStatusMessage.value = '未获取到有效的 Workbench 链接'
      if (serverRuntimeStatusTraceId) serverRuntimeStatusTraceId.value = tid
      return
    }
    open(data.workbench_url)
  } catch (error) {
    console.error('打开 Workbench 失败:', error)
    serverRuntimeStatusMessage.value = '打开 Workbench 失败，请稍后重试'
    if (serverRuntimeStatusTraceId) {
      serverRuntimeStatusTraceId.value = extractTraceId(error) || ''
    }
  } finally {
    isWorkbenchLinkLoading.value = false
  }
}

export async function stopServerApi({
  props,
  route,
  resolveRuntimeApiWorkspaceId,
  fetchServerRuntimeStatus,
  fetchServerStartHistory,
  commentId,
}) {
  const taskId = resolveComputeTaskId(props, route)
  if (!taskId) {
    return
  }
  if (!hasComputeCommentId(commentId)) {
    return
  }
  try {
    const tenantId = route.params.tenant
    const workspaceId = resolveRuntimeApiWorkspaceId()
    const response = await apiFetch(
      withCommentIdQuery(
        `/api/cloud/compute/stop-vm/tenant_id/${tenantId}/workspace_id/${workspaceId}`,
        commentId,
      ),
      {
      method: 'POST',
      credentials: 'include',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(stopVmBodyWithCommentId(taskId, commentId)),
    })
    if (!response.ok) {
      const result = await response.json().catch(() => ({}))
      console.error('停止服务器失败:', result.message)
      if (props.updateServerStatus) {
        props.updateServerStatus({
          status: 'error',
          message: result.message || '停止服务器失败',
          progress: 0,
        })
      }
      return
    }
    void fetchServerRuntimeStatus()
    void fetchServerStartHistory()
  } catch (error) {
    console.error('停止服务器出错:', error)
    if (props.updateServerStatus) {
      props.updateServerStatus({
        status: 'error',
        message: '停止服务器失败，请检查网络连接或联系管理员',
        progress: 0,
      })
    }
  }
}

export async function fetchServerRuntimeStatusApi({
  props,
  route,
  resolveRuntimeApiWorkspaceId,
  commentId,
  serverRuntimeStatus,
  serverRuntimeStatusMessage,
  serverRuntimeStatusTraceId,
  serverRuntimeStatusResponse,
  isServerRuntimeStatusLoading,
  applyDefaultServerTabByRuntime,
}) {
  const taskId = resolveComputeTaskId(props, route)
  if (!taskId) {
    serverRuntimeStatus.value = ''
    serverRuntimeStatusMessage.value = '缺少任务ID'
    serverRuntimeStatusTraceId.value = ''
    return
  }
  if (!hasComputeCommentId(commentId)) {
    serverRuntimeStatus.value = ''
    serverRuntimeStatusMessage.value = '缺少评论ID'
    serverRuntimeStatusTraceId.value = ''
    return
  }
  try {
    isServerRuntimeStatusLoading.value = true
    const tenantId = route.params.tenant
    const workspaceId = resolveRuntimeApiWorkspaceId()
    const response = await apiFetch(
      withCommentIdQuery(
        `/api/cloud/compute/server-runtime-status/tenant_id/${tenantId}/workspace_id/${workspaceId}?task_id=${taskId}`,
        commentId,
      ),
      {
        credentials: 'include',
        headers: {
          Accept: 'application/json',
        },
      },
    )
    const data = await response.json()
    const reqTraceId = extractTraceId(response) || extractTraceId(data) || ''
    serverRuntimeStatusTraceId.value = reqTraceId
    if (!response.ok || data.status === 'error') {
      const errMsg = data.message || '查询服务器运行状态失败'
      const aligned = resolveRuntimeStatusOnAuthError({
        isServerRunning: props.isServerRunning,
        errMsg,
      })
      if (aligned) {
        serverRuntimeStatus.value = aligned.runtimeStatus
        serverRuntimeStatusMessage.value = aligned.message
        serverRuntimeStatusResponse.value = {
          status: 'error',
          runtime_status: aligned.runtimeStatus,
          message: aligned.message,
          auth_missing: true,
          trace_id: reqTraceId || undefined,
        }
        applyDefaultServerTabByRuntime()
        return
      }
      serverRuntimeStatus.value = ''
      serverRuntimeStatusMessage.value = errMsg
      serverRuntimeStatusResponse.value = null
      applyDefaultServerTabByRuntime()
      return
    }
    serverRuntimeStatusResponse.value = data
    serverRuntimeStatus.value = data.runtime_status || data.instance_attribute?.body?.Status || ''
    serverRuntimeStatusMessage.value = data.message || ''
    applyDefaultServerTabByRuntime()
    if (serverRuntimeStatus.value) {
      notifyRuntimeHydrate(props.updateServerStatus, serverRuntimeStatus.value)
    } else if (props.isServerRunning && isServerRuntimeAbsentPayload(data)) {
      notifyRuntimeAbsentHydrate(props.updateServerStatus)
    }
  } catch (error) {
    console.error('获取服务器运行状态出错:', error)
    serverRuntimeStatus.value = ''
    serverRuntimeStatusMessage.value = '获取服务器运行状态失败'
    serverRuntimeStatusTraceId.value = extractTraceId(error) || ''
    serverRuntimeStatusResponse.value = null
    applyDefaultServerTabByRuntime()
  } finally {
    isServerRuntimeStatusLoading.value = false
  }
}

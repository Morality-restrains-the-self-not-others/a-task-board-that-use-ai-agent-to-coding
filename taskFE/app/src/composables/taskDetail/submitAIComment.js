/**
 * Pure function: submits an AI comment for the TaskDetail page.
 */
import { getCookie } from '../../utils/cookieUtils.js'
import { resolveLayerGraphCommandPayload } from '../../utils/resolveLayerGraphCommandPayload.js'
import { showRequestError } from '../../utils/requestErrorDisplay.js'
import { messageFromFailedResponse } from '../../utils/httpError.js'
import { handleActionChunkLoadError } from '../../utils/chunkLoadGuard.js'
import { getActiveToken } from '../../domain/auth/services/saved_accounts_store.js'

export async function submitAIComment(deps) {
  const {
    effectiveTenantId, effectiveWorkspaceId, effectiveTaskId,
    selectedLayerGraphNode, layerGraphSnapshot,
    layerGraphAutoIterationCount, layerGraphSelectedModel,
    layerGraphCommandKind, layerGraphModelProvider,
    newComment, commentComposerChips, activeAiInstructId,
    aiStreamBuffer, aiStreamBusy, localTask,
    normalizeSelectedAgentModels, buildCommentBodyWithChips,
    markContainerTransportUnreachableIfForwardingFailed,
    fetchTaskDetail} = deps

  const tenantId = effectiveTenantId.value
  const workspaceId = effectiveWorkspaceId.value
  const taskId = effectiveTaskId.value
  if (!tenantId || !workspaceId || !taskId) {
    console.error('发送给 AI 失败: 缺少 tenantId/workspaceId/taskId')
    return
  }
  const node = selectedLayerGraphNode.value
  const snap = layerGraphSnapshot.value
  if (!node || !snap) {
    window.alert('请先在层级图中选中可写层或任务节点（与 onlineServiceJS POST /api/jobs 锚点一致），再使用对话区发送给 AI')
    return
  }
  const resolved = resolveLayerGraphCommandPayload(node, snap.jobs || [])
  if (resolved.error) {
    window.alert(resolved.error)
    return
  }
  const parsedAutoIterationCount = layerGraphAutoIterationCount.value
    ? Number.parseInt(layerGraphAutoIterationCount.value, 10)
    : null
  if (layerGraphAutoIterationCount.value) {
    if (!Number.isFinite(parsedAutoIterationCount) || parsedAutoIterationCount <= 0) {
      window.alert('智能体自动迭代次数需为正整数')
      return
    }
  }
  const selectedModels = normalizeSelectedAgentModels(
    layerGraphSelectedModel.value.trim() ? [layerGraphSelectedModel.value.trim()] : []
  )
  if (layerGraphCommandKind.value === 'trae' && selectedModels.length > 0 && !layerGraphModelProvider.value) {
    window.alert('缺少模型提供商，请先到智能体资源配置页面配置智能体模型提供商')
    return
  }
  let content
  try {
    content = buildCommentBodyWithChips()
  } catch (error) {
    // 点击路径 chunk 失败兜底（OPT-20260823-010）：发布后旧 SPA 引用的
    // 动态 chunk 404 时提示刷新，而非静默吞掉；非 chunk 错误如实展示。
    if (handleActionChunkLoadError(error, { showError: showRequestError })) return
    console.error('构建评论内容失败:', error)
    showRequestError(`发送给 AI 失败：${error?.message || '构建评论内容失败'}`, error)
    return
  }
  if (!content) {
    window.alert('请先填写要发送给 AI 的内容')
    return
  }
  const baseUrl = window.config?.API_BASE_URL ?? ''
  const path = `/api/ai-comment/task-detail/tenant_id/${tenantId}/workspace_id/${workspaceId}/task_id/${taskId}/`
  const fullUrl = `${baseUrl}${path.startsWith('/') ? path : `/${path}`}`
  const headers = {
    'Content-Type': 'application/json',
    Accept: 'application/octet-stream, application/json'}
  const authToken = String(await getActiveToken() || '')
  if (authToken) {
    headers.Authorization = `Token ${authToken}`
  }
  if (deps.aiInstructAbortController) {
    deps.aiInstructAbortController.abort()
    deps.aiInstructAbortController = null
  }
  const ac = new AbortController()
  deps.aiInstructAbortController = ac
  aiStreamBuffer.value = ''
  activeAiInstructId.value = null
  aiStreamBusy.value = true
  try {
    const body = {
      task: localTask.value.id,
      content,
      command_kind: layerGraphCommandKind.value === 'shell' ? 'shell' : 'trae',
      // 默认串行；用户可在执行细节中切换并 PATCH 持久化
      execution_mode: deps.defaultExecutionMode === 'independent' ? 'independent' : 'wait_previous'}
    // OPT-20260722-045: support dependency picker for AI comments (reuses same depends_on_comment_ids
    // field as regular comments; defaults to empty → wait_previous).
    if (deps.dependsOnCommentIds && deps.dependsOnCommentIds.length) {
      body.depends_on_comment_ids = deps.dependsOnCommentIds
    }
    if (resolved.parent_job_id) {
      body.parent_job_id = resolved.parent_job_id
    } else {
      body.repo_layer_id = resolved.repo_layer_id
    }
    if (parsedAutoIterationCount) {
      body.agent_auto_iteration_count = parsedAutoIterationCount
    }
    if (layerGraphCommandKind.value === 'trae' && selectedModels.length > 0) {
      body.agent_models = selectedModels.map((model) => ({
        provider: layerGraphModelProvider.value,
        model
      }))
    }
    const response = await fetch(fullUrl, {
      method: 'POST', credentials: 'include',
      headers: { ...headers, Accept: 'application/json' },
      signal: ac.signal,
      body: JSON.stringify(body)
    })
    if (!response.ok) {
      let errorData = {}
      try {
        errorData = await response.clone().json()
      } catch {
        const t = await response.clone().text().catch(() => '')
        errorData = t ? { _rawErrorText: t } : {}
      }
      const msg = messageFromFailedResponse(
        { status: response.status, _errorData: errorData, headers: response.headers },
        `HTTP ${response.status}`,
      )
      console.error('发送给 AI 失败', msg)
      markContainerTransportUnreachableIfForwardingFailed(response.status, msg)
      showRequestError(`发送给 AI 失败：${msg}`, response)
      aiStreamBusy.value = false
      return
    }
    const ct = (response.headers.get('content-type') || '').toLowerCase()
    if (!ct.includes('application/json')) {
      console.warn('ai-comments 响应非 JSON', ct)
    }
    const data = await response.json()
    if (data.kind !== 'ai_comment_created' || !data.id) {
      console.warn('意外的 ai-comments JSON', data)
      aiStreamBusy.value = false
      return
    }
    activeAiInstructId.value = String(data.id)
    newComment.value = ''
    commentComposerChips.value = []
    await fetchTaskDetail()
  } catch (error) {
    if (error?.name === 'AbortError') {
      aiStreamBusy.value = false
      activeAiInstructId.value = null
      return
    }
    if (handleActionChunkLoadError(error, { showError: showRequestError })) {
      aiStreamBusy.value = false
      activeAiInstructId.value = null
      return
    }
    console.error('发送给 AI 出错:', error)
    showRequestError('发送给 AI 失败，请稍后重试', error)
    aiStreamBusy.value = false
    activeAiInstructId.value = null
  } finally {
    if (deps.aiInstructAbortController === ac) {
      deps.aiInstructAbortController = null
    }
  }
}

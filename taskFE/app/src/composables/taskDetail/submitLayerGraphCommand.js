/**
 * Pure function: submits a layer graph command for TaskDetail.
 */
import { resolveLayerGraphCommandPayload } from '../../utils/resolveLayerGraphCommandPayload.js'
import { messageFromFailedResponse } from '../../utils/httpError.js'
import { extractTraceId } from '../../utils/traceId.js'
import { postContainerCompute } from './containerComputeRequest.js'
import { resolveActiveExecutionCommentId } from './useCommentExecutionContext.js'
import { isReleasedFlag } from './taskDetailReleasedFlag.js'

function setLayerGraphCmdError(deps, message, traceId = '') {
  deps.layerGraphCmdError.value = message
  if (deps.layerGraphCmdErrorTraceId) {
    deps.layerGraphCmdErrorTraceId.value = typeof traceId === 'string' ? traceId : ''
  }
}

function applyLayerGraphCommandHttpError(resp, deps) {
  const raw = messageFromFailedResponse(resp, `HTTP ${resp.status}`)
  const formatted = deps.formatLayerGraphCommandErrorForUser(resp.status, raw)
  const traceId = resp.traceId || extractTraceId(resp) || ''
  setLayerGraphCmdError(deps, formatted, traceId)
  console.warn('发送给AI失败', { status: resp.status, traceId, detail: raw.slice(0, 200) })
  deps.markContainerTransportUnreachableIfForwardingFailed(resp.status, raw)
}

export async function submitLayerGraphCommand(deps) {
  const {
    effectiveTenantId, effectiveWorkspaceId, effectiveTaskId,
    selectedLayerGraphNode, layerGraphSnapshot,
    layerGraphCommandText, layerGraphCommandKind, layerGraphCmdSending,
    layerGraphAutoIterationCount, layerGraphSelectedModel, layerGraphModelProvider,
    layerGraphEditRunTargetJobId,
    validateLinkedProjectReposBeforeSendToAi, normalizeSelectedAgentModels,
    markContainerTransportOk,
    refreshLayerGraphFromServer, bumpProjectFileTreeRefresh,
    displayComments, activeContainerAgentId, localTask,
    bindingStatusFor, bindingCscIdFor,
  } = deps

  const tenantId = effectiveTenantId.value
  const workspaceId = effectiveWorkspaceId.value
  const taskId = effectiveTaskId.value
  const node = selectedLayerGraphNode.value
  const snap = layerGraphSnapshot.value
  if (!tenantId || !workspaceId || !taskId || !node || !snap) {
    setLayerGraphCmdError(deps, '缺少任务或层级上下文'); return
  }
  if (isReleasedFlag(deps.containerReleased) || isReleasedFlag(deps.serverRuntimeNotServing)) {
    setLayerGraphCmdError(deps, '服务器已释放，无法发送指令')
    return
  }
  const content = layerGraphCommandText.value.trim()
  if (!content) { setLayerGraphCmdError(deps, '请输入指令'); return }
  const resolved = resolveLayerGraphCommandPayload(node, snap.jobs || [])
  if (resolved.error) { setLayerGraphCmdError(deps, resolved.error); return }
  if (layerGraphCommandKind.value === 'trae') {
    const repoValidation = await validateLinkedProjectReposBeforeSendToAi()
    if (!repoValidation.ok) { setLayerGraphCmdError(deps, repoValidation.message); return }
  }
  setLayerGraphCmdError(deps, '')
  layerGraphCmdSending.value = true
  try {
    const parsedAutoIterationCount = layerGraphAutoIterationCount.value
      ? Number.parseInt(layerGraphAutoIterationCount.value, 10) : null
    if (layerGraphAutoIterationCount.value) {
      if (!Number.isFinite(parsedAutoIterationCount) || parsedAutoIterationCount <= 0) {
        setLayerGraphCmdError(deps, '智能体自动迭代次数需为正整数'); return
      }
    }
    const selectedModels = normalizeSelectedAgentModels(
      layerGraphSelectedModel.value.trim() ? [layerGraphSelectedModel.value.trim()] : [])
    if (layerGraphCommandKind.value === 'trae' && selectedModels.length > 0 && !layerGraphModelProvider.value) {
      setLayerGraphCmdError(deps, '缺少模型提供商，请先到智能体资源配置页面配置智能体模型提供商'); return
    }
    const body = { command: content, command_kind: layerGraphCommandKind.value === 'shell' ? 'shell' : 'trae' }
    if (parsedAutoIterationCount) body.agent_auto_iteration_count = parsedAutoIterationCount
    if (layerGraphCommandKind.value === 'trae' && selectedModels.length > 0) {
      body.agent_models = selectedModels.map((model) => ({ provider: layerGraphModelProvider.value, model }))
    }
    if (layerGraphEditRunTargetJobId.value) {
      const comments = typeof displayComments?.value !== 'undefined' ? displayComments.value : displayComments
      const activeAgentId = typeof activeContainerAgentId?.value !== 'undefined'
        ? activeContainerAgentId.value
        : activeContainerAgentId
      const parentCommentId = resolveActiveExecutionCommentId(comments, activeAgentId, {
        bindingStatusFor: typeof bindingStatusFor === 'function' ? bindingStatusFor : undefined,
        bindingCscIdFor: typeof bindingCscIdFor === 'function' ? bindingCscIdFor : undefined,
      })
      const taskObj = typeof localTask?.value !== 'undefined' ? localTask.value : localTask
      const installedImageId = String(
        taskObj?.container_image_id || taskObj?.container_image?.id || taskObj?.installed_image_id || '',
      ).trim()
      const resp = await postContainerCompute(deps, 'container-job-edit-run', {
            job_id: layerGraphEditRunTargetJobId.value, command: content,
            command_kind: layerGraphCommandKind.value === 'shell' ? 'shell' : 'trae',
            ...(parsedAutoIterationCount ? { agent_auto_iteration_count: parsedAutoIterationCount } : {}),
            ...(layerGraphCommandKind.value === 'trae' && selectedModels.length > 0
              ? { agent_models: selectedModels.map((m) => ({ provider: layerGraphModelProvider.value, model: m })) } : {}),
            ...(parentCommentId ? { parent_comment_id: parentCommentId } : {}),
            ...(installedImageId ? { installed_image_id: installedImageId } : {}),
          })
      if (!resp.ok) {
        applyLayerGraphCommandHttpError(resp, deps)
        return
      }
      layerGraphCommandText.value = ''; setLayerGraphCmdError(deps, ''); layerGraphEditRunTargetJobId.value = ''
      markContainerTransportOk(); void refreshLayerGraphFromServer(true, { commentId: String(deps.commentId || '').trim() }); bumpProjectFileTreeRefresh(); return
    } else if (resolved.parent_job_id) {
      body.parent_job_id = resolved.parent_job_id
    } else {
      body.repo_layer_id = resolved.repo_layer_id
    }
    const resp = await postContainerCompute(deps, 'container-layer-command', body)
    if (!resp.ok) {
      applyLayerGraphCommandHttpError(resp, deps)
      return
    }
    layerGraphCommandText.value = ''; setLayerGraphCmdError(deps, ''); layerGraphEditRunTargetJobId.value = ''
    markContainerTransportOk(); void refreshLayerGraphFromServer(true, { commentId: String(deps.commentId || '').trim() }); bumpProjectFileTreeRefresh()
  } catch (err) {
    console.error('发送给AI失败', err)
    setLayerGraphCmdError(deps, '网络错误，请稍后重试', err?.traceId || '')
  }
  finally { layerGraphCmdSending.value = false }
}

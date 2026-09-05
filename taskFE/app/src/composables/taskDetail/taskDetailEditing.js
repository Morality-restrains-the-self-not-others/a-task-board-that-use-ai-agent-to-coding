/**
 * Pure functions: task editing and forking for TaskDetail.
 */
import { apiFetch } from '../../utils/apiUtils.js'
import { showRequestError, humanizeRequestErrorMessage } from '../../utils/requestErrorDisplay.js'
import { queryClientPublicIpForAutoSg } from '../../utils/publicClientIp.js'
import { normalizeMemberPk } from '../../utils/taskDetailBranchAndRepoUtils.js'
import { resolveApiErrorMessage } from '../../utils/workPanelFormat.js'
import {
  appendParentTaskToCreatePayload,
  resolveParentDeliverableBlockedReason,
} from '../../utils/workPanelCreateTaskParent.js'
import { alertAutoRunStartSkippedIfNeeded } from '../../utils/autoRunGateHints.js'
import { buildCreateTaskRepoIdentitiesPayload } from '../../utils/createTaskGitIdentityGate.js'
import { consumeSessionGrantTicket, sessionGrantTicketAny } from '../../utils/grantTicketSession.js'
import { mergeIdempotencyHeaders, newIdempotencyKey } from '../../utils/clickGuard.js'
import { clampForkCopyCount, forkCopyIdempotencyKey } from '../../utils/forkCopyCount.js'
import modalService from '../../utils/modalService.js'
import { resolveSavedTaskProjects } from '../../utils/resolveSavedTaskProjects.js'
import { openTaskDetailInNewTab } from './taskDetailRoute.js'

export {
  TASK_DETAIL_PRESERVED_QUERY_KEYS,
  buildTaskDetailRouteQuery,
  openTaskDetailInNewTab,
} from './taskDetailRoute.js'

export async function saveEdit(deps) {
  const {
    effectiveTenantId, effectiveWorkspaceId, effectiveTaskId,
    editingTask, editError, isSaving, localTask, cancelEdit,
    getProjectRepos, buildProjectsApiPayload,
    deliverableCategoryOptions, workspaceTodos,
    onTaskUpdated,
  } = deps
  const tenantId = effectiveTenantId.value; const wid = effectiveWorkspaceId.value
  const taskId = effectiveTaskId.value
  if (!tenantId || !wid || !taskId || !editingTask.value) { editError.value = '缺少必要的参数'; return }
  const linkedProjects = editingTask.value.linkedProjects || []
  for (const project of linkedProjects) {
    const projectId = project.project_id ? String(project.project_id) : ''
    if (!projectId) continue
    const repos = getProjectRepos(projectId)
    const repoBranches = project.repo_branches || {}
    for (const repoUrl of repos) {
      if (!(repoBranches[repoUrl] || '').trim()) { editError.value = '请为所有关联项目的仓库选择基准分支'; return }
    }
  }
  const categories = Array.isArray(deliverableCategoryOptions?.value)
    ? deliverableCategoryOptions.value
    : (Array.isArray(deliverableCategoryOptions) ? deliverableCategoryOptions : [])
  const todos = Array.isArray(workspaceTodos?.value)
    ? workspaceTodos.value
    : (Array.isArray(workspaceTodos) ? workspaceTodos : [])
  const parentBlocked = resolveParentDeliverableBlockedReason({
    taskTypes: categories,
    todos,
    categoryId: editingTask.value.deliverable_obj_id,
    parentTaskId: editingTask.value.parent_task,
  })
  if (parentBlocked) {
    editError.value = parentBlocked
    return
  }
  isSaving.value = true; editError.value = ''
  try {
    const nwbn = String(editingTask.value.workBranchName || '').trim()
    const nmtn = String(editingTask.value.mergeTargetName || '').trim()
    const assigneePks = Array.isArray(editingTask.value.assignees)
      ? editingTask.value.assignees.map(normalizeMemberPk).filter(Boolean)
      : []
    const ownerPk = normalizeMemberPk(editingTask.value.owner)
    const operatorPk = normalizeMemberPk(editingTask.value.operator)
    const deliverableRaw = editingTask.value.deliverable_obj_id
    const deliverableObjId =
      deliverableRaw == null || deliverableRaw === '' ? '' : String(deliverableRaw)
    const payload = {
      title: editingTask.value.title, description: editingTask.value.description,
      priority: Number(editingTask.value.priority),
      task_kind: editingTask.value.task_kind || '',
      code_lang: editingTask.value.code_lang || '',
      deliverable_obj_id: deliverableObjId,
      projects: buildProjectsApiPayload(editingTask.value.linkedProjects, nwbn),
      branch_strategy: { work_branch_name: nwbn, merge_target_branch_name: nmtn, target_branch_name: nwbn },
      owner: ownerPk,
      operator: operatorPk,
      assignees: assigneePks,
    }
    appendParentTaskToCreatePayload(payload, {
      deliverable_obj_id: deliverableObjId,
      parent_task: editingTask.value.parent_task,
    }, categories)
    const resp = await apiFetch(`/api/tasks/todos/tenant_id/${tenantId}/workspace_id/${wid}/${taskId}/`, {
      method: 'PATCH', credentials: 'include', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(payload) })
    if (!resp.ok) {
      const data = await resp.json().catch(() => ({}))
      const err = new Error(resolveApiErrorMessage(data, { fallback: '保存失败', httpStatus: resp.status }))
      err.traceId = resp.traceId || data._traceId || ''
      throw err
    }
    const updatedTask = await resp.json()
    const nextLocal = {
      ...updatedTask,
      projects: resolveSavedTaskProjects(updatedTask, payload.projects),
      branch_strategy: payload.branch_strategy,
    }
    localTask.value = nextLocal
    cancelEdit()
    if (typeof onTaskUpdated === 'function') {
      onTaskUpdated(nextLocal)
    }
  } catch (error) {
    if (error?.name === 'AbortError') {
      editError.value = '保存请求超时，请检查网络后重试'
    } else {
      editError.value = humanizeRequestErrorMessage(error.message || '保存失败，请稍后重试')
    }
    console.error('保存任务失败:', error)
  }
  finally { isSaving.value = false }
}

/**
 * Fork 新任务使用的进度列：工作区进度系统第一列。
 * 选项为空时返回空串，调用方不得回退到源任务列。
 * @param {unknown} progressStatusOptions ref 或数组
 * @returns {string}
 */
export function resolveForkProgressColumnId(progressStatusOptions) {
  const list = Array.isArray(progressStatusOptions)
    ? progressStatusOptions
    : (Array.isArray(progressStatusOptions?.value) ? progressStatusOptions.value : [])
  const first = list[0]
  if (!first || first.id == null || first.id === '') return ''
  return String(first.id)
}

function forkCreateErrorDetail(errText) {
  try {
    const errJson = JSON.parse(errText)
    if (typeof errJson.detail === 'string') return errJson.detail
    if (typeof errJson.error === 'string') return errJson.error
    if (errJson.owner) return String(errJson.owner)
    if (errJson.projects) return String(errJson.projects)
  } catch {
    /* ignore */
  }
  return ''
}

/**
 * @param {object} deps
 * @param {boolean} [deps.autoRun] 用户在确认模态中选择的是否自动运行
 * @param {number} [deps.copyCount] 一次派生的副本数，默认 1，最大 99
 * @param {string} [deps.batchIdempotencyKey] 同一次点击的 batch 幂等前缀
 * @param {function} [deps.onForkProgress] (current, total) => void
 * @param {string} [deps.featureParamsSource]
 * @param {string} [deps.personalFeatureParamsConfigId]
 * @param {string} [deps.agentModelProvider]
 * @param {Array<{model: string}>} [deps.agents]
 * @returns {Promise<boolean>} 是否派生成功（含已创建任务）
 */
export async function forkTask(deps) {
  const {
    effectiveTenantId, effectiveWorkspaceId, effectiveTaskId,
    localTask, isForking, router, detailQuery,
    progressStatusOptions,
    fetchProgressStatusOptions,
    autoRun = false,
    copyCount: copyCountRaw,
    batchIdempotencyKey,
    repoIdentities,
    onForked,
    onForkProgress,
    featureParamsSource,
    personalFeatureParamsConfigId,
    agentModelProvider,
    agents: agentsRaw,
  } = deps
  const tenantId = effectiveTenantId.value; const wid = effectiveWorkspaceId.value
  const sourceTaskId = effectiveTaskId.value; const task = localTask.value
  if (!tenantId || !wid || !sourceTaskId || !task) {
    showRequestError('无法派生：缺少租户、工作空间或任务信息')
    return false
  }
  const ownerPk = normalizeMemberPk(task.owner)
  if (!ownerPk) {
    showRequestError('无法派生：当前任务缺少负责人')
    return false
  }
  const operatorPk = normalizeMemberPk(task.operator)
  const assignees = Array.isArray(task.assignees) ? task.assignees : []
  const assigneePks = assignees.map(normalizeMemberPk).filter(Boolean)
  const wantAutoRun = autoRun === true
  // 必须在任何 await 之前同步置位，否则确认按钮双击会在 fetch 进度列 / 公网 IP
  // 期间再次进入并打出第二次 POST（见 task_877835528962076672 / task_877835529347952640，间隔 ~92ms）。
  if (isForking.value) return false
  isForking.value = true
  const wantAgents = wantAutoRun
    ? (Array.isArray(agentsRaw) ? agentsRaw : [])
      .map((item) => ({
        provider: String(item?.provider || agentModelProvider || '').trim(),
        model: String(item?.model || '').trim(),
      }))
      .filter((item) => item.model)
    : []
  const copyCount = wantAutoRun
    ? (wantAgents.length > 0 ? clampForkCopyCount(wantAgents.length) : 1)
    : 1
  const batchKey = String(batchIdempotencyKey || '').trim() || newIdempotencyKey()
  try {
  let firstProgressColumnId = resolveForkProgressColumnId(progressStatusOptions)
  if (!firstProgressColumnId && typeof fetchProgressStatusOptions === 'function') {
    await fetchProgressStatusOptions()
    firstProgressColumnId = resolveForkProgressColumnId(progressStatusOptions)
  }
  const payload = {
    title: task.title, description: task.description ?? '', completed: false,
    priority: task.priority,
    workspace_id: String(wid), owner: ownerPk, operator: operatorPk, assignees: assigneePks,
    fork_from: String(sourceTaskId),
    auto_run: wantAutoRun,
  }
  if (firstProgressColumnId) {
    payload.progress_column_id = firstProgressColumnId
  }
  console.info(
    '[forkTask] event=fork_progress_column_reset source_task_id=%s workspace_id=%s source_progress_column_id=%s first_progress_column_id=%s',
    String(sourceTaskId),
    String(wid),
    task.progress_column_id == null ? '' : String(task.progress_column_id),
    firstProgressColumnId,
  )
  if (task.due_date) payload.due_date = task.due_date
  if (task.parameters != null) payload.parameters = task.parameters
  if (task.parent_task != null && task.parent_task !== '') payload.parent_task = task.parent_task
  const deliverableId = task.deliverable_obj?.id ?? task.deliverable_obj_id
  if (deliverableId != null && deliverableId !== '') payload.deliverable_obj_id = deliverableId
  const imageId = task.container_image_id ?? task.container_image?.id
  if (imageId != null && imageId !== '') payload.container_image_id = String(imageId)
  if (task.feature_params_source != null && task.feature_params_source !== '') {
    payload.feature_params_source = task.feature_params_source
  }
  if (task.personal_feature_params_config_id != null && task.personal_feature_params_config_id !== '') {
    payload.personal_feature_params_config_id = String(task.personal_feature_params_config_id)
  }
  if (wantAutoRun && featureParamsSource) {
    payload.feature_params_source = featureParamsSource
    if (featureParamsSource === 'personal' && personalFeatureParamsConfigId) {
      payload.personal_feature_params_config_id = String(personalFeatureParamsConfigId)
    } else if (featureParamsSource !== 'personal') {
      delete payload.personal_feature_params_config_id
    }
  }
  if (task.task_kind != null && task.task_kind !== '') {
    payload.task_kind = task.task_kind
  }
  const bs = task.branch_strategy || {}
  payload.branch_strategy = { work_branch_name: bs.work_branch_name || '', merge_target_branch_name: bs.merge_target_branch_name || '', target_branch_name: bs.target_branch_name || bs.work_branch_name || '' }
  payload.projects = Array.isArray(task.projects)
    ? task.projects.map((p) => ({
      project_id: p.project_id,
      repo_index: p.repo_index,
      base_branch: p.base_branch,
      target_branch: p.target_branch,
    }))
    : []
  if (wantAutoRun) {
    const ip = await queryClientPublicIpForAutoSg()
    if (ip) payload.client_public_ip = ip
    const payloadIdents = buildCreateTaskRepoIdentitiesPayload(true, repoIdentities) || []
    payload.repo_identities = payloadIdents
    const grantTicket = sessionGrantTicketAny()
    if (grantTicket) payload.grant_ticket = grantTicket
    console.info(
      '[forkTask] event=fork_auto_run_repo_identities source_task_id=%s repo_count=%s identity_count=%s',
      String(sourceTaskId),
      String(payload.projects.length),
      String(payloadIdents.length),
    )
  }
    if (copyCount > 1 && wantAgents.length === 0) {
      // 同构批仍走服务端 fork_count（本弹窗自动运行异构副本不走此路径）。
      return await forkTaskBatchSend({
        tenantId,
        wid,
        sourceTaskId,
        payload,
        copyCount,
        batchKey,
        onForkProgress,
        router,
        detailQuery,
        onForked,
      })
    }
    console.info(
      '[forkTask] event=fork_copies_begin source_task_id=%s copy_count=%s auto_run=%s agent_count=%s',
      String(sourceTaskId),
      String(copyCount),
      String(wantAutoRun),
      String(wantAgents.length),
    )
    const createdTasks = []
    for (let i = 1; i <= copyCount; i += 1) {
      if (typeof onForkProgress === 'function') {
        onForkProgress(i, copyCount)
      }
      const copyPayload = { ...payload }
      if (wantAgents.length > 0) {
        const agent = wantAgents[i - 1]
        copyPayload.agent_models = [{
          provider: agent.provider,
          model: agent.model,
        }]
        console.info(
          '[forkTask] event=fork_copy_model source_task_id=%s copy_index=%s model=%s provider=%s',
          String(sourceTaskId),
          String(i),
          agent.model,
          agent.provider,
        )
      }
      const resp = await apiFetch(`/api/tasks/todos/tenant_id/${tenantId}/workspace_id/${wid}`, {
        method: 'POST',
        credentials: 'include',
        headers: mergeIdempotencyHeaders(
          { 'Content-Type': 'application/json', Accept: 'application/json' },
          forkCopyIdempotencyKey(batchKey, i),
        ),
        body: JSON.stringify(copyPayload),
      })
      if (!resp.ok) {
        const errText = await resp.text().catch(() => '')
        const detail = forkCreateErrorDetail(errText)
        console.error(
          '[forkTask] event=fork_copy_failed source_task_id=%s copy_index=%s copy_count=%s status=%s',
          String(sourceTaskId),
          String(i),
          String(copyCount),
          String(resp.status),
          errText,
        )
        if (createdTasks.length === 0) {
          showRequestError(detail ? `派生任务失败：${detail}` : '派生任务失败，请稍后重试', resp)
          return false
        }
        showRequestError(
          detail
            ? `已派生 ${createdTasks.length}/${copyCount} 个副本，后续失败：${detail}`
            : `已派生 ${createdTasks.length}/${copyCount} 个副本，后续失败，请稍后重试`,
          resp,
        )
        break
      }
      const created = await resp.json()
      const newId = created?.id
      if (!newId) {
        if (createdTasks.length === 0) {
          showRequestError('派生成功但未返回新任务 ID')
          return false
        }
        showRequestError(`已派生 ${createdTasks.length}/${copyCount} 个副本，后续未返回新任务 ID`)
        break
      }
      createdTasks.push(created)
    }
    if (createdTasks.length === 0) return false
    // OPT-20260902-025：fork 已携带 grant_ticket 创建成功 → ticket 已消费，从 session 移除
    if (payload.grant_ticket) consumeSessionGrantTicket(payload.grant_ticket)
    const created = createdTasks[0]
    const newId = created.id
    console.info(
      '[forkTask] event=fork_copies_created source_task_id=%s copy_count=%s created_count=%s first_task_id=%s',
      String(sourceTaskId),
      String(copyCount),
      String(createdTasks.length),
      String(newId),
    )
    // Fork 与工作面板创建一致：软跳过启服时必须弹窗，否则新标签页冷打开只见「未启动」无原因。
    alertAutoRunStartSkippedIfNeeded(created, modalService)
    openTaskDetailInNewTab(router, {
      tenantId,
      workspaceId: wid,
      taskId: newId,
      query: detailQuery.value,
    })
    // 浏览器拦截新标签页时静默处理：任务已创建成功，
    // onForked 回调会刷新工作面板，用户可从中手动打开新任务
    // Same-tab work-panel: refresh board immediately (SSE is best-effort / cross-tab).
    if (typeof onForked === 'function') {
      try {
        onForked(created)
      } catch {
        /* ignore listener errors */
      }
    }
    return true
  } catch (error) {
    console.error('派生任务出错:', error)
    showRequestError('派生任务失败，请检查网络后重试', error)
    return false
  } finally {
    isForking.value = false
  }
}

/**
 * OPT-20260823-006：服务端一次请求批量派生（copyCount>1）。
 * 单次 POST 携带 fork_count，服务端返回 ids/created_count/first_id；
 * 部分成功（created_count < copyCount）时提示已派生数并照常打开首个副本。
 * @param {object} deps
 * @returns {Promise<boolean>}
 */
async function forkTaskBatchSend({
  tenantId,
  wid,
  sourceTaskId,
  payload,
  copyCount,
  batchKey,
  onForkProgress,
  router,
  detailQuery,
  onForked,
}) {
  if (typeof onForkProgress === 'function') onForkProgress(1, copyCount)
  const batchPayload = { ...payload, fork_count: copyCount }
  console.info(
    '[forkTask] event=fork_batch_begin source_task_id=%s copy_count=%s batch_key=%s',
    String(sourceTaskId),
    String(copyCount),
    batchKey,
  )
  const resp = await apiFetch(`/api/tasks/todos/tenant_id/${tenantId}/workspace_id/${wid}`, {
    method: 'POST',
    credentials: 'include',
    headers: mergeIdempotencyHeaders(
      { 'Content-Type': 'application/json', Accept: 'application/json' },
      batchKey,
    ),
    body: JSON.stringify(batchPayload),
  })
  const body = await resp.json().catch(() => ({}))
  const createdCount = Number(body.created_count ?? 0)
  const ids = Array.isArray(body.ids) ? body.ids : []
  const firstId = body.first_id || ids[0]
  if (resp.ok) {
    if (createdCount < copyCount) {
      const errDetail = body.error?.detail || body.error?.error || body.error?.message || ''
      showRequestError(
        errDetail
          ? `已派生 ${createdCount}/${copyCount} 个副本，后续失败：${errDetail}`
          : `已派生 ${createdCount}/${copyCount} 个副本，后续失败`,
      )
    } else if (!firstId) {
      showRequestError('派生成功但未返回新任务 ID')
      return false
    }
    // OPT-20260902-025：批量派生成功同样视为 grant_ticket 已消费
    if (batchPayload.grant_ticket) consumeSessionGrantTicket(batchPayload.grant_ticket)
    const created = { id: firstId }
    console.info(
      '[forkTask] event=fork_batch_created source_task_id=%s copy_count=%s created_count=%s first_task_id=%s',
      String(sourceTaskId),
      String(copyCount),
      String(createdCount),
      String(firstId),
    )
    alertAutoRunStartSkippedIfNeeded(created, modalService)
    openTaskDetailInNewTab(router, {
      tenantId,
      workspaceId: wid,
      taskId: firstId,
      query: detailQuery.value,
    })
    if (typeof onForked === 'function') {
      try {
        onForked(created)
      } catch {
        /* ignore listener errors */
      }
    }
    return true
  }
  const detail = forkCreateErrorDetail(JSON.stringify(body))
  if (createdCount > 0) {
    showRequestError(
      detail
        ? `已派生 ${createdCount}/${copyCount} 个副本，后续失败：${detail}`
        : `已派生 ${createdCount}/${copyCount} 个副本，后续失败，请稍后重试`,
      resp,
    )
    return false
  }
  showRequestError(detail ? `派生任务失败：${detail}` : '派生任务失败，请稍后重试', resp)
  return false
}

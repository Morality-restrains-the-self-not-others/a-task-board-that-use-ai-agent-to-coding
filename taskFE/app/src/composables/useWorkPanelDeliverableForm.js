/**
 * WorkPanel：创建/编辑交付物表单提交与编辑态组装（从 WorkPanel.vue 抽出以削行数）。
 */
import { getStoredUserId } from '../utils/sessionUserIdUtils.js'
import { apiFetch } from '../utils/apiUtils.js'
import { parseJsonSafe, warnOptionalApiFailure } from '../utils/workPanelApiUtils.js'
import {
  applyTaskCreatedDateToWorkBranchName,
  buildDefaultMergeTargetBranchName,
  getDefaultTaskDeadline,
  resolveBranchNamePlaceholders,
  sanitizeBranchSegment,
} from '../utils/workPanelBranchHelpers.js'
import { appendParentTaskToCreatePayload, buildCreateTaskDraft } from '../utils/workPanelCreateTaskParent.js'
import {
  pickCreateTaskDefaultProjectId,
  readCreateTaskPreferredProject,
} from '../utils/createTaskPreferredProject.js'
import { buildFeatureParamsPayloadFields, normalizeFeatureParamsSourceForSelect } from '../utils/envParamsSourceSelection.js'
import { pickTaskMetaFields } from './useWorkPanelTaskMetaOptions.js'
import { resolveApiErrorMessage } from '../utils/workPanelFormat.js'
import { alertAutoRunStartSkippedIfNeeded } from '../utils/autoRunGateHints.js'
import { appendQueuedAutoRunToCreatePayload } from '../utils/createTaskQueuedAutoRun.js'
import { buildCreateTaskRepoIdentitiesPayload } from '../utils/createTaskGitIdentityGate.js'
import { consumeSessionGrantTicket, sessionGrantTicketAny } from '../utils/grantTicketSession.js'
import modalService from '../utils/modalService.js'
import { showRequestError } from '../utils/requestErrorDisplay.js'

/**
 * @param {object} deps
 */
export function useWorkPanelDeliverableForm(deps) {
  const {
    tenantId,
    currentWorkspace,
    todos,
    taskStatuses,
    taskTypes,
    projects,
    deliverableFilterBars,
    editingTask,
    showCreateTaskModal,
    fetchTodos,
    buildDefaultWorkBranchName,
    buildBranchStrategyProjectMappings,
  } = deps

  const closeCreateTaskModal = () => {
    showCreateTaskModal.value = false
    editingTask.value = null
    document.body.style.overflow = ''
  }

  const handleCreateDeliverable = () => {
    editingTask.value = buildCreateTaskDraft({
      taskStatuses: taskStatuses.value,
      taskTypes: taskTypes.value,
      todos: todos.value,
      projects: projects.value,
      filterBars: deliverableFilterBars.value,
      preferredProjectId: readCreateTaskPreferredProject(tenantId.value),
      ownerId: getStoredUserId() || '',
      operatorId: getStoredUserId() || '',
      dueDate: getDefaultTaskDeadline(),
      workBranchName: buildDefaultWorkBranchName('feature'),
    })
    showCreateTaskModal.value = true
    document.body.style.overflow = 'hidden'
  }

  const openEditTaskModal = (task) => {
    const normalizedContainerImageId = task?.container_image_id || task?.container_image?.id || null
    // D2=B 回填：技能名与稳定 ID 优先取详情容器对象（container_image.skill），
    // 老数据回退保存时快照（container_image_snapshot）/ 扁平列（image_skill_id）。
    // 使编辑弹窗 chips 提示与提交 container_image_skill_id 不丢失绑定。
    const snapshotSkill = task?.container_image_snapshot && typeof task.container_image_snapshot === 'object'
      ? task.container_image_snapshot
      : {}
    const containerImageSkill = task?.container_image?.skill && typeof task.container_image.skill === 'object'
      ? task.container_image.skill
      : {}
    const containerImageSkillName =
      String(containerImageSkill?.name || snapshotSkill?.skill_name || '').trim()
    const containerImageSkillId =
      String(containerImageSkill?.id || snapshotSkill?.skill_id || task?.image_skill_id || '').trim()
    const branchStrategy = task?.branch_strategy || {}
    const branchProjects = Array.isArray(task?.projects) ? task.projects : []
    const grouped = new Map()
    for (const item of branchProjects) {
      const pid = item?.project_id != null ? String(item.project_id) : ''
      if (!pid) continue
      if (!grouped.has(pid)) grouped.set(pid, [])
      grouped.get(pid).push(item)
    }
    const projectSelections = []
    const firstGroupedEntry = grouped.entries().next().value
    if (firstGroupedEntry) {
      const [pid, items] = firstGroupedEntry
      const sorted = [...items]
        .filter((it) => it && Number.isFinite(Number(it.repo_index)))
        .sort((a, b) => Number(a.repo_index) - Number(b.repo_index))
      if (sorted.length > 0) {
        const repoBranches = sorted.map((it) => ({
          repoIndex: Number(it.repo_index),
          baseBranch: it?.base_branch ? String(it.base_branch) : '',
        }))
        projectSelections.push({ projectId: pid, repoBranches })
      }
    }
    const defaultProjectSelection = {
      projectId: pickCreateTaskDefaultProjectId(
        projects.value,
        new Set(),
        readCreateTaskPreferredProject(tenantId.value),
      ),
      repoBranches: [],
    }

    const rawOwner = task?.owner ?? task?.parameters?.owner_id
    const normalizedOwner = rawOwner != null && rawOwner !== '' ? String(rawOwner) : ''
    const rawOperator = task?.operator ?? task?.parameters?.operator_id
    const normalizedOperator = rawOperator != null && rawOperator !== '' ? String(rawOperator) : ''

    editingTask.value = {
      ...task,
      due_date: task?.due_date || task?.deadline || '',
      ...pickTaskMetaFields(task, { trim: false }),
      container_image: {
        id: normalizedContainerImageId,
        skill: containerImageSkillName,
        skillId: containerImageSkillId,
      },
      projectSelections: projectSelections.length > 0 ? projectSelections : [defaultProjectSelection],
      workBranchName:
        branchStrategy.work_branch_name ||
        branchStrategy.target_branch_name ||
        buildDefaultWorkBranchName('feature', task?.title || '', task?.created_at),
      mergeTargetName:
        branchStrategy.merge_target_branch_name ||
        buildDefaultMergeTargetBranchName('develop'),
      assignees: Array.isArray(task?.assignees) ? task.assignees : [],
      owner: normalizedOwner,
      operator: normalizedOperator,
      auto_run: Boolean(task?.auto_run),
      queued_auto_run: Boolean(task?.queued_auto_run),
      force_auto_run: false,
      feature_params_source: normalizeFeatureParamsSourceForSelect(task?.feature_params_source),
      personal_feature_params_config_id: String(task?.personal_feature_params_config_id || '').trim(),
    }
    showCreateTaskModal.value = true
  }

  const submitDeliverableForm = async (taskData) => {
    try {
      const task = taskData || editingTask.value
      const url = task?.id
        ? `/api/tasks/todos/tenant_id/${tenantId.value}/workspace_id/${currentWorkspace.value?.id}/${task.id}/`
        : `/api/tasks/todos/tenant_id/${tenantId.value}/workspace_id/${currentWorkspace.value?.id}`
      const method = task?.id ? 'PUT' : 'POST'

      const payload = {
        title: task.title,
        description: task.description,
        ...pickTaskMetaFields(task),
        priority: task.priority,
        progress_column_id: task.progressColumn.id,
        due_date: task.due_date,
        workspace_id: currentWorkspace.value?.id,
        assignees: Array.isArray(task.assignees) ? task.assignees : [],
        owner: task.owner,
        operator: task.operator || '',
      }

      const workBranchNameForSubmit =
        task.workBranchPreset === 'custom'
          ? task.workBranchName
          : applyTaskCreatedDateToWorkBranchName(task.workBranchName, task)
      const titleSegmentForBranch = sanitizeBranchSegment(task.title || '', 'task')
      const normalizedWorkBranchName = resolveBranchNamePlaceholders(
        String(workBranchNameForSubmit || '').trim(),
        {
          taskId: task?.id ? String(task.id) : '',
          taskTitleSegment: titleSegmentForBranch,
        },
      )
      const normalizedMergeTargetName = resolveBranchNamePlaceholders(
        String(task.mergeTargetName || '').trim(),
        {
          taskId: task?.id ? String(task.id) : '',
          taskTitleSegment: titleSegmentForBranch,
        },
      )
      const projectBranchMappings = buildBranchStrategyProjectMappings(task)
      payload.branch_strategy = {
        work_branch_name: normalizedWorkBranchName,
        merge_target_branch_name: normalizedMergeTargetName,
        target_branch_name: normalizedWorkBranchName,
      }
      payload.projects = projectBranchMappings

      if (task.task_type?.id) {
        payload.deliverable_obj_id = task.task_type.id
      }
      appendParentTaskToCreatePayload(payload, task, taskTypes.value)

      const containerImageId = task.container_image?.id || task.container_image_id
      if (containerImageId) {
        payload.container_image_id = containerImageId
        // D4 契约：携带稳定技能 ID（创建/编辑共用）。无 id（老数据/目录缺失）时不发，
        // 服务端按描述 /token 反解兜底。
        const containerImageSkillId = task.container_image?.skillId || task.container_image_skill_id
        if (containerImageSkillId) {
          payload.container_image_skill_id = containerImageSkillId
        }
      }

      if (typeof task.auto_run === 'boolean') {
        payload.auto_run = task.auto_run
      }
      appendQueuedAutoRunToCreatePayload(payload, task)
      if (task.force_auto_run === true) {
        payload.force_auto_run = true
      }
      const repoIdentities = buildCreateTaskRepoIdentitiesPayload(task.auto_run, task.repo_identities)
      if (repoIdentities) {
        payload.repo_identities = repoIdentities
      }
      if (task.auto_run === true) {
        const grantTicket = sessionGrantTicketAny()
        if (grantTicket) payload.grant_ticket = grantTicket
      }

      Object.assign(payload, buildFeatureParamsPayloadFields(task))

      const response = await apiFetch(url, {
        method,
        credentials: 'include',
        headers: {
          Accept: 'application/json',
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(payload),
      })

      if (response.ok) {
        // OPT-20260902-025：create-task 携带的 grant_ticket 已随请求交给后端
        // （成功即失效/已由 L2 覆盖），从 session 移除，避免 Fork 门禁误当「已绑定」。
        if (payload.grant_ticket) consumeSessionGrantTicket(payload.grant_ticket)
        alertAutoRunStartSkippedIfNeeded(await parseJsonSafe(response), modalService)
        await fetchTodos()
        closeCreateTaskModal()
      } else {
        const errorData = await parseJsonSafe(response)
        warnOptionalApiFailure('保存交付物', response)
        const isTaskCreateInsufficientBalance =
          response.status === 402 &&
          method === 'POST' &&
          (errorData?.code === 'INSUFFICIENT_BALANCE' ||
           errorData?.code === 'INSUFFICIENT_TASK_POST_QUOTA')
        if (isTaskCreateInsufficientBalance) {
          const baseMsg = resolveApiErrorMessage(errorData, {
            fallback: '当前资源配额不足以创建任务帖，请先购买资源。',
            httpStatus: response.status,
          })
          const tid = tenantId.value
          modalService.alert({
            title: '资源配额不足',
            message: baseMsg,
            additionalActions: tid
              ? [
                  {
                    text: '去购买',
                    callback: () => {
                      modalService.close()
                      window.location.assign(`/tenant/${tid}/billing/orders/create/`)
                    },
                  },
                ]
              : [],
          })
        } else {
          const errorMessage = resolveApiErrorMessage(errorData, {
            fallback: '保存交付物失败，请检查输入内容',
            httpStatus: response.status,
          })
          showRequestError(errorMessage, {
            ...errorData,
            traceId: errorData.trace_id || errorData.traceId,
          })
        }
      }
    } catch (error) {
      console.error('保存交付物出错:', error)
      showRequestError(error?.message || '保存交付物出错，请稍后重试', error)
    }
  }

  return {
    handleCreateDeliverable,
    openEditTaskModal,
    submitDeliverableForm,
    closeCreateTaskModal,
  }
}

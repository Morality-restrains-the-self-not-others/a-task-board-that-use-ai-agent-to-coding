/**
 * WorkPanel 数据拉取（从 WorkPanel.vue 抽出以降低单文件行数）。
 * 所有函数接收同一套 deps（Vue ref 与工具函数），在调用处解构传入。
 */

import {
  DEFAULT_TASK_STATUSES,
  mapProgressColumnsToTaskStatuses,
} from './workPanelKanbanUtils.js'

/**
 * 看板空态提示：仅在任务列表**成功加载且为空**时展示。
 * 加载失败不得伪装成「暂无任务」（否则会与机器摘要已启动数打架）。
 * @param {{ workspaceId?: string|number|null, statusCount?: number, todoCount?: number, loadError?: unknown }} input
 * @returns {boolean}
 */
export function shouldShowEmptyTaskHint({ workspaceId, statusCount, todoCount, loadError }) {
  const wsId = workspaceId == null ? '' : String(workspaceId).trim()
  if (!wsId || wsId === 'default') return false
  if (loadError) return false
  return Number(statusCount) > 0 && Number(todoCount) === 0
}

export async function fetchCurrentCompanyUserName(deps, tenantIdValue, userData) {
  const { companyUserName } = deps
  if (!tenantIdValue) {
    companyUserName.value = ''
    return
  }

  // 工作面板 initData 已拉 /me/；分支名默认用 username/email，不额外请求 profile
  companyUserName.value = userData?.username || userData?.email || ''
}

export async function fetchTodos(deps) {
  const {
    tenantId,
    currentWorkspace,
    filterOptions,
    todos,
    todosError,
    todosErrorTraceId,
    apiFetch,
    parseJsonSafe,
    normalizeListPayload,
    warnOptionalApiFailure,
    warnNetworkFailure,
    formatApiErrorMessage,
    extractTraceId,
  } = deps

  const setTodosError = (message, source) => {
    if (todosError) todosError.value = message || '任务列表加载失败'
    if (todosErrorTraceId) {
      todosErrorTraceId.value = extractTraceId?.(source) || ''
    }
  }
  const clearTodosError = () => {
    if (todosError) todosError.value = null
    if (todosErrorTraceId) todosErrorTraceId.value = ''
  }

  try {
    if (!tenantId.value || !currentWorkspace.value?.id || currentWorkspace.value.id === 'default') {
      clearTodosError()
      return
    }

    console.log('开始获取任务列表')
    console.log('当前工作空间ID:', currentWorkspace.value?.id)
    console.log('当前租户ID:', tenantId.value)

    let queryParams = ''

    if (
      filterOptions.value.priority !== null ||
      filterOptions.value.search
    ) {
      queryParams = '?'
      if (filterOptions.value.priority !== null) {
        queryParams += `priority=${filterOptions.value.priority}&`
      }
      if (filterOptions.value.search) {
        queryParams += `search=${filterOptions.value.search}&`
      }
      queryParams = queryParams.replace(/&$/, '')
    }

    const url = `/api/tasks/todos/tenant_id/${tenantId.value}/workspace_id/${currentWorkspace.value.id}${queryParams}`
    console.log('请求URL:', url)
    const response = await apiFetch(url, {
      credentials: 'include',
      headers: {
        Accept: 'application/json',
      },
    })

    console.log('任务列表请求状态:', response.status)
    if (response.ok) {
      const data = await parseJsonSafe(response)
      if (data == null) {
        warnOptionalApiFailure('任务列表(JSON)', response)
        todos.value = []
        setTodosError('任务列表响应异常', response)
        return
      }
      console.log('任务列表数据:', data)
      todos.value = normalizeListPayload(data)
      clearTodosError()
    } else {
      warnOptionalApiFailure('任务列表', response)
      const errBody = await parseJsonSafe(response)
      todos.value = []
      const message =
        formatApiErrorMessage?.(errBody) || `任务列表加载失败（HTTP ${response.status}）`
      if (todosError) todosError.value = message
      if (todosErrorTraceId) {
        todosErrorTraceId.value =
          extractTraceId?.(response) || extractTraceId?.(errBody) || ''
      }
    }
  } catch (error) {
    warnNetworkFailure('任务列表', error)
    todos.value = []
    setTodosError('网络错误，请稍后重试', error)
  }
}

export async function fetchWorkspaceProgressSystem(deps) {
  const {
    tenantId,
    currentWorkspace,
    apiFetch,
    parseJsonSafe,
    warnOptionalApiFailure,
    warnNetworkFailure,
  } = deps

  try {
    if (!tenantId.value || !currentWorkspace.value?.id || currentWorkspace.value.id === 'default') {
      console.log('跳过获取工作空间进度体系：租户ID或工作空间ID无效')
      return null
    }

    console.log('开始获取工作空间进度体系')
    console.log('租户ID:', tenantId.value)
    console.log('工作空间ID:', currentWorkspace.value.id)

    const url = `/api/projects/workspaces/tenant_id/${tenantId.value}/${currentWorkspace.value.id}/progress-system/`
    console.log('请求URL:', url)
    const response = await apiFetch(url, {
      credentials: 'include',
      headers: {
        Accept: 'application/json',
      },
    })

    console.log('工作空间进度体系请求状态:', response.status)
    if (response.ok) {
      const data = await parseJsonSafe(response)
      if (data == null) {
        warnOptionalApiFailure('工作空间进度体系(JSON)', response)
        return null
      }
      console.log('工作空间进度体系数据:', data)
      return data
    }
    warnOptionalApiFailure('工作空间进度体系', response)
    return null
  } catch (error) {
    warnNetworkFailure('工作空间进度体系', error)
    return null
  }
}

export async function fetchTaskStatuses(deps) {
  const {
    tenantId,
    currentWorkspace,
    taskStatuses,
    apiFetch,
    warnNetworkFailure,
  } = deps

  try {
    if (!tenantId.value || !currentWorkspace.value?.id || currentWorkspace.value.id === 'default') {
      console.log('跳过获取任务状态：租户ID或工作空间ID无效')
      return
    }

    console.log('开始获取任务状态列表（从进度体系）')

    const workspaceProgressSystem = await fetchWorkspaceProgressSystem(deps)

    const statuses = mapProgressColumnsToTaskStatuses(workspaceProgressSystem?.columns)
    if (statuses?.length) {
      console.log('从工作空间进度体系获取到 columns:', workspaceProgressSystem.columns)
      console.log('从进度体系生成的任务状态:', statuses)
      taskStatuses.value = statuses
    } else {
      console.log('没有找到进度体系或列为空，使用默认任务状态')
      taskStatuses.value = DEFAULT_TASK_STATUSES
    }
  } catch (error) {
    warnNetworkFailure('任务状态列表', error)
    taskStatuses.value = DEFAULT_TASK_STATUSES
  }
}

export async function fetchTaskTypes(deps) {
  const {
    tenantId,
    currentWorkspace,
    taskTypes,
    isTaskTypesLoading,
    taskTypesError,
    taskTypesErrorTraceId,
    apiFetch,
    parseJsonSafe,
    warnOptionalApiFailure,
    warnNetworkFailure,
    formatApiErrorMessage,
    extractTraceId,
  } = deps

  try {
    if (!tenantId.value || !currentWorkspace.value?.id || currentWorkspace.value.id === 'default') {
      console.log('跳过获取任务类型：租户ID或工作空间ID无效')
      return
    }

    console.log('开始获取任务类型列表')
    isTaskTypesLoading.value = true
    taskTypesError.value = null
    if (taskTypesErrorTraceId) taskTypesErrorTraceId.value = ''

    const url = `/api/projects/manage-deliverable-system/tenant_id/${tenantId.value}?workspace_id=${currentWorkspace.value.id}`

    console.log('请求URL:', url)
    const response = await apiFetch(url, {
      credentials: 'include',
      headers: {
        Accept: 'application/json',
      },
    })

    console.log('任务类型列表请求状态:', response.status)
    if (response.ok) {
      const data = await parseJsonSafe(response)
      if (data == null) {
        warnOptionalApiFailure('任务类型列表(JSON)', response)
        taskTypesError.value = '响应格式异常'
        if (taskTypesErrorTraceId) {
          taskTypesErrorTraceId.value = extractTraceId?.(response) || ''
        }
        taskTypes.value = []
      } else {
        console.log('任务类型列表数据:', data)
        if (data.current_deliverable_objs) {
          console.log('从响应中获取到交付物类型:', data.current_deliverable_objs)
          taskTypes.value = data.current_deliverable_objs
        } else {
          console.log('响应中没有任务类型数据，使用默认值')
          taskTypes.value = []
        }
      }
    } else {
      warnOptionalApiFailure('任务类型列表', response)
      const errBody = await parseJsonSafe(response)
      taskTypesError.value =
        formatApiErrorMessage(errBody) || `获取任务类型失败（HTTP ${response.status}）`
      if (taskTypesErrorTraceId) {
        taskTypesErrorTraceId.value =
          extractTraceId?.(response) || extractTraceId?.(errBody) || ''
      }
      taskTypes.value = []
    }
  } catch (error) {
    warnNetworkFailure('任务类型列表', error)
    taskTypesError.value = '网络错误，请稍后重试'
    if (taskTypesErrorTraceId) {
      taskTypesErrorTraceId.value = extractTraceId?.(error) || ''
    }
    taskTypes.value = []
  } finally {
    isTaskTypesLoading.value = false
  }
}

export async function fetchProjects(deps) {
  const {
    tenantId,
    currentWorkspace,
    projects,
    isProjectsLoading,
    projectsError,
    projectsErrorTraceId,
    apiFetch,
    parseJsonSafe,
    normalizeListPayload,
    warnOptionalApiFailure,
    warnNetworkFailure,
    formatApiErrorMessage,
    extractTraceId,
  } = deps

  try {
    if (!tenantId.value || !currentWorkspace.value?.id || currentWorkspace.value.id === 'default') {
      console.log('跳过获取项目：租户ID或工作空间ID无效')
      return
    }

    console.log('开始获取项目列表')
    isProjectsLoading.value = true
    projectsError.value = null
    if (projectsErrorTraceId) projectsErrorTraceId.value = ''

    const url = `/api/projects/tenant_id/${tenantId.value}?workspace_id=${currentWorkspace.value.id}`

    console.log('请求URL:', url)
    const response = await apiFetch(url, {
      credentials: 'include',
      headers: {
        Accept: 'application/json',
      },
    })

    console.log('项目列表请求状态:', response.status)
    if (response.ok) {
      const data = await parseJsonSafe(response)
      if (data == null) {
        warnOptionalApiFailure('项目列表(JSON)', response)
        projectsError.value = '项目列表响应异常'
        if (projectsErrorTraceId) {
          projectsErrorTraceId.value = extractTraceId?.(response) || ''
        }
        projects.value = []
      } else {
        console.log('项目列表数据:', data)
        projects.value = normalizeListPayload(data)
      }
    } else {
      warnOptionalApiFailure('项目列表', response)
      const errBody = await parseJsonSafe(response)
      projectsError.value =
        formatApiErrorMessage(errBody) || `获取项目失败（HTTP ${response.status}）`
      if (projectsErrorTraceId) {
        projectsErrorTraceId.value =
          extractTraceId?.(response) || extractTraceId?.(errBody) || ''
      }
      projects.value = []
    }
  } catch (error) {
    warnNetworkFailure('项目列表', error)
    projectsError.value = '网络错误，请稍后重试'
    if (projectsErrorTraceId) {
      projectsErrorTraceId.value = extractTraceId?.(error) || ''
    }
    projects.value = []
  } finally {
    isProjectsLoading.value = false
  }
}

export async function fetchInstalledImages(deps) {
  const {
    tenantId,
    installedImages,
    isInstalledImagesLoading,
    installedImagesError,
    installedImagesErrorTraceId,
    apiFetch,
    parseJsonSafe,
    normalizeListPayload,
    warnOptionalApiFailure,
    warnNetworkFailure,
    formatApiErrorMessage,
    extractTraceId,
  } = deps

  try {
    if (!tenantId.value) {
      console.log('跳过获取已安装镜像：租户ID无效')
      return
    }
    console.log('开始获取已安装镜像列表')
    isInstalledImagesLoading.value = true
    installedImagesError.value = null
    if (installedImagesErrorTraceId) installedImagesErrorTraceId.value = ''

    const url = `/api/cloud/installed-images/tenant_id/${tenantId.value}`

    console.log('请求URL:', url)
    const response = await apiFetch(url, {
      credentials: 'include',
      headers: {
        Accept: 'application/json',
      },
    })

    console.log('已安装镜像列表请求状态:', response.status)
    if (response.ok) {
      const data = await parseJsonSafe(response)
      if (data == null) {
        warnOptionalApiFailure('已安装镜像列表(JSON)', response)
        installedImagesError.value = '已安装镜像列表响应异常'
        if (installedImagesErrorTraceId) {
          installedImagesErrorTraceId.value = extractTraceId?.(response) || ''
        }
        installedImages.value = []
      } else {
        console.log('已安装镜像列表数据:', data)
        installedImages.value = normalizeListPayload(data)
      }
    } else {
      warnOptionalApiFailure('已安装镜像列表', response)
      const errBody = await parseJsonSafe(response)
      installedImagesError.value =
        formatApiErrorMessage(errBody) || `获取已安装镜像失败（HTTP ${response.status}）`
      if (installedImagesErrorTraceId) {
        installedImagesErrorTraceId.value =
          extractTraceId?.(response) || extractTraceId?.(errBody) || ''
      }
      installedImages.value = []
    }
  } catch (error) {
    warnNetworkFailure('已安装镜像列表', error)
    installedImagesError.value = '网络错误，请稍后重试'
    if (installedImagesErrorTraceId) {
      installedImagesErrorTraceId.value = extractTraceId?.(error) || ''
    }
    installedImages.value = []
  } finally {
    isInstalledImagesLoading.value = false
  }
}

export async function fetchCollaborators(deps) {
  const {
    tenantId,
    currentWorkspace,
    collaborators,
    apiFetch,
    parseJsonSafe,
    normalizeListPayload,
    warnOptionalApiFailure,
    warnNetworkFailure,
  } = deps

  try {
    if (!tenantId.value || !currentWorkspace.value?.id || currentWorkspace.value.id === 'default') {
      console.log('跳过获取协作人员：租户ID或工作空间ID无效')
      return
    }

    console.log('开始获取协作人员列表')

    const url = `/api/projects/workspace-access/workspace-collaborators/tenant_id/${tenantId.value}/?workspace_id=${currentWorkspace.value.id}`

    console.log('请求URL:', url)
    const response = await apiFetch(url, {
      credentials: 'include',
      headers: {
        Accept: 'application/json',
      },
    })

    console.log('协作人员列表请求状态:', response.status)
    if (response.ok) {
      const data = await parseJsonSafe(response)
      if (data == null) {
        warnOptionalApiFailure('协作人员列表(JSON)', response)
        collaborators.value = []
      } else {
        console.log('协作人员列表数据:', data)
        collaborators.value = normalizeListPayload(data)
      }
    } else {
      warnOptionalApiFailure('协作人员列表', response)
      collaborators.value = []
    }
  } catch (error) {
    warnNetworkFailure('协作人员列表', error)
    collaborators.value = []
  }
}

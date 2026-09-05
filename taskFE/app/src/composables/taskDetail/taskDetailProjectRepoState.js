import { ref, computed, watch } from 'vue'
import { apiFetch } from '../../utils/apiUtils.js'
import { resolveProjectServerRunTemplateFromProjects } from '../../utils/projectRunTemplateUtils.js'
import { syncTaskRepoAddressesFromProjects } from '../../utils/taskRepoAddressMismatch.js'
import {
  resolveRepoCloneIdentityFromMap,
  githubRepoBindingsBySlug,
  githubRepoSlugFromUrl,
  validateTaskRepoSavedAssociations,
  collectLinkedRepoBranchTargets,
  intersectBranchNameLists,
} from '../../utils/taskDetailBranchAndRepoUtils.js'
import { extractTraceId } from '../../utils/traceId.js'
import { humanizeRequestErrorMessage } from '../../utils/requestErrorDisplay.js'
import { buildTaskProjectsWithDetails } from '../../utils/taskProjectsWithDetails.js'
import { buildTaskRepoRows } from '../../utils/taskRepoRows.js'
import { mergeTaskDetailUpdate } from './mergeTaskDetailUpdate.js'
import { linkedProjectRowsFromTask } from '../../utils/unwrapTaskDetailPayload.js'

export function repoBranchMapKey(projectId, repoUrl) {
  return `${projectId}-${repoUrl}`
}

export function createTaskDetailProjectRepoState(deps) {
  const {
    effectiveTenantId,
    effectiveWorkspaceId,
    effectiveTaskId,
    localTask,
    editingTask,
    isEditing,
    // OPT-20260829-025: 同步成功后按需触发容器重新克隆的桥接 + 容器已注册标记
    repoRecloneBridge,
    containerEndpointRegistered,
  } = deps

  const linkedProjectsPanelRef = ref(null)
  const workspaceProjects = ref([])
  const workspaceProjectsLoading = ref(false)
  const repoCloneIdentityByUrl = ref({})
  const repoCloneIdentitySaveError = ref('')
  const repoCloneIdentitySaving = ref(false)
  const repoCloneIdentityUserTouchedByUrl = ref({})
  const repoCloneIdentityAutoApplyInFlight = ref(false)
  const recloneLoadingByUrl = ref({})
  const recloneStatusByUrl = ref({})
  const recloneErrorTraceIdByUrl = ref({})
  const repoRecloneGlobalLoading = ref(false)

  const linkedRows = () => linkedProjectRowsFromTask(localTask.value, effectiveTaskId.value)
  const projectServerRunTemplate = computed(() =>
    resolveProjectServerRunTemplateFromProjects(workspaceProjects.value, linkedRows()),
  )
  const taskProjectsWithDetails = computed(() =>
    buildTaskProjectsWithDetails(linkedRows(), workspaceProjects.value),
  )

  const staleRepoSyncLoading = ref(false)
  const staleRepoSyncError = ref('')
  // OPT-20260829-025: 同步触发重新克隆后为 true，直至所有关联仓库克隆完成
  const staleRepoSyncNeedsReclone = ref(false)

  const syncStaleTaskRepoAddresses = async () => {
    const tenantId = String(effectiveTenantId.value || '').trim()
    const workspaceId = String(effectiveWorkspaceId.value || '').trim()
    const taskId = String(effectiveTaskId.value || '').trim()
    if (!tenantId || !workspaceId || !taskId) {
      staleRepoSyncError.value = '缺少任务上下文，无法同步'
      return
    }
    staleRepoSyncLoading.value = true
    staleRepoSyncError.value = ''
    try {
      const data = await syncTaskRepoAddressesFromProjects({
        apiFetch,
        tenantId,
        workspaceId,
        taskId,
        projects: linkedRows(),
      })
      localTask.value = mergeTaskDetailUpdate(localTask.value, data)
      // OPT-20260829-025: PATCH 只改 task_projects.repo_address，运行中容器 origin 仍指向旧仓。
      // 容器已注册时对当前 git_repos 触发重新克隆，避免「徽章消失但推送仍失败」。
      if (repoRecloneBridge?.fn && containerEndpointRegistered?.value) {
        staleRepoSyncNeedsReclone.value = true
        for (const row of taskRepoRows.value) {
          repoRecloneBridge.fn({ repoUrl: row.url })
        }
      }
    } catch (error) {
      staleRepoSyncError.value = error?.message || '同步仓库地址失败'
    } finally {
      staleRepoSyncLoading.value = false
    }
  }

  const taskRepoRows = computed(() => buildTaskRepoRows(linkedRows(), workspaceProjects.value))

  // OPT-20260829-025: 全部关联仓库克隆完成后清除「需重新克隆」标记
  watch(
    recloneStatusByUrl,
    () => {
      const rows = taskRepoRows.value
      if (!staleRepoSyncNeedsReclone.value || !rows.length) return
      const allOk = rows.every((r) => String(recloneStatusByUrl.value[r.url] || '') === 'ok')
      if (allOk) staleRepoSyncNeedsReclone.value = false
    },
    { deep: true },
  )

  const syncRepoCloneIdentityMapFromTask = () => {
    const raw = localTask.value?.parameters?.repo_clone_git_identities
    const fromTask =
      raw && typeof raw === 'object' && !Array.isArray(raw) ? { ...raw } : {}
    const next = {}
    for (const row of taskRepoRows.value) {
      next[row.url] = resolveRepoCloneIdentityFromMap(fromTask, row.url)
    }
    repoCloneIdentityByUrl.value = next
  }

  const fetchWorkspaceProjects = async () => {
    const tid = effectiveTenantId.value
    const wid = effectiveWorkspaceId.value
    if (!tid || !wid) {
      workspaceProjects.value = []
      return
    }
    workspaceProjectsLoading.value = true
    repoCloneIdentitySaveError.value = ''
    try {
      const response = await apiFetch(
        `/api/projects/tenant_id/${tid}?workspace_id=${encodeURIComponent(String(wid))}`,
        {
          credentials: 'include',
          headers: { Accept: 'application/json' },
        },
      )
      const data = await response.json().catch(() => ({}))
      if (!response.ok) {
        throw new Error(
          typeof data.detail === 'string' ? data.detail : '加载项目列表失败',
        )
      }
      workspaceProjects.value = Array.isArray(data) ? data : data.results || []
      syncRepoCloneIdentityMapFromTask()
    } catch (error) {
      console.error('加载工作区项目失败:', error)
      repoCloneIdentitySaveError.value = humanizeRequestErrorMessage(error.message || '加载项目列表失败')
      workspaceProjects.value = []
    } finally {
      workspaceProjectsLoading.value = false
    }
  }

  const savedRepoCloneIdentityIdForUrl = (repoUrl) => {
    const m = localTask.value?.parameters?.repo_clone_git_identities
    return resolveRepoCloneIdentityFromMap(m, repoUrl)
  }

  const fetchTaskGithubRepoBindingsForValidation = async () => {
    const tenantId = effectiveTenantId.value
    const workspaceId = effectiveWorkspaceId.value
    const taskId = effectiveTaskId.value
    if (!tenantId || !workspaceId || !taskId) {
      return []
    }
    try {
      const response = await apiFetch(
        `/api/cloud/compute/github-credential-status/tenant_id/${tenantId}/workspace_id/${workspaceId}/task_id/${taskId}/`,
        { credentials: 'include', headers: { Accept: 'application/json' } },
      )
      const data = await response.json().catch(() => ({}))
      if (!response.ok) {
        return []
      }
      return Array.isArray(data.repo_bindings) ? data.repo_bindings : []
    } catch {
      return []
    }
  }

  const validateLinkedProjectReposBeforeSendToAi = async () => {
    const repoUrls = taskRepoRows.value.map((row) => row.url)
    if (repoUrls.length === 0) {
      return { ok: true, message: '' }
    }
    const repoBindings = await fetchTaskGithubRepoBindingsForValidation()
    void linkedProjectsPanelRef.value?.refreshGithubRepoBindingStatus?.()
    const bindingsBySlug = githubRepoBindingsBySlug(repoBindings)
    return validateTaskRepoSavedAssociations(
      repoUrls,
      (url) => savedRepoCloneIdentityIdForUrl(url),
      (url) => {
        const slug = githubRepoSlugFromUrl(url)
        if (!slug) return null
        const row = bindingsBySlug[slug]
        const raw = row?.selected_github_user_id
        if (raw == null) return null
        const normalized = String(raw).trim()
        return normalized || null
      },
    )
  }

  const repoCloneIdentityIdForUrl = (repoUrl) => {
    const u = String(repoUrl || '').trim()
    if (!u) return ''
    const fromUi = String(repoCloneIdentityByUrl.value[u] || '').trim()
    if (fromUi) return fromUi
    const m = localTask.value?.parameters?.repo_clone_git_identities
    return resolveRepoCloneIdentityFromMap(m, u)
  }

  function firstTaskRepoCloneIdentityId() {
    for (const r of taskRepoRows.value) {
      const id = String(repoCloneIdentityIdForUrl(r.url) || '').trim()
      if (id) return id
    }
    return ''
  }

  const getProjectRepos = (projectId) => {
    const project = workspaceProjects.value.find((p) => String(p.id) === String(projectId))
    if (project && Array.isArray(project.git_repos)) {
      return project.git_repos.filter((u) => u && String(u).trim())
    }
    return []
  }

  const buildProjectsApiPayload = (linkedProjects, workBranchName) => {
    const target = String(workBranchName || '').trim()
    const out = []
    for (const row of linkedProjects || []) {
      const projectId = row.project_id ? String(row.project_id) : ''
      if (!projectId) continue
      const repos = getProjectRepos(projectId)
      const rb = row.repo_branches || {}
      if (repos.length === 0) {
        out.push({
          project_id: projectId,
          repo_index: 0,
          base_branch: '',
          target_branch: target,
        })
        continue
      }
      for (let i = 0; i < repos.length; i += 1) {
        const url = repos[i]
        const base = (rb[url] || '').trim()
        out.push({
          project_id: projectId,
          repo_index: i,
          base_branch: base,
          target_branch: target,
        })
      }
    }
    return out
  }

  const getRepoBranchValue = (project, repoUrl) => {
    if (!project.repo_branches) {
      return ''
    }
    return project.repo_branches[repoUrl] || ''
  }

  const setRepoBranch = (project, repoUrl, value) => {
    if (!project.repo_branches) {
      project.repo_branches = {}
    }
    if (value && value.trim()) {
      project.repo_branches[repoUrl] = value.trim()
    } else {
      delete project.repo_branches[repoUrl]
    }
  }

  const repoBranchesCache = ref({})
  const repoBranchesLoading = ref({})
  const repoBranchesErrors = ref({})
  const repoBranchesErrorTraceIds = ref({})

  const getRepoBranches = (projectId, repoUrl) => {
    const key = repoBranchMapKey(projectId, repoUrl)
    return repoBranchesCache.value[key] || []
  }

  const getRepoBranchError = (projectId, repoUrl) => {
    const key = repoBranchMapKey(projectId, repoUrl)
    return repoBranchesErrors.value[key] || ''
  }

  const getRepoBranchErrorTraceId = (projectId, repoUrl) => {
    const key = repoBranchMapKey(projectId, repoUrl)
    return repoBranchesErrorTraceIds.value[key] || ''
  }

  const isLoadingRepoBranches = (projectId, repoUrl) => {
    const key = repoBranchMapKey(projectId, repoUrl)
    return repoBranchesLoading.value[key] || false
  }

  const fetchRepoBranches = async (projectId, repoUrl, force = false) => {
    const tenantId = effectiveTenantId.value
    if (!tenantId || !projectId || !repoUrl) return

    const key = repoBranchMapKey(projectId, repoUrl)
    if (!force && Array.isArray(repoBranchesCache.value[key]) && repoBranchesCache.value[key].length > 0) {
      return
    }
    repoBranchesLoading.value = { ...repoBranchesLoading.value, [key]: true }

    try {
      const response = await apiFetch(`/api/projects/tenant_id/${tenantId}/${projectId}/branches/?repo_url=${encodeURIComponent(repoUrl)}`, {
        method: 'GET',
        credentials: 'include',
        headers: { Accept: 'application/json' },
      })
      const data = await response.json().catch(() => ({}))
      if (response.ok) {
        repoBranchesCache.value = {
          ...repoBranchesCache.value,
          [key]: Array.isArray(data.branches) ? data.branches : [],
        }
        repoBranchesErrors.value = {
          ...repoBranchesErrors.value,
          [key]: data.error ? String(data.error) : '',
        }
        repoBranchesErrorTraceIds.value = {
          ...repoBranchesErrorTraceIds.value,
          [key]: data.error ? (extractTraceId(response) || extractTraceId(data) || '') : '',
        }
        return
      }
      const errMsg = typeof data.error === 'string' && data.error.trim()
        ? data.error.trim()
        : (typeof data.detail === 'string' && data.detail.trim()
          ? data.detail.trim()
          : `获取分支列表失败（HTTP ${response.status}）`)
      repoBranchesCache.value = {
        ...repoBranchesCache.value,
        [key]: Array.isArray(data.branches) ? data.branches : [],
      }
      repoBranchesErrors.value = {
        ...repoBranchesErrors.value,
        [key]: errMsg,
      }
      repoBranchesErrorTraceIds.value = {
        ...repoBranchesErrorTraceIds.value,
        [key]: extractTraceId(response) || extractTraceId(data) || '',
      }
    } catch (error) {
      console.error('Failed to fetch repo branches:', error)
      repoBranchesErrors.value = {
        ...repoBranchesErrors.value,
        [key]: error?.message || '网络错误，请稍后重试',
      }
      repoBranchesErrorTraceIds.value = {
        ...repoBranchesErrorTraceIds.value,
        [key]: extractTraceId(error) || '',
      }
    } finally {
      repoBranchesLoading.value = { ...repoBranchesLoading.value, [key]: false }
    }
  }

  const onProjectChange = (project) => {
    project.repo_branches = {}
    const repos = getProjectRepos(project.project_id)
    for (const repoUrl of repos) {
      const key = repoBranchMapKey(project.project_id, repoUrl)
      repoBranchesErrors.value = { ...repoBranchesErrors.value, [key]: '' }
      repoBranchesErrorTraceIds.value = { ...repoBranchesErrorTraceIds.value, [key]: '' }
      fetchRepoBranches(project.project_id, repoUrl, true)
    }
  }

  const linkedRepoBranchTargets = computed(() => {
    if (!isEditing.value || !editingTask.value) return []
    return collectLinkedRepoBranchTargets(editingTask.value.linkedProjects, getProjectRepos)
  })

  const fetchAllLinkedRepoBranches = async (force = false) => {
    const targets = linkedRepoBranchTargets.value
    if (targets.length === 0) return
    await Promise.all(
      targets.map(({ projectId, repoUrl }) => fetchRepoBranches(projectId, repoUrl, force)),
    )
  }

  const commonMergeTargetBranchesLoading = computed(() => {
    const targets = linkedRepoBranchTargets.value
    if (targets.length === 0) return false
    return targets.some(({ projectId, repoUrl }) => isLoadingRepoBranches(projectId, repoUrl))
  })

  const commonMergeTargetBranchesError = computed(() => {
    const targets = linkedRepoBranchTargets.value
    const errors = []
    for (const { projectId, repoUrl } of targets) {
      const err = getRepoBranchError(projectId, repoUrl)
      if (err) errors.push(err)
    }
    if (errors.length === 0) return ''
    return errors.length === 1 ? errors[0] : `部分仓库分支拉取失败（${errors.length}）`
  })

  const commonMergeTargetBranchesErrorTraceId = computed(() => {
    const targets = linkedRepoBranchTargets.value
    for (const { projectId, repoUrl } of targets) {
      const tid = getRepoBranchErrorTraceId(projectId, repoUrl)
      if (tid) return tid
    }
    return ''
  })

  const commonMergeTargetBranches = computed(() => {
    const targets = linkedRepoBranchTargets.value
    if (targets.length === 0) return []
    if (commonMergeTargetBranchesLoading.value) return []
    if (targets.some(({ projectId, repoUrl }) => getRepoBranchError(projectId, repoUrl))) {
      return []
    }
    const lists = targets.map(({ projectId, repoUrl }) => getRepoBranches(projectId, repoUrl))
    return intersectBranchNameLists(lists)
  })

  return {
    linkedProjectsPanelRef,
    workspaceProjects,
    workspaceProjectsLoading,
    projectServerRunTemplate,
    repoCloneIdentityByUrl,
    repoCloneIdentitySaveError,
    repoCloneIdentitySaving,
    repoCloneIdentityUserTouchedByUrl,
    repoCloneIdentityAutoApplyInFlight,
    recloneLoadingByUrl,
    recloneStatusByUrl,
    recloneErrorTraceIdByUrl,
    repoRecloneGlobalLoading,
    taskProjectsWithDetails,
    staleRepoSyncLoading,
    staleRepoSyncError,
    staleRepoSyncNeedsReclone,
    syncStaleTaskRepoAddresses,
    taskRepoRows,
    syncRepoCloneIdentityMapFromTask,
    fetchWorkspaceProjects,
    savedRepoCloneIdentityIdForUrl,
    validateLinkedProjectReposBeforeSendToAi,
    repoCloneIdentityIdForUrl,
    firstTaskRepoCloneIdentityId,
    getProjectRepos,
    buildProjectsApiPayload,
    getRepoBranchValue,
    setRepoBranch,
    repoBranchesCache,
    repoBranchesLoading,
    repoBranchesErrors,
    getRepoBranches,
    getRepoBranchError,
    getRepoBranchErrorTraceId,
    isLoadingRepoBranches,
    fetchRepoBranches,
    onProjectChange,
    linkedRepoBranchTargets,
    fetchAllLinkedRepoBranches,
    commonMergeTargetBranchesLoading,
    commonMergeTargetBranchesError,
    commonMergeTargetBranchesErrorTraceId,
    commonMergeTargetBranches,
  }
}

export function installTaskDetailProjectRepoWatchers(state, deps) {
  const { localTask, isEditing, effectiveTaskId } = deps

  watch(effectiveTaskId, () => {
    state.repoCloneIdentityUserTouchedByUrl.value = {}
  })

  watch([state.workspaceProjects, localTask], () => state.syncRepoCloneIdentityMapFromTask(), { deep: true })

  watch(
    state.linkedRepoBranchTargets,
    (targets, prev) => {
      if (!isEditing.value) return
      const nextKey = targets.map((t) => `${t.projectId}::${t.repoUrl}`).join('|')
      const prevKey = (prev || []).map((t) => `${t.projectId}::${t.repoUrl}`).join('|')
      if (nextKey === prevKey) return
      if (targets.length === 0) return
      void state.fetchAllLinkedRepoBranches(true)
    },
    { deep: true },
  )
}

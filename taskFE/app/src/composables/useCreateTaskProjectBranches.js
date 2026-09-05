import { ref, watch } from 'vue'
import { apiFetch } from '../utils/apiUtils.js'
import { createDatalistDismissController } from '../utils/datalistDismiss.js'
import { parseJsonSafe, warnNetworkFailure, warnOptionalApiFailure } from '../utils/workPanelApiUtils.js'
import { resolveRepoOAuthAuthorizeLabel } from '../utils/repoOAuthAuthorizeUtils.js'
import { loadProjectBranchListState } from '../utils/projectBranchListFetch.js'
import { mapRepoAccessApiToLabel } from '../utils/repoAccessLabelUtils.js'
import { normalizeEditingTaskNestedObjects as normalizeEditingTaskNestedObjectsImpl } from '../utils/createTaskModalNormalize.js'
import { collectProjectGitRepoUrls } from '../utils/gitRepoUrlUtils.js'
import {
  pickCreateTaskDefaultProjectId,
  readCreateTaskPreferredProject,
  rememberCreateTaskPreferredProject,
} from '../utils/createTaskPreferredProject.js'
import { useBaseBranchCommitCheck } from './useBaseBranchCommitCheck.js'

/**
 * @param {{
 *   editingTask: () => object|null,
 *   projects: () => array,
 *   taskStatuses: () => array,
 *   taskTypes: () => array,
 *   tenantId: () => string|number|null,
 *   show: () => boolean,
 *   isProjectsLoading: () => boolean,
 *   initBranchDefaults?: () => void,
 *   datalistController?: ReturnType<typeof createDatalistDismissController>,
 * }} opts
 */
export function useCreateTaskProjectBranches({
  editingTask,
  projects,
  taskStatuses,
  taskTypes,
  tenantId,
  show,
  isProjectsLoading,
  initBranchDefaults,
  datalistController,
}) {
  const projectSelectionRowKeySeq = ref(0)
  const projectRepoAccessibility = ref({})
  const projectBranchesMap = ref({})

  const nextProjectSelectionRowKey = () => {
    projectSelectionRowKeySeq.value += 1
    return `project-selection-row-${projectSelectionRowKeySeq.value}`
  }

  const branchMapKey = (projectId, repoIndex) => `${String(projectId)}::${Number(repoIndex)}`

  const getProjectById = (projectId) => {
    if (!projectId) return null
    return projects().find((p) => String(p.id) === String(projectId))
  }

  const getRepoUrl = (projectId, repoIndex) => {
    const project = getProjectById(projectId)
    const repos = collectProjectGitRepoUrls(project)
    const idx = Number(repoIndex)
    if (repos.length > 0 && idx >= 0 && idx < repos.length) {
      return String(repos[idx])
    }
    return ''
  }

  const {
    isBaseBranchCommitChecking,
    isBaseBranchCommitVerified,
    isBaseBranchCommitMissing,
    getBaseBranchCommitCheckTraceId,
    getBaseBranchCommitMissingHint,
    scheduleBaseBranchCommitCheck,
    resetAllCommitChecks,
  } = useBaseBranchCommitCheck({
    tenantId,
    getRepoUrl,
  })

  const {
    dismissIfPicked: dismissDatalistIfPicked,
    restoreOnFocus: restoreDatalistOnFocus,
    listAttr: datalistListAttr,
  } = datalistController || createDatalistDismissController()

  const getAllSelectedProjectIds = () => {
    const selections = Array.isArray(editingTask()?.projectSelections) ? editingTask().projectSelections : []
    const set = new Set()
    selections.forEach((sel) => {
      const id = sel?.projectId
      if (id) set.add(String(id))
    })
    return set
  }

  const getOtherSelectedProjectIds = (rowIndex) => {
    const selections = Array.isArray(editingTask()?.projectSelections) ? editingTask().projectSelections : []
    const set = new Set()
    selections.forEach((sel, idx) => {
      if (idx === rowIndex) return
      const id = sel?.projectId
      if (id) set.add(String(id))
    })
    return set
  }

  const getProjectsForSelectionRow = (rowIndex) => {
    const otherIds = getOtherSelectedProjectIds(rowIndex)
    const currentId = editingTask()?.projectSelections?.[rowIndex]?.projectId
    const cur = currentId ? String(currentId) : ''
    return projects().filter(
      (p) => !otherIds.has(String(p.id)) || (cur !== '' && String(p.id) === cur),
    )
  }

  const pickFirstUnusedProjectId = () => {
    const used = getAllSelectedProjectIds()
    const preferred = readCreateTaskPreferredProject(tenantId())
    return pickCreateTaskDefaultProjectId(projects(), used, preferred)
  }

  const createEmptyProjectSelection = () => ({
    selectionRowKey: nextProjectSelectionRowKey(),
    projectId: pickFirstUnusedProjectId(),
    repoBranches: [],
  })

  const syncSelectionRepoBranches = (selection) => {
    if (!selection) return
    const project = getProjectById(selection.projectId)
    const repos = collectProjectGitRepoUrls(project)
    const existing = Array.isArray(selection.repoBranches) ? selection.repoBranches : []

    const pickPrev = (idx) => {
      const byIndex = existing.find((r) => Number(r.repoIndex) === idx)
      if (byIndex?.baseBranch != null && String(byIndex.baseBranch).trim() !== '') {
        return String(byIndex.baseBranch).trim()
      }
      const byPos = existing[idx]
      if (byPos?.baseBranch != null && String(byPos.baseBranch).trim() !== '') {
        return String(byPos.baseBranch).trim()
      }
      return ''
    }

    if (repos.length === 0) {
      selection.repoBranches = [{ repoIndex: 0, baseBranch: pickPrev(0) }]
      return
    }

    selection.repoBranches = repos.map((_, idx) => ({
      repoIndex: idx,
      baseBranch: pickPrev(idx),
    }))
  }

  const normalizeEditingTaskNestedObjects = () => {
    normalizeEditingTaskNestedObjectsImpl(editingTask(), {
      taskStatuses: taskStatuses(),
      taskTypes: taskTypes(),
    })
  }

  const ensureTaskBranchConfig = () => {
    const task = editingTask()
    if (!task) return

    normalizeEditingTaskNestedObjects()

    const currentSelections = Array.isArray(task.projectSelections) ? task.projectSelections : []

    if (currentSelections.length === 0) {
      task.projectSelections = [createEmptyProjectSelection()]
    } else {
      const firstSelection = currentSelections[0]
      task.projectSelections = [{
        selectionRowKey: firstSelection?.selectionRowKey || nextProjectSelectionRowKey(),
        projectId: firstSelection?.projectId ? String(firstSelection.projectId) : '',
        repoBranches: Array.isArray(firstSelection?.repoBranches) ? firstSelection.repoBranches : [],
      }]
    }

    task.projectSelections.forEach((sel) => {
      syncSelectionRepoBranches(sel)
    })

    if (typeof initBranchDefaults === 'function') {
      initBranchDefaults()
    }
  }

  const getProjectDisplayLabel = (project) => {
    const accessibility = projectRepoAccessibility.value[project.id] ?? '检查中'
    return `(${accessibility})  ${project.name}`
  }

  const fetchProjectRepoAccessibility = async (projectId) => {
    if (!tenantId()) return
    try {
      const response = await apiFetch(`/api/projects/tenant_id/${tenantId()}/${projectId}/repo-access-check/`, {
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
      })
      if (response.ok) {
        const data = await parseJsonSafe(response)
        if (!data || typeof data !== 'object') {
          warnOptionalApiFailure(`repo-access-check JSON project=${projectId}`, response)
          projectRepoAccessibility.value[projectId] = '不可访问'
          return
        }
        projectRepoAccessibility.value[projectId] = mapRepoAccessApiToLabel(data)
      } else {
        if (response.status >= 500) {
          warnOptionalApiFailure(`repo-access-check project=${projectId}`, response)
        }
        projectRepoAccessibility.value[projectId] = '不可访问'
      }
    } catch (e) {
      warnNetworkFailure(`repo-access-check project=${projectId}`, e)
      projectRepoAccessibility.value[projectId] = '不可访问'
    }
  }

  const fetchAllProjectsRepoAccessibility = async () => {
    const list = projects()
    if (!tenantId() || !list?.length) return
    list.forEach((p) => {
      projectRepoAccessibility.value[p.id] = '检查中'
    })
    projectRepoAccessibility.value = { ...projectRepoAccessibility.value }
    await Promise.all(list.map((p) => fetchProjectRepoAccessibility(p.id)))
    projectRepoAccessibility.value = { ...projectRepoAccessibility.value }
  }

  const fetchProjectBranches = async (projectId, repoIndex = 0, force = false) => {
    if (!projectId || !tenantId()) return

    const normalizedProjectId = String(projectId)
    const idx = Number(repoIndex)
    const mapKey = branchMapKey(normalizedProjectId, idx)
    const existingState = projectBranchesMap.value[mapKey]
    if (!force && existingState?.loaded) return

    projectBranchesMap.value = {
      ...projectBranchesMap.value,
      [mapKey]: {
        branches: existingState?.branches || [],
        loading: true,
        error: null,
        loaded: false,
      },
    }

    const nextState = await loadProjectBranchListState(
      tenantId(),
      normalizedProjectId,
      getRepoUrl(normalizedProjectId, idx),
    )
    projectBranchesMap.value = {
      ...projectBranchesMap.value,
      [mapKey]: nextState,
    }
  }

  const getProjectBranches = (projectId, repoIndex = 0) => {
    if (!projectId) return []
    const branchState = projectBranchesMap.value[branchMapKey(projectId, repoIndex)]
    return Array.isArray(branchState?.branches) ? branchState.branches : []
  }

  const getProjectBranchesError = (projectId, repoIndex = 0) => {
    if (!projectId) return ''
    return projectBranchesMap.value[branchMapKey(projectId, repoIndex)]?.error || ''
  }

  const getProjectBranchesErrorTraceId = (projectId, repoIndex = 0) => {
    if (!projectId) return ''
    return projectBranchesMap.value[branchMapKey(projectId, repoIndex)]?.traceId || ''
  }

  const isProjectBranchesLoading = (projectId, repoIndex = 0) => {
    if (!projectId) return false
    return Boolean(projectBranchesMap.value[branchMapKey(projectId, repoIndex)]?.loading)
  }

  const refreshRepoBranches = (projectId, repoIndex) => {
    if (!projectId) return
    fetchProjectBranches(projectId, repoIndex, true)
  }

  const getRepoBranchRowLabel = (projectId, repoIndex) => {
    const url = getRepoUrl(projectId, repoIndex)
    const n = Number(repoIndex) + 1
    if (url) {
      const short = url.length > 56 ? `${url.slice(0, 53)}…` : url
      return `仓库 ${n}：${short}`
    }
    return `仓库 ${n}（未配置 URL，可手动填写基准分支）`
  }

  const getRepoOAuthSiteLabel = (projectId, repoIndex = 0) =>
    resolveRepoOAuthAuthorizeLabel(getRepoUrl(projectId, repoIndex))

  const onProjectSelectionChange = (selection) => {
    if (!selection?.projectId) return
    selection.projectId = String(selection.projectId)
    rememberCreateTaskPreferredProject(tenantId(), selection.projectId)
    syncSelectionRepoBranches(selection)
    if (Array.isArray(selection.repoBranches)) {
      selection.repoBranches.forEach((rb) => {
        fetchProjectBranches(selection.projectId, rb.repoIndex, true)
      })
    }
  }

  const onBaseBranchDatalistEvent = (selection, rb, event) => {
    const inputEl = event?.target
    const value = String(inputEl?.value ?? '')
    const candidates = getProjectBranches(selection.projectId, rb.repoIndex)
    dismissDatalistIfPicked(inputEl, candidates, value, event)
    scheduleBaseBranchCommitCheck(selection?.projectId, rb?.repoIndex, value)
  }

  const fetchBranchesForAllSelections = () => {
    editingTask()?.projectSelections?.forEach((sel) => {
      if (!sel?.projectId || !Array.isArray(sel.repoBranches)) return
      sel.repoBranches.forEach((rb) => {
        fetchProjectBranches(sel.projectId, rb.repoIndex)
      })
    })
  }

  const onPrimaryProjectChange = (newProjectId) => {
    if (!newProjectId) return
    rememberCreateTaskPreferredProject(tenantId(), newProjectId)
    const task = editingTask()
    if (!task) return
    const firstSel = task.projectSelections?.[0]
    if (firstSel?.projectId === String(newProjectId) && Array.isArray(firstSel.repoBranches)) {
      firstSel.repoBranches.forEach((rb) => fetchProjectBranches(newProjectId, rb.repoIndex))
    } else {
      fetchProjectBranches(newProjectId, 0)
    }
  }

  watch(
    () => [show(), tenantId()],
    () => {
      if (!show()) return
      fetchAllProjectsRepoAccessibility()
      ensureTaskBranchConfig()
      fetchBranchesForAllSelections()
    },
    { immediate: true },
  )

  watch(
    () => projects(),
    () => {
      if (!show() || !editingTask() || isProjectsLoading()) return
      ensureTaskBranchConfig()
      fetchBranchesForAllSelections()
    },
    { deep: true },
  )

  watch(
    () => editingTask()?.projectSelections,
    (newSelections) => {
      if (!Array.isArray(newSelections)) return
      newSelections.forEach((selection) => {
        if (selection?.projectId && Array.isArray(selection.repoBranches)) {
          selection.repoBranches.forEach((rb) => {
            fetchProjectBranches(selection.projectId, rb.repoIndex)
          })
        }
      })
    },
    { deep: true },
  )

  watch(
    () => show(),
    (visible) => {
      if (!visible) {
        resetAllCommitChecks()
      }
    },
  )

  return {
    branchMapKey,
    getProjectById,
    getRepoUrl,
    getProjectsForSelectionRow,
    getProjectDisplayLabel,
    getProjectBranches,
    getProjectBranchesError,
    getProjectBranchesErrorTraceId,
    isProjectBranchesLoading,
    getRepoBranchRowLabel,
    getRepoOAuthSiteLabel,
    refreshRepoBranches,
    onProjectSelectionChange,
    onBaseBranchDatalistEvent,
    datalistListAttr,
    restoreDatalistOnFocus,
    ensureTaskBranchConfig,
    fetchBranchesForAllSelections,
    onPrimaryProjectChange,
    resetAllCommitChecks,
    isBaseBranchCommitChecking,
    isBaseBranchCommitVerified,
    isBaseBranchCommitMissing,
    getBaseBranchCommitCheckTraceId,
    getBaseBranchCommitMissingHint,
    scheduleBaseBranchCommitCheck,
  }
}

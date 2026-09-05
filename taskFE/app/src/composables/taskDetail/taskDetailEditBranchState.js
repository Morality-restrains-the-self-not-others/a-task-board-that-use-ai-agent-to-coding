/**
 * Factory: task edit shell + work/merge branch naming from title presets.
 */
import { ref, watch } from 'vue'
import { apiFetch } from '../../utils/apiUtils.js'
import {
  extractCompanyNickname,
  sanitizeBranchSegment,
  hasChineseCharacter,
  buildWorkBranchName as buildWorkBranchNameUtil,
  buildMergeTargetBranchName as buildMergeTargetBranchNameUtil,
} from '../../utils/taskDetailBranchAndRepoUtils.js'
import { fetchTranslatedTaskTitleSegment as _fetchTranslatedTaskTitleSegment, startEdit as _startEdit } from './taskDetailFetchFns.js'
import {
  isTitleTranslateAbortError,
  userFacingTitleTranslationError,
} from '../../utils/titleTranslationError.js'

const TITLE_TRANSLATION_DEBOUNCE_MS = 500

/**
 * @param {object} deps
 * @param {import('vue').Ref} deps.effectiveTenantId
 * @param {import('vue').Ref} deps.effectiveTaskId
 * @param {import('vue').Ref} deps.localTask
 * @param {import('vue').Ref} deps.editingTask
 * @param {import('vue').Ref} deps.isEditing
 * @param {import('vue').Ref} deps.editError
 * @param {import('vue').Ref} deps.workspaceProjects
 * @param {Function} deps.getProjectRepos
 * @param {Function} deps.inferWorkBranchPreset
 * @param {Function} deps.inferMergeTargetPreset
 * @param {Function} [deps.fetchWorkspaceTodosForParent]
 * @param {Function} [deps.fetchAllLinkedRepoBranches]
 */
export function createTaskDetailEditBranchState(deps) {
  const {
    effectiveTenantId,
    effectiveTaskId,
    localTask,
    editingTask,
    isEditing,
    editError,
    workspaceProjects,
    getProjectRepos,
    inferWorkBranchPreset,
    inferMergeTargetPreset,
    fetchWorkspaceTodosForParent,
    fetchAllLinkedRepoBranches,
  } = deps

  const companyUserName = ref('')
  const isTaskTitleTranslating = ref(false)
  const taskTitleTranslationError = ref('')
  const translatedTaskTitleSegmentMap = ref({})
  let taskTitleTranslationTimer = null
  let taskTitleTranslationRequestId = 0
  let titleTranslateAbort = null

  const buildWorkBranchName = (presetType, taskTitleSegment = '', taskId = '') =>
    buildWorkBranchNameUtil(presetType, taskTitleSegment, taskId, {
      companyUserName: companyUserName.value,
      localTaskTitle: localTask.value?.title,
      customWorkBranchFallback: editingTask.value?.workBranchName || '',
      taskCreatedAt: localTask.value?.created_at || editingTask.value?.created_at,
    })

  const buildMergeTargetBranchName = (presetType, taskId = '') =>
    buildMergeTargetBranchNameUtil(presetType, taskId, {
      customMergeTargetFallback: editingTask.value?.mergeTargetName || '',
    })

  const fetchCurrentCompanyUserName = async (tenantIdValue) => {
    if (!tenantIdValue) {
      companyUserName.value = ''
      return
    }

    try {
      const profileResponse = await apiFetch('/api/accounts/users/profile/', {
        credentials: 'include',
        headers: {
          Accept: 'application/json',
        },
      })

      if (profileResponse.ok) {
        const profileData = await profileResponse.json()
        const companyNickname = extractCompanyNickname(profileData, tenantIdValue)
        if (companyNickname) {
          companyUserName.value = companyNickname
          return
        }
      }
    } catch (error) {
      console.warn('获取公司昵称失败:', error)
    }

    companyUserName.value = ''
  }

  const fetchTranslatedTaskTitleSegment = (taskTitle) => {
    titleTranslateAbort?.abort()
    const ac = new AbortController()
    titleTranslateAbort = ac
    return _fetchTranslatedTaskTitleSegment(taskTitle, {
      effectiveTenantId,
      sanitizeBranchSegment,
      signal: ac.signal,
    })
  }

  const resolveTaskTitleSegment = async () => {
    const rawTitle = String(editingTask.value?.title || '').trim()
    if (!rawTitle) {
      return 'task'
    }
    if (!hasChineseCharacter(rawTitle)) {
      taskTitleTranslationError.value = ''
      return sanitizeBranchSegment(rawTitle, 'task')
    }
    if (translatedTaskTitleSegmentMap.value[rawTitle]) {
      taskTitleTranslationError.value = ''
      return translatedTaskTitleSegmentMap.value[rawTitle]
    }
    isTaskTitleTranslating.value = true
    try {
      const translatedSegment = await fetchTranslatedTaskTitleSegment(rawTitle)
      translatedTaskTitleSegmentMap.value = {
        ...translatedTaskTitleSegmentMap.value,
        [rawTitle]: translatedSegment,
      }
      taskTitleTranslationError.value = ''
      return translatedSegment
    } catch (error) {
      if (isTitleTranslateAbortError(error)) {
        throw error
      }
      taskTitleTranslationError.value = userFacingTitleTranslationError(error?.message)
      throw error
    } finally {
      isTaskTitleTranslating.value = false
    }
  }

  const updateWorkBranchNameByPreset = async () => {
    if (!editingTask.value) return
    const preset = editingTask.value.workBranchPreset
    if (!preset || preset === 'custom') return
    const currentRequestId = taskTitleTranslationRequestId + 1
    taskTitleTranslationRequestId = currentRequestId
    try {
      const titleSegment = await resolveTaskTitleSegment()
      if (currentRequestId !== taskTitleTranslationRequestId) {
        return
      }
      const taskId = effectiveTaskId.value
      editingTask.value.workBranchName = buildWorkBranchName(preset, titleSegment, taskId)
    } catch (error) {
      if (isTitleTranslateAbortError(error)) {
        return
      }
      const fallbackSegment = sanitizeBranchSegment(editingTask.value?.title, 'task')
      const taskId = effectiveTaskId.value
      editingTask.value.workBranchName = buildWorkBranchName(preset, fallbackSegment, taskId)
    }
  }

  const scheduleWorkBranchNameUpdate = () => {
    if (taskTitleTranslationTimer) {
      clearTimeout(taskTitleTranslationTimer)
    }
    taskTitleTranslationTimer = setTimeout(() => {
      void updateWorkBranchNameByPreset()
    }, TITLE_TRANSLATION_DEBOUNCE_MS)
  }

  const updateMergeTargetBranchNameByPreset = () => {
    if (!editingTask.value) return
    const preset = editingTask.value.mergeTargetPreset
    if (!preset || preset === 'custom') return
    const taskId = effectiveTaskId.value
    editingTask.value.mergeTargetName = buildMergeTargetBranchName(preset, taskId)
  }

  const applyWorkBranchPreset = () => {
    scheduleWorkBranchNameUpdate()
  }

  const applyMergeTargetPreset = () => {
    updateMergeTargetBranchNameByPreset()
  }

  const startEdit = () => {
    _startEdit({
      localTask,
      workspaceProjects,
      editingTask,
      isEditing,
      editError,
      getProjectRepos,
      inferWorkBranchPreset,
      inferMergeTargetPreset,
    })
    void fetchWorkspaceTodosForParent?.()
    void fetchAllLinkedRepoBranches?.(true)
  }

  const cancelEdit = () => {
    titleTranslateAbort?.abort()
    isEditing.value = false
    editingTask.value = null
    editError.value = ''
  }

  const addProjectAssociation = () => {
    if (!editingTask.value) return
    if (!Array.isArray(editingTask.value.linkedProjects)) {
      editingTask.value.linkedProjects = []
    }
    if (editingTask.value.linkedProjects.length >= 1) return
    editingTask.value.linkedProjects.push({ project_id: '', repo_branches: {} })
  }

  const removeProjectAssociation = (index) => {
    if (editingTask.value.linkedProjects && editingTask.value.linkedProjects.length > 0) {
      editingTask.value.linkedProjects.splice(index, 1)
    }
  }

  return {
    companyUserName,
    isTaskTitleTranslating,
    taskTitleTranslationError,
    translatedTaskTitleSegmentMap,
    buildWorkBranchName,
    buildMergeTargetBranchName,
    fetchCurrentCompanyUserName,
    fetchTranslatedTaskTitleSegment,
    resolveTaskTitleSegment,
    updateWorkBranchNameByPreset,
    scheduleWorkBranchNameUpdate,
    updateMergeTargetBranchNameByPreset,
    applyWorkBranchPreset,
    applyMergeTargetPreset,
    startEdit,
    cancelEdit,
    addProjectAssociation,
    removeProjectAssociation,
  }
}

/**
 * @param {ReturnType<typeof createTaskDetailEditBranchState>} state
 * @param {{ editingTask: import('vue').Ref }} deps
 */
export function installTaskDetailEditBranchWatchers(state, deps) {
  const { editingTask } = deps
  watch(
    () => editingTask.value?.title,
    () => {
      if (!editingTask.value) return
      if (editingTask.value.workBranchPreset && editingTask.value.workBranchPreset !== 'custom') {
        state.scheduleWorkBranchNameUpdate()
      }
    },
  )
}

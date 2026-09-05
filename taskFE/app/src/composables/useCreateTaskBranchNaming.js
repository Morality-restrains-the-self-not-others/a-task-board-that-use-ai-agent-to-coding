import { computed, getCurrentInstance, onBeforeUnmount, ref, watch } from 'vue'
import { apiFetch } from '../utils/apiUtils.js'
import { WORK_BRANCH_PRESET_OPTIONS } from '../utils/workBranchPresetOptions.js'
import { resolveTaskCreatedAtDate } from '../utils/workPanelBranchHelpers.js'
import {
  intersectBranchNameLists,
  pickPreferredCommonMergeTargetBranch,
} from '../utils/taskDetailBranchAndRepoUtils.js'
import { parseJsonSafe, warnOptionalApiFailure } from '../utils/workPanelApiUtils.js'
import { createDatalistDismissController } from '../utils/datalistDismiss.js'
import { extractTraceId } from '../utils/traceId.js'
import {
  isTitleTranslateAbortError,
  userFacingTitleTranslationError,
} from '../utils/titleTranslationError.js'

/**
 * @param {{
 *   editingTask: () => object|null,
 *   companyUserName: () => string,
 *   tenantId: () => string|number|null,
 *   show: () => boolean,
 *   getProjectBranches: (projectId: string, repoIndex?: number) => string[],
 *   isProjectBranchesLoading: (projectId: string, repoIndex?: number) => boolean,
 *   getProjectBranchesError: (projectId: string, repoIndex?: number) => string,
 *   branchMapKey: (projectId: string, repoIndex: number) => string,
 *   datalistController?: ReturnType<typeof createDatalistDismissController>,
 * }} opts
 */
export function useCreateTaskBranchNaming({
  editingTask,
  companyUserName,
  tenantId,
  show,
  getProjectBranches,
  isProjectBranchesLoading,
  getProjectBranchesError,
  branchMapKey,
  datalistController,
}) {
  const isTaskTitleTranslating = ref(false)
  const taskTitleTranslationError = ref('')
  const taskTitleTranslationErrorTraceId = ref('')
  const translatedTaskTitleSegmentMap = ref({})
  const mergeTargetUserEdited = ref(false)
  const lastAutoMergeTargetName = ref('')
  let taskTitleTranslationTimer = null
  let taskTitleTranslationRequestId = 0
  let titleTranslateAbort = null
  const TITLE_TRANSLATION_DEBOUNCE_MS = 500

  const workBranchPresetOptions = WORK_BRANCH_PRESET_OPTIONS
  const mergeTargetPresetOptions = [
    { value: 'develop', label: 'develop' },
    { value: 'release', label: 'release/${日期}_daydaymoney${taskId}' },
    { value: 'main', label: 'main' },
    { value: 'custom', label: '自定义' },
  ]

  const sanitizeBranchSegment = (rawValue, fallbackValue = 'unknown') => {
    const normalizedValue = String(rawValue || '').trim().replace(/\s+/g, '_')
    const sanitizedValue = normalizedValue.replace(/[^a-zA-Z0-9._-]/g, '_')
    return sanitizedValue || fallbackValue
  }

  const hasChineseCharacter = (rawValue) => /[\u3400-\u9fff]/.test(String(rawValue || ''))

  const formatDateYmd = (date = new Date()) => {
    const pad = (value) => String(value).padStart(2, '0')
    return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`
  }

  const buildWorkBranchName = (presetType, taskTitleSegment = '') => {
    const task = editingTask()
    const taskCreatedDateYmd = formatDateYmd(resolveTaskCreatedAtDate(task))
    const companyName = sanitizeBranchSegment(companyUserName(), 'company_user_name')
    const taskId = task?.id ? String(task.id) : '${taskId}'
    const fallbackTitle = task?.title ? sanitizeBranchSegment(task.title, 'task') : 'task'
    const taskTitle = taskTitleSegment || fallbackTitle

    if (presetType === 'release') {
      return `release/${taskCreatedDateYmd}_daydaymoney${taskId}`
    }
    if (presetType === 'bugfix' || presetType === 'hotfix' || presetType === 'feature') {
      return `${presetType}/${taskCreatedDateYmd}_${companyName}_daydaymoney${taskId}_${taskTitle}`
    }
    return task?.workBranchName || ''
  }

  const buildMergeTargetBranchName = (presetType) => {
    const task = editingTask()
    const today = formatDateYmd()
    const taskId = task?.id ? String(task.id) : '${taskId}'
    if (presetType === 'develop' || presetType === 'main') {
      return presetType
    }
    if (presetType === 'release') {
      return `release/${today}_daydaymoney${taskId}`
    }
    return task?.mergeTargetName || ''
  }

  const resolveSyncTitleSegmentForBranch = () => {
    const task = editingTask()
    const rawTitle = String(task?.title || '').trim()
    if (!rawTitle) return 'task'
    if (translatedTaskTitleSegmentMap.value[rawTitle]) {
      return translatedTaskTitleSegmentMap.value[rawTitle]
    }
    if (!hasChineseCharacter(rawTitle)) {
      return sanitizeBranchSegment(rawTitle, 'task')
    }
    return sanitizeBranchSegment(rawTitle, 'task')
  }

  const workBranchDatalistOptions = computed(() => {
    if (!editingTask()) return []
    const titleSegment = resolveSyncTitleSegmentForBranch()
    return workBranchPresetOptions
      .filter((preset) => preset.value !== 'custom')
      .map((preset) => ({
        preset: preset.value,
        value: buildWorkBranchName(preset.value, titleSegment),
        label: preset.label,
      }))
  })

  const commonMergeTargetBranchTargets = computed(() => {
    const out = []
    const seen = new Set()
    for (const sel of editingTask()?.projectSelections || []) {
      const projectId = sel?.projectId ? String(sel.projectId) : ''
      if (!projectId || !Array.isArray(sel.repoBranches)) continue
      for (const rb of sel.repoBranches) {
        const repoIndex = Number(rb?.repoIndex)
        if (!Number.isFinite(repoIndex)) continue
        const key = branchMapKey(projectId, repoIndex)
        if (seen.has(key)) continue
        seen.add(key)
        out.push({ projectId, repoIndex })
      }
    }
    return out
  })

  const commonMergeTargetBranchesLoading = computed(() =>
    commonMergeTargetBranchTargets.value.some(({ projectId, repoIndex }) =>
      isProjectBranchesLoading(projectId, repoIndex),
    ),
  )

  const commonMergeTargetBranches = computed(() => {
    const targets = commonMergeTargetBranchTargets.value
    if (targets.length === 0) return []
    if (commonMergeTargetBranchesLoading.value) return []
    if (targets.some(({ projectId, repoIndex }) => getProjectBranchesError(projectId, repoIndex))) {
      return []
    }
    const lists = targets.map(({ projectId, repoIndex }) => getProjectBranches(projectId, repoIndex))
    return intersectBranchNameLists(lists)
  })

  const mergeTargetDatalistOptions = computed(() => {
    if (!editingTask()) return []
    return commonMergeTargetBranches.value.map((branch) => ({
      preset: `common:${branch}`,
      value: branch,
      label: `${branch} [共有]`,
    }))
  })

  const syncWorkBranchPresetFromName = () => {
    const task = editingTask()
    if (!task) return
    const branchName = String(task.workBranchName || '')
    const titleSegment = resolveSyncTitleSegmentForBranch()
    let matchedPreset = 'custom'
    for (const preset of workBranchPresetOptions) {
      if (preset.value === 'custom') continue
      if (buildWorkBranchName(preset.value, titleSegment) === branchName) {
        matchedPreset = preset.value
        break
      }
    }
    task.workBranchPreset = matchedPreset
  }

  const syncMergeTargetPresetFromName = () => {
    const task = editingTask()
    if (!task) return
    const branchName = String(task.mergeTargetName || '')
    let matchedPreset = 'custom'
    for (const preset of mergeTargetPresetOptions) {
      if (preset.value === 'custom') continue
      if (buildMergeTargetBranchName(preset.value) === branchName) {
        matchedPreset = preset.value
        break
      }
    }
    task.mergeTargetPreset = matchedPreset
  }

  const markMergeTargetUserEdited = () => {
    mergeTargetUserEdited.value = true
  }

  const applyPreferredMergeTargetFromCommonBranches = () => {
    const task = editingTask()
    if (!show() || !task) return
    if (task.id) return
    if (commonMergeTargetBranchesLoading.value) return
    if (mergeTargetUserEdited.value) return
    if (commonMergeTargetBranchTargets.value.length === 0) return

    const preferred = pickPreferredCommonMergeTargetBranch(commonMergeTargetBranches.value)
    const current = String(task.mergeTargetName || '').trim()
    const canOverwrite =
      current === ''
      || current === lastAutoMergeTargetName.value
      || current === 'develop'
    if (!canOverwrite && current !== preferred) return

    task.mergeTargetName = preferred
    task.mergeTargetPreset = 'custom'
    lastAutoMergeTargetName.value = preferred
    syncMergeTargetPresetFromName()
  }

  watch(
    [commonMergeTargetBranches, commonMergeTargetBranchesLoading, () => show()],
    () => {
      applyPreferredMergeTargetFromCommonBranches()
    },
  )

  const {
    listAttr: datalistListAttr,
    dismissIfPicked: dismissDatalistIfPicked,
    restoreOnFocus: restoreDatalistOnFocus,
  } = datalistController || createDatalistDismissController()

  const onWorkBranchDatalistChange = (event) => {
    const inputEl = event?.target
    const value = String(inputEl?.value ?? '')
    syncWorkBranchPresetFromName()
    const candidates = workBranchDatalistOptions.value.map((opt) => opt.value)
    dismissDatalistIfPicked(inputEl, candidates, value, event)
  }

  const onMergeTargetBranchChange = (event) => {
    const inputEl = event?.target
    const value = String(inputEl?.value ?? '')
    markMergeTargetUserEdited()
    syncMergeTargetPresetFromName()
    const candidates = mergeTargetDatalistOptions.value.map((opt) => opt.value)
    dismissDatalistIfPicked(inputEl, candidates, value, event)
  }

  const fetchTranslatedTaskTitleSegment = async (taskTitle) => {
    if (!tenantId()) {
      throw new Error('tenantId 不能为空')
    }
    titleTranslateAbort?.abort()
    const ac = new AbortController()
    titleTranslateAbort = ac
    const response = await apiFetch(`/api/projects/translate-branch-title/tenant_id/${tenantId()}/`, {
      method: 'POST',
      credentials: 'include',
      headers: {
        'Content-Type': 'application/json',
        Accept: 'application/json',
      },
      body: JSON.stringify({ title: taskTitle }),
      signal: ac.signal,
    })
    if (!response.ok) {
      let errorMessage = '任务标题翻译失败'
      const errorData = await parseJsonSafe(response)
      warnOptionalApiFailure('translate-branch-title', response)
      if (errorData && typeof errorData === 'object' && errorData.error) {
        errorMessage = String(errorData.error)
      } else if (errorData && typeof errorData === 'object' && errorData.detail) {
        errorMessage = String(errorData.detail)
      } else {
        errorMessage = `任务标题翻译失败（HTTP ${response.status}）`
      }
      const tid = extractTraceId(response) || extractTraceId(errorData) || ''
      taskTitleTranslationErrorTraceId.value = tid
      const err = new Error(userFacingTitleTranslationError(errorMessage))
      if (tid) err.traceId = tid
      throw err
    }
    const data = await parseJsonSafe(response)
    if (!data) {
      throw new Error(userFacingTitleTranslationError('任务标题翻译响应无效'))
    }
    return sanitizeBranchSegment(data?.translated_title, 'task')
  }

  const resolveTaskTitleSegment = async () => {
    const task = editingTask()
    const rawTitle = String(task?.title || '').trim()
    if (!rawTitle) {
      return 'task'
    }

    if (!hasChineseCharacter(rawTitle)) {
      taskTitleTranslationError.value = ''
      taskTitleTranslationErrorTraceId.value = ''
      return sanitizeBranchSegment(rawTitle, 'task')
    }

    if (translatedTaskTitleSegmentMap.value[rawTitle]) {
      taskTitleTranslationError.value = ''
      taskTitleTranslationErrorTraceId.value = ''
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
      taskTitleTranslationErrorTraceId.value = ''
      return translatedSegment
    } catch (error) {
      if (isTitleTranslateAbortError(error)) {
        throw error
      }
      taskTitleTranslationError.value = userFacingTitleTranslationError(error?.message)
      if (!taskTitleTranslationErrorTraceId.value) {
        taskTitleTranslationErrorTraceId.value = extractTraceId(error) || ''
      }
      throw error
    } finally {
      isTaskTitleTranslating.value = false
    }
  }

  const updateWorkBranchNameByPreset = async () => {
    const task = editingTask()
    if (!task) return
    const preset = task.workBranchPreset
    if (!preset || preset === 'custom') return
    const currentRequestId = taskTitleTranslationRequestId + 1
    taskTitleTranslationRequestId = currentRequestId

    try {
      const titleSegment = await resolveTaskTitleSegment()
      if (currentRequestId !== taskTitleTranslationRequestId) {
        return
      }
      task.workBranchName = buildWorkBranchName(preset, titleSegment)
      syncWorkBranchPresetFromName()
    } catch (error) {
      if (isTitleTranslateAbortError(error)) {
        return
      }
      const fallbackSegment = sanitizeBranchSegment(task?.title, 'task')
      task.workBranchName = buildWorkBranchName(preset, fallbackSegment)
      syncWorkBranchPresetFromName()
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
    const task = editingTask()
    if (!task) return
    const preset = task.mergeTargetPreset
    if (!preset || preset === 'custom') return
    task.mergeTargetName = buildMergeTargetBranchName(preset)
    syncMergeTargetPresetFromName()
  }

  const initBranchDefaults = () => {
    const task = editingTask()
    if (!task) return
    if (!task.workBranchPreset) {
      task.workBranchPreset = 'feature'
    }
    if (!task.workBranchName) {
      task.workBranchName = buildWorkBranchName(task.workBranchPreset)
    }
    if (!task.mergeTargetPreset) {
      task.mergeTargetPreset = 'custom'
    }
    if (!task.id && task.mergeTargetName == null) {
      task.mergeTargetName = ''
    }
    syncWorkBranchPresetFromName()
    syncMergeTargetPresetFromName()
  }

  watch(
    () => show(),
    (visible) => {
      if (!visible) {
        mergeTargetUserEdited.value = false
        lastAutoMergeTargetName.value = ''
        return
      }
      if (editingTask()?.workBranchPreset && editingTask().workBranchPreset !== 'custom') {
        scheduleWorkBranchNameUpdate()
      }
      if (
        editingTask()?.id
        && editingTask()?.mergeTargetPreset
        && editingTask().mergeTargetPreset !== 'custom'
      ) {
        updateMergeTargetBranchNameByPreset()
      }
    },
  )

  watch(
    () => editingTask()?.title,
    () => {
      const task = editingTask()
      if (!task) return
      if (task.workBranchPreset && task.workBranchPreset !== 'custom') {
        scheduleWorkBranchNameUpdate()
      }
    },
  )

  if (getCurrentInstance()) {
    onBeforeUnmount(() => {
      if (taskTitleTranslationTimer) {
        clearTimeout(taskTitleTranslationTimer)
        taskTitleTranslationTimer = null
      }
      titleTranslateAbort?.abort()
    })
  }

  return {
    isTaskTitleTranslating,
    taskTitleTranslationError,
    taskTitleTranslationErrorTraceId,
    workBranchDatalistOptions,
    mergeTargetDatalistOptions,
    commonMergeTargetBranches,
    commonMergeTargetBranchesLoading,
    datalistListAttr,
    restoreDatalistOnFocus,
    onWorkBranchDatalistChange,
    onMergeTargetBranchChange,
    syncWorkBranchPresetFromName,
    syncMergeTargetPresetFromName,
    markMergeTargetUserEdited,
    updateWorkBranchNameByPreset,
    updateMergeTargetBranchNameByPreset,
    resolveSyncTitleSegmentForBranch,
    initBranchDefaults,
    scheduleWorkBranchNameUpdate,
  }
}

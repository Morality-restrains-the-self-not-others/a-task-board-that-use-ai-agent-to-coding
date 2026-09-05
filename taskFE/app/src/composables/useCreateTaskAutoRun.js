import { computed, ref, watch } from 'vue'
import { apiFetch } from '../utils/apiUtils.js'
import {
  projectHasConfiguredRunTemplate,
  summarizeRunTemplate,
} from '../utils/projectRunTemplateUtils.js'
import {
  resolveAutoRunDisabledReasons,
  resolveAutoRunEnabledHint,
  resolveForceAutoRunHint,
} from '../utils/autoRunGateHints.js'
import {
  resolveQueuedAutoRunHint,
  shouldShowCreateTaskQueuedAutoRun,
} from '../utils/createTaskQueuedAutoRun.js'
import { fetchWorkspaceAutoScheduleEnabled } from '../utils/workspaceAutoScheduleEnabled.js'

/** @param {{ editingTask: () => object|null, projects: () => array, installedImages: () => array, onPrimaryProjectChange?: (projectId: string) => void, tenantId?: () => string, workspaceId?: () => string, show?: () => boolean }} opts */
export function useCreateTaskAutoRun({
  editingTask,
  projects,
  installedImages,
  onPrimaryProjectChange,
  tenantId,
  workspaceId,
  show,
}) {
  const primaryProjectId = computed(() => {
    const firstProjectId = editingTask()?.projectSelections?.[0]?.projectId
    return firstProjectId ? String(firstProjectId) : ''
  })

  const selectedLinkedProject = computed(() => {
    if (!primaryProjectId.value) return null
    return projects().find((project) => String(project.id) === primaryProjectId.value) || null
  })

  const selectedProjectRunTemplateSummary = computed(() =>
    summarizeRunTemplate(selectedLinkedProject.value?.server_run_template),
  )

  const projectAllowsAutoRun = computed(() =>
    Boolean(selectedLinkedProject.value?.server_run_template?.default_auto_run),
  )

  const canEnableAutoRun = computed(() => {
    if (!selectedLinkedProject.value) return false
    if (!projectHasConfiguredRunTemplate(selectedLinkedProject.value)) return false
    if (!projectAllowsAutoRun.value) return false
    const imageId = editingTask()?.container_image?.id
    return imageId != null && String(imageId).trim() !== ''
  })

  const autoRunDisabledReasons = computed(() => resolveAutoRunDisabledReasons({
    hasLinkedProject: Boolean(primaryProjectId.value),
    hasConfiguredRunTemplate: projectHasConfiguredRunTemplate(selectedLinkedProject.value),
    projectAllowsAutoRun: projectAllowsAutoRun.value,
    hasInstalledImage: Boolean(
      editingTask()?.container_image?.id != null
      && String(editingTask().container_image.id).trim() !== '',
    ),
  }))

  const autoRunEnabledHint = computed(() => resolveAutoRunEnabledHint({
    templateSummary: selectedProjectRunTemplateSummary.value,
  }))

  const forceAutoRunHint = computed(() => resolveForceAutoRunHint())

  const showForceAutoRunOption = computed(() => (
    Boolean(editingTask()?.id)
    && canEnableAutoRun.value
    && editingTask()?.auto_run === true
  ))

  const workspaceScheduleEnabled = ref(false)
  const scheduleLoadError = ref('')
  const scheduleLoadErrorTraceId = ref('')

  const showQueuedAutoRunOption = computed(() => shouldShowCreateTaskQueuedAutoRun({
    canEnableAutoRun: canEnableAutoRun.value,
    autoRun: editingTask()?.auto_run === true,
    workspaceScheduleEnabled: workspaceScheduleEnabled.value,
  }))

  const queuedAutoRunHint = computed(() => resolveQueuedAutoRunHint())

  watch(showForceAutoRunOption, (visible) => {
    const task = editingTask()
    if (!visible && task) {
      task.force_auto_run = false
    }
  })

  watch(canEnableAutoRun, (enabled) => {
    const task = editingTask()
    if (!enabled && task) {
      task.auto_run = false
      task.force_auto_run = false
      task.queued_auto_run = false
    }
  })

  watch(() => editingTask()?.auto_run, (on) => {
    const task = editingTask()
    if (!on && task) {
      task.queued_auto_run = false
    }
  })

  watch(
    () => ({
      open: typeof show === 'function' ? Boolean(show()) : false,
      tid: typeof tenantId === 'function' ? String(tenantId() || '').trim() : '',
      wid: typeof workspaceId === 'function' ? String(workspaceId() || '').trim() : '',
    }),
    async ({ open, tid, wid }) => {
      if (!open || !tid || !wid) {
        workspaceScheduleEnabled.value = false
        scheduleLoadError.value = ''
        scheduleLoadErrorTraceId.value = ''
        return
      }
      scheduleLoadError.value = ''
      scheduleLoadErrorTraceId.value = ''
      try {
        const got = await fetchWorkspaceAutoScheduleEnabled({
          tenantId: tid,
          workspaceId: wid,
          apiFetch,
        })
        workspaceScheduleEnabled.value = got.enabled === true
      } catch (e) {
        workspaceScheduleEnabled.value = false
        scheduleLoadError.value = e?.message || '加载自动调度失败'
        scheduleLoadErrorTraceId.value = e?.traceId || ''
      }
    },
    { immediate: true },
  )

  watch(primaryProjectId, (newProjectId) => {
    const task = editingTask()
    if (newProjectId && task) {
      const currentProject = projects().find((project) => String(project.id) === String(newProjectId))
      if (currentProject) {
        if (currentProject.container_image) {
          const correspondingImage = installedImages().find((image) => image.name === currentProject.container_image)
          if (task.container_image) {
            if (correspondingImage) {
              task.container_image.id = correspondingImage.id
            } else {
              task.container_image.id = null
            }
          }
        } else if (currentProject.container_image_id) {
          if (task.container_image) {
            task.container_image.id = String(currentProject.container_image_id)
          }
        }

        const imageId = String(
          task.container_image?.id || currentProject.container_image_id || '',
        ).trim()
        task.auto_run = Boolean(
          currentProject.server_run_template?.default_auto_run
          && projectHasConfiguredRunTemplate(currentProject)
          && imageId,
        )
      }
    }
    if (typeof onPrimaryProjectChange === 'function') {
      onPrimaryProjectChange(newProjectId || '')
    }
  }, { immediate: true })

  return {
    primaryProjectId,
    canEnableAutoRun,
    autoRunDisabledReasons,
    autoRunEnabledHint,
    forceAutoRunHint,
    showForceAutoRunOption,
    showQueuedAutoRunOption,
    queuedAutoRunHint,
    workspaceScheduleEnabled,
    scheduleLoadError,
    scheduleLoadErrorTraceId,
  }
}

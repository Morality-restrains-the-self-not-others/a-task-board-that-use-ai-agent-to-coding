import { watch, onMounted, onBeforeUnmount, nextTick } from 'vue'
import { buildDefaultRelayToTraeEnvItems } from '../../utils/relayToTraeUtils.js'

/**
 * ServerConfig.logic 生命周期、watch 与初始化编排。
 */
export function useServerConfigLifecycle(deps) {
  const {
    props,
    route,
    serverHardwarePanelRef,
    isInitialized,
    selectedImageId,
    installedImages,
    fetchInstalledImages,
    initFeatureParamsSource,
    isRelayToTraeEnabled,
    activeServerSection,
    relayToTraeDefaultApplied,
    fetchServerStartHistory,
    fetchRelayToTraeStatusOnMount,
    fetchRelayToTraeServiceStatus,
    loadRelayToTraeEnvDefaults,
    resetRuntimeForNewTask,
    resetRelayForNewTask,
    resetStaleRepoAck,
    stopRelayToTraePoll,
    watchStatusMessageForRefresh,
    disposeRelayTimers,
    disposeRuntimeTimers,
    applyDefaultServerTabByRuntime,
    isServerRuntimeRunning,
  } = deps

  onMounted(async () => {
    if (isRelayToTraeEnabled.value) {
      activeServerSection.value = 'relayDirect'
      relayToTraeDefaultApplied.value = true
    }
    await fetchInstalledImages()
    if (props.task) {
      selectedImageId.value = props.task.container_image_id || props.task.container_image?.id || ''
    }
    isInitialized.value = true
    if (props.task) {
      initFeatureParamsSource()
      await nextTick()
      await serverHardwarePanelRef.value?.initHardwareFromParent?.()
      if (isRelayToTraeEnabled.value) {
        void fetchRelayToTraeStatusOnMount()
      }
    }
  })

  onBeforeUnmount(() => {
    stopRelayToTraePoll?.()
    disposeRelayTimers?.()
    disposeRuntimeTimers?.()
  })

  watch(
    () => [props.statusMessage, props.statusCommentId, props.statusRuntimeStatus],
    ([message, commentId, runtimeStatus]) => {
      watchStatusMessageForRefresh(message, commentId, runtimeStatus)
    },
  )

  watch(isRelayToTraeEnabled, (enabled) => {
    if (enabled) {
      activeServerSection.value = 'relayDirect'
      relayToTraeDefaultApplied.value = true
      deps.relayToTraeEnvItems.value = buildDefaultRelayToTraeEnvItems(route.query)
      void fetchRelayToTraeStatusOnMount()
    } else {
      stopRelayToTraePoll?.()
    }
  })

  watch(
    () => props.task?.projects,
    () => resetStaleRepoAck(),
    { deep: true },
  )

  watch(activeServerSection, async (section) => {
    if (section === 'relayDirect' && isRelayToTraeEnabled.value) {
      void fetchRelayToTraeServiceStatus()
    }
  })

  watch(
    () => props.task,
    (newTask, oldTask) => {
      if (!newTask) {
        return
      }
      const newId = String(newTask.id ?? newTask.pk ?? '').trim()
      const oldId = oldTask ? String(oldTask.id ?? oldTask.pk ?? '').trim() : ''
      const sameTask = Boolean(newId) && newId === oldId

      if (!sameTask) {
        resetRuntimeForNewTask()
        if (activeServerSection.value === 'history') {
          fetchServerStartHistory()
        }
      }

      selectedImageId.value = newTask.container_image_id || newTask.container_image?.id || ''
      initFeatureParamsSource()
      void serverHardwarePanelRef.value?.initHardwareFromParent?.()

      if (isRelayToTraeEnabled.value) {
        if (!sameTask) {
          resetRelayForNewTask()
        }
        void loadRelayToTraeEnvDefaults().then(() => fetchRelayToTraeServiceStatus())
      }
    },
  )
}

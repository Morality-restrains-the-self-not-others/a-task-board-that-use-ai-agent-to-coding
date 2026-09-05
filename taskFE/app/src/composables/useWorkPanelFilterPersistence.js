import { ref, watch, onUnmounted } from 'vue'
import { defaultDeliverableFilterBars } from '../utils/workPanelDeliverableFilterBars.js'
import {
  accessFilterToPref,
  fetchWorkPanelFilters,
  saveWorkPanelFilters,
} from '../utils/workPanelFilterPersistence.js'
import { warnNetworkFailure } from '../utils/workPanelApiUtils.js'

/**
 * 工作面板过滤偏好：交付物栏 + access_filter，按用户×工作空间拉取 / 防抖保存。
 *
 * @param {{
 *   apiFetch: typeof fetch,
 *   tenantId: import('vue').Ref<string|null>,
 *   currentWorkspace: import('vue').Ref<{id?: string|null}|null>,
 *   deliverableFilterBars: import('vue').Ref<unknown[]>,
 *   accessFilter: import('vue').Ref<unknown>,
 *   initialDataLoaded: import('vue').Ref<boolean>,
 * }} opts
 */
export function useWorkPanelFilterPersistence({
  apiFetch,
  tenantId,
  currentWorkspace,
  deliverableFilterBars,
  accessFilter,
  initialDataLoaded,
}) {
  const filterPrefsLoading = ref(false)
  let filterPrefsSaveTimer = null
  /** @type {import('vue').Ref<{kind:string,id:string,label:string}|null>} */
  const pendingAccessFilterPref = ref(null)

  const loadDeliverableFilterPrefs = async (workspace) => {
    const tid = tenantId.value
    const wid = workspace?.id != null ? String(workspace.id) : ''
    if (!tid || !wid || wid === 'default') {
      deliverableFilterBars.value = defaultDeliverableFilterBars()
      pendingAccessFilterPref.value = null
      return null
    }
    filterPrefsLoading.value = true
    try {
      const payload = await fetchWorkPanelFilters({
        apiFetch,
        tenantId: tid,
        workspaceId: wid,
      })
      deliverableFilterBars.value = payload.deliverable_filter_bars
      pendingAccessFilterPref.value = payload.access_filter
      return payload
    } catch (e) {
      warnNetworkFailure('work-panel-filters GET', e)
      deliverableFilterBars.value = defaultDeliverableFilterBars()
      pendingAccessFilterPref.value = null
      return null
    } finally {
      filterPrefsLoading.value = false
    }
  }

  const scheduleSaveFilterPrefs = () => {
    if (filterPrefsLoading.value || !initialDataLoaded.value) return
    const tid = tenantId.value
    const wid = currentWorkspace.value?.id != null ? String(currentWorkspace.value.id) : ''
    if (!tid || !wid || wid === 'default') return
    if (filterPrefsSaveTimer) clearTimeout(filterPrefsSaveTimer)
    filterPrefsSaveTimer = setTimeout(async () => {
      filterPrefsSaveTimer = null
      try {
        const accessForSave =
          accessFilter?.value ?? pendingAccessFilterPref.value ?? null
        await saveWorkPanelFilters({
          apiFetch,
          tenantId: tid,
          workspaceId: wid,
          bars: deliverableFilterBars.value,
          accessFilter: accessForSave,
        })
        console.info(
          `[WorkPanel:filter-prefs] PUT ok access=${accessFilterToPref(accessForSave)?.kind || 'none'}`,
        )
      } catch (e) {
        warnNetworkFailure('work-panel-filters PUT', e)
      }
    }, 400)
  }

  watch(deliverableFilterBars, () => scheduleSaveFilterPrefs(), { deep: true })
  if (accessFilter) {
    watch(accessFilter, () => scheduleSaveFilterPrefs(), { deep: true })
  }

  onUnmounted(() => {
    if (filterPrefsSaveTimer) {
      clearTimeout(filterPrefsSaveTimer)
      filterPrefsSaveTimer = null
    }
  })

  return {
    filterPrefsLoading,
    pendingAccessFilterPref,
    loadDeliverableFilterPrefs,
    scheduleSaveFilterPrefs,
  }
}

import { ref, computed } from 'vue'
import {
  accessFilterChipLabel,
  normalizeAccessFilter,
  parseAccessFilterSubjects,
  resolveCompanyMemberIdsFromGroupMembers,
  toggleAccessFilter,
} from '../utils/workPanelAccessFilter.js'
import { warnNetworkFailure, warnOptionalApiFailure, parseJsonSafe } from '../utils/workPanelApiUtils.js'

/**
 * @param {{
 *   apiFetch: typeof fetch,
 *   tenantId: import('vue').Ref<string|null>,
 *   currentWorkspace: import('vue').Ref<{id?: string|null}|null>,
 *   collaborators: import('vue').Ref<unknown[]>,
 * }} opts
 */
export function useWorkPanelAccessFilter({ apiFetch, tenantId, currentWorkspace, collaborators }) {
  /** @type {import('vue').Ref<import('../utils/workPanelAccessFilter.js').AccessFilter | null>} */
  const accessFilter = ref(null)
  const accessPeople = ref([])
  const accessGroups = ref([])
  const accessSubjectsLoading = ref(false)
  const accessFilterPanelOpen = ref(false)
  /** @type {import('vue').Ref<'person' | 'group'>} */
  const accessFilterTab = ref('person')

  const accessFilterChip = computed(() => accessFilterChipLabel(accessFilter.value))

  /** 双击会先触发两次 click：短窗内忽略第二次 click，由 dblclick 收起 */
  let accessFilterClickOpenedAt = 0

  const refreshAccessSubjects = async () => {
    const tid = tenantId.value
    const wid = currentWorkspace.value?.id
    if (!tid || !wid || wid === 'default') {
      accessPeople.value = []
      accessGroups.value = []
      return
    }
    accessSubjectsLoading.value = true
    try {
      const url = `/api/projects/workspace-access/workspace-permissions/tenant_id/${tid}/?workspace_id=${wid}`
      const response = await apiFetch(url, {
        credentials: 'include',
        headers: { Accept: 'application/json' },
      })
      if (!response.ok) {
        warnOptionalApiFailure('workspace-permissions', response)
        accessPeople.value = []
        accessGroups.value = []
        return
      }
      const data = await parseJsonSafe(response)
      const { people, groups } = parseAccessFilterSubjects(data, collaborators.value || [])
      accessPeople.value = people
      accessGroups.value = groups
      console.info(
        `[WorkPanel:access-filter] subjects loaded people=${people.length} groups=${groups.length} workspace=${wid}`,
      )
    } catch (e) {
      warnNetworkFailure('workspace-permissions', e)
      accessPeople.value = []
      accessGroups.value = []
    } finally {
      accessSubjectsLoading.value = false
    }
  }

  const clearAccessFilter = () => {
    accessFilter.value = null
  }

  const resetAccessFilter = () => {
    accessFilter.value = null
    accessFilterPanelOpen.value = false
    accessFilterTab.value = 'person'
  }

  /**
   * @param {{ kind: 'person'|'group', id: string, label?: string }} subject
   * @param {{ replace?: boolean }} [opts] replace=true 时强制设置（用于偏好恢复，不 toggle）
   */
  const selectAccessSubject = async (subject, opts = {}) => {
    if (!subject || (subject.kind !== 'person' && subject.kind !== 'group')) {
      return
    }
    const replace = Boolean(opts.replace)
    const tid = tenantId.value
    if (subject.kind === 'person') {
      const next = normalizeAccessFilter({
        kind: 'person',
        id: subject.id,
        label: subject.label || '',
        memberIds: [subject.id],
      })
      accessFilter.value = replace ? next : toggleAccessFilter(accessFilter.value, next)
      accessFilterPanelOpen.value = false
      console.info(
        `[WorkPanel:access-filter] select kind=person id=${subject.id} active=${Boolean(accessFilter.value)} replace=${replace}`,
      )
      return
    }
    // group
    let memberIds = []
    if (tid && subject.id) {
      try {
        const url = `/api/tenant/${tid}/accounts/groups/${subject.id}/members/`
        const response = await apiFetch(url, {
          credentials: 'include',
          headers: { Accept: 'application/json' },
        })
        if (!response.ok) {
          warnOptionalApiFailure('group-members', response)
        } else {
          const data = await parseJsonSafe(response)
          memberIds = resolveCompanyMemberIdsFromGroupMembers(
            Array.isArray(data) ? data : [],
            collaborators.value || [],
          )
        }
      } catch (e) {
        warnNetworkFailure('group-members', e)
      }
    }
    const next = normalizeAccessFilter({
      kind: 'group',
      id: subject.id,
      label: subject.label || '',
      memberIds,
    })
    accessFilter.value = replace ? next : toggleAccessFilter(accessFilter.value, next)
    accessFilterPanelOpen.value = false
    console.info(
      `[WorkPanel:access-filter] select kind=group id=${subject.id} members=${memberIds.length} active=${Boolean(accessFilter.value)} replace=${replace}`,
    )
  }

  /**
   * 从持久化偏好恢复（重算 memberIds）。
   * @param {{ kind?: string, id?: string, label?: string } | null | undefined} pref
   */
  const hydrateAccessFilterPref = async (pref) => {
    const n = normalizeAccessFilterPrefCompat(pref)
    if (!n) {
      accessFilter.value = null
      return
    }
    await selectAccessSubject(n, { replace: true })
  }

  function normalizeAccessFilterPrefCompat(pref) {
    if (pref == null || typeof pref !== 'object') return null
    const kind = String(pref.kind || '').trim()
    const id = pref.id != null ? String(pref.id).trim() : ''
    if ((kind !== 'person' && kind !== 'group') || !id) return null
    return { kind, id, label: pref.label != null ? String(pref.label) : '' }
  }

  const toggleAccessFilterPanel = () => {
    const now = Date.now()
    if (accessFilterPanelOpen.value && now - accessFilterClickOpenedAt < 350) {
      return
    }
    if (!accessFilterPanelOpen.value) {
      accessFilterPanelOpen.value = true
      accessFilterClickOpenedAt = now
      return
    }
    accessFilterPanelOpen.value = false
    accessFilterClickOpenedAt = 0
  }

  const openAccessFilterPanel = () => {
    accessFilterPanelOpen.value = true
    accessFilterClickOpenedAt = Date.now()
  }

  const closeAccessFilterPanel = () => {
    accessFilterPanelOpen.value = false
    accessFilterClickOpenedAt = 0
  }

  const setAccessFilterTab = (tab) => {
    if (tab === 'person' || tab === 'group') {
      accessFilterTab.value = tab
    }
  }

  return {
    accessFilter,
    accessPeople,
    accessGroups,
    accessSubjectsLoading,
    accessFilterPanelOpen,
    accessFilterTab,
    accessFilterChip,
    refreshAccessSubjects,
    selectAccessSubject,
    hydrateAccessFilterPref,
    clearAccessFilter,
    resetAccessFilter,
    toggleAccessFilterPanel,
    openAccessFilterPanel,
    closeAccessFilterPanel,
    setAccessFilterTab,
  }
}

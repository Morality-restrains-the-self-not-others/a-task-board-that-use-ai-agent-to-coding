import { ref } from 'vue'
import {
  createTaskFieldSettingsFromResponse,
  defaultCreateTaskFieldSettings,
} from '../utils/createTaskFieldSettings.js'

/**
 * @param {{
 *   apiFetch: typeof import('../utils/apiUtils.js').apiFetch,
 *   tenantId: import('vue').Ref<unknown>,
 *   currentWorkspace: import('vue').Ref<{ id?: unknown }|null>,
 *   parseJsonSafe: (r: Response) => Promise<unknown>,
 *   warnOptionalApiFailure: Function,
 *   warnNetworkFailure: Function,
 * }} deps
 */
export function useWorkPanelCreateTaskFieldSettings(deps) {
  const createTaskFieldSettings = ref(defaultCreateTaskFieldSettings())

  const resetCreateTaskFieldSettings = () => {
    createTaskFieldSettings.value = defaultCreateTaskFieldSettings()
  }

  const fetchCreateTaskFieldSettings = async () => {
    const tid = deps.tenantId.value
    const wid = deps.currentWorkspace.value?.id
    if (!tid || !wid || wid === 'default') {
      resetCreateTaskFieldSettings()
      return
    }
    try {
      const response = await deps.apiFetch(
        `/api/projects/workspaces/tenant_id/${tid}/${wid}/create-task-field-settings/`,
        {
          method: 'GET',
          credentials: 'include',
          headers: { Accept: 'application/json' },
        },
      )
      const body = await deps.parseJsonSafe(response)
      if (!response.ok) {
        deps.warnOptionalApiFailure('创建任务字段设置', response.status, body)
        resetCreateTaskFieldSettings()
        return
      }
      createTaskFieldSettings.value = createTaskFieldSettingsFromResponse(body)
    } catch (error) {
      deps.warnNetworkFailure('创建任务字段设置', error)
      resetCreateTaskFieldSettings()
    }
  }

  return {
    createTaskFieldSettings,
    fetchCreateTaskFieldSettings,
    resetCreateTaskFieldSettings,
  }
}

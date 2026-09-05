import { ref } from 'vue'
import { DEFAULT_TASK_KIND_OPTIONS, taskKindOptionsFromResponse } from '../utils/taskKindOptions.js'

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
export function useWorkPanelTaskKindOptions(deps) {
  const taskKindOptions = ref([...DEFAULT_TASK_KIND_OPTIONS])

  const resetTaskKindOptions = () => {
    taskKindOptions.value = [...DEFAULT_TASK_KIND_OPTIONS]
  }

  const fetchTaskKindOptions = async () => {
    const tid = deps.tenantId.value
    const wid = deps.currentWorkspace.value?.id
    if (!tid || !wid || wid === 'default') {
      resetTaskKindOptions()
      return
    }
    try {
      const response = await deps.apiFetch(`/api/projects/workspaces/tenant_id/${tid}/${wid}/task-kind-options/`, {
        method: 'GET',
        credentials: 'include',
        headers: { Accept: 'application/json' },
      })
      const body = await deps.parseJsonSafe(response)
      if (!response.ok) {
        deps.warnOptionalApiFailure('任务类型可选值', response.status, body)
        resetTaskKindOptions()
        return
      }
      taskKindOptions.value = taskKindOptionsFromResponse(body)
    } catch (error) {
      deps.warnNetworkFailure('任务类型可选值', error)
      resetTaskKindOptions()
    }
  }

  return {
    taskKindOptions,
    fetchTaskKindOptions,
    resetTaskKindOptions,
  }
}

import { ref } from 'vue'
import { DEFAULT_CODE_LANG_OPTIONS, codeLangOptionsFromResponse } from '../utils/codeLangOptions.js'

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
export function useWorkPanelCodeLangOptions(deps) {
  const codeLangOptions = ref([...DEFAULT_CODE_LANG_OPTIONS])

  const resetCodeLangOptions = () => {
    codeLangOptions.value = [...DEFAULT_CODE_LANG_OPTIONS]
  }

  const fetchCodeLangOptions = async () => {
    const tid = deps.tenantId.value
    const wid = deps.currentWorkspace.value?.id
    if (!tid || !wid || wid === 'default') {
      resetCodeLangOptions()
      return
    }
    try {
      const response = await deps.apiFetch(`/api/projects/workspaces/tenant_id/${tid}/${wid}/code-lang-options/`, {
        method: 'GET',
        credentials: 'include',
        headers: { Accept: 'application/json' },
      })
      const body = await deps.parseJsonSafe(response)
      if (!response.ok) {
        deps.warnOptionalApiFailure('主要编程语言可选值', response.status, body)
        resetCodeLangOptions()
        return
      }
      codeLangOptions.value = codeLangOptionsFromResponse(body)
    } catch (error) {
      deps.warnNetworkFailure('主要编程语言可选值', error)
      resetCodeLangOptions()
    }
  }

  return {
    codeLangOptions,
    fetchCodeLangOptions,
    resetCodeLangOptions,
  }
}

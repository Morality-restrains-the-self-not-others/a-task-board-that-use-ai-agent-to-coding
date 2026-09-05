import { apiFetch } from '../../utils/apiUtils.js'
import { extractTraceId } from '../../utils/traceId.js'
import { userFacingTitleTranslationError } from '../../utils/titleTranslationError.js'

export async function fetchTranslatedTaskTitleSegment(taskTitle, deps) {
  const { effectiveTenantId, sanitizeBranchSegment, signal } = deps
  const tenantId = effectiveTenantId.value
  if (!tenantId) throw new Error('tenantId 不能为空')
  const response = await apiFetch(`/api/projects/translate-branch-title/tenant_id/${tenantId}/`, {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
    body: JSON.stringify({ title: taskTitle }),
    ...(signal ? { signal } : {}),
  })
  if (!response.ok) {
    let errorMessage = '任务标题翻译失败'
    try {
      const errorData = await response.json()
      if (errorData && typeof errorData === 'object') {
        if (errorData.error) errorMessage = String(errorData.error)
        else if (errorData.detail) errorMessage = String(errorData.detail)
      }
    } catch {
      errorMessage = `任务标题翻译失败（HTTP ${response.status}）`
    }
    const err = new Error(userFacingTitleTranslationError(errorMessage))
    const tid = extractTraceId(response)
    if (tid) err.traceId = tid
    throw err
  }
  const data = await response.json()
  if (!data) throw new Error('任务标题翻译响应无效')
  return sanitizeBranchSegment(data?.translated_title, 'task')
}

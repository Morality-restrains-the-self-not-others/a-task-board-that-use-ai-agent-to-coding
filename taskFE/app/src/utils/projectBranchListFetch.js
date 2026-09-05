import { apiFetch } from './apiUtils.js'
import { parseJsonSafe, warnNetworkFailure, warnOptionalApiFailure } from './workPanelApiUtils.js'
import { extractTraceId } from './traceId.js'

/** @returns {{ branches: string[], loading: false, error: string|null, loaded: boolean, traceId: string }} */
export async function loadProjectBranchListState(tenantId, projectId, repoUrl) {
  if (!repoUrl) {
    return {
      branches: [],
      loading: false,
      error: '未找到对应仓库地址',
      loaded: true,
      traceId: '',
    }
  }

  const normalizedProjectId = String(projectId)
  try {
    const params = new URLSearchParams({ repo_url: repoUrl })
    const response = await apiFetch(
      `/api/projects/tenant_id/${tenantId}/${normalizedProjectId}/branches/?${params.toString()}`,
      {
        credentials: 'include',
        headers: { Accept: 'application/json' },
      },
    )

    if (!response.ok) {
      warnOptionalApiFailure(`项目分支 project=${normalizedProjectId} repo_url`, response)
      const fallbackMsg = `获取分支列表失败（HTTP ${response.status}）`
      const errData = await parseJsonSafe(response)
      let errMsg = fallbackMsg
      let loaded = false
      if (errData && typeof errData === 'object' && typeof errData.error === 'string' && errData.error.trim()) {
        errMsg = errData.error.trim()
        loaded = true
      }
      const tid = extractTraceId(response) || extractTraceId(errData)
      return {
        branches: Array.isArray(errData?.branches) ? errData.branches : [],
        loading: false,
        error: errMsg,
        loaded,
        traceId: tid,
      }
    }

    const data = await parseJsonSafe(response)
    if (!data) {
      warnOptionalApiFailure(`项目分支(JSON) project=${normalizedProjectId}`, response)
      return {
        branches: [],
        loading: false,
        error: '分支列表响应无效',
        loaded: false,
        traceId: extractTraceId(response) || '',
      }
    }
    return {
      branches: Array.isArray(data.branches) ? data.branches : [],
      loading: false,
      error: data.error || null,
      loaded: true,
      traceId: data.error ? (extractTraceId(response) || extractTraceId(data)) : '',
    }
  } catch (e) {
    warnNetworkFailure(`项目分支 project=${normalizedProjectId}`, e)
    return {
      branches: [],
      loading: false,
      error: '网络错误，请稍后重试',
      loaded: false,
      traceId: extractTraceId(e) || '',
    }
  }
}

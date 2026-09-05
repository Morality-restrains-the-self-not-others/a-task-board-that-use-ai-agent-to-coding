import { ref } from 'vue'
import { BRANCH_PREVIEW_REQUEST_TIMEOUT_MS, resolveBranchPreviewFetchError } from '../utils/branchPreviewUtils.js'
import { extractTraceId } from '../utils/traceId.js'

export function useProjectRepoBranchPreview({ apiFetch, route, project, projectGitRepoList }) {
  const branchPreviewLoading = ref(false)
  const branchPreviewError = ref('')
  const repoBranchPreviews = ref([])

  const fetchBranchPreviewWithTimeout = async (tenantId, projectId, repoUrl, timeoutMs = BRANCH_PREVIEW_REQUEST_TIMEOUT_MS) => {
    const params = new URLSearchParams({ repo_url: repoUrl })
    const controller = new AbortController()
    const timer = setTimeout(() => controller.abort(), timeoutMs)
    try {
      return await apiFetch(`/api/projects/tenant_id/${tenantId}/${projectId}/branches/?${params.toString()}`, {
        credentials: 'include',
        headers: { Accept: 'application/json' },
        signal: controller.signal,
      })
    } finally {
      clearTimeout(timer)
    }
  }

  const fetchProjectRepoBranchesPreview = async () => {
    const tenantId = route.params.tenant
    const projectId = route.params.id || project.value?.id
    const repoUrls = [...new Set(projectGitRepoList.value.map((url) => String(url || '').trim()).filter(Boolean))]
    branchPreviewError.value = ''
    repoBranchPreviews.value = []
    if (!tenantId || !projectId) {
      branchPreviewError.value = '项目信息未加载完成，请稍后再试'
      return
    }
    if (!repoUrls.length) {
      branchPreviewError.value = '当前项目未配置 Git 仓库'
      return
    }
    branchPreviewLoading.value = true
    try {
      repoBranchPreviews.value = await Promise.all(
        repoUrls.map(async (repoUrl) => {
          try {
            const response = await fetchBranchPreviewWithTimeout(tenantId, projectId, repoUrl)
            const data = await response.json().catch(() => ({}))
            if (!response.ok) {
              const tid = extractTraceId(response) || extractTraceId(data)
              return { repoUrl, branches: [], error: data?.error || data?.detail || `查询失败（HTTP ${response.status}）`, traceId: tid }
            }
            return {
              repoUrl,
              branches: Array.isArray(data?.branches) ? data.branches : [],
              error: data?.error ? String(data.error) : '',
              traceId: data?.error ? (extractTraceId(response) || extractTraceId(data)) : '',
            }
          } catch (repoErr) {
            return { repoUrl, branches: [], error: resolveBranchPreviewFetchError(repoErr), traceId: extractTraceId(repoErr) }
          }
        }),
      )
    } catch (err) {
      branchPreviewError.value = err?.message || '查询分支列表失败'
    } finally {
      branchPreviewLoading.value = false
    }
  }

  return { branchPreviewLoading, branchPreviewError, repoBranchPreviews, fetchProjectRepoBranchesPreview }
}

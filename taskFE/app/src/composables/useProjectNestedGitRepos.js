import { ref, watch, unref } from 'vue'
import { apiFetch } from '../utils/apiUtils.js'
import { extractTraceId } from '../utils/traceId.js'

/**
 * 项目详情：发现父仓的子 Git 仓库列表。
 * @param {{ tenantId: import('vue').Ref|string|Function, projectId: import('vue').Ref|string|Function, repoUrl: import('vue').Ref|string|Function, auto?: boolean }} options
 */
export function useProjectNestedGitRepos({ tenantId, projectId, repoUrl, auto = true } = {}) {
  const nestedRepos = ref([])
  const nestedLoading = ref(false)
  const nestedError = ref('')
  /** 最近一次 nested-git-repos 请求失败/业务错误对应的 traceId；无请求产生的提示保持空 */
  const nestedErrorTraceId = ref('')
  const parentRepoUrl = ref('')

  const resolveOpt = (v) => {
    if (typeof v === 'function') return String(v() ?? '').trim()
    return String(unref(v) ?? '').trim()
  }

  const clearNestedError = () => {
    nestedError.value = ''
    nestedErrorTraceId.value = ''
  }

  const assignNestedError = (message, ...sources) => {
    nestedError.value = String(message || '').trim()
    let tid = ''
    for (const src of sources) {
      tid = extractTraceId(src)
      if (tid) break
    }
    nestedErrorTraceId.value = tid
  }

  const fetchNestedGitRepos = async () => {
    const tid = resolveOpt(tenantId)
    const pid = resolveOpt(projectId)
    const repo = resolveOpt(repoUrl)
    clearNestedError()
    nestedRepos.value = []
    parentRepoUrl.value = repo
    if (!tid || !pid) {
      // 纯前端门禁，未发请求 — 不得伪造 data-traceId
      nestedError.value = '项目信息未加载完成，请稍后再试'
      return
    }
    if (!repo) {
      nestedRepos.value = []
      return
    }
    nestedLoading.value = true
    try {
      const params = new URLSearchParams({ repo_url: repo })
      const response = await apiFetch(
        `/api/projects/tenant_id/${tid}/${pid}/nested-git-repos/?${params.toString()}`,
        {
          credentials: 'include',
          headers: { Accept: 'application/json' },
        },
      )
      const data = await response.json().catch(() => ({}))
      if (!response.ok) {
        assignNestedError(
          data?.error || data?.detail || `查询失败（HTTP ${response.status}）`,
          response,
          data,
        )
        return
      }
      parentRepoUrl.value = String(data?.parent_repo_url || repo).trim()
      nestedRepos.value = Array.isArray(data?.nested_repos) ? data.nested_repos : []
      if (data?.error) {
        // HTTP 200 但业务体带 error（如父仓不可访问）——仍属请求产出的可展示错误
        assignNestedError(data.error, response, data)
      } else {
        clearNestedError()
      }
    } catch (err) {
      assignNestedError(err?.message || '查询子 Git 仓库失败', err)
    } finally {
      nestedLoading.value = false
    }
  }

  if (auto) {
    watch(
      () => [resolveOpt(tenantId), resolveOpt(projectId), resolveOpt(repoUrl)],
      () => {
        void fetchNestedGitRepos()
      },
      { immediate: true },
    )
  }

  return {
    nestedRepos,
    nestedLoading,
    nestedError,
    nestedErrorTraceId,
    parentRepoUrl,
    fetchNestedGitRepos,
  }
}

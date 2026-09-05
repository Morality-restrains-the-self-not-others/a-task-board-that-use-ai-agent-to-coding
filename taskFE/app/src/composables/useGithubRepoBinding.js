import { apiFetch } from '../utils/apiUtils'
import { githubRepoBindingsBySlug, githubRepoSlugFromUrl } from '../utils/taskDetailBranchAndRepoUtils.js'
import {
  assignGithubRepoBindingCatchError,
  assignGithubRepoBindingError,
  makeGithubRepoBindingHttpError,
} from '../utils/githubRepoBindingError.js'

/**
 * GitHub credential status/approve 调用与错误 data-traceId 赋值。
 */
export function createGithubRepoBindingApi({
  getIds,
  refs,
  selectedGithubUserIdForRepo,
  fetchRepoOAuthConnectionStatusByRepoUrl,
}) {
  const {
    githubRepoBindingLoading,
    githubRepoBindingSaving,
    githubBindingSavingRepoUrl,
    githubRepoBindingError,
    githubRepoBindingErrorTraceId,
    githubConnectedOptions,
    repoBindingBySlug,
    selectedGithubUserIdBySlug,
  } = refs

  const fetchGithubRepoBindingStatus = async () => {
    const { tenantId, workspaceId, taskId } = getIds()
    if (!tenantId || !workspaceId || !taskId) return
    githubRepoBindingLoading.value = true
    githubRepoBindingError.value = ''
    githubRepoBindingErrorTraceId.value = ''
    try {
      // taskCloudService 约定：funcName 在前、kv 键值对在后
      const response = await apiFetch(
        `/api/cloud/compute/github-credential-status/tenant_id/${tenantId}/workspace_id/${workspaceId}/task_id/${taskId}/`,
        { credentials: 'include', headers: { Accept: 'application/json' } },
      )
      const data = await response.json().catch(() => ({}))
      if (!response.ok) {
        assignGithubRepoBindingError(githubRepoBindingError, githubRepoBindingErrorTraceId, data, {
          fallback: '无法加载 GitHub 账号绑定状态',
          response,
        })
        return
      }
      githubConnectedOptions.value = Array.isArray(data.github_connections)
        ? data.github_connections.filter((x) => x && x.connected && x.github_user_id != null)
        : []
      repoBindingBySlug.value = githubRepoBindingsBySlug(data.repo_bindings)
      selectedGithubUserIdBySlug.value = {}
    } catch (e) {
      assignGithubRepoBindingCatchError(
        githubRepoBindingError,
        githubRepoBindingErrorTraceId,
        e,
        '加载 GitHub 账号绑定状态失败',
      )
    } finally {
      githubRepoBindingLoading.value = false
    }
    try {
      await fetchRepoOAuthConnectionStatusByRepoUrl()
    } catch {
      // ignore: per-repo OAuth check errors are kept in per-row status map
    }
  }

  const saveGithubRepoBinding = async (repoUrl) => {
    const slug = githubRepoSlugFromUrl(repoUrl)
    if (!slug) return
    const githubUserIdRaw = selectedGithubUserIdForRepo(repoUrl)
    if (!githubUserIdRaw) {
      window.alert('请先选择 GitHub 账号')
      return
    }
    const { tenantId, workspaceId, taskId } = getIds()
    if (!tenantId || !workspaceId || !taskId) return
    githubRepoBindingSaving.value = true
    githubBindingSavingRepoUrl.value = repoUrl
    githubRepoBindingError.value = ''
    githubRepoBindingErrorTraceId.value = ''
    try {
      const response = await apiFetch(
        `/api/cloud/compute/github-credential-approve/tenant_id/${tenantId}/workspace_id/${workspaceId}/task_id/${taskId}/`,
        {
          method: 'POST',
          credentials: 'include',
          headers: {
            Accept: 'application/json',
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({
            repo_url: String(repoUrl || ''),
            repo_slug: slug,
            github_user_id: String(githubUserIdRaw),
          }),
        },
      )
      const data = await response.json().catch(() => ({}))
      if (!response.ok) {
        throw makeGithubRepoBindingHttpError(data, response, '保存 GitHub 账号失败')
      }
      await fetchGithubRepoBindingStatus()
    } catch (e) {
      assignGithubRepoBindingCatchError(
        githubRepoBindingError,
        githubRepoBindingErrorTraceId,
        e,
        '保存 GitHub 账号失败',
      )
    } finally {
      githubBindingSavingRepoUrl.value = ''
      githubRepoBindingSaving.value = false
    }
  }

  return { fetchGithubRepoBindingStatus, saveGithubRepoBinding }
}

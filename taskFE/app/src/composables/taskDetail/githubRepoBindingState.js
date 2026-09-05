/**
 * GitHub credential status/approve binding for linked project repos.
 */
import { ref } from 'vue'
import { apiFetch } from '../../utils/apiUtils.js'
import { extractTraceId } from '../../utils/traceId.js'
import { resolveApiErrorMessage } from '../../utils/workPanelFormat.js'
import { githubRepoBindingsBySlug, githubRepoSlugFromUrl } from '../../utils/taskDetailBranchAndRepoUtils.js'

/**
 * @param {object} deps
 * @param {() => { tenantId: string, workspaceId: string, taskId: string }} deps.getRouteIds
 * @param {() => Promise<void>} [deps.fetchRepoOAuthConnectionStatusByRepoUrl]
 */
export function createGithubRepoBindingState(deps) {
  const { getRouteIds, fetchRepoOAuthConnectionStatusByRepoUrl } = deps

  const githubRepoBindingLoading = ref(false)
  const githubRepoBindingSaving = ref(false)
  const githubBindingSavingRepoUrl = ref('')
  const githubRepoBindingError = ref('')
  const githubRepoBindingErrorTraceId = ref('')
  const githubConnectedOptions = ref([])
  const repoBindingBySlug = ref({})
  const selectedGithubUserIdBySlug = ref({})

  const selectedGithubUserIdForRepo = (repoUrl) => {
    const slug = githubRepoSlugFromUrl(repoUrl)
    if (!slug) return ''
    const draft = selectedGithubUserIdBySlug.value[slug]
    if (draft != null && String(draft).trim()) return String(draft)
    const bound = repoBindingBySlug.value[slug]
    if (bound?.selected_github_user_id != null) return String(bound.selected_github_user_id)
    return ''
  }

  const hasGithubRepoBindingSaved = (repoUrl) => {
    const slug = githubRepoSlugFromUrl(repoUrl)
    if (!slug) return false
    const bound = repoBindingBySlug.value[slug]
    return bound?.selected_github_user_id != null && String(bound.selected_github_user_id).trim() !== ''
  }

  const setSelectedGithubUserIdForRepo = (repoUrl, value) => {
    const slug = githubRepoSlugFromUrl(repoUrl)
    if (!slug) return
    selectedGithubUserIdBySlug.value = {
      ...selectedGithubUserIdBySlug.value,
      [slug]: String(value || '').trim(),
    }
  }

  const fetchGithubRepoBindingStatus = async () => {
    const { tenantId, workspaceId, taskId } = getRouteIds()
    if (!tenantId || !workspaceId || !taskId) return
    githubRepoBindingLoading.value = true
    githubRepoBindingError.value = ''
    githubRepoBindingErrorTraceId.value = ''
    try {
      const response = await apiFetch(
        `/api/cloud/compute/github-credential-status/tenant_id/${tenantId}/workspace_id/${workspaceId}/task_id/${taskId}/`,
        { credentials: 'include', headers: { Accept: 'application/json' } },
      )
      const data = await response.json().catch(() => ({}))
      if (!response.ok) {
        githubRepoBindingError.value = resolveApiErrorMessage(data, { fallback: '无法加载 GitHub 账号绑定状态' })
        githubRepoBindingErrorTraceId.value = extractTraceId(response) || extractTraceId(data) || ''
        return
      }
      githubConnectedOptions.value = Array.isArray(data.github_connections)
        ? data.github_connections.filter((x) => x && x.connected && x.github_user_id != null)
        : []
      repoBindingBySlug.value = githubRepoBindingsBySlug(data.repo_bindings)
      selectedGithubUserIdBySlug.value = {}
    } catch (e) {
      githubRepoBindingError.value = e?.message || '加载 GitHub 账号绑定状态失败'
      githubRepoBindingErrorTraceId.value = extractTraceId(e) || ''
    } finally {
      githubRepoBindingLoading.value = false
    }
    if (typeof fetchRepoOAuthConnectionStatusByRepoUrl === 'function') {
      try {
        await fetchRepoOAuthConnectionStatusByRepoUrl()
      } catch {
        // ignore: per-repo OAuth check errors are kept in per-row status map
      }
    }
  }

  const sleep = (ms) =>
    new Promise((resolve) => {
      setTimeout(resolve, ms)
    })

  const fetchGithubRepoBindingStatusAfterOauth = async () => {
    const attempts = 6
    const delayMs = 350
    for (let i = 0; i < attempts; i += 1) {
      await fetchGithubRepoBindingStatus()
      if (i < attempts - 1) {
        await sleep(delayMs)
      }
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
    const { tenantId, workspaceId, taskId } = getRouteIds()
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
        const err = new Error(resolveApiErrorMessage(data, { fallback: '保存 GitHub 账号失败' }))
        err.traceId = extractTraceId(response) || extractTraceId(data) || ''
        throw err
      }
      await fetchGithubRepoBindingStatus()
    } catch (e) {
      githubRepoBindingError.value = e?.message || '保存 GitHub 账号失败'
      githubRepoBindingErrorTraceId.value = extractTraceId(e) || ''
    } finally {
      githubBindingSavingRepoUrl.value = ''
      githubRepoBindingSaving.value = false
    }
  }

  return {
    githubRepoBindingLoading,
    githubRepoBindingSaving,
    githubBindingSavingRepoUrl,
    githubRepoBindingError,
    githubRepoBindingErrorTraceId,
    githubConnectedOptions,
    repoBindingBySlug,
    selectedGithubUserIdBySlug,
    selectedGithubUserIdForRepo,
    hasGithubRepoBindingSaved,
    setSelectedGithubUserIdForRepo,
    fetchGithubRepoBindingStatus,
    fetchGithubRepoBindingStatusAfterOauth,
    saveGithubRepoBinding,
  }
}

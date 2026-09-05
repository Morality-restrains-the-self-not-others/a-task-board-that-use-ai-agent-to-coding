import { ref } from 'vue'
import { apiFetch } from '../../utils/apiUtils.js'
import { resolveAuthenticatedUserId } from '../../utils/sessionUserIdUtils.js'
import { gitCloneRefMatchKey } from '../../utils/taskDetailContainerCloneProgress.js'

/**
 * Layer-graph / 任务关联面板：Git 身份选项与容器侧按仓库匹配。
 * @param {{ effectiveTenantId: import('vue').Ref }} deps
 */
export function createLayerGraphGitIdentityState(deps) {
  const { effectiveTenantId } = deps

  const layerGitIdentityOptions = ref([])
  const layerGitIdentityLoading = ref(false)
  const layerGitGithubAppOauthConnected = ref(false)
  const layerRepoGitIdentityLoading = ref(false)
  const layerRepoGitIdentityFetchError = ref('')
  const layerRepoGitIdentityRows = ref([])
  const perRepoGitIdentitySyncing = ref(false)
  const perRepoGitIdentitySyncError = ref('')

  const fetchLayerGitIdentityOptions = async () => {
    const tenantId = effectiveTenantId.value
    if (!tenantId) {
      layerGitIdentityOptions.value = []
      layerGitGithubAppOauthConnected.value = false
      return
    }
    layerGitIdentityLoading.value = true
    try {
      const userId = await resolveAuthenticatedUserId()
      if (!userId) {
        throw new Error('缺少 userId，无法获取 Git 身份')
      }
      const apiPath = `/api/git-identities/user/${encodeURIComponent(userId)}/`
      const response = await apiFetch(
        `${apiPath}?company_id=${encodeURIComponent(String(tenantId))}`,
        {
          credentials: 'include',
          headers: { Accept: 'application/json' },
        },
      )
      const data = await response.json().catch(() => ({}))
      if (!response.ok) {
        throw new Error(data.detail || '获取 Git 身份失败')
      }
      layerGitIdentityOptions.value = Array.isArray(data.identities) ? data.identities : []
      layerGitGithubAppOauthConnected.value = Boolean(data.github_app_user_oauth_connected)
    } catch (error) {
      console.error('获取 Git 身份失败:', error)
      layerGitIdentityOptions.value = []
      layerGitGithubAppOauthConnected.value = false
    } finally {
      layerGitIdentityLoading.value = false
    }
  }

  const containerGitIdentityRowForRepoUrl = (repoUrl) => {
    const want = gitCloneRefMatchKey(repoUrl)
    if (!want) return null
    const rows = layerRepoGitIdentityRows.value
    if (!Array.isArray(rows)) return null
    return (
      rows.find((r) => {
        if (!r || typeof r !== 'object') return false
        const key =
          String(r.repo_match_key || '').trim() || gitCloneRefMatchKey(r.origin_url || '')
        return key === want
      }) || null
    )
  }

  const containerGitIdentityLineForRepoUrl = (repoUrl) => {
    const row = containerGitIdentityRowForRepoUrl(repoUrl)
    if (!row) return ''
    const name = typeof row.user_name === 'string' ? row.user_name.trim() : ''
    const email = typeof row.user_email === 'string' ? row.user_email.trim() : ''
    if (name && email) return `${name} <${email}>`
    if (name) return name
    if (email) return email
    if (row.error) return String(row.error)
    return ''
  }

  return {
    layerGitIdentityOptions,
    layerGitIdentityLoading,
    layerGitGithubAppOauthConnected,
    layerRepoGitIdentityLoading,
    layerRepoGitIdentityFetchError,
    layerRepoGitIdentityRows,
    perRepoGitIdentitySyncing,
    perRepoGitIdentitySyncError,
    fetchLayerGitIdentityOptions,
    containerGitIdentityRowForRepoUrl,
    containerGitIdentityLineForRepoUrl,
  }
}

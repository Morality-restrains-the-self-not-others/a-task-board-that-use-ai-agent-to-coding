import { ref, computed } from 'vue'
import { apiFetch } from '../utils/apiUtils.js'
import { extractTraceId } from '../utils/traceId.js'
import { normalizePurchasedGitlabRegions } from './gitlabPurchasedRegions.js'

export { normalizePurchasedGitlabRegions }

const GITLAB_SYNC_REGION_PREFIX = 'gitlab-sync-region:'

export const gitlabSyncRegionStorageKey = (tenantId) =>
  `${GITLAB_SYNC_REGION_PREFIX}${String(tenantId || '').trim()}`

/** 上次为该租户选用的 GitLab 同步区域 slug；存储/配额异常时静默降级为空串。 */
export const readStoredGitlabSyncRegion = (tenantId) => {
  try {
    return window.localStorage.getItem(gitlabSyncRegionStorageKey(tenantId)) || ''
  } catch {
    return ''
  }
}

const writeStoredGitlabSyncRegion = (tenantId, slug) => {
  const key = gitlabSyncRegionStorageKey(tenantId)
  const value = String(slug || '').trim()
  if (!key || !value) return
  try {
    window.localStorage.setItem(key, value)
  } catch {
    // storage disabled/quota exceeded：仅丢失「记住上次区域」，不阻断同步流程
  }
}

export function useGitlabProjectSync({ tenantId, onProjectsCreated }) {
  const showModal = ref(false)
  const loading = ref(false)
  const creating = ref(false)
  const error = ref('')
  const errorTraceId = ref('')
  const oauthBound = ref(false)
  const gitlabWebsite = ref('')
  const gitlabLogin = ref('')
  const providerKey = ref('')
  const repos = ref([])
  /** @type {import('vue').Ref<Record<string, true>>} */
  const selectedRepoKeys = ref({})
  const workspaces = ref([])
  const selectedWorkspaceId = ref('')
  const createResult = ref(null)
  const combinedProjectName = ref('')
  const combinedProjectNameTouched = ref(false)

  /** @type {import('vue').Ref<ReturnType<typeof normalizePurchasedGitlabRegions>>} */
  const purchasedRegions = ref([])
  const selectedRegionSlug = ref('')
  const regionsLoading = ref(false)

  let remoteLoadGeneration = 0

  const selectedRegion = computed(
    () => purchasedRegions.value.find((r) => r.region === selectedRegionSlug.value) || null,
  )

  const batchSelectableRepos = computed(() =>
    repos.value.filter((r) => !r.imported_in_single_repo_project),
  )

  const selectedReposForMerge = computed(() =>
    repos.value.filter((r) => selectedRepoKeys.value[r.http_url_to_repo]),
  )

  const selectedReposForBatch = computed(() =>
    selectedReposForMerge.value.filter((r) => !r.imported_in_single_repo_project),
  )

  const selectedCount = computed(() => selectedReposForMerge.value.length)

  const batchEligibleCount = computed(() => selectedReposForBatch.value.length)

  const allSelectableChecked = computed(() => {
    if (!repos.value.length) return false
    return repos.value.every((r) => selectedRepoKeys.value[r.http_url_to_repo])
  })

  const suggestedCombinedProjectName = computed(() => {
    const selected = selectedReposForMerge.value
    if (!selected.length) return ''
    if (selected.length === 1) return selected[0].name
    const groupPrefixes = selected
      .map((r) => {
        const parts = String(r.path_with_namespace || '').split('/')
        return parts.length > 1 ? parts.slice(0, -1).join('/') : parts[0]
      })
      .filter(Boolean)
    const uniquePrefixes = [...new Set(groupPrefixes)]
    if (uniquePrefixes.length === 1) {
      const leaf = uniquePrefixes[0].split('/').pop()
      return leaf || selected[0].name
    }
    return `${selected[0].name} 等`
  })

  const syncCombinedProjectName = () => {
    if (selectedReposForMerge.value.length >= 2) {
      if (!combinedProjectNameTouched.value) {
        combinedProjectName.value = suggestedCombinedProjectName.value
      }
      return
    }
    if (!combinedProjectNameTouched.value) {
      combinedProjectName.value = ''
    }
  }

  const setCombinedProjectName = (value) => {
    combinedProjectName.value = value
    combinedProjectNameTouched.value = true
  }

  const resetSelectionState = () => {
    selectedRepoKeys.value = {}
    combinedProjectName.value = ''
    combinedProjectNameTouched.value = false
    createResult.value = null
    repos.value = []
    oauthBound.value = false
    gitlabLogin.value = ''
    providerKey.value = ''
  }

  const loadPurchasedRegions = async () => {
    const tid = tenantId.value
    if (!tid) {
      purchasedRegions.value = []
      selectedRegionSlug.value = ''
      return
    }
    regionsLoading.value = true
    try {
      const response = await apiFetch(
        `/api/tenant/${encodeURIComponent(tid)}/billing/gitlab-resources/`,
        { headers: { Accept: 'application/json' }, timeout: 15000 },
      )
      const data = await response.json().catch(() => ({}))
      const requestTraceId = extractTraceId(response) || extractTraceId(data) || ''
      if (!response.ok) {
        purchasedRegions.value = []
        selectedRegionSlug.value = ''
        error.value = data.error || data.detail || '加载已购买 GitLab 列表失败'
        errorTraceId.value = requestTraceId
        return
      }
      const next = normalizePurchasedGitlabRegions(data.resources)
      purchasedRegions.value = next
      if (!next.length) {
        selectedRegionSlug.value = ''
        gitlabWebsite.value = ''
        return
      }
      const storedSlug = readStoredGitlabSyncRegion(tid)
      const preferred =
        storedSlug && next.some((r) => r.region === storedSlug)
          ? storedSlug
          : selectedRegionSlug.value
      const stillValid = next.some((r) => r.region === preferred)
      const nextSlug = stillValid ? preferred : next[0].region
      selectedRegionSlug.value = nextSlug
      writeStoredGitlabSyncRegion(tid, nextSlug)
      const picked = next.find((r) => r.region === selectedRegionSlug.value) || next[0]
      gitlabWebsite.value = picked.gitlab_web_url
    } catch (err) {
      purchasedRegions.value = []
      selectedRegionSlug.value = ''
      error.value =
        err?.name === 'TimeoutError'
          ? '加载已购买 GitLab 列表超时，请稍后重试'
          : '网络错误，无法加载已购买 GitLab 列表'
      errorTraceId.value = extractTraceId(err) || ''
    } finally {
      regionsLoading.value = false
    }
  }

  const openModal = async () => {
    showModal.value = true
    error.value = ''
    errorTraceId.value = ''
    resetSelectionState()
    await Promise.all([loadWorkspaces(), loadPurchasedRegions()])
    if (!purchasedRegions.value.length) {
      loading.value = false
      return
    }
    await loadRemoteRepos()
  }

  const closeModal = () => {
    if (creating.value) return
    showModal.value = false
  }

  const loadWorkspaces = async () => {
    const tid = tenantId.value
    if (!tid) return
    try {
      const response = await apiFetch(`/api/projects/workspaces/tenant_id/${tid}`)
      if (response.ok) {
        const data = await response.json()
        workspaces.value = Array.isArray(data) ? data : []
        if (!selectedWorkspaceId.value && workspaces.value.length > 0) {
          selectedWorkspaceId.value = String(workspaces.value[0].id)
        }
      }
    } catch {
      // workspace load failure handled at create time
    }
  }

  const loadRemoteRepos = async () => {
    const tid = tenantId.value
    if (!tid) {
      error.value = '缺少租户 ID'
      return
    }
    const host = String(selectedRegion.value?.gitlab_web_url || gitlabWebsite.value || '').trim()
    if (!host) {
      error.value = '请先选择已购买的 GitLab 区域'
      repos.value = []
      oauthBound.value = false
      return
    }
    const generation = ++remoteLoadGeneration
    loading.value = true
    error.value = ''
    errorTraceId.value = ''
    try {
      const qs = new URLSearchParams({ gitlab_host: host })
      const response = await apiFetch(
        `/api/projects/gitlab-remote-repos/tenant_id/${tid}/?${qs.toString()}`,
        { timeout: 90000 },
      )
      const data = await response.json().catch(() => ({}))
      const requestTraceId =
        extractTraceId(response) || extractTraceId(data) || ''
      // Always extract provider metadata — the backend includes them even on
      // 401/502 errors so the OAuth start flow can resolve the correct provider.
      gitlabWebsite.value = data.gitlab_website || host
      providerKey.value = data.provider_key || ''
      gitlabLogin.value = data.gitlab_login || ''
      if (!response.ok) {
        error.value = data.detail || data.error || '加载 GitLab 仓库失败'
        errorTraceId.value = requestTraceId
        repos.value = []
        oauthBound.value = false
        return
      }
      oauthBound.value = Boolean(data.oauth_bound)
      repos.value = Array.isArray(data.repos) ? data.repos : []
      if (data.error) {
        error.value = data.error
        errorTraceId.value = requestTraceId
      }
      selectedRepoKeys.value = Object.fromEntries(
        repos.value.map((repo) => [repo.http_url_to_repo, true]),
      )
      syncCombinedProjectName()
    } catch (err) {
      error.value =
        err?.name === 'TimeoutError'
          ? '加载 GitLab 仓库超时，请稍后重试'
          : '网络错误，请稍后重试'
      errorTraceId.value = extractTraceId(err) || ''
      repos.value = []
    } finally {
      if (generation === remoteLoadGeneration) {
        loading.value = false
      }
    }
  }

  const selectRegion = async (slug) => {
    const next = String(slug || '').trim()
    if (!next || next === selectedRegionSlug.value) return
    const meta = purchasedRegions.value.find((r) => r.region === next)
    if (!meta) return
    selectedRegionSlug.value = next
    writeStoredGitlabSyncRegion(tenantId.value, next)
    gitlabWebsite.value = meta.gitlab_web_url
    resetSelectionState()
    gitlabWebsite.value = meta.gitlab_web_url
    await loadRemoteRepos()
  }

  const toggleRepo = (repoUrl, checked) => {
    const next = { ...selectedRepoKeys.value }
    if (checked) {
      next[repoUrl] = true
    } else {
      delete next[repoUrl]
    }
    selectedRepoKeys.value = next
    syncCombinedProjectName()
  }

  const toggleSelectAll = (checked) => {
    if (!checked) {
      selectedRepoKeys.value = {}
      combinedProjectName.value = ''
      combinedProjectNameTouched.value = false
      return
    }
    selectedRepoKeys.value = Object.fromEntries(
      repos.value.map((r) => [r.http_url_to_repo, true]),
    )
    syncCombinedProjectName()
  }

  /**
   * Real anchor href for starting GitLab OAuth from the sync modal.
   * Browser GET with Accept: text/html is 302'd to the provider authorize page
   * （与全站 repoOAuthAuthorizeUtils.buildRepoOAuthStartHref 同导航模式）。
   */
  const gitlabOAuthStartHref = computed(() => {
    const tid = tenantId.value
    const params = new URLSearchParams()
    if (providerKey.value) {
      const sp = providerKey.value.split(':')[1]
      if (sp) params.set('service_provider', sp)
    }
    const repoUrl = String(
      selectedRegion.value?.gitlab_web_url || gitlabWebsite.value || '',
    ).trim()
    if (repoUrl) {
      params.set('repo_url', repoUrl)
    }
    params.set('next', `/tenant/${tid}/projects/`)
    return `/api/git-oauth/gitlab-start-from-gateway/?${params.toString()}`
  })

  const createSelectedProjects = async () => {
    const tid = tenantId.value
    if (!tid) {
      error.value = '缺少租户 ID'
      return
    }
    if (!selectedWorkspaceId.value) {
      error.value = '请选择工作空间'
      return
    }
    const selected = selectedReposForBatch.value
    if (!selected.length) {
      error.value = '请至少选择一个可单独创建项目的仓库（单仓已占用的仓库不可批量创建）'
      return
    }

    creating.value = true
    error.value = ''
    errorTraceId.value = ''
    createResult.value = null
    try {
      const response = await apiFetch(`/api/projects/batch-from-gitlab-repos/tenant_id/${tid}/`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({
          workspace_id: selectedWorkspaceId.value,
          repos: selected.map((r) => ({
            name: r.name,
            description: r.description || `从 GitLab 同步：${r.path_with_namespace || r.name}`,
            http_url_to_repo: r.http_url_to_repo,
          })),
        }),
      })
      const data = await response.json().catch(() => ({}))
      const requestTraceId =
        extractTraceId(response) || extractTraceId(data) || ''
      if (!response.ok) {
        error.value = data.error || data.detail || '创建项目失败'
        errorTraceId.value = requestTraceId
        return
      }
      createResult.value = data
      if (typeof onProjectsCreated === 'function') {
        await onProjectsCreated(data)
      }
      if (!data.errors?.length) {
        showModal.value = false
      }
    } catch (err) {
      error.value = '网络错误，请稍后重试'
      errorTraceId.value = extractTraceId(err) || ''
    } finally {
      creating.value = false
    }
  }

  const createCombinedProject = async () => {
    const tid = tenantId.value
    if (!tid) {
      error.value = '缺少租户 ID'
      return
    }
    if (!selectedWorkspaceId.value) {
      error.value = '请选择工作空间'
      return
    }
    const selected = selectedReposForMerge.value
    if (selected.length < 2) {
      error.value = '合并创建请至少选择 2 个仓库'
      return
    }
    const name = combinedProjectName.value.trim()
    if (!name) {
      error.value = '请输入合并项目的名称'
      return
    }

    creating.value = true
    error.value = ''
    errorTraceId.value = ''
    createResult.value = null
    try {
      const response = await apiFetch(`/api/projects/combined-from-gitlab-repos/tenant_id/${tid}/`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({
          workspace_id: selectedWorkspaceId.value,
          name,
          description: `从 GitLab 同步合并：${selected.map((r) => r.path_with_namespace || r.name).join('、')}`,
          repos: selected.map((r) => ({
            name: r.name,
            description: r.description || '',
            http_url_to_repo: r.http_url_to_repo,
          })),
        }),
      })
      const data = await response.json().catch(() => ({}))
      const requestTraceId =
        extractTraceId(response) || extractTraceId(data) || ''
      if (!response.ok) {
        error.value = data.error || data.detail || data.errors?.[0]?.error || '合并创建项目失败'
        errorTraceId.value = requestTraceId
        return
      }
      createResult.value = {
        created: data.created ? [data.created] : [],
        skipped: data.skipped || [],
        errors: data.errors || [],
      }
      if (typeof onProjectsCreated === 'function') {
        await onProjectsCreated(createResult.value)
      }
      if (!data.errors?.length) {
        showModal.value = false
      }
    } catch (err) {
      error.value = '网络错误，请稍后重试'
      errorTraceId.value = extractTraceId(err) || ''
    } finally {
      creating.value = false
    }
  }

  return {
    showModal,
    loading,
    creating,
    error,
    errorTraceId,
    oauthBound,
    gitlabWebsite,
    gitlabLogin,
    purchasedRegions,
    selectedRegionSlug,
    regionsLoading,
    selectedRegion,
    repos,
    batchSelectableRepos,
    selectedCount,
    batchEligibleCount,
    allSelectableChecked,
    selectedRepoKeys,
    workspaces,
    selectedWorkspaceId,
    createResult,
    combinedProjectName,
    setCombinedProjectName,
    suggestedCombinedProjectName,
    selectedReposForMerge,
    selectedReposForBatch,
    openModal,
    closeModal,
    loadPurchasedRegions,
    loadRemoteRepos,
    selectRegion,
    toggleRepo,
    toggleSelectAll,
    gitlabOAuthStartHref,
    createSelectedProjects,
    createCombinedProject,
  }
}

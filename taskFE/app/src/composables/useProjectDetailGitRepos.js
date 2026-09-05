import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { apiFetch } from '../utils/apiUtils.js'
import { resolveRepoOAuthAuthorizeUrl, resolveRepoOAuthProviderInfo } from '../utils/repoOAuthAuthorizeUtils.js'
import { useProjectGitOAuthCatalog } from './useProjectGitOAuthCatalog.js'
import { useProjectRepoBranchPreview } from './useProjectRepoBranchPreview.js'
import { createGithubAppReturnKey, setGithubAppReturnTarget } from '../utils/githubAppReturnStorage.js'
import { buildOauthReturnPath } from '../utils/githubAppContinueNav.js'
import {
  gitRepoOAuthStatusBadgeClass,
  isPlaceholderGitRepoOAuthStatus,
  resolveGitRepoOAuthStatusLabel,
} from '../utils/gitRepoOAuthStatusUtils.js'
import { extractTraceId } from '../utils/traceId.js'
import {
  indexValidateResultsByUrl,
  lookupValidateResultByUrl,
  resolveValidateGitReposRequestFailure,
  VALIDATE_GIT_REPOS_TIMEOUT_MS,
} from '../utils/gitRepoValidateError.js'
/**
 * 项目详情页 Git 仓库行：OAuth 状态、授权跳转、分支预览。
 * @param {{ project: import('vue').Ref|import('vue').ComputedRef }} options
 */
export function useProjectDetailGitRepos({ project }) {
  const route = useRoute()
  const router = useRouter()

  const projectGitRepoList = computed(() => {
    const p = project.value
    return Array.isArray(p?.git_repos) ? p.git_repos.filter(Boolean) : []
  })

  const diskSizeByUrl = ref({})

  /** Prefer structured entries (url + clone_alias) when API returns them. */
  const projectGitRepoEntries = computed(() => {
    const p = project.value
    const overlay = diskSizeByUrl.value
    const entries = Array.isArray(p?.git_repo_entries) ? p.git_repo_entries : null
    if (entries && entries.length) {
      return entries
        .map((e) => {
          if (!e || typeof e !== 'object') return null
          const url = String(e.url || e.repo_url || '').trim()
          if (!url) return null
          return {
            url,
            cloneAlias: String(e.clone_alias || e.cloneAlias || '').trim(),
            isInternal: Boolean(e.is_internal ?? e.isInternal),
            diskSizeBytes:
              overlay[url] ??
              normalizeDiskSizeBytes(e.disk_size_bytes ?? e.diskSizeBytes),
          }
        })
        .filter(Boolean)
    }
    return projectGitRepoList.value.map((url) => ({
      url,
      cloneAlias: '',
      isInternal: false,
      diskSizeBytes: overlay[url] ?? null,
    }))
  })

  /** @param {unknown} raw */
  function normalizeDiskSizeBytes(raw) {
    if (raw == null || raw === '') return null
    const n = typeof raw === 'number' ? raw : Number(raw)
    if (!Number.isFinite(n) || n < 0) return null
    return n
  }

  const repoOAuthActionLoadingByUrl = ref({})
  const repoOAuthActionErrorByUrl = ref({})
  const repoOAuthActionErrorTraceIdByUrl = ref({})
  const gitRepoTokenStatus = ref({})
  const gitRepoStatusLoading = ref({})
  const { touchRepoOAuthButtons, bootstrapGitOAuthCatalog } = useProjectGitOAuthCatalog(
    apiFetch,
    () => String(project.value?.company || route.params.tenant || ''),
  )
  watch(
    () => String(project.value?.company || '').trim(),
    (cid, prev) => {
      if (cid && cid !== prev) {
        void bootstrapGitOAuthCatalog()
      }
    },
  )

  const fetchGitRepoDiskSizes = async () => {
    const tenantId = route.params.tenant
    const projectId = String(project.value?.id || '').trim()
    if (!tenantId || !projectId) return
    try {
      const response = await apiFetch(
        `/api/projects/${encodeURIComponent(projectId)}/git-repo-disk-sizes/tenant_id/${encodeURIComponent(tenantId)}/`,
        {
          method: 'GET',
          credentials: 'include',
          headers: { Accept: 'application/json' },
          timeout: VALIDATE_GIT_REPOS_TIMEOUT_MS,
        },
      )
      const data = await response.json().catch(() => ({}))
      if (!response.ok || !Array.isArray(data?.git_repo_entries)) {
        return
      }
      const overlay = {}
      data.git_repo_entries.forEach((entry) => {
        if (!entry || typeof entry !== 'object') return
        const url = String(entry.url || entry.repo_url || '').trim()
        if (!url) return
        const size = normalizeDiskSizeBytes(entry.disk_size_bytes ?? entry.diskSizeBytes)
        if (size != null) overlay[url] = size
      })
      diskSizeByUrl.value = overlay
    } catch {
      return
    }
  }

  watch(
    () => String(project.value?.id || '').trim(),
    (pid) => {
      if (pid) void fetchGitRepoDiskSizes()
    },
    { immediate: true },
  )
  const { branchPreviewLoading, branchPreviewError, repoBranchPreviews, fetchProjectRepoBranchesPreview } =
    useProjectRepoBranchPreview({ apiFetch, route, project, projectGitRepoList })

  const normalizeRepoUrlKey = (repoUrl) => String(repoUrl || '').trim()

  const setRepoOAuthActionLoading = (repoUrl, loading) => {
    const key = normalizeRepoUrlKey(repoUrl)
    repoOAuthActionLoadingByUrl.value = {
      ...repoOAuthActionLoadingByUrl.value,
      [key]: Boolean(loading),
    }
  }

  const setRepoOAuthActionError = (repoUrl, errorMessage = '', traceId = '') => {
    const key = normalizeRepoUrlKey(repoUrl)
    const message = String(errorMessage || '').trim()
    const tid = message ? String(traceId || '').trim() : ''
    repoOAuthActionErrorByUrl.value = {
      ...repoOAuthActionErrorByUrl.value,
      [key]: message,
    }
    repoOAuthActionErrorTraceIdByUrl.value = {
      ...repoOAuthActionErrorTraceIdByUrl.value,
      [key]: tid,
    }
  }

  const isRepoOAuthActionDisabled = (repoUrl) => {
    const key = normalizeRepoUrlKey(repoUrl)
    return Boolean(repoOAuthActionLoadingByUrl.value[key])
  }

  const repoOAuthButtonLabel = (repoUrl) => {
    const key = normalizeRepoUrlKey(repoUrl)
    if (repoOAuthActionLoadingByUrl.value[key]) return '跳转中...'
    if (gitRepoTokenStatus.value[key] === 'token_error') return '重试'
    return 'OAuth 授权'
  }

  const shouldShowRepoOAuthButton = (repoUrl) => {
    touchRepoOAuthButtons()
    const info = resolveRepoOAuthProviderInfo(repoUrl)
    if (!info) return false
    const key = normalizeRepoUrlKey(repoUrl)
    const ts = gitRepoTokenStatus.value[key]
    if (ts === 'token_available') return false
    if (ts === 'not_applicable') return false
    return Boolean(resolveRepoOAuthAuthorizeUrl(info.provider))
  }

  const repoOAuthErrorByUrl = (repoUrl) => {
    const key = normalizeRepoUrlKey(repoUrl)
    return String(repoOAuthActionErrorByUrl.value[key] || '').trim()
  }

  const repoOAuthErrorTraceIdByUrl = (repoUrl) => {
    const key = normalizeRepoUrlKey(repoUrl)
    return String(repoOAuthActionErrorTraceIdByUrl.value[key] || '').trim()
  }

  const isRepoOAuthStatusLoading = (repoUrl) => {
    const key = normalizeRepoUrlKey(repoUrl)
    return Boolean(gitRepoStatusLoading.value[key])
  }

  const repoOAuthTokenStatus = (repoUrl) => {
    const key = normalizeRepoUrlKey(repoUrl)
    return String(gitRepoTokenStatus.value[key] || '').trim()
  }

  const repoOAuthStatusLabel = (repoUrl) => {
    const key = normalizeRepoUrlKey(repoUrl)
    return resolveGitRepoOAuthStatusLabel(gitRepoTokenStatus.value[key], {
      loading: isRepoOAuthStatusLoading(repoUrl),
    })
  }

  const repoOAuthStatusBadgeClass = (repoUrl) => {
    const key = normalizeRepoUrlKey(repoUrl)
    return gitRepoOAuthStatusBadgeClass(gitRepoTokenStatus.value[key], {
      loading: isRepoOAuthStatusLoading(repoUrl),
    })
  }

  const applyGitReposStatusFromApi = (statusList) => {
    if (!Array.isArray(statusList) || !statusList.length) return
    const next = { ...gitRepoTokenStatus.value }
    statusList.forEach((entry) => {
      const repoUrl = String(entry?.repo_url || '').trim()
      if (!repoUrl) return
      next[normalizeRepoUrlKey(repoUrl)] = String(entry?.token_status || '').trim() || 'not_applicable'
    })
    gitRepoTokenStatus.value = next
  }

  const repoUrlsNeedingLiveOAuthStatus = (repoUrls, { force = false } = {}) =>
    (Array.isArray(repoUrls) ? repoUrls : []).filter((url) => {
      const key = normalizeRepoUrlKey(url)
      if (isRepoOAuthStatusLoading(key)) return false
      const status = gitRepoTokenStatus.value[key]
      if (!force && !isPlaceholderGitRepoOAuthStatus(status)) return false
      return Boolean(resolveRepoOAuthProviderInfo(url))
    })

  const fetchGitReposOAuthStatus = async (repoUrls, opts = {}) => {
    const tenantId = route.params.tenant
    const targets = repoUrlsNeedingLiveOAuthStatus(repoUrls, opts)
    if (!tenantId || !targets.length) return

    const loadingPatch = { ...gitRepoStatusLoading.value }
    targets.forEach((url) => {
      loadingPatch[normalizeRepoUrlKey(url)] = true
    })
    gitRepoStatusLoading.value = loadingPatch

    let results = []
    try {
      // taskProjectService 分发器要求 tenant_id kv 键值对（无 kv 会 400 tenant_id required）；
      // 约定：action 位置段在前、kv 键值对在后
      const response = await apiFetch(
        `/api/projects/validate-git-repos/tenant_id/${encodeURIComponent(tenantId)}/`,
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
          credentials: 'include',
          body: JSON.stringify({
            urls: targets,
            probe_access: true,
            project_id: String(project.value?.id || '').trim(),
          }),
          timeout: VALIDATE_GIT_REPOS_TIMEOUT_MS,
        },
      )
      const data = await response.json().catch(() => ({}))
      const byUrl = indexValidateResultsByUrl(response.ok ? data?.results : null)
      if (!response.ok || !Array.isArray(data?.results)) {
        console.error('Batch validate-git-repos failed', response.status, data)
      }
      const requestFail = resolveValidateGitReposRequestFailure({ response, data })
      targets.forEach((url) => {
        const entry = lookupValidateResultByUrl(byUrl, url)
        if (entry) {
          setRepoOAuthActionError(url, '')
          results.push({
            url,
            token_status: String(entry?.token_status || '').trim() || 'not_applicable',
          })
          return
        }
        setRepoOAuthActionError(url, requestFail.message, requestFail.traceId)
      })
    } catch (err) {
      console.error('Error batch fetching git repo oauth status:', err)
      const failure = resolveValidateGitReposRequestFailure({ err })
      targets.forEach((url) => {
        setRepoOAuthActionError(url, failure.message, failure.traceId)
      })
    }

    const statusPatch = { ...gitRepoTokenStatus.value }
    const loadingDone = { ...gitRepoStatusLoading.value }
    const seen = new Set()
    results.forEach(({ url, token_status }) => {
      const key = normalizeRepoUrlKey(url)
      statusPatch[key] = token_status
      delete loadingDone[key]
      seen.add(key)
    })
    targets.forEach((url) => {
      const key = normalizeRepoUrlKey(url)
      if (!seen.has(key)) {
        delete loadingDone[key]
      }
    })
    gitRepoTokenStatus.value = statusPatch
    gitRepoStatusLoading.value = loadingDone
  }

  const refreshGitReposOAuthStatus = async (repoUrls, opts = {}) => {
    await fetchGitReposOAuthStatus(repoUrls, opts)
  }

  const resolveCurrentPagePathForOauth = () =>
    buildOauthReturnPath({
      routePath: String(route.path || '').trim(),
      pathname: String(window.location?.pathname || '').trim(),
      search: String(window.location?.search || ''),
      query: route.query,
    })

  const startRepoOAuthConnect = async (repoUrl) => {
    const info = resolveRepoOAuthProviderInfo(repoUrl)
    if (!info) return
    const startApiUrl = resolveRepoOAuthAuthorizeUrl(info.provider)
    if (!startApiUrl) return
    setRepoOAuthActionError(repoUrl, '')
    setRepoOAuthActionLoading(repoUrl, true)
    try {
      const nextPath = resolveCurrentPagePathForOauth()
      const returnKey = createGithubAppReturnKey()
      setGithubAppReturnTarget(returnKey, nextPath)
      const projectId = String(project.value?.id || '').trim()
      const targetUrl =
        `${startApiUrl}?next=${encodeURIComponent(nextPath)}` +
        `&return_key=${encodeURIComponent(returnKey)}` +
        `&repo_url=${encodeURIComponent(repoUrl)}` +
        `&service_provider=${encodeURIComponent(info.service_provider)}` +
        `&grant_kind=project` +
        `&grant_id=${encodeURIComponent(projectId)}`
      const response = await apiFetch(targetUrl, {
        credentials: 'include',
        headers: { Accept: 'application/json' },
        timeout: 10000,
      })
      const data = await response.json().catch(() => ({}))
      const requestTraceId =
        extractTraceId(response) || extractTraceId(data) || ''
      if (!response.ok) {
        setRepoOAuthActionError(
          repoUrl,
          typeof data?.detail === 'string' ? data.detail : '无法启动 OAuth 授权',
          requestTraceId,
        )
        return
      }
      if (data.authorize_url) {
        window.location.href = data.authorize_url
        return
      }
      setRepoOAuthActionError(repoUrl, '无法启动 OAuth 授权', requestTraceId)
    } catch (e) {
      const isTimeout = e?.name === 'TimeoutError'
      setRepoOAuthActionError(
        repoUrl,
        isTimeout
          ? 'OAuth 授权服务响应超时，请检查网络后重试'
          : (e?.message || '启动 OAuth 授权失败'),
        extractTraceId(e),
      )
    } finally {
      setRepoOAuthActionLoading(repoUrl, false)
    }
  }

  return {
    route,
    router,
    projectGitRepoList,
    projectGitRepoEntries,
    fetchGitRepoDiskSizes,
    branchPreviewLoading,
    branchPreviewError,
    repoBranchPreviews,
    fetchProjectRepoBranchesPreview,
    bootstrapGitOAuthCatalog,
    applyGitReposStatusFromApi,
    refreshGitReposOAuthStatus,
    isRepoOAuthActionDisabled,
    repoOAuthButtonLabel,
    shouldShowRepoOAuthButton,
    repoOAuthErrorByUrl,
    repoOAuthErrorTraceIdByUrl,
    repoOAuthTokenStatus,
    repoOAuthStatusLabel,
    repoOAuthStatusBadgeClass,
    startRepoOAuthConnect,
  }
}

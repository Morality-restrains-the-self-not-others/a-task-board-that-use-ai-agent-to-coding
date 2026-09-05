/**
 * Per-repo OAuth bind status / start-authorize helpers for linked projects panel.
 */
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { showRequestError } from '../../utils/requestErrorDisplay.js'
import { apiFetch } from '../../utils/apiUtils.js'
import { fetchAndFollowOauthAuthorizeUrl } from '../../utils/startRepoOauthAuthorize.js'
import { fetchGitOAuthUserAppConnected } from '../../utils/gitOAuthUserAppConnection.js'
import {
  createGithubAppReturnKey,
  setGithubAppReturnTarget,
} from '../../utils/githubAppReturnStorage.js'
import { buildOauthReturnPath } from '../../utils/githubAppContinueNav.js'
import {
  resolveRepoOAuthAuthorizeLabel,
  resolveRepoOAuthAuthorizeUrl,
  resolveRepoOAuthProvider,
  resolveRepoOAuthProviderInfo,
  shouldShowRepoOAuthAuthorizeButton,
  supportsRepoOAuthAuthorize,
} from '../../utils/repoOAuthAuthorizeUtils.js'
import { hasSessionGrantForRepo, rememberGrantTicketFromSearch } from '../../utils/grantTicketSession.js'

/**
 * @param {object} deps
 * @param {object} deps.props
 * @param {(event: string, ...args: any[]) => void} deps.emit
 */
const OAUTH_RETURN_RETRY_ATTEMPTS = 8
const OAUTH_RETURN_RETRY_DELAY_MS = 400

function sleep(ms) {
  return new Promise((resolve) => {
    setTimeout(resolve, ms)
  })
}

export function useLinkedProjectsRepoOAuth({ props, emit }) {
  const route = useRoute()
  const router = useRouter()
  if (typeof window !== 'undefined') {
    rememberGrantTicketFromSearch(window.location?.search || '')
  }
  const repoOAuthBoundByUrl = ref({})
  const repoOAuthCheckLoadingByUrl = ref({})
  const repoOAuthCheckErrorByUrl = ref({})
  const repoOAuthActionLoadingByUrl = ref({})
  const editRepoOAuthActionLoadingByUrl = ref({})

  const normalizeRepoUrlKey = (repoUrl) => String(repoUrl || '').trim()

  const collectTaskRepoOAuthTargets = () => {
    const rows = Array.isArray(props.taskProjectsWithDetails) ? props.taskProjectsWithDetails : []
    const deduped = new Map()
    rows.forEach((taskProject) => {
      const repos = Array.isArray(taskProject?.project?.git_repos) ? taskProject.project.git_repos : []
      repos.forEach((repoUrlRaw) => {
        const repoUrl = String(repoUrlRaw || '').trim()
        if (!repoUrl) return
        const provider = resolveRepoOAuthProvider(repoUrl)
        const startApiUrl = resolveRepoOAuthAuthorizeUrl(provider)
        if (!provider || !startApiUrl) return
        const domainLabel = resolveRepoOAuthAuthorizeLabel(repoUrl)
        const key = normalizeRepoUrlKey(repoUrl)
        if (deduped.has(key)) return
        deduped.set(key, {
          key,
          provider,
          repoUrl,
          domainLabel,
          startApiUrl,
        })
      })
    })
    return Array.from(deduped.values())
  }

  const isProviderConnectedForCurrentUser = async (_provider, repoUrl = '') => {
    // 网关凭 session cookie 注入 X-User-Id；不要因前端读不到 JS userId 而跳过查询，
    // 否则 OAuth 回流后（尤其 accessCode 页）会一直误显未绑定。
    return fetchGitOAuthUserAppConnected(String(repoUrl || '').trim())
  }

  const setRepoOAuthActionLoading = (repoUrl, loading) => {
    const key = normalizeRepoUrlKey(repoUrl)
    repoOAuthActionLoadingByUrl.value = {
      ...repoOAuthActionLoadingByUrl.value,
      [key]: Boolean(loading),
    }
  }

  const setRepoOAuthBoundState = (repoUrl, connected) => {
    const key = normalizeRepoUrlKey(repoUrl)
    repoOAuthBoundByUrl.value = {
      ...repoOAuthBoundByUrl.value,
      [key]: Boolean(connected),
    }
    repoOAuthCheckErrorByUrl.value = {
      ...repoOAuthCheckErrorByUrl.value,
      [key]: '',
    }
  }

  const fetchRepoOAuthConnectionStatusByRepoUrl = async () => {
    const targets = collectTaskRepoOAuthTargets()
    const loadingMap = {}
    const boundMap = {}
    const errorMap = {}
    for (const target of targets) {
      loadingMap[target.key] = true
      boundMap[target.key] = false
      errorMap[target.key] = ''
    }
    repoOAuthCheckLoadingByUrl.value = loadingMap
    repoOAuthBoundByUrl.value = boundMap
    repoOAuthCheckErrorByUrl.value = errorMap
    for (const target of targets) {
      try {
        const connected = await isProviderConnectedForCurrentUser(target.provider, target.repoUrl)
        boundMap[target.key] = Boolean(connected)
        errorMap[target.key] = ''
      } catch (e) {
        boundMap[target.key] = false
        errorMap[target.key] = e?.message || 'OAuth 绑定状态检查失败'
      } finally {
        loadingMap[target.key] = false
        repoOAuthCheckLoadingByUrl.value = { ...loadingMap }
        repoOAuthBoundByUrl.value = { ...boundMap }
        repoOAuthCheckErrorByUrl.value = { ...errorMap }
      }
    }
  }

  const resolveCurrentPagePathForOauth = () =>
    buildOauthReturnPath({
      routePath: String(route?.path || '').trim(),
      pathname: String(window.location?.pathname || '').trim(),
      search: String(window.location?.search || ''),
      query: route?.query,
    })

  const startProviderOauthForRepo = async ({ startApiUrl, repoUrl }) => {
    const nextPath = resolveCurrentPagePathForOauth()
    const returnKey = createGithubAppReturnKey()
    setGithubAppReturnTarget(returnKey, nextPath)
    const targetUrl =
      `${startApiUrl}?next=${encodeURIComponent(nextPath)}` +
      `&return_key=${encodeURIComponent(returnKey)}` +
      `&repo_url=${encodeURIComponent(repoUrl)}`
    await fetchAndFollowOauthAuthorizeUrl(apiFetch, targetUrl)
  }

  const shouldShowEditRepoOAuthAuthorize = (projectId, repoUrl) => {
    void props.gitOAuthCatalogVersion
    const errorMessage = props.getRepoBranchError(projectId, repoUrl)
    if (!shouldShowRepoOAuthAuthorizeButton(errorMessage)) return false
    if (!supportsRepoOAuthAuthorize(repoUrl)) return false
    const provider = resolveRepoOAuthProvider(repoUrl)
    return Boolean(resolveRepoOAuthAuthorizeUrl(provider))
  }

  const setEditRepoOAuthActionLoading = (repoUrl, loading) => {
    const key = normalizeRepoUrlKey(repoUrl)
    editRepoOAuthActionLoadingByUrl.value = {
      ...editRepoOAuthActionLoadingByUrl.value,
      [key]: Boolean(loading),
    }
  }

  const startEditRepoOAuthConnect = async (repoUrl) => {
    const info = resolveRepoOAuthProviderInfo(repoUrl)
    if (!info) {
      window.alert('当前仓库暂不支持 OAuth 一键授权')
      return
    }
    const startApiUrl = resolveRepoOAuthAuthorizeUrl(info.provider)
    if (!startApiUrl) {
      window.alert('当前仓库暂不支持 OAuth 一键授权')
      return
    }
    setEditRepoOAuthActionLoading(repoUrl, true)
    try {
      const nextPath = resolveCurrentPagePathForOauth()
      const returnKey = createGithubAppReturnKey()
      setGithubAppReturnTarget(returnKey, nextPath)
      const targetUrl =
        `${startApiUrl}?next=${encodeURIComponent(nextPath)}` +
        `&return_key=${encodeURIComponent(returnKey)}` +
        `&repo_url=${encodeURIComponent(repoUrl)}` +
        `&service_provider=${encodeURIComponent(info.service_provider)}`
      await fetchAndFollowOauthAuthorizeUrl(apiFetch, targetUrl)
    } catch (e) {
      showRequestError(e?.message || '启动 OAuth 授权失败', e)
    } finally {
      setEditRepoOAuthActionLoading(repoUrl, false)
    }
  }

  const startRepoOAuthConnect = async (repoUrl) => {
    const repoKey = normalizeRepoUrlKey(repoUrl)
    const target = collectTaskRepoOAuthTargets().find((item) => item.key === repoKey)
    if (!target) {
      window.alert('当前仓库暂不支持 OAuth 一键授权')
      return
    }
    setRepoOAuthActionLoading(repoUrl, true)
    try {
      const connected = await isProviderConnectedForCurrentUser(target.provider, target.repoUrl)
      if (connected) {
        setRepoOAuthBoundState(target.repoUrl, true)
        window.alert('该仓库地址已完成 OAuth 绑定')
        return
      }
      await startProviderOauthForRepo(target)
    } catch (e) {
      showRequestError(e?.message || '启动 GitHub 授权失败', e)
    } finally {
      setRepoOAuthActionLoading(repoUrl, false)
    }
  }

  const isRepoOAuthBindActionDisabled = (repoUrl) => {
    const key = normalizeRepoUrlKey(repoUrl)
    return Boolean(repoOAuthCheckLoadingByUrl.value[key] || repoOAuthActionLoadingByUrl.value[key])
  }

  const repoOAuthBindButtonLabel = (repoUrl) => {
    const key = normalizeRepoUrlKey(repoUrl)
    if (repoOAuthActionLoadingByUrl.value[key]) return '跳转中...'
    if (repoOAuthCheckLoadingByUrl.value[key]) return '检测中...'
    return 'OAuth 绑定'
  }

  const shouldShowRepoOAuthBindButton = (repoUrl) => {
    const key = normalizeRepoUrlKey(repoUrl)
    if (!key) return false
    const provider = resolveRepoOAuthProvider(repoUrl)
    const startApiUrl = resolveRepoOAuthAuthorizeUrl(provider)
    if (!provider || !startApiUrl) return false
    if (repoOAuthBoundByUrl.value[key] !== true) return true
    return !hasSessionGrantForRepo(repoUrl)
  }

  const shouldShowRepoOAuthBoundLabel = (repoUrl) => {
    const key = normalizeRepoUrlKey(repoUrl)
    if (!key) return false
    const provider = resolveRepoOAuthProvider(repoUrl)
    const startApiUrl = resolveRepoOAuthAuthorizeUrl(provider)
    if (!provider || !startApiUrl) return false
    return repoOAuthBoundByUrl.value[key] === true
  }

  const repoOAuthStartReadiness = computed(() => {
    void props.gitOAuthCatalogVersion
    const rows = Array.isArray(props.taskProjectsWithDetails) ? props.taskProjectsWithDetails : []
    let projectOAuthRepoCount = 0
    for (const taskProject of rows) {
      const repos = Array.isArray(taskProject?.project?.git_repos) ? taskProject.project.git_repos : []
      for (const repoUrlRaw of repos) {
        const repoUrl = String(repoUrlRaw || '').trim()
        if (!repoUrl) continue
        const provider = resolveRepoOAuthProvider(repoUrl)
        const startApiUrl = resolveRepoOAuthAuthorizeUrl(provider)
        if (provider && startApiUrl) projectOAuthRepoCount += 1
      }
    }
    const targets = collectTaskRepoOAuthTargets()
    const catalogPending = projectOAuthRepoCount > 0 && targets.length === 0
    const checkPending = targets.some(
      (target) => repoOAuthCheckLoadingByUrl.value[target.key] === undefined,
    )
    const loading =
      catalogPending
      || checkPending
      || targets.some((target) => Boolean(repoOAuthCheckLoadingByUrl.value[target.key]))
    const unboundRepoUrls = targets
      .filter((target) => repoOAuthBoundByUrl.value[target.key] !== true)
      .map((target) => target.repoUrl)
    const allBound = !catalogPending && (targets.length === 0 || unboundRepoUrls.length === 0)
    const hasOAuthRepos = projectOAuthRepoCount > 0 || targets.length > 0
    const startBlocked =
      catalogPending
      || (targets.length > 0 && (loading || unboundRepoUrls.length > 0))
      || (projectOAuthRepoCount > 0 && targets.length === 0)
    return {
      allBound,
      loading,
      unboundRepoUrls,
      hasOAuthRepos,
      startBlocked,
    }
  })

  watch(
    repoOAuthStartReadiness,
    (readiness) => {
      emit('repo-oauth-readiness', readiness)
      props.onReadinessChange?.(readiness)
    },
    { immediate: true, deep: true },
  )

  let fetchQueue = Promise.resolve()
  const enqueueConnectionFetch = () => {
    const job = fetchQueue.then(() => fetchRepoOAuthConnectionStatusByRepoUrl())
    fetchQueue = job.then(() => undefined, () => undefined)
    return job
  }

  const clearOauthOkQuery = async () => {
    const query = route?.query || {}
    if (query.gitlab == null && query.github == null) return
    const nextQuery = { ...query }
    delete nextQuery.gitlab
    delete nextQuery.github
    try {
      await router.replace({
        path: route?.path,
        query: nextQuery,
        hash: route?.hash || undefined,
      })
    } catch {
      // duplicated navigation
    }
  }

  // 用户从 OAuth 提供方跳回（?gitlab=ok / ?github=ok）后凭据可能晚一拍；
  // 有限次重试，不是后台轮询。
  const fetchConnectionAfterOauthReturn = async () => {
    for (let attempt = 0; attempt < OAUTH_RETURN_RETRY_ATTEMPTS; attempt += 1) {
      await enqueueConnectionFetch()
      const readiness = repoOAuthStartReadiness.value
      if (readiness.allBound && readiness.loading !== true) return
      if (attempt < OAUTH_RETURN_RETRY_ATTEMPTS - 1) {
        await sleep(OAUTH_RETURN_RETRY_DELAY_MS)
      }
    }
  }

  watch(
    () => [props.taskProjectsWithDetails, props.gitOAuthCatalogVersion],
    () => {
      void enqueueConnectionFetch()
    },
    { immediate: true, deep: true },
  )

  watch(
    () => [route?.query?.gitlab, route?.query?.github],
    async ([gitlabFlag, githubFlag]) => {
      if (typeof window !== 'undefined') {
        rememberGrantTicketFromSearch(window.location?.search || '')
      }
      const gitlabOk = String(gitlabFlag || '') === 'ok'
      const githubOk = String(githubFlag || '') === 'ok'
      if (!gitlabOk && !githubOk) return
      await fetchConnectionAfterOauthReturn()
      await clearOauthOkQuery()
    },
    { immediate: true },
  )

  return {
    normalizeRepoUrlKey,
    editRepoOAuthActionLoadingByUrl,
    fetchRepoOAuthConnectionStatusByRepoUrl,
    shouldShowEditRepoOAuthAuthorize,
    startEditRepoOAuthConnect,
    startRepoOAuthConnect,
    isRepoOAuthBindActionDisabled,
    repoOAuthBindButtonLabel,
    shouldShowRepoOAuthBindButton,
    shouldShowRepoOAuthBoundLabel,
    resolveRepoOAuthAuthorizeLabel,
    repoOAuthStartReadiness,
  }
}

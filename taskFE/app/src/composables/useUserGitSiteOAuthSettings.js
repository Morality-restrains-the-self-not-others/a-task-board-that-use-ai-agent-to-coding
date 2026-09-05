import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'

import { apiFetch } from '../utils/apiUtils'
import { showRequestError } from '../utils/requestErrorDisplay.js'
import { extractTraceId } from '../utils/traceId.js'
import { resolveAuthenticatedUserId } from '../utils/sessionUserIdUtils.js'
import {
  createGithubAppReturnKey,
  setGithubAppReturnTarget,
} from '../utils/githubAppReturnStorage'
import {
  GITHUB_CALLBACK_HINTS,
  GITLAB_CALLBACK_HINTS,
} from '../utils/gitSiteOAuthCallbackUtils.js'
import {
  buildProviderCatalogRequest,
  normalizeProviderCatalog,
} from '../utils/gitSiteOAuthProviderCatalog.js'
import {
  splitScope,
  normalizeConnectionStatus,
  isConnectionPayloadUsable,
} from '../utils/gitSiteOAuthConnectionUtils.js'
const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms))

/**
 * 构造 gitOauth 授权启动 URL：/api/git-oauth/{provider}-app-start/?{params}
 * （provider ∈ github|gitlab，对应后端 github-app-start/gitlab-app-start 处理器）。
 * 旧路径 /api/accounts/{provider}/app/start/ 网关无路由 → 502。
 * @param {string} provider
 * @param {URLSearchParams} [params]
 * @returns {string}
 */
export function buildProviderAppStartUrl(provider, params) {
  const qs = params ? params.toString() : ''
  return `/api/git-oauth/${provider}-app-start/${qs ? `?${qs}` : ''}`
}

export function useUserGitSiteOAuthSettings() {
  const route = useRoute()
  const router = useRouter()
  const providerOptions = ref([])
  const selectedProviderKey = ref('')
  const selectedProviderEntry = computed(
    () => providerOptions.value.find((it) => it.provider_key === selectedProviderKey.value) || null,
  )
  const selectedProvider = computed(() => String(selectedProviderEntry.value?.provider || '').trim())
  const selectedServiceProvider = computed(() =>
    String(selectedProviderEntry.value?.service_provider || 'default').trim(),
  )
  const selectedProviderLabel = computed(() => selectedProviderEntry.value?.label || 'Git')
  const sessionUserId = ref('')

  const apiConnectionUrl = computed(() => {
    const userId = sessionUserId.value
    const provider = selectedProvider.value
    const serviceProvider = selectedServiceProvider.value
    if (!provider || !userId || !serviceProvider) return ''
    const qs = new URLSearchParams({ service_provider: serviceProvider })
    return `/api/git-oauth/user-app-connection/?${qs.toString()}`
  })

  const tenantId = computed(() => String(route.params.tenant || ''))

  const profileUserId = computed(() => {
    const fromRoute = String(route.params.id || route.params.userId || '').trim()
    if (fromRoute) return fromRoute
    return sessionUserId.value
  })

  const isOwnProfile = computed(() => {
    const a = profileUserId.value
    const b = sessionUserId.value
    return Boolean(a && b && a === b)
  })

  const statusLoading = ref(true)
  const statusLoaded = ref(false)
  const connecting = ref(false)
  const disconnecting = ref(false)
  const status = ref(null)
  const errorMessage = ref('')
  const errorTraceId = ref('')
  const successMessage = ref('')

  const scopeTokensGranted = computed(() => splitScope(status.value?.scope))
  const scopeTokensRequested = computed(() => {
    const fromApi = splitScope(status.value?.authorize_scope)
    if (fromApi.length) return fromApi
    return splitScope('repo read:user')
  })
  const connectionList = computed(() => status.value?.connections || [])

  const clearInlineError = () => {
    errorMessage.value = ''
    errorTraceId.value = ''
  }

  const setInlineError = (message, source) => {
    errorMessage.value = message || ''
    errorTraceId.value = extractTraceId(source) || ''
  }

  const fetchProviderCatalog = async () => {
    try {
      const { path, headers } = buildProviderCatalogRequest(tenantId.value)
      const response = await apiFetch(path, { headers })
      const data = await response.json().catch(() => ({}))
      if (!response.ok) {
        const msg = typeof data.detail === 'string' ? data.detail : '无法加载 Git OAuth 站点列表'
        setInlineError(msg, extractTraceId(response) || extractTraceId(data) || '')
        providerOptions.value = []
        return
      }
      const normalized = normalizeProviderCatalog(data.providers)
      if (!normalized.length) {
        setInlineError('服务端未配置任何 Git OAuth 站点', '')
        providerOptions.value = []
        return
      }
      providerOptions.value = normalized
      if (!normalized.some((it) => it.provider_key === selectedProviderKey.value)) {
        selectedProviderKey.value = normalized[0].provider_key
      }
    } catch (e) {
      setInlineError(e?.message || '无法加载 Git OAuth 站点列表', e)
      providerOptions.value = []
    }
  }

  const applyGithubQueryMessage = async () => {
    const provider = selectedProvider.value
    const queryKey = provider
    const callbackHints = provider === 'gitlab' ? GITLAB_CALLBACK_HINTS : GITHUB_CALLBACK_HINTS
    const raw = route.query[queryKey]
    if (raw == null || raw === '') return ''
    const key = String(raw)
    const queryTraceId = String(route.query.trace_id || route.query.traceId || '').trim()
    if (key === 'ok') {
      successMessage.value = callbackHints.ok
      clearInlineError()
    } else if (callbackHints[key]) {
      setInlineError(callbackHints[key], queryTraceId)
    } else {
      setInlineError(`授权未完成（${key}）`, queryTraceId)
    }
    const nextQuery = { ...route.query }
    delete nextQuery[queryKey]
    delete nextQuery.trace_id
    delete nextQuery.traceId
    await router.replace({ path: route.path, query: nextQuery })
    return key
  }

  const fetchStatus = async () => {
    if (!isOwnProfile.value) return
    statusLoading.value = true
    statusLoaded.value = false
    clearInlineError()
    try {
      if (!apiConnectionUrl.value) {
        setInlineError('缺少 userId，无法获取绑定状态', '')
        status.value = null
        statusLoaded.value = false
        return
      }
      const response = await apiFetch(apiConnectionUrl.value, {
        headers: { Accept: 'application/json' },
      })
      const data = await response.json().catch(() => ({}))
      const responseTraceId = extractTraceId(response) || extractTraceId(data) || ''
      if (!response.ok) {
        if (isConnectionPayloadUsable(data)) {
          status.value = normalizeConnectionStatus(data)
          statusLoaded.value = true
          if (typeof data.detail === 'string' && data.detail.trim()) {
            setInlineError(data.detail, responseTraceId)
          }
          return
        }
        setInlineError(
          typeof data.detail === 'string' ? data.detail : '无法获取绑定状态',
          responseTraceId,
        )
        status.value = null
        statusLoaded.value = false
        return
      }
      status.value = normalizeConnectionStatus(data)
      statusLoaded.value = true
    } catch (e) {
      setInlineError(e?.message || '无法获取绑定状态', e)
      status.value = null
      statusLoaded.value = false
    } finally {
      statusLoading.value = false
    }
  }

  /** OAuth 回跳后 gitOauth 与主站摘要可能晚一拍；短暂重试避免误显「尚未绑定」 */
  const fetchStatusAfterGithubOk = async () => {
    const maxAttempts = 8
    const delayMs = 400
    for (let attempt = 0; attempt < maxAttempts; attempt++) {
      await fetchStatus()
      if (status.value?.connected) {
        return
      }
      if (attempt < maxAttempts - 1) {
        await sleep(delayMs)
      }
    }
    const okMessage = selectedProvider.value === 'gitlab' ? GITLAB_CALLBACK_HINTS.ok : GITHUB_CALLBACK_HINTS.ok
    if (successMessage.value === okMessage && !status.value?.connected) {
      successMessage.value = ''
      // 纯前端时序提示（非单次请求失败），不得伪造 data-traceId
      setInlineError(
        'GitHub 回调已成功，但读取绑定状态仍为未绑定。请点击「重新加载」或刷新页面；若仍如此，请核对主站与 gitOauth 是否指向同一环境。',
        '',
      )
    }
  }

  const startProviderAuthorize = async () => {
    if (!isOwnProfile.value) return
    connecting.value = true
    clearInlineError()
    successMessage.value = ''
    try {
      const nextPath = `${window.location.pathname}${window.location.search || ''}`
      const returnKey = createGithubAppReturnKey()
      setGithubAppReturnTarget(returnKey, nextPath)
      const startParams = new URLSearchParams({
        next: nextPath,
        return_key: returnKey,
        service_provider: selectedServiceProvider.value || 'default',
      })
      const startUrl = buildProviderAppStartUrl(selectedProvider.value, startParams)
      const response = await apiFetch(startUrl, {
        headers: { Accept: 'application/json' },
      })
      const data = await response.json().catch(() => ({}))
      if (!response.ok) {
        showRequestError(typeof data.detail === 'string' ? data.detail : '无法启动 OAuth 授权', data)
        return
      }
      if (data.authorize_url) {
        window.location.href = data.authorize_url
      }
    } catch (e) {
      showRequestError(e?.message || '启动 OAuth 授权失败', e)
    } finally {
      connecting.value = false
    }
  }

  const disconnectConnection = async (connection = null) => {
    if (!isOwnProfile.value) return
    const displayName =
      connection?.github_login ||
      (connection?.github_user_id ? `用户 #${connection.github_user_id}` : '当前账号')
    if (!window.confirm(`确定取消 ${displayName} 的 ${selectedProviderLabel.value} 授权？取消后需重新授权才能访问私有仓库。`)) {
      return
    }
    disconnecting.value = true
    clearInlineError()
    successMessage.value = ''
    try {
      const response = await apiFetch(apiConnectionUrl.value, {
        method: 'DELETE',
        headers: {
          Accept: 'application/json',
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          github_user_id: connection?.github_user_id || null,
        }),
      })
      const data = await response.json().catch(() => ({}))
      if (!response.ok) {
        setInlineError(
          typeof data.detail === 'string' ? data.detail : '取消授权失败',
          extractTraceId(response) || extractTraceId(data) || '',
        )
        return
      }
      successMessage.value = data.was_connected
        ? `已取消 ${displayName} 的 ${selectedProviderLabel.value} 授权`
        : `${displayName} 当前未绑定`
      await fetchStatus()
    } catch (e) {
      setInlineError(e?.message || '取消授权失败', e)
    } finally {
      disconnecting.value = false
    }
  }

  const switchProvider = async (providerKey) => {
    const next = String(providerKey || '').trim().toLowerCase()
    if (!next || next === selectedProviderKey.value) return
    selectedProviderKey.value = next
    await fetchStatus()
  }

  const pickProviderKeyFromRoute = () => {
    const fromQuery = String(route.query.provider || '').trim().toLowerCase()
    const options = providerOptions.value
    if (!options.length) return
    const fromServiceProvider = String(route.query.service_provider || '').trim().toLowerCase()
    if (fromServiceProvider) {
      const matched = options.find(
        (it) =>
          it.service_provider === fromServiceProvider &&
          (!fromQuery || it.provider === fromQuery),
      )
      if (matched) {
        selectedProviderKey.value = matched.provider_key
        return
      }
    }
    if (fromQuery && options.some((p) => p.provider === fromQuery)) {
      const first = options.find((p) => p.provider === fromQuery)
      if (first) selectedProviderKey.value = first.provider_key
      return
    }
    if (route.query.gitlab != null) {
      const firstGitlab = options.find((p) => p.provider === 'gitlab')
      if (firstGitlab) selectedProviderKey.value = firstGitlab.provider_key
    } else if (route.query.github != null) {
      const firstGithub = options.find((p) => p.provider === 'github')
      if (firstGithub) selectedProviderKey.value = firstGithub.provider_key
    }
  }

  onMounted(async () => {
    sessionUserId.value = await resolveAuthenticatedUserId()
    await fetchProviderCatalog()
    pickProviderKeyFromRoute()
    const githubKey = await applyGithubQueryMessage()
    if (!isOwnProfile.value) {
      statusLoading.value = false
      return
    }
    if (githubKey === 'ok') {
      await fetchStatusAfterGithubOk()
    } else {
      const callbackError = errorMessage.value
      const callbackTraceId = errorTraceId.value
      await fetchStatus()
      if (callbackError) {
        errorMessage.value = callbackError
        errorTraceId.value = callbackTraceId
      }
    }
  })

  return {
    tenantId,
    isOwnProfile,
    providerOptions,
    selectedProviderKey,
    selectedProviderEntry,
    selectedProviderLabel,
    statusLoading,
    statusLoaded,
    status,
    connecting,
    disconnecting,
    errorMessage,
    errorTraceId,
    successMessage,
    scopeTokensGranted,
    scopeTokensRequested,
    connectionList,
    fetchStatus,
    startProviderAuthorize,
    disconnectConnection,
    switchProvider,
  }
}

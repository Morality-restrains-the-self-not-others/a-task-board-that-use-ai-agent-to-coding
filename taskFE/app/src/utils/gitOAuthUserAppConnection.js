// OPT-20260819-001: 共享 Git OAuth user-app 绑定查询。
// 创建任务门禁（useCreateTaskRepoOAuth）与评论侧（useLinkedProjectsRepoOAuth）
// 复用同一超时/解析逻辑，避免绑定判定分叉。
import { apiFetch } from './apiUtils.js'
import { extractTraceId } from './traceId.js'
import { isGitOAuthUserAppConnectedPayload } from './createTaskOauthGate.js'

const OAUTH_CONNECTION_REQUEST_TIMEOUT_MS = 6000

/**
 * @param {string} repoUrl
 * @param {{ baseUrl?: string, probeAccessToken?: boolean }} [options]
 * @returns {string}
 */
export function buildGitOAuthUserAppConnectionUrl(repoUrl, options = {}) {
  const base = String(options.baseUrl || '/api/git-oauth/user-app-connection/')
  const params = new URLSearchParams()
  const trimmed = String(repoUrl || '').trim()
  if (trimmed) params.set('repo_url', trimmed)
  if (options.probeAccessToken) params.set('probe_access_token', '1')
  const query = params.toString()
  return query ? `${base}?${query}` : base
}

export function classifyGitOAuthProbePayload(data) {
  const payload = data && typeof data === 'object' ? data : {}
  const networkStatus = String(payload.network_status || '').trim()
  if (networkStatus === 'unreachable') {
    return { networkUnreachable: true, tokenValid: false }
  }
  if (Object.prototype.hasOwnProperty.call(payload, 'access_token_valid')) {
    return { networkUnreachable: false, tokenValid: payload.access_token_valid === true }
  }
  return { networkUnreachable: false, tokenValid: isGitOAuthUserAppConnectedPayload(payload) }
}

/**
 * 查询当前用户对指定 repo 的 Git OAuth user-app 连接状态。
 *
 * @param {string} repoUrl 仓库 URL（可为空字符串，查询全部连接）
 * @param {object} [options]
 * @param {string} [options.baseUrl='/api/git-oauth/user-app-connection/']
 * @param {number} [options.timeoutMs=6000]
 * @param {boolean} [options.probeAccessToken=false] 为 true 时实时校验 AccessToken
 * @returns {Promise<boolean>} connected；probe 时为 access_token_valid
 * @throws {Error} 失败 Error 携带 .traceId 供 Loki 关联
 */
export async function fetchGitOAuthUserAppConnectionData(repoUrl, options = {}) {
  const {
    baseUrl = '/api/git-oauth/user-app-connection/',
    timeoutMs = OAUTH_CONNECTION_REQUEST_TIMEOUT_MS,
    probeAccessToken = false,
  } = options

  const controller = new AbortController()
  const timeoutId = setTimeout(
    () => controller.abort(),
    Math.max(1000, Number(timeoutMs) || OAUTH_CONNECTION_REQUEST_TIMEOUT_MS),
  )
  let response
  try {
    const apiUrl = buildGitOAuthUserAppConnectionUrl(repoUrl, { baseUrl, probeAccessToken })
    response = await apiFetch(apiUrl, {
      credentials: 'include',
      headers: { Accept: 'application/json' },
      signal: controller.signal,
      skipSessionExpiredRedirect: true,
    })
  } catch (error) {
    if (error?.name === 'AbortError') {
      const err = new Error('检查 OAuth 绑定状态超时，请确认服务可用后重试')
      err.traceId = extractTraceId(error)
      throw err
    }
    throw error
  } finally {
    clearTimeout(timeoutId)
  }

  const data = await response.json().catch(() => ({}))
  if (!response.ok) {
    if (response.status === 401) return { status: 401, data }
    const detail = typeof data?.detail === 'string' ? data.detail : '无法检查 OAuth 绑定状态'
    const err = new Error(detail)
    err.traceId = extractTraceId(response) || extractTraceId(data)
    throw err
  }
  return { status: response.status, data }
}

/**
 * 查询当前用户对指定 repo 的 Git OAuth user-app 连接状态。
 *
 * @param {string} repoUrl 仓库 URL（可为空字符串，查询全部连接）
 * @param {object} [options]
 * @returns {Promise<boolean>} connected；probe 时为 access_token_valid
 */
export async function fetchGitOAuthUserAppConnected(repoUrl, options = {}) {
  const { probeAccessToken = false } = options
  const { status, data } = await fetchGitOAuthUserAppConnectionData(repoUrl, options)
  if (status === 401) return false
  if (probeAccessToken) {
    if (String(data?.network_status || '').trim() === 'unreachable') {
      return isGitOAuthUserAppConnectedPayload(data)
    }
    if (Object.prototype.hasOwnProperty.call(data, 'access_token_valid')) {
      return data.access_token_valid === true
    }
    return isGitOAuthUserAppConnectedPayload(data)
  }
  return isGitOAuthUserAppConnectedPayload(data)
}

/**
 * AccessToken 探测：区分 token 失效与 GitLab 网络不可达。
 *
 * @returns {Promise<{ networkUnreachable: boolean, tokenValid: boolean }>}
 */
export async function fetchGitOAuthUserAppProbeOutcome(repoUrl, options = {}) {
  const { status, data } = await fetchGitOAuthUserAppConnectionData(repoUrl, {
    ...options,
    probeAccessToken: true,
  })
  if (status === 401) return { networkUnreachable: false, tokenValid: false }
  return classifyGitOAuthProbePayload(data)
}

import { extractTraceId } from './traceId.js'
import { messageFromFailedResponse } from './httpError.js'
import { INVALID_REPO_URL_MSG } from './gitRepoUrlUtils.js'
import { gitCloneRefMatchKey } from './taskDetailContainerCloneProgress.js'

export const REPO_INACCESSIBLE_MSG = '该仓库无法访问，后续需要授权后才能访问'
export const VALIDATE_GIT_REPO_NETWORK_MSG = '校验失败，请检查网络后重试'
export const VALIDATE_GIT_REPO_TIMEOUT_MSG = '仓库校验超时，请稍后重试'
export const VALIDATE_GIT_REPO_MISSING_RESULT_MSG = '未返回该仓库的校验结果，请重试'
/** probe_access 远端探测可达 25s，前端须留余量避免误报超时 */
export const VALIDATE_GIT_REPOS_TIMEOUT_MS = 60_000

export function indexValidateResultsByUrl(results) {
  const byUrl = {}
  if (!Array.isArray(results)) return byUrl
  for (const entry of results) {
    const u = String(entry?.url || entry?.repo_url || '').trim()
    if (u) byUrl[u] = entry
  }
  return byUrl
}

/**
 * Exact URL first, then host+path without .git (same key as clone progress).
 * @param {Record<string, unknown>} byUrl
 * @param {string} url
 * @returns {unknown|null}
 */
export function lookupValidateResultByUrl(byUrl, url) {
  const exact = String(url || '').trim()
  if (!exact || !byUrl || typeof byUrl !== 'object') return null
  if (byUrl[exact]) return byUrl[exact]
  const want = gitCloneRefMatchKey(exact)
  if (!want) return null
  for (const [key, entry] of Object.entries(byUrl)) {
    if (gitCloneRefMatchKey(key) === want) return entry
  }
  return null
}

/**
 * 批量校验请求失败（传输 / HTTP / 200 但缺行）→ 用户文案 + traceId。
 * 禁止把 HTTP 业务错误一律说成「请检查网络」。
 */
export function resolveValidateGitReposRequestFailure({ response, data, err } = {}) {
  if (err) {
    const isTimeout = err?.name === 'TimeoutError'
    return {
      message: isTimeout ? VALIDATE_GIT_REPO_TIMEOUT_MSG : VALIDATE_GIT_REPO_NETWORK_MSG,
      traceId: extractTraceId(err),
    }
  }
  if (response && response.ok === false) {
    const fakeResp = {
      status: response.status,
      _errorData: data && typeof data === 'object' ? data : response._errorData,
    }
    return {
      message: messageFromFailedResponse(fakeResp, VALIDATE_GIT_REPO_NETWORK_MSG),
      traceId: extractTraceId(response) || extractTraceId(data),
    }
  }
  return {
    message: VALIDATE_GIT_REPO_MISSING_RESULT_MSG,
    traceId: extractTraceId(response) || extractTraceId(data),
  }
}

export function isGitRepoValidateRetryable(message) {
  const m = String(message || '').trim()
  if (!m) return false
  if (m === INVALID_REPO_URL_MSG) return false
  if (m === REPO_INACCESSIBLE_MSG) return false
  return true
}

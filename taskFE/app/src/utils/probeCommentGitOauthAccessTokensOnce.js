import { commentOauthUrlsNeedingAccessTokenProbe } from './commentExecutionGitOauth.js'
import { fetchGitOAuthUserAppProbeOutcome } from './gitOAuthUserAppConnection.js'
import { extractTraceId } from './traceId.js'

/**
 * 对「看起来已绑定」的评论仓库各触发一次 AccessToken 探测。同一 URL 不会重试（含失败）。
 * 单仓探测抛错（超时等）记入 checkFailedRepoUrls，不向上抛，避免打开页被错误弹窗挡住。
 *
 * @param {object} args
 * @param {string[]} args.urls
 * @param {object} [args.readiness]
 * @param {Set<string>} args.probedUrls
 * @param {boolean} [args.force] 用户显式「重试」：绕过「仅已绑定才探测」门禁，强制探测这些 URL（OPT-20260902-011）
 * @param {(repoUrl: string, options: { probeAccessToken: boolean }) => Promise<boolean>} [args.fetchConnected]
 * @param {(repoUrl: string) => Promise<{ networkUnreachable: boolean, tokenValid: boolean }>} [args.fetchOutcome]
 * @returns {Promise<{
 *   probed: string[],
 *   invalidRepoUrls: string[],
 *   unreachableRepoUrls: string[],
 *   checkFailedRepoUrls: string[],
 *   probeError: string,
 *   probeTraceId: string,
 * }>}
 */
export async function probeCommentGitOauthAccessTokensOnce({
  urls,
  readiness,
  probedUrls,
  force = false,
  fetchConnected,
  fetchOutcome,
}) {
  const need = force
    ? (Array.isArray(urls) ? urls : []).map((url) => String(url || '').trim()).filter(Boolean)
    : commentOauthUrlsNeedingAccessTokenProbe(urls, readiness)
  const toProbe = need.filter((url) => !probedUrls.has(url))
  const invalidRepoUrls = []
  const unreachableRepoUrls = []
  const checkFailedRepoUrls = []
  let probeError = ''
  let probeTraceId = ''
  const resolveOutcome = async (url) => {
    if (typeof fetchOutcome === 'function') return fetchOutcome(url)
    if (typeof fetchConnected === 'function') {
      const valid = await fetchConnected(url, { probeAccessToken: true })
      return { networkUnreachable: false, tokenValid: Boolean(valid) }
    }
    return fetchGitOAuthUserAppProbeOutcome(url)
  }
  for (const url of toProbe) {
    probedUrls.add(url)
    try {
      const outcome = await resolveOutcome(url)
      if (outcome?.networkUnreachable) unreachableRepoUrls.push(url)
      else if (!outcome?.tokenValid) invalidRepoUrls.push(url)
    } catch (error) {
      checkFailedRepoUrls.push(url)
      if (!probeError) {
        probeError = error?.message || '无法确认 Git OAuth AccessToken 是否有效'
        probeTraceId = extractTraceId(error)
      }
    }
  }
  return {
    probed: toProbe,
    invalidRepoUrls,
    unreachableRepoUrls,
    checkFailedRepoUrls,
    probeError,
    probeTraceId,
  }
}

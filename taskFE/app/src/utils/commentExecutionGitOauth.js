/**
 * 评论执行细节摘要：把评论关联仓库解析为 Git OAuth 绑定芯片。
 * 摘要行本身不发请求；评论区显示且 DB 显示已绑定时，由评论区触发一次 AccessToken probe。
 */
import {
  buildRepoOAuthStartHref,
  resolveRepoOAuthAuthorizeLabel,
  shouldShowRepoOAuthAuthorizeButton,
  supportsRepoOAuthAuthorize,
} from './repoOAuthAuthorizeUtils.js'
import { isGitPushPermissionDenied } from './layerZtreePushError.js'

/**
 * @param {unknown} rows
 * @returns {string[]}
 */
function repoUrlsFromIdentities(rows) {
  if (!Array.isArray(rows)) return []
  const out = []
  const seen = new Set()
  for (const row of rows) {
    if (!row || typeof row !== 'object') continue
    const url = String(row.repo_url || '').trim()
    if (!url || seen.has(url)) continue
    seen.add(url)
    out.push(url)
  }
  return out
}

/**
 * @param {unknown} repoIdentities
 * @param {unknown} [fallbackRepoIdentities]
 * @returns {string[]}
 */
export function collectCommentOauthRepoUrls(repoIdentities, fallbackRepoIdentities = []) {
  const own = repoUrlsFromIdentities(repoIdentities).filter((url) => supportsRepoOAuthAuthorize(url))
  if (own.length > 0) return own
  return repoUrlsFromIdentities(fallbackRepoIdentities).filter((url) => supportsRepoOAuthAuthorize(url))
}

/**
 * @param {string[]} repoUrls
 * @param {{ loading?: boolean, allBound?: boolean, hasOAuthRepos?: boolean, unboundRepoUrls?: unknown }} [readiness]
 * @param {unknown} [pushErrorDetail] 评论层快照 last_push_error；已绑定时若为仓库写权限拒绝则 overlay 无写权限
 * @returns {{
 *   kind: 'loading' | 'bound' | 'bound_no_write' | 'unbound' | 'unreachable' | 'check_failed',
 *   text: string,
 *   title: string,
 *   bindLabel: string,
 *   unboundRepoUrls: string[],
 *   traceId?: string,
 * } | null}
 */
export function commentGitOauthSummary(repoUrls, readiness, pushErrorDetail = '') {
  const urls = (Array.isArray(repoUrls) ? repoUrls : [])
    .map((url) => String(url || '').trim())
    .filter((url) => url && supportsRepoOAuthAuthorize(url))
  if (urls.length === 0) return null

  if (!readiness || readiness.loading !== false) {
    return {
      kind: 'loading',
      text: 'Git OAuth · 检查中',
      title: urls.join('\n'),
      bindLabel: '',
      unboundRepoUrls: [],
    }
  }

  const reportedUnbound = new Set(
    (Array.isArray(readiness.unboundRepoUrls) ? readiness.unboundRepoUrls : [])
      .map((url) => String(url || '').trim())
      .filter(Boolean),
  )
  const reportedUnreachable = new Set(
    (Array.isArray(readiness.unreachableRepoUrls) ? readiness.unreachableRepoUrls : [])
      .map((url) => String(url || '').trim())
      .filter(Boolean),
  )
  let unbound = urls.filter((url) => reportedUnbound.has(url))
  if (unbound.length === 0 && readiness.allBound !== true && readiness.hasOAuthRepos !== true) {
    unbound = urls.slice()
  }

  if (unbound.length === 0) {
    const reportedCheckFailed = new Set(
      (Array.isArray(readiness.checkFailedRepoUrls) ? readiness.checkFailedRepoUrls : [])
        .map((url) => String(url || '').trim())
        .filter(Boolean),
    )
    const checkFailed = urls.filter((url) => reportedCheckFailed.has(url))
    if (checkFailed.length > 0) {
      const probeError = String(readiness.probeError || '').trim()
      const timeout = probeError.includes('超时')
      const host = checkFailed.length === 1 ? resolveRepoOAuthAuthorizeLabel(checkFailed[0]) : ''
      const baseTitle = host ? `${host}\n${checkFailed[0]}` : checkFailed.join('\n')
      return {
        kind: 'check_failed',
        text: timeout ? 'Git OAuth · 检查超时' : 'Git OAuth · 检查失败',
        title: probeError ? `${baseTitle}\n\n${probeError}` : baseTitle,
        bindLabel: '',
        unboundRepoUrls: [],
        traceId: String(readiness.probeTraceId || '').trim(),
      }
    }
    const unreachable = urls.filter((url) => reportedUnreachable.has(url))
    if (unreachable.length > 0) {
      const host = unreachable.length === 1 ? resolveRepoOAuthAuthorizeLabel(unreachable[0]) : ''
      const baseTitle = host ? `${host}\n${unreachable[0]}` : unreachable.join('\n')
      return {
        kind: 'unreachable',
        text: 'Git OAuth · 网络不可达',
        title: baseTitle,
        bindLabel: '',
        unboundRepoUrls: [],
      }
    }
    const host = urls.length === 1 ? resolveRepoOAuthAuthorizeLabel(urls[0]) : ''
    const baseTitle = host ? `${host}\n${urls[0]}` : urls.join('\n')
    if (isGitPushPermissionDenied(pushErrorDetail)) {
      const err = String(pushErrorDetail || '').trim()
      return {
        kind: 'bound_no_write',
        text: 'Git OAuth · 无写权限',
        title: err ? `${baseTitle}\n\n${err}` : baseTitle,
        bindLabel: host ? `换账号授权 ${host}` : '换账号授权',
        unboundRepoUrls: urls.slice(),
      }
    }
    return {
      kind: 'bound',
      text: 'Git OAuth · 已绑定',
      title: baseTitle,
      bindLabel: '',
      unboundRepoUrls: [],
    }
  }

  const first = unbound[0]
  const host = resolveRepoOAuthAuthorizeLabel(first)
  return {
    kind: 'unbound',
    text: unbound.length > 1 ? `Git OAuth · 未绑定 ${unbound.length}` : 'Git OAuth · 未绑定',
    title: unbound.join('\n'),
    bindLabel: `去绑定 ${host}`,
    unboundRepoUrls: unbound,
  }
}

/**
 * @param {string} repoUrl
 * @param {{ nextPath?: string }} [opts]
 * @returns {string}
 */
export function commentGitOauthBindHref(repoUrl, opts = {}) {
  return buildRepoOAuthStartHref(repoUrl, opts)
}

/**
 * @param {unknown} displayComments
 * @param {unknown} [fallbackRepoIdentities]
 * @returns {string[]}
 */
export function collectDisplayedCommentOauthRepoUrls(displayComments, fallbackRepoIdentities = []) {
  const comments = Array.isArray(displayComments) ? displayComments : []
  if (comments.length === 0) return []
  const seen = new Set()
  const out = []
  for (const comment of comments) {
    const urls = collectCommentOauthRepoUrls(comment?.repo_identities, fallbackRepoIdentities)
    for (const url of urls) {
      if (seen.has(url)) continue
      seen.add(url)
      out.push(url)
    }
  }
  return out
}

/**
 * 仅当任务级检查已结束且摘要会显示「已绑定」时，才需要对这些 URL 做 AccessToken 探测。
 *
 * @param {string[]} repoUrls
 * @param {{ loading?: boolean, allBound?: boolean, hasOAuthRepos?: boolean, unboundRepoUrls?: unknown }} [readiness]
 * @returns {string[]}
 */
export function commentOauthUrlsNeedingAccessTokenProbe(repoUrls, readiness) {
  const summary = commentGitOauthSummary(repoUrls, readiness)
  if (!summary || summary.kind !== 'bound') return []
  return (Array.isArray(repoUrls) ? repoUrls : [])
    .map((url) => String(url || '').trim())
    .filter((url) => url && supportsRepoOAuthAuthorize(url))
}

/**
 * @param {{
 *   loading?: boolean,
 *   allBound?: boolean,
 *   hasOAuthRepos?: boolean,
 *   unboundRepoUrls?: unknown,
 *   startBlocked?: boolean,
 * }} [readiness]
 * @param {{
 *   probing?: boolean,
 *   invalidRepoUrls?: unknown,
 *   unreachableRepoUrls?: unknown,
 *   checkFailedRepoUrls?: unknown,
 *   probeError?: unknown,
 *   probeTraceId?: unknown,
 * }} [probe]
 */
export function mergeReadinessAfterAccessTokenProbe(readiness, probe = {}) {
  const base = readiness && typeof readiness === 'object' ? readiness : {}
  const invalid = (Array.isArray(probe.invalidRepoUrls) ? probe.invalidRepoUrls : [])
    .map((url) => String(url || '').trim())
    .filter(Boolean)
  const unreachableReported = (Array.isArray(probe.unreachableRepoUrls) ? probe.unreachableRepoUrls : [])
    .map((url) => String(url || '').trim())
    .filter(Boolean)
  const unbound = []
  const unreachable = []
  const seenUnbound = new Set()
  const seenUnreachable = new Set()
  const addUnbound = (url) => {
    const key = String(url || '').trim()
    if (!key || seenUnbound.has(key)) return
    seenUnbound.add(key)
    unbound.push(key)
  }
  const addUnreachable = (url) => {
    const key = String(url || '').trim()
    if (!key || seenUnreachable.has(key) || seenUnbound.has(key)) return
    seenUnreachable.add(key)
    unreachable.push(key)
  }
  const reported = Array.isArray(base.unboundRepoUrls) ? base.unboundRepoUrls : []
  for (const url of reported) addUnbound(url)
  for (const url of invalid) addUnbound(url)
  const reportedUnreachable = Array.isArray(base.unreachableRepoUrls) ? base.unreachableRepoUrls : []
  for (const url of reportedUnreachable) addUnreachable(url)
  for (const url of unreachableReported) addUnreachable(url)
  const checkFailed = []
  const seenCheckFailed = new Set()
  const addCheckFailed = (url) => {
    const key = String(url || '').trim()
    if (!key || seenCheckFailed.has(key) || seenUnbound.has(key)) return
    seenCheckFailed.add(key)
    checkFailed.push(key)
  }
  const reportedCheckFailed = Array.isArray(probe.checkFailedRepoUrls) ? probe.checkFailedRepoUrls : []
  for (const url of reportedCheckFailed) addCheckFailed(url)
  const probing = Boolean(probe.probing)
  const allBound = unbound.length === 0 && base.allBound === true
  return {
    ...base,
    loading: probing ? true : base.loading === true,
    unboundRepoUrls: unbound,
    unreachableRepoUrls: unreachable,
    checkFailedRepoUrls: checkFailed,
    probeError: String(probe.probeError || '').trim(),
    probeTraceId: String(probe.probeTraceId || '').trim(),
    allBound,
    startBlocked: probing
      ? true
      : (Boolean(base.startBlocked) || unbound.length > 0 || checkFailed.length > 0),
  }
}

/**
 * 多评论共用同一用户级 Git OAuth AccessToken：任务级检查已绑定时，
 * PR 回复不得再展示「去绑定 / 授权已失效」。
 *
 * @param {{ loading?: boolean, allBound?: boolean, unboundRepoUrls?: unknown }} [readiness]
 * @returns {boolean}
 */
export function sharedUserGitOAuthIsBound(readiness) {
  if (!readiness || readiness.loading !== false) return false
  const unbound = (Array.isArray(readiness.unboundRepoUrls) ? readiness.unboundRepoUrls : [])
    .map((url) => String(url || '').trim())
    .filter(Boolean)
  return readiness.allBound === true && unbound.length === 0
}

/**
 * @param {unknown} displayError
 * @param {{ loading?: boolean, allBound?: boolean, unboundRepoUrls?: unknown }} [oauthReadiness]
 * @returns {boolean}
 */
export function shouldShowGitPrOauthBind(displayError, oauthReadiness) {
  if (!shouldShowRepoOAuthAuthorizeButton(displayError)) return false
  if (sharedUserGitOAuthIsBound(oauthReadiness)) return false
  return true
}

/**
 * @param {unknown} displayError
 * @param {{ loading?: boolean, allBound?: boolean, unboundRepoUrls?: unknown }} [oauthReadiness]
 * @returns {string}
 */
export function gitPrOauthStatusErrorText(displayError, oauthReadiness) {
  const text = String(displayError || '').trim()
  if (!text) return ''
  if (sharedUserGitOAuthIsBound(oauthReadiness) && shouldShowRepoOAuthAuthorizeButton(text)) {
    return ''
  }
  return text
}

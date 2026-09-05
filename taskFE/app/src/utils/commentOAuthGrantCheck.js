import { supportsRepoOAuthAuthorize } from './repoOAuthAuthorizeUtils.js'
import { gitsiteFromRepoUrl, hasSessionGrantForRepo } from './grantTicketSession.js'
import { resolveContainerUiContextCommentId } from '../composables/taskDetail/resolveContainerUiContextCommentId.js'

function collectOAuthRepoUrls(repoUrls) {
  const urls = []
  const seen = new Set()
  for (const raw of Array.isArray(repoUrls) ? repoUrls : []) {
    const url = String(raw || '').trim()
    if (!url || seen.has(url)) continue
    if (!supportsRepoOAuthAuthorize(url)) continue
    seen.add(url)
    urls.push(url)
  }
  return urls
}

function unwrapMaybeRef(v) {
  if (v == null) return v
  if (typeof v === 'object' && 'value' in v) return v.value
  return v
}

export function commentOAuthGrantMissingMessage(gitsite, actionLabel) {
  const site = String(gitsite || 'git').trim() || 'git'
  const action = String(actionLabel || '').trim() || '操作'
  return `${action}前，请先完成该任务评论的 Git OAuth 使用授权（gitsite=${site}）`
}

export function sessionGrantUnboundRepoUrls(repoUrls) {
  return collectOAuthRepoUrls(repoUrls).filter((url) => !hasSessionGrantForRepo(url))
}

export function mergeSessionGrantIntoReadiness(readiness = {}, repoUrls = []) {
  const next = {
    hasOAuthRepos: Boolean(readiness.hasOAuthRepos),
    loading: Boolean(readiness.loading),
    allBound: Boolean(readiness.allBound),
    checkError: String(readiness.checkError || '').trim(),
    unboundRepoUrls: Array.isArray(readiness.unboundRepoUrls) ? [...readiness.unboundRepoUrls] : [],
    startBlocked: Boolean(readiness.startBlocked),
  }
  if (!next.hasOAuthRepos || next.loading || next.checkError) return next
  const missing = sessionGrantUnboundRepoUrls(repoUrls)
  if (missing.length === 0) return next
  const seen = new Set(next.unboundRepoUrls.map((url) => String(url || '').trim()).filter(Boolean))
  for (const url of missing) {
    if (seen.has(url)) continue
    seen.add(url)
    next.unboundRepoUrls.push(url)
  }
  next.allBound = false
  next.startBlocked = true
  return next
}

export function commentHasOAuthGrantForSite(comment, gitsite) {
  const site = String(gitsite || '').trim().toLowerCase()
  if (!site) return false
  const idents = Array.isArray(comment?.repo_identities) ? comment.repo_identities : []
  return idents.some((row) => String(row?.oauth_gitsite || '').trim().toLowerCase() === site)
}

export function findCommentForOAuthGrantCheck(comments, commentId) {
  const list = Array.isArray(comments) ? comments : []
  const cid = String(commentId || '').trim()
  if (!cid) return null
  return list.find((row) => String(row?.id || '').trim() === cid) || null
}

export function missingCommentOAuthGrantSite(comment, repoUrls) {
  const sites = []
  const seen = new Set()
  for (const url of collectOAuthRepoUrls(repoUrls)) {
    const site = gitsiteFromRepoUrl(url)
    if (!site || seen.has(site)) continue
    seen.add(site)
    if (!commentHasOAuthGrantForSite(comment, site)) sites.push(site)
  }
  return sites[0] || ''
}

export function anyCommentHasOAuthGrantForSite(comments, gitsite) {
  const list = Array.isArray(comments) ? comments : []
  return list.some((row) => commentHasOAuthGrantForSite(row, gitsite))
}

/**
 * 云端开发提交/推送前：运行评论须已打 L2。缺标记时不得先 git commit。
 * @returns {string} empty when ready
 */
export function commentOAuthGrantMissingReason(deps, repoUrls, actionLabel) {
  const urls = collectOAuthRepoUrls(repoUrls)
  if (urls.length === 0) return ''
  const comments = unwrapMaybeRef(deps?.displayComments)
    || unwrapMaybeRef(deps?.comments)
    || unwrapMaybeRef(unwrapMaybeRef(deps?.localTask)?.comments)
    || []
  const commentId = resolveContainerUiContextCommentId(deps)
  const comment = findCommentForOAuthGrantCheck(comments, commentId)
  if (comment) {
    const site = missingCommentOAuthGrantSite(comment, urls)
    if (site) return commentOAuthGrantMissingMessage(site, actionLabel)
    return ''
  }
  for (const url of urls) {
    const site = gitsiteFromRepoUrl(url)
    if (!site) continue
    if (!anyCommentHasOAuthGrantForSite(comments, site)) {
      return commentOAuthGrantMissingMessage(site, actionLabel)
    }
  }
  return ''
}

import { fetchGitOAuthUserAppConnected } from './gitOAuthUserAppConnection.js'
import { supportsRepoOAuthAuthorize } from './repoOAuthAuthorizeUtils.js'
import { collectLinkedRepoUrls } from './commentRepoIdentity.js'
import { showRequestError } from './requestErrorDisplay.js'
import { sessionGrantUnboundRepoUrls } from './commentOAuthGrantCheck.js'

/**
 * @param {string} [actionLabel]
 * @returns {string}
 */
export function gitOauthUnboundActionMessage(actionLabel) {
  const action = String(actionLabel || '').trim() || '操作'
  return `${action}前，请先完成 GitHub/GitLab OAuth 授权，否则无法推送到 HTTPS 远端`
}

/**
 * @param {unknown[]} repoUrls
 * @returns {string[]}
 */
export function collectOAuthRepoUrls(repoUrls) {
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

/**
 * @param {{ url?: string }[]} rows
 * @returns {string[]}
 */
export function repoUrlsFromTaskRepoRows(rows) {
  return (Array.isArray(rows) ? rows : [])
    .map((row) => String(row?.url || '').trim())
    .filter(Boolean)
}

/**
 * @param {unknown[]} repoUrls
 * @param {object} [options]
 * @returns {Promise<{ ok: boolean, unboundRepoUrls: string[] }>}
 */
export async function assertGitOauthBoundForRepoUrls(repoUrls, options = {}) {
  const urls = collectOAuthRepoUrls(repoUrls)
  if (urls.length === 0) return { ok: true, unboundRepoUrls: [] }
  const unboundRepoUrls = []
  for (const url of urls) {
    const connected = await fetchGitOAuthUserAppConnected(url, options)
    if (!connected) unboundRepoUrls.push(url)
  }
  return { ok: unboundRepoUrls.length === 0, unboundRepoUrls }
}

/**
 * @param {unknown[]} repoUrls
 * @param {{ allowBare?: boolean, actionLabel?: string, timeoutMs?: number }} [options]
 * @returns {Promise<string>} empty when ready
 */
export async function gitOauthUnboundReasonForRepoUrls(repoUrls, options = {}) {
  if (options.allowBare === true) return ''
  const result = await assertGitOauthBoundForRepoUrls(repoUrls, options)
  if (result.ok) return ''
  return gitOauthUnboundActionMessage(options.actionLabel || '推送')
}

export function gitOauthCommentGrantMissingMessage(actionLabel) {
  const action = String(actionLabel || '').trim() || '操作'
  return `${action}前，请先完成该任务评论的 Git OAuth 使用授权，否则无法推送到 HTTPS 远端`
}

/**
 * $镜像 提交并运行：未绑定则 showRequestError 并返回 true（调用方不得 POST 评论）。
 * L1 connected 不够：ADR-0049 要求本条评论的 grant_ticket / L2。
 * @param {unknown} taskProjectsWithDetails
 * @returns {Promise<boolean>}
 */
export async function blockCommentRunIfGitOauthUnbound(taskProjectsWithDetails) {
  try {
    const repoUrls = collectLinkedRepoUrls(taskProjectsWithDetails)
    const reason = await gitOauthUnboundReasonForRepoUrls(
      repoUrls,
      { actionLabel: '提交并运行' },
    )
    if (reason) {
      showRequestError(reason)
      return true
    }
    if (sessionGrantUnboundRepoUrls(repoUrls).length > 0) {
      showRequestError(gitOauthCommentGrantMissingMessage('提交并运行'))
      return true
    }
    return false
  } catch (err) {
    showRequestError(err?.message || '无法检查 Git OAuth 绑定', err)
    return true
  }
}

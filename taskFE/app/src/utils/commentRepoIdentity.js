/**
 * Collect linked git repo URLs and validate comment-run identity draft.
 */
import { mergeSessionGrantIntoReadiness } from './commentOAuthGrantCheck.js'

export function collectLinkedRepoUrls(taskProjectsWithDetails) {
  const rows = Array.isArray(taskProjectsWithDetails) ? taskProjectsWithDetails : []
  const urls = []
  const seen = new Set()
  rows.forEach((tp) => {
    const repos = Array.isArray(tp?.project?.git_repos) ? tp.project.git_repos : []
    repos.forEach((raw) => {
      const url = String(raw || '').trim()
      if (!url || seen.has(url)) return
      seen.add(url)
      urls.push(url)
    })
  })
  return urls
}

/**
 * Map a linked repo URL back to its parent project for project-level clone config.
 * auto_clone_nested_repos 缺省 true，与 project_entries / CreateProject 默认一致。
 * @returns {{ projectId: string, autoCloneNestedRepos: boolean } | null}
 */
export function resolveLinkedRepoProject(taskProjectsWithDetails, repoUrl) {
  const url = String(repoUrl || '').trim()
  if (!url) return null
  const rows = Array.isArray(taskProjectsWithDetails) ? taskProjectsWithDetails : []
  for (const tp of rows) {
    const repos = Array.isArray(tp?.project?.git_repos) ? tp.project.git_repos : []
    const hit = repos.some((raw) => String(raw || '').trim() === url)
    if (!hit) continue
    const projectId = String(tp?.project_id || tp?.id || '').trim()
    if (!projectId) return null
    return {
      projectId,
      autoCloneNestedRepos: tp?.project?.auto_clone_nested_repos !== false,
    }
  }
  return null
}

/**
 * 任务关联项目只要有一个关闭自动克隆子仓，评论进度条就不得把 nested 行计入分母。
 * @param {object[]} taskProjectsWithDetails
 * @returns {boolean}
 */
export function taskProjectsAllowNestedClone(taskProjectsWithDetails) {
  const rows = Array.isArray(taskProjectsWithDetails) ? taskProjectsWithDetails : []
  if (!rows.length) return true
  return rows.every((tp) => {
    const project = tp?.project && typeof tp.project === 'object' ? tp.project : tp
    return project?.auto_clone_nested_repos !== false
  })
}

export function isGithubRepoUrl(repoUrl) {
  return /github\.com/i.test(String(repoUrl || ''))
}

/**
 * @param {{ hasOAuthRepos?: boolean, loading?: boolean, allBound?: boolean, checkError?: string }} [readiness]
 * @param {unknown[]} [repoUrls]
 * @returns {string} empty if send is allowed
 */
export function resolveCommentRunOauthBlockedReason(readiness = {}, repoUrls = []) {
  const merged = mergeSessionGrantIntoReadiness(readiness, repoUrls)
  if (!merged.hasOAuthRepos) return ''
  if (merged.loading) return '正在检查仓库 Git OAuth 绑定…'
  const checkError = String(merged.checkError || '').trim()
  if (checkError) return `无法检查 Git OAuth 绑定：${checkError}`
  if (!merged.allBound) {
    return '提交并运行前，请先完成该任务评论的 Git OAuth 使用授权，否则无法推送到 HTTPS 远端'
  }
  return ''
}

/**
 * @param {{ hasOAuthRepos?: boolean, loading?: boolean, allBound?: boolean }} [readiness]
 * @param {unknown[]} [repoUrls]
 * @returns {{ kind: 'none' | 'loading' | 'bound' | 'unbound', text: string }}
 */
export function commentComposerGitOauthHint(readiness, repoUrls = []) {
  const merged = mergeSessionGrantIntoReadiness(readiness, repoUrls)
  if (!merged?.hasOAuthRepos) {
    return { kind: 'none', text: '' }
  }
  if (merged.loading) {
    return { kind: 'loading', text: '正在检查是否已绑定 Git OAuth…' }
  }
  if (merged.allBound) {
    return { kind: 'bound', text: '已绑定 Git OAuth，私有仓库可使用你的授权克隆与推送。' }
  }
  return {
    kind: 'unbound',
    text: '尚未完成该任务评论的 Git OAuth 使用授权。请先完成授权后再提交并运行；账号中心已连接时仍需再授权一次（通常秒过）。',
  }
}

/**
 * Advisory only: submitComment must not use this as a send blocker.
 * @param {{ repo_url: string, git_identity_id?: string, github_user_id?: string }[]} selections
 * @param {string[]} requiredRepoUrls
 * @returns {string} empty if ok
 */
export function validateCommentRepoIdentities(selections, requiredRepoUrls) {
  const required = Array.isArray(requiredRepoUrls) ? requiredRepoUrls.map((u) => String(u || '').trim()).filter(Boolean) : []
  if (required.length === 0) return ''
  const list = Array.isArray(selections) ? selections : []
  const byUrl = new Map()
  list.forEach((row) => {
    const url = String(row?.repo_url || '').trim()
    if (url) byUrl.set(url, row)
  })
  for (const url of required) {
    const row = byUrl.get(url)
    if (!row || !String(row.git_identity_id || '').trim()) {
      return `请为仓库选择 Git 提交身份：${url}`
    }
    if (isGithubRepoUrl(url) && !String(row.github_user_id || '').trim()) {
      return `请为 GitHub 仓库选择授权账号：${url}`
    }
  }
  return ''
}

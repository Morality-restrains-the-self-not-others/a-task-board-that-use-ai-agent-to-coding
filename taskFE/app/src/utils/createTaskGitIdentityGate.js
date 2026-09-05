import { collectLinkedRepoUrls } from './commentRepoIdentity.js'
import { collectProjectGitRepoUrls } from './gitRepoUrlUtils.js'
import { gitsiteFromRepoUrl } from './grantTicketSession.js'

/**
 * 创建/编辑任务：勾选自动运行时须为每个关联仓选择 Git 提交身份。
 * 未勾选自动运行不校验、不拦截。
 */

/**
 * @param {boolean|undefined|null} autoRun
 * @returns {boolean}
 */
export function shouldRequireAutoRunGitIdentityGate(autoRun) {
  return autoRun === true
}

/**
 * Fork 确认弹窗：任务已关联仓库的全部 git URL（含 generic git）。
 * @param {unknown[]} taskProjectsWithDetails
 * @returns {string[]}
 */
export function collectForkTaskRepoUrls(taskProjectsWithDetails) {
  return collectLinkedRepoUrls(taskProjectsWithDetails)
}

/**
 * Fork「自动运行并派生」Git 身份门禁。加载中/失败/未选齐均拦截该按钮。
 * @param {{
 *   requiredRepoUrls?: string[],
 *   selections?: unknown,
 *   loading?: boolean,
 *   loadError?: string,
 * }} [readiness]
 * @returns {string}
 */
export function resolveForkAutoRunGitIdentityBlockedReason(readiness = {}) {
  if (readiness.loading) return '正在加载 Git 提交身份…'
  const loadError = String(readiness.loadError || '').trim()
  if (loadError) return `无法加载 Git 提交身份：${loadError}`
  return validateCreateTaskGitIdentities(true, readiness.selections, readiness.requiredRepoUrls)
}

/**
 * 当前所选项目的全部 git 仓库 URL（不去掉 generic git）。
 * @param {object|null} editingTask
 * @param {unknown[]} projects
 * @returns {string[]}
 */
export function collectCreateTaskRepoUrls(editingTask, projects) {
  const selections = Array.isArray(editingTask?.projectSelections) ? editingTask.projectSelections : []
  const projectList = Array.isArray(projects) ? projects : []
  const urls = []
  const seen = new Set()
  for (const selection of selections) {
    const projectId = String(selection?.projectId || '').trim()
    if (!projectId) continue
    const project = projectList.find((row) => String(row?.id || '') === projectId)
    for (const url of collectProjectGitRepoUrls(project)) {
      if (seen.has(url)) continue
      seen.add(url)
      urls.push(url)
    }
  }
  return urls
}

/**
 * @param {unknown} raw
 * @returns {{ repo_url: string, git_identity_id: string }[]}
 */
export function normalizeCreateTaskRepoIdentities(raw) {
  if (!Array.isArray(raw)) return []
  const out = []
  const seen = new Set()
  for (const row of raw) {
    const repoUrl = String(row?.repo_url || '').trim()
    const gitIdentityId = String(row?.git_identity_id || '').trim()
    if (!repoUrl || seen.has(repoUrl)) continue
    seen.add(repoUrl)
    const oauthGitsite = String(row?.oauth_gitsite || '').trim().toLowerCase()
    const item = { repo_url: repoUrl, git_identity_id: gitIdentityId }
    if (oauthGitsite) item.oauth_gitsite = oauthGitsite
    const remote = String(row?.oauth_remote_user_id || '').trim()
    if (remote) item.oauth_remote_user_id = remote
    out.push(item)
  }
  return out
}

/**
 * OAuth 能力站点判定：github.com / *.github.com，或 host 含 gitlab。
 * 与后端 taskTaskService isOAuthCapableGitsite（comment_oauth_grant.go）契约孪生——
 * 两侧漂移时须同步修改（OPT-20260902-028）。
 * @param {string} site
 * @returns {boolean}
 */
export function isOAuthCapableGitsite(site) {
  const s = String(site || '').trim().toLowerCase()
  if (!s) return false
  if (s === 'github.com' || s.endsWith('.github.com')) return true
  return s.includes('gitlab')
}

/**
 * After the auto-run / 提交并运行 OAuth gate passed, record the repo host as
 * comment L2 so layer-oauth and cloud push do not treat the comment as unbound.
 * 只给 OAuth 能力站点（github/gitlab）盖章；generic git（如 git.example）不写
 * oauth_gitsite，避免 credential FetchAccessToken 打未知 host。
 * @param {{ repo_url?: string, git_identity_id?: string, oauth_gitsite?: string, oauth_remote_user_id?: string }[]} rows
 * @returns {typeof rows}
 */
export function stampRepoIdentityOauthGitsite(rows) {
  if (!Array.isArray(rows)) return []
  return rows.map((row) => {
    const repoUrl = String(row?.repo_url || '').trim()
    const existing = String(row?.oauth_gitsite || '').trim().toLowerCase()
    const site = existing || gitsiteFromRepoUrl(repoUrl)
    if (!site) return { ...row }
    if (!existing && !isOAuthCapableGitsite(site)) return { ...row }
    const next = { ...row, repo_url: repoUrl, oauth_gitsite: site }
    return next
  })
}

/**
 * @param {boolean|undefined|null} autoRun
 * @param {unknown} selections
 * @param {string[]} requiredRepoUrls
 * @returns {string} empty if ok
 */
export function validateCreateTaskGitIdentities(autoRun, selections, requiredRepoUrls) {
  if (!shouldRequireAutoRunGitIdentityGate(autoRun)) return ''
  const required = Array.isArray(requiredRepoUrls)
    ? requiredRepoUrls.map((u) => String(u || '').trim()).filter(Boolean)
    : []
  if (required.length === 0) return ''
  const byUrl = new Map()
  normalizeCreateTaskRepoIdentities(selections).forEach((row) => {
    byUrl.set(row.repo_url, row)
  })
  for (const url of required) {
    const row = byUrl.get(url)
    if (!row || !row.git_identity_id) {
      return `请为仓库选择 Git 提交身份：${url}`
    }
  }
  return ''
}

/**
 * @param {boolean|undefined|null} autoRun
 * @param {unknown} selections
 * @returns {{ repo_url: string, git_identity_id: string }[] | undefined}
 */
export function buildCreateTaskRepoIdentitiesPayload(autoRun, selections) {
  if (!shouldRequireAutoRunGitIdentityGate(autoRun)) return undefined
  const rows = normalizeCreateTaskRepoIdentities(selections).filter((row) => row.git_identity_id)
  return stampRepoIdentityOauthGitsite(rows)
}

/**
 * @param {{ git_user_name?: string, name?: string, git_user_email?: string, email?: string, id?: string }} identity
 * @returns {string}
 */
export function formatGitIdentityOptionLabel(identity) {
  const name = String(identity?.git_user_name || identity?.name || '').trim()
  const email = String(identity?.git_user_email || identity?.email || '').trim()
  if (name && email) return `${name} <${email}>`
  return name || email || String(identity?.id || '')
}

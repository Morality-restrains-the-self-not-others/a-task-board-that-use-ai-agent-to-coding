import { extractTraceId } from './traceId.js'
import { collectProjectGitRepoUrls } from './gitRepoUrlUtils.js'
import { gitCloneRefMatchKey } from './taskDetailContainerCloneProgress.js'
import {
  indexValidateResultsByUrl,
  lookupValidateResultByUrl,
  resolveValidateGitReposRequestFailure,
  VALIDATE_GIT_REPOS_TIMEOUT_MS,
} from './gitRepoValidateError.js'
import { supportsRepoOAuthAuthorize } from './repoOAuthAuthorizeUtils.js'

export const CREATE_TASK_OAUTH_MISSING_TENANT_MSG = '缺少 tenantId，无法检查项目 Git OAuth 授权'
export const CREATE_TASK_OAUTH_MISSING_PROJECT_MSG = '未选择项目，无法对照项目详情的 Git OAuth 授权'

/**
 * 自动运行会在创建/派生后立即引导克隆，须事先完成 OAuth。
 * @param {boolean|undefined|null} autoRun
 * @returns {boolean}
 */
export function shouldRequireAutoRunOauthGate(autoRun) {
  return autoRun === true
}

function pushOAuthRepoRow(rows, seen, url, projectId) {
  const trimmed = String(url || '').trim()
  if (!trimmed || seen.has(trimmed)) return
  if (!supportsRepoOAuthAuthorize(trimmed)) return
  seen.add(trimmed)
  rows.push({ url: trimmed, projectId: String(projectId || '').trim() })
}

/**
 * GitHub/GitLab URLs + owning project from task linked projects (fork / task detail).
 * @param {unknown[]} taskProjectsWithDetails
 * @returns {{ url: string, projectId: string }[]}
 */
export function collectForkTaskOAuthRepoRows(taskProjectsWithDetails) {
  const source = Array.isArray(taskProjectsWithDetails) ? taskProjectsWithDetails : []
  const rows = []
  const seen = new Set()
  for (const row of source) {
    const projectId = String(row?.project?.id || row?.project_id || '').trim()
    for (const url of collectProjectGitRepoUrls(row?.project)) {
      pushOAuthRepoRow(rows, seen, url, projectId)
    }
  }
  return rows
}

/**
 * GitHub/GitLab URLs from task linked projects (fork / task detail).
 * @param {unknown[]} taskProjectsWithDetails
 * @returns {string[]}
 */
export function collectForkTaskOAuthRepoUrls(taskProjectsWithDetails) {
  return collectForkTaskOAuthRepoRows(taskProjectsWithDetails).map((row) => row.url)
}

/**
 * GitHub/GitLab URLs + owning project from currently selected create-task projects.
 * @param {object|null} editingTask
 * @param {unknown[]} projects
 * @returns {{ url: string, projectId: string }[]}
 */
export function collectCreateTaskOAuthRepoRows(editingTask, projects) {
  const selections = Array.isArray(editingTask?.projectSelections) ? editingTask.projectSelections : []
  const projectList = Array.isArray(projects) ? projects : []
  const rows = []
  const seen = new Set()
  for (const selection of selections) {
    const projectId = String(selection?.projectId || '').trim()
    if (!projectId) continue
    const project = projectList.find((row) => String(row?.id || '') === projectId)
    for (const url of collectProjectGitRepoUrls(project)) {
      pushOAuthRepoRow(rows, seen, url, projectId)
    }
  }
  return rows
}

/**
 * GitHub/GitLab URLs from currently selected create-task projects.
 * @param {object|null} editingTask
 * @param {unknown[]} projects
 * @returns {string[]}
 */
export function collectCreateTaskOAuthRepoUrls(editingTask, projects) {
  return collectCreateTaskOAuthRepoRows(editingTask, projects).map((row) => row.url)
}

/**
 * @param {unknown} data
 * @returns {boolean}
 */
export function isGitOAuthUserAppConnectedPayload(data) {
  if (!data || typeof data !== 'object') return false
  if (data.connected === true) return true
  if (Array.isArray(data.connections)) {
    return data.connections.some((item) => Boolean(item && item.connected))
  }
  return false
}

/** Project L2 + L1 exchange — not L1 `connected` alone. */
export function isCreateTaskOAuthTokenAvailable(tokenStatus) {
  return String(tokenStatus || '').trim() === 'token_available'
}

function rowProjectTokenStatus(project, url) {
  if (!project || typeof project !== 'object') return ''
  const want = gitCloneRefMatchKey(String(url || '').trim())
  if (!want) return ''
  const rows = Array.isArray(project.git_repos_status) ? project.git_repos_status : []
  for (const row of rows) {
    const repoUrl = String(row?.repo_url || row?.url || '').trim()
    if (!repoUrl) continue
    if (gitCloneRefMatchKey(repoUrl) === want) return String(row?.token_status || '').trim()
  }
  return ''
}

/**
 * 列表/详情缓存已带真实 token_available 时，URL 视为 bound（跳过 validate-git-repos POST）。
 * 逐行匹配 repo_url（gitCloneRefMatchKey 归一化，忽略 .git / 尾斜杠差异）。
 * @param {{ url: string, projectId: string }[]} rows
 * @param {unknown[]} projects
 * @returns {Record<string, boolean>}
 */
export function buildCreateTaskProjectListBoundMap(rows, projects) {
  const byId = new Map()
  for (const project of Array.isArray(projects) ? projects : []) {
    if (project && project.id != null) byId.set(String(project.id), project)
  }
  const bound = {}
  for (const row of Array.isArray(rows) ? rows : []) {
    const url = String(row?.url || '').trim()
    const project = byId.get(String(row?.projectId || '').trim())
    if (!url || !project) continue
    if (rowProjectTokenStatus(project, url) === 'token_available') bound[url] = true
  }
  return bound
}

/**
 * Batch-check project L2 via validate-git-repos (probe_access false).
 * @param {{
 *   apiFetch: Function,
 *   tenantId: string,
 *   rows: { url: string, projectId: string }[],
 *   unboundUrls: string[],
 * }} opts
 * @returns {Promise<{
 *   bound: Record<string, boolean>,
 *   errorByUrl: Record<string, string>,
 *   errorTraceIdByUrl: Record<string, string>,
 * }>}
 */
export async function fetchCreateTaskProjectL2BoundByUrl({
  apiFetch,
  tenantId,
  rows,
  unboundUrls,
} = {}) {
  const bound = {}
  const errorByUrl = {}
  const errorTraceIdByUrl = {}
  const tid = String(tenantId || '').trim()
  const unbound = new Set((Array.isArray(unboundUrls) ? unboundUrls : []).map((u) => String(u || '').trim()).filter(Boolean))
  if (unbound.size === 0 || typeof apiFetch !== 'function') {
    return { bound, errorByUrl, errorTraceIdByUrl }
  }
  if (!tid) {
    for (const url of unbound) {
      bound[url] = false
      errorByUrl[url] = CREATE_TASK_OAUTH_MISSING_TENANT_MSG
    }
    return { bound, errorByUrl, errorTraceIdByUrl }
  }
  const byProject = new Map()
  const assigned = new Set()
  for (const row of Array.isArray(rows) ? rows : []) {
    const url = String(row?.url || '').trim()
    const projectId = String(row?.projectId || '').trim()
    if (!url || !projectId || !unbound.has(url)) continue
    const list = byProject.get(projectId) || []
    list.push(url)
    byProject.set(projectId, list)
    assigned.add(url)
  }
  for (const url of unbound) {
    if (assigned.has(url)) continue
    bound[url] = false
    errorByUrl[url] = CREATE_TASK_OAUTH_MISSING_PROJECT_MSG
  }
  for (const [projectId, urls] of byProject.entries()) {
    let response
    let data = {}
    try {
      response = await apiFetch(
        `/api/projects/validate-git-repos/tenant_id/${encodeURIComponent(tid)}/`,
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
          credentials: 'include',
          body: JSON.stringify({
            urls,
            probe_access: false,
            project_id: projectId,
          }),
          timeout: VALIDATE_GIT_REPOS_TIMEOUT_MS,
        },
      )
      data = await response.json().catch(() => ({}))
    } catch (err) {
      const fail = resolveValidateGitReposRequestFailure({ err })
      for (const url of urls) {
        errorByUrl[url] = fail.message
        errorTraceIdByUrl[url] = fail.traceId || extractTraceId(err) || ''
        bound[url] = false
      }
      continue
    }
    const byUrl = indexValidateResultsByUrl(response?.ok ? data?.results : null)
    const requestFail = resolveValidateGitReposRequestFailure({ response, data })
    for (const url of urls) {
      const entry = lookupValidateResultByUrl(byUrl, url)
      if (entry) {
        bound[url] = isCreateTaskOAuthTokenAvailable(entry.token_status)
        errorByUrl[url] = ''
        errorTraceIdByUrl[url] = ''
        continue
      }
      bound[url] = false
      errorByUrl[url] = requestFail.message
      errorTraceIdByUrl[url] = requestFail.traceId || extractTraceId(response) || extractTraceId(data) || ''
    }
  }
  return { bound, errorByUrl, errorTraceIdByUrl }
}

/**
 * @param {{
 *   hasOAuthRepos?: boolean,
 *   loading?: boolean,
 *   allBound?: boolean,
 *   checkError?: string,
 * }} [readiness]
 * @returns {string}
 */
export function resolveCreateTaskOauthBlockedReason(readiness = {}) {
  if (!readiness.hasOAuthRepos) return ''
  if (readiness.loading) return '正在检查仓库 Git OAuth 绑定…'
  const checkError = String(readiness.checkError || '').trim()
  if (checkError) return `无法检查 Git OAuth 绑定：${checkError}`
  if (!readiness.allBound) {
    return '开启自动运行前，请为未授权的仓库完成 Git OAuth（可在项目详情授权，或点下方绑定）'
  }
  return ''
}

/**
 * Fork 派生并自动运行时的 OAuth 门禁文案（与创建任务语义一致）。
 * @param {Parameters<typeof resolveCreateTaskOauthBlockedReason>[0]} [readiness]
 * @returns {string}
 */
export function resolveForkAutoRunOauthBlockedReason(readiness = {}) {
  if (!readiness.hasOAuthRepos) return ''
  if (readiness.loading) return '正在检查仓库 Git OAuth 绑定…'
  const checkError = String(readiness.checkError || '').trim()
  if (checkError) return `无法检查 Git OAuth 绑定：${checkError}`
  if (!readiness.allBound) {
    return '自动运行并派生前，请为未授权的仓库完成 Git OAuth（可在项目详情授权，或点下方绑定）'
  }
  return ''
}

/** 与后端 ``repo_match_key._GIT_SCP_RE`` / ``gitCloneRefMatchKey`` 对齐 */
const GIT_SCP_RE = /^git@([^:]+):(.+?)(?:\.git)?\/?$/i

export const INVALID_REPO_URL_MSG = '请输入有效的 Git 仓库 URL'

export const GIT_REPO_URL_EXAMPLES = [
  'https://github.com/owner/repo.git',
  'git@gitlab.com:group/project.git',
  'ssh://git@gitlab.daydaymoney.com:2222/group/project.git',
]

export const GIT_REPO_URL_FORMAT_HINT =
  `支持 https://、git@主机:路径 或 ssh:// 格式，例如 ${GIT_REPO_URL_EXAMPLES.join('、')}`

/**
 * git_repos 可能是字符串，也可能是 {url, clone_alias} / {repo_url}。
 * @param {unknown} raw
 * @returns {string}
 */
export function gitRepoUrlFromUnknown(raw) {
  if (raw == null) return ''
  if (typeof raw === 'string' || typeof raw === 'number') {
    return String(raw).trim()
  }
  if (typeof raw === 'object') {
    return String(raw.url || raw.repo_url || raw.git_repo || '').trim()
  }
  return ''
}

/**
 * 项目详情与创建任务共用：优先 git_repo_entries，否则 git_repos。
 * @param {unknown} project
 * @returns {string[]}
 */
export function collectProjectGitRepoUrls(project) {
  const out = []
  const seen = new Set()
  const push = (raw) => {
    const url = gitRepoUrlFromUnknown(raw)
    if (!url || seen.has(url)) return
    seen.add(url)
    out.push(url)
  }
  const entries = Array.isArray(project?.git_repo_entries) ? project.git_repo_entries : []
  if (entries.length > 0) {
    for (const entry of entries) push(entry)
    return out
  }
  const repos = Array.isArray(project?.git_repos) ? project.git_repos : []
  for (const raw of repos) push(raw)
  return out
}

function hasGitRepoPath(pathname) {
  const path = String(pathname || '').replace(/\/+$/, '')
  return Boolean(path && path !== '/')
}

/**
 * @param {string} url
 * @returns {boolean} 空字符串视为合法（可选字段）
 */
export function isValidGitRepoUrl(url) {
  const trimmed = String(url || '').trim()
  if (!trimmed) return true
  if (GIT_SCP_RE.test(trimmed)) return true
  try {
    const parsed = new URL(trimmed)
    if (!parsed.hostname) return false
    if (parsed.protocol === 'ssh:') return hasGitRepoPath(parsed.pathname)
    return (parsed.protocol === 'http:' || parsed.protocol === 'https:')
  } catch {
    return false
  }
}

/** 多行仓库输入：任一非空 URL 非法则返回格式错误文案，否则空串。 */
export function gitRepoRowsFormatError(rows) {
  for (const row of rows || []) {
    const url = String(row?.url || '').trim()
    if (url && !isValidGitRepoUrl(url)) return INVALID_REPO_URL_MSG
  }
  return ''
}

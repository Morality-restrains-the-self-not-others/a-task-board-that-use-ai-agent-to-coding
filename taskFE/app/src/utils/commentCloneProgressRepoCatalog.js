/**
 * 评论级克隆进度行补全 git URL（冷打开 binding 日志只有仓库名、没有 URL）。
 * 无 Vue 依赖。
 */
import {
  gitCloneRefMatchKey,
  parseBootstrapCloneLogSections,
  shortCloneRepoLabel,
} from './taskDetailContainerCloneProgress.js'
import {
  extractGitRepoRefFromText,
  looksLikeGitRepoRef,
  resolveCommentCloneProgress,
  applyBootstrapCloneDoneToRows,
} from './commentCloneProgressFromLogs.js'

function basenameOfRepo(urlOrLabel) {
  const s = String(urlOrLabel || '').trim().toLowerCase()
  if (!s) return ''
  const noGit = s.replace(/\.git$/i, '')
  const parts = noGit.split(/[/:]/).filter(Boolean)
  return parts.length ? parts[parts.length - 1] : ''
}

function catalogMatchKeys(entry) {
  const url = String(entry?.repoUrl || '').trim()
  const keys = new Set()
  if (!url) return keys
  const matchKey = gitCloneRefMatchKey(url)
  if (matchKey) keys.add(matchKey)
  const short = shortCloneRepoLabel(url).toLowerCase()
  if (short) keys.add(short)
  const base = basenameOfRepo(url)
  if (base) keys.add(base)
  const alias = String(entry?.cloneAlias || '').trim().toLowerCase()
  if (alias) keys.add(alias)
  return keys
}

function rowMatchNeedles(row) {
  const needles = new Set()
  const label = String(row?.label || '').trim().toLowerCase()
  const key = String(row?.key || '').trim().toLowerCase()
  if (label && label !== '项目克隆' && label !== '__global__') needles.add(label)
  if (key && key !== '__global__' && key !== '项目克隆') needles.add(key)
  const base = basenameOfRepo(label || key)
  if (base) needles.add(base)
  return needles
}

function addCatalogEntry(out, seen, repoUrl, extra = {}) {
  const u = String(repoUrl || '').trim()
  if (!looksLikeGitRepoRef(u)) return
  const k = gitCloneRefMatchKey(u)
  if (!k) return
  if (seen.has(k)) {
    const existing = out.find((e) => gitCloneRefMatchKey(e.repoUrl) === k)
    if (!existing) return
    if (!existing.parentRepoUrl && extra.parentRepoUrl) {
      existing.parentRepoUrl = extra.parentRepoUrl
    }
    if (!existing.cloneAlias && extra.cloneAlias) {
      existing.cloneAlias = extra.cloneAlias
    }
    return
  }
  seen.add(k)
  out.push({
    repoUrl: u,
    parentRepoUrl: String(extra.parentRepoUrl || '').trim(),
    cloneAlias: String(extra.cloneAlias || '').trim(),
  })
}

function walkProjectLike(out, seen, value) {
  if (!value || typeof value !== 'object') return
  const project = value.project && typeof value.project === 'object' ? value.project : value
  const entries = Array.isArray(project.git_repo_entries) ? project.git_repo_entries : []
  for (const e of entries) {
    if (!e || typeof e !== 'object') continue
    addCatalogEntry(out, seen, e.url || e.repo_url, {
      parentRepoUrl: e.parent_repo_url || e.parentRepoUrl || '',
      cloneAlias: e.clone_alias || e.cloneAlias || '',
    })
  }
  const repos = Array.isArray(project.git_repos) ? project.git_repos : []
  for (const u of repos) addCatalogEntry(out, seen, u)
}

/**
 * @param {{
 *   projects?: object[],
 *   taskRepoRows?: Array<{ url?: string, repoUrl?: string }>,
 *   extraEntries?: Array<{ repoUrl?: string, key?: string, parentRepoUrl?: string, cloneAlias?: string }>,
 *   bootstrapLog?: string,
 *   statusLogs?: string[],
 * }} [sources]
 * @returns {Array<{ repoUrl: string, parentRepoUrl: string, cloneAlias: string }>}
 */
export function collectCloneRepoCatalog(sources = {}) {
  const out = []
  const seen = new Set()
  for (const row of Array.isArray(sources.taskRepoRows) ? sources.taskRepoRows : []) {
    addCatalogEntry(out, seen, row?.url || row?.repoUrl)
  }
  for (const row of Array.isArray(sources.projects) ? sources.projects : []) {
    walkProjectLike(out, seen, row)
  }
  for (const e of Array.isArray(sources.extraEntries) ? sources.extraEntries : []) {
    addCatalogEntry(out, seen, e?.repoUrl || e?.key, {
      parentRepoUrl: e?.parentRepoUrl || e?.parent_repo_url || '',
      cloneAlias: e?.cloneAlias || e?.clone_alias || '',
    })
  }
  if (typeof sources.bootstrapLog === 'string' && sources.bootstrapLog.trim()) {
    const { sections } = parseBootstrapCloneLogSections(sources.bootstrapLog)
    for (const s of sections) addCatalogEntry(out, seen, s.url)
  }
  for (const line of Array.isArray(sources.statusLogs) ? sources.statusLogs : []) {
    const fromLine = extractGitRepoRefFromText(line)
    if (fromLine) addCatalogEntry(out, seen, fromLine)
  }
  return out
}

function matchCatalogEntry(row, catalog) {
  if (!Array.isArray(catalog) || !catalog.length) return null
  const needles = rowMatchNeedles(row)
  const hits = catalog.filter((entry) => {
    const keys = catalogMatchKeys(entry)
    for (const n of needles) {
      if (keys.has(n)) return true
    }
    return false
  })
  if (hits.length === 1) return hits[0]
  return null
}

/**
 * 失败行若只有仓库名，用任务关联仓库目录补 git URL，以便显示「手动重试」。
 * @param {object[]} rows
 * @param {Array<{ repoUrl: string, parentRepoUrl?: string, cloneAlias?: string }>} catalog
 * @returns {object[]}
 */
export function enrichCloneProgressRowsWithCatalog(rows, catalog) {
  const list = Array.isArray(rows) ? rows : []
  const cat = Array.isArray(catalog) ? catalog : []
  const uniqueFallback =
    list.length === 1
    && cat.length === 1
    && Number(list[0]?.repoTotal || 0) <= 1
    ? cat[0]
    : null
  return list.map((row) => {
    const existingUrl = looksLikeGitRepoRef(row?.repoUrl) ? row.repoUrl : extractGitRepoRefFromText(row?.message)
    const hit = existingUrl
      ? (cat.find((e) => gitCloneRefMatchKey(e.repoUrl) === gitCloneRefMatchKey(existingUrl)) || {
        repoUrl: existingUrl,
        parentRepoUrl: row?.parentRepoUrl || '',
        cloneAlias: row?.cloneAlias || '',
      })
      : (matchCatalogEntry(row, cat) || uniqueFallback)
    if (!hit?.repoUrl) return row
    return {
      ...row,
      repoUrl: row.repoUrl || hit.repoUrl,
      key: looksLikeGitRepoRef(row.key) ? row.key : (hit.repoUrl || row.key),
      parentRepoUrl: row.parentRepoUrl || hit.parentRepoUrl || '',
      cloneAlias: row.cloneAlias || hit.cloneAlias || '',
    }
  })
}

/**
 * @param {Parameters<typeof resolveCommentCloneProgress>[0]} opts
 * @param {Parameters<typeof collectCloneRepoCatalog>[0]} catalogSources
 */
export function resolveCommentCloneProgressWithCatalog(opts = {}, catalogSources = {}) {
  const rows = resolveCommentCloneProgress(opts)
  const logs = Array.isArray(opts?.statusLogs) ? opts.statusLogs : []
  const catalog = collectCloneRepoCatalog({
    ...catalogSources,
    statusLogs: Array.isArray(catalogSources.statusLogs) ? catalogSources.statusLogs : logs,
    extraEntries: [
      ...(Array.isArray(catalogSources.extraEntries) ? catalogSources.extraEntries : []),
      ...(Array.isArray(opts?.taskLevelEntries) ? opts.taskLevelEntries : []),
    ],
  })
  return applyBootstrapCloneDoneToRows(
    enrichCloneProgressRowsWithCatalog(rows, catalog),
    catalogSources.bootstrapLog || '',
  )
}

/**
 * 关闭 auto_clone_nested_repos 时，带 parentRepoUrl 的子仓不得进入总分母
 *（否则 1 个父仓 100% + 33 个空子仓 ≈ 3%）。
 * @param {object[]} rows
 * @param {{ autoCloneNestedRepos?: boolean }} [opts]
 */
export function omitSkippedNestedCloneProgressRows(rows, opts = {}) {
  const list = Array.isArray(rows) ? rows : []
  if (opts.autoCloneNestedRepos !== false) return list
  const parents = list.filter((row) => !String(row?.parentRepoUrl || '').trim())
  return parents.length ? parents : list
}

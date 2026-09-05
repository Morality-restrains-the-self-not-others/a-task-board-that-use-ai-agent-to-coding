/**
 * 任务详情编辑：分支命名预设、仓库克隆身份映射等纯逻辑（无 Vue 依赖）。
 */

/**
 * @typedef {Object} GitIdentityOption
 * @property {string|number} id - 身份唯一标识
 * @property {string} [display_name] - API 遗留展示名；有 name/email/label 时不再优先使用
 * @property {string} [label] - 身份标签（展示时排在用户名、邮箱之后）
 * @property {string} [git_user_name] - Git user.name 字段
 * @property {string} [git_name] - 历史字段，与 git_user_name 同义（旧版本未统一前使用）
 * @property {string} [name] - 更早期的字段，与 git_user_name 同义
 * @property {string} [git_user_email] - Git user.email 字段
 * @property {string} [git_email] - 历史字段，与 git_user_email 同义（旧版本未统一前使用）
 * @property {string} [email] - 更早期的字段，与 git_user_email 同义
 * @property {boolean} [is_default] - 是否为公司默认身份
 */

export function extractCompanyNickname(profileData, tenantIdValue) {
  const profileItems = Array.isArray(profileData?.company_nicknames) ? profileData.company_nicknames : []
  const matchedItem = profileItems.find((item) => String(item.company_id) === String(tenantIdValue))
  if (matchedItem?.member_name) {
    return String(matchedItem.member_name).trim()
  }
  return ''
}

export function repoCloneFieldId(url) {
  return String(url || '')
    .replace(/[^a-zA-Z0-9]+/g, '-')
    .slice(0, 80)
}

export function sanitizeBranchSegment(rawValue, fallbackValue = 'unknown') {
  const normalizedValue = String(rawValue || '').trim().replace(/\s+/g, '_')
  const sanitizedValue = normalizedValue.replace(/[^a-zA-Z0-9._-]/g, '_')
  return sanitizedValue || fallbackValue
}

/**
 * 解析工作分支名中的 ${taskId} / ${taskTitle}。
 * 兼容历史 bug：空标题时把 '${taskTitle}' sanitize 成了 '__taskTitle_'。
 *
 * @param {string} branchName
 * @param {{ taskId?: string, taskTitleSegment?: string }} [opts]
 */
export function resolveBranchNamePlaceholders(branchName, opts = {}) {
  let name = String(branchName || '').trim()
  if (!name) return name
  const tid = String(opts.taskId || '').trim()
  const titleSeg = sanitizeBranchSegment(opts.taskTitleSegment ?? '', 'task')
  if (tid) {
    name = name.replace(/\$\{taskId\}/g, tid)
  }
  name = name.replace(/\$\{taskTitle\}/g, titleSeg)
  name = name.replace(/__taskTitle_/g, titleSeg)
  return name
}

export function hasChineseCharacter(rawValue) {
  return /[\u3400-\u9fff]/.test(String(rawValue || ''))
}

export function formatDateYmd(date = new Date()) {
  const pad = (value) => String(value).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`
}

/**
 * @param {string} presetType
 * @param {string} [taskTitleSegment]
 * @param {string} [taskId]
 * @param {{ companyUserName: string, localTaskTitle: string, customWorkBranchFallback: string, taskCreatedAt?: string|Date }} ctx
 */
export function buildWorkBranchName(presetType, taskTitleSegment = '', taskId = '', ctx) {
  const createdAt = ctx.taskCreatedAt ? new Date(ctx.taskCreatedAt) : new Date()
  const taskCreatedDateYmd = formatDateYmd(
    Number.isNaN(createdAt.getTime()) ? new Date() : createdAt
  )
  const userName = sanitizeBranchSegment(ctx.companyUserName, 'company_user_name')
  const normalizedTaskId = taskId ? String(taskId) : '${taskId}'
  const fallbackTitle = sanitizeBranchSegment(ctx.localTaskTitle || '', 'task')
  const taskTitle = taskTitleSegment || fallbackTitle

  if (presetType === 'release') {
    return `release/${taskCreatedDateYmd}_daydaymoney${normalizedTaskId}`
  }
  if (presetType === 'bugfix' || presetType === 'hotfix' || presetType === 'feature') {
    return `${presetType}/${taskCreatedDateYmd}_${userName}_daydaymoney${normalizedTaskId}_${taskTitle}`
  }
  return ctx.customWorkBranchFallback || ''
}

/**
 * @param {string} presetType
 * @param {string} [taskId]
 * @param {{ customMergeTargetFallback: string }} ctx
 */
export function buildMergeTargetBranchName(presetType, taskId = '', ctx) {
  const today = formatDateYmd()
  const normalizedTaskId = taskId ? String(taskId) : '${taskId}'
  if (presetType === 'develop' || presetType === 'main') {
    return presetType
  }
  if (presetType === 'release') {
    return `release/${today}_daydaymoney${normalizedTaskId}`
  }
  return ctx.customMergeTargetFallback || ''
}

/** 从仓库 URL 解析 GitHub owner/repo（小写），非 GitHub 返回空字符串 */
export function githubRepoSlugFromUrl(repoUrl) {
  const raw = String(repoUrl || '').trim()
  if (!raw) return ''
  const match = raw.match(/github\.com[:/]([\w.-]+)\/([\w.-]+?)(?:\.git)?\/?$/i)
  if (!match) return ''
  const owner = String(match[1] || '').trim().toLowerCase()
  const repo = String(match[2] || '').trim().replace(/\.git$/i, '').toLowerCase()
  if (!owner || !repo) return ''
  return `${owner}/${repo}`
}

export function isGithubRepoUrl(repoUrl) {
  return Boolean(githubRepoSlugFromUrl(repoUrl))
}

/**
 * 将 github-credential-status 的 repo_bindings 转为 slug -> row 映射。
 * @param {Array<{ repo_slug?: string, selected_github_user_id?: string|null }>} repoBindings
 */
export function githubRepoBindingsBySlug(repoBindings) {
  const map = {}
  if (!Array.isArray(repoBindings)) return map
  for (const row of repoBindings) {
    const slug = String(row?.repo_slug || '').trim().toLowerCase()
    if (!slug) continue
    map[slug] = row
  }
  return map
}

/**
 * 发送给 AI 前：校验各关联仓库是否已保存克隆身份与（GitHub 仓）PR 授权账号。
 * 仅依据已持久化数据（任务 parameters / repo_bindings），不含下拉框未保存的草稿。
 *
 * @param {string[]} repoUrls
 * @param {(url: string) => string} resolveSavedGitIdentityId
 * @param {(url: string) => string|null|undefined} resolveSavedGithubUserId
 */
export function validateTaskRepoSavedAssociations(
  repoUrls,
  resolveSavedGitIdentityId,
  resolveSavedGithubUserId,
) {
  const seen = new Set()
  const missingGit = []
  const missingGithub = []
  for (const raw of repoUrls || []) {
    const url = String(raw || '').trim()
    if (!url || seen.has(url)) continue
    seen.add(url)
    if (!String(resolveSavedGitIdentityId(url) || '').trim()) {
      missingGit.push(url)
    }
    if (isGithubRepoUrl(url) && resolveSavedGithubUserId(url) == null) {
      missingGithub.push(url)
    }
  }
  if (missingGit.length === 0 && missingGithub.length === 0) {
    return { ok: true, missingGit, missingGithub, message: '' }
  }
  const lines = ['请先在「关联项目」中为下列仓库保存配置后再发送给 AI：']
  for (const url of missingGit) {
    lines.push(`· ${url}：未保存「Git 提交身份」`)
  }
  for (const url of missingGithub) {
    lines.push(`· ${url}：未保存「GitHub App 授权账号」`)
  }
  return {
    ok: false,
    missingGit,
    missingGithub,
    message: lines.join('\n'),
  }
}

/**
 * 仓库克隆身份下拉展示文案：依序用户名、邮箱、标签。
 * 参数 identity 符合 {@link GitIdentityOption} 类型约定。
 */
export function repoCloneIdentityOptionLabel(identity) {
  if (!identity || typeof identity !== 'object') return ''
  const name = String(
    identity.git_user_name || identity.git_name || identity.name || '',
  ).trim()
  const email = String(
    identity.git_user_email || identity.git_email || identity.email || '',
  ).trim()
  const label = String(identity.label || '').trim()

  const parts = []
  if (name && email) {
    parts.push(`${name} <${email}>`)
  } else if (name) {
    parts.push(name)
  } else if (email) {
    parts.push(`<${email}>`)
  }
  if (label) parts.push(label)
  if (parts.length) return parts.join(' · ')

  const displayName = String(identity.display_name || '').trim()
  if (displayName) return displayName
  return '未命名身份'
}

/** 从公司 Git 身份列表中解析 is_default 标记的默认身份 id；无默认时返回空字符串 */
export function resolveDefaultCompanyGitIdentityId(identities) {
  if (!Array.isArray(identities)) return ''
  for (const item of identities) {
    if (!item || !item.is_default) continue
    const id = String(item.id || '').trim()
    if (id) return id
  }
  return ''
}

/**
 * 解析仓库克隆身份的选中值：保存值 → 仅有一个可选项时自动选中该选项。
 * 模式参考 {@link resolveSelectedGithubUserIdForRepo}。
 *
 * @param {{ savedId?: string|undefined|null, identityOptions?: Array<GitIdentityOption> }} args
 * @returns {string}
 */
export function resolveSelectedGitIdentityIdForRepo({ savedId, identityOptions = [] } = {}) {
  if (savedId != null && String(savedId).trim()) return String(savedId)
  if (Array.isArray(identityOptions) && identityOptions.length === 1) {
    const onlyId = identityOptions[0]?.id
    if (onlyId != null && String(onlyId).trim()) return String(onlyId)
  }
  return ''
}

/** 与后端 repo_clone_git_identities 键一致：兼容 parameters 键与项目仓库 URL 尾部斜杠等差异 */
export function resolveRepoCloneIdentityFromMap(m, rowUrl) {
  const u = String(rowUrl || '').trim()
  if (!u || !m || typeof m !== 'object') return ''
  const candidates = [u, u.replace(/\/$/, ''), u.endsWith('/') ? u : `${u}/`]
  for (const k of candidates) {
    const v = m[k]
    if (v != null && String(v).trim()) return String(v).trim()
  }
  return ''
}

export function normalizeMemberPk(raw) {
  if (raw == null) {
    return null
  }
  if (typeof raw === 'object') {
    const v = raw.id ?? raw.pk
    return v != null ? String(v) : null
  }
  return String(raw)
}

export function inferWorkBranchPreset(workBranchName) {
  if (!workBranchName) return 'feature'
  const lowerName = String(workBranchName).toLowerCase()
  if (lowerName.startsWith('feature/')) return 'feature'
  if (lowerName.startsWith('bugfix/')) return 'bugfix'
  if (lowerName.startsWith('hotfix/')) return 'hotfix'
  if (lowerName.startsWith('release/')) return 'release'
  return 'custom'
}

export function inferMergeTargetPreset(mergeTargetName) {
  if (!mergeTargetName) return 'develop'
  const lowerName = String(mergeTargetName).toLowerCase()
  if (lowerName === 'develop') return 'develop'
  if (lowerName === 'main') return 'main'
  if (lowerName.startsWith('release/')) return 'release'
  return 'custom'
}

/** 层图推送目标分支：与任务 target_branch / branch_strategy / projects 对齐 */
export function resolveLayerGraphPushTargetBranch(task, taskId) {
  const tid = String(taskId || '')
  const titleSegment = sanitizeBranchSegment(task?.title || '', 'task')
  const subst = (raw) =>
    resolveBranchNamePlaceholders(raw, { taskId: tid, taskTitleSegment: titleSegment })
  const direct = typeof task?.target_branch === 'string' ? task.target_branch.trim() : ''
  if (direct) {
    return subst(direct)
  }
  const bs = task?.branch_strategy
  const projects = Array.isArray(task?.projects) ? task.projects : []
  for (const p of projects) {
    const t = typeof p?.target_branch === 'string' ? p.target_branch.trim() : ''
    if (t) {
      return subst(t)
    }
  }
  if (!bs || typeof bs !== 'object') return ''
  const workShared = typeof bs.work_branch_name === 'string' ? bs.work_branch_name.trim() : ''
  if (workShared) {
    return subst(workShared)
  }
  const shared = typeof bs.target_branch_name === 'string' ? bs.target_branch_name.trim() : ''
  return subst(shared) || ''
}

/**
 * 解析「合并到目标分支」所用分支名（merge_target_branch_name）。
 * @param {object|null|undefined} task
 * @param {string} [taskId]
 * @returns {string}
 */
export function resolveLayerGraphMergeTargetBranch(task, taskId) {
  const tid = String(taskId || '')
  const titleSegment = sanitizeBranchSegment(task?.title || '', 'task')
  const subst = (raw) =>
    resolveBranchNamePlaceholders(raw, { taskId: tid, taskTitleSegment: titleSegment })
  const bs = task?.branch_strategy
  if (bs && typeof bs === 'object') {
    const mergeShared =
      typeof bs.merge_target_branch_name === 'string' ? bs.merge_target_branch_name.trim() : ''
    if (mergeShared) {
      return subst(mergeShared)
    }
  }
  const projects = Array.isArray(task?.projects) ? task.projects : []
  for (const p of projects) {
    const t =
      typeof p?.merge_target_branch === 'string'
        ? p.merge_target_branch.trim()
        : typeof p?.merge_target_branch_name === 'string'
          ? p.merge_target_branch_name.trim()
          : ''
    if (t) {
      return subst(t)
    }
  }
  return ''
}

/**
 * 多仓库分支名求交：仅保留每个列表都出现的分支名，去重后按字典序排序。
 * 空入参或任一列表无有效分支名时返回 []（无法形成共有分支）。
 *
 * @param {Array<Array<string>|null|undefined>} branchLists
 * @returns {string[]}
 */
export function intersectBranchNameLists(branchLists) {
  if (!Array.isArray(branchLists) || branchLists.length === 0) {
    return []
  }
  const normalized = branchLists.map((list) => {
    const names = Array.isArray(list) ? list : []
    return [
      ...new Set(
        names
          .map((b) => String(b ?? '').trim())
          .filter(Boolean),
      ),
    ]
  })
  if (normalized.some((list) => list.length === 0)) {
    return []
  }
  let acc = new Set(normalized[0])
  for (let i = 1; i < normalized.length; i += 1) {
    const next = new Set(normalized[i])
    acc = new Set([...acc].filter((name) => next.has(name)))
  }
  return [...acc].sort((a, b) => a.localeCompare(b))
}

/**
 * 从共有分支中按优先级挑选默认合并目标：
 * develop → release/*（多个时取字典序最大，倾向较新的日期后缀）→ main → 空字符串。
 *
 * @param {Array<string>|null|undefined} commonBranches
 * @returns {string}
 */
export function pickPreferredCommonMergeTargetBranch(commonBranches) {
  const names = [
    ...new Set(
      (Array.isArray(commonBranches) ? commonBranches : [])
        .map((b) => String(b ?? '').trim())
        .filter(Boolean),
    ),
  ]
  if (names.includes('develop')) return 'develop'
  const releaseBranches = names
    .filter((name) => name === 'release' || name.startsWith('release/'))
    .sort((a, b) => a.localeCompare(b))
  if (releaseBranches.length > 0) {
    return releaseBranches[releaseBranches.length - 1]
  }
  if (names.includes('main')) return 'main'
  return ''
}

/**
 * 从关联项目行收集应拉取分支的 (projectId, repoUrl) 列表。
 *
 * @param {Array<{ project_id?: string }>|null|undefined} linkedProjects
 * @param {(projectId: string) => string[]} getProjectRepos
 * @returns {Array<{ projectId: string, repoUrl: string }>}
 */
/**
 * 解析仓库行选中的 GitHub 账号：草稿 → 已保存绑定 → 仅一个已连接账号时自动选中。
 *
 * @param {{ draft?: unknown, boundUserId?: unknown, connectedOptions?: Array<{ github_user_id?: unknown }> }} args
 * @returns {string}
 */
export function resolveSelectedGithubUserIdForRepo({
  draft,
  boundUserId,
  connectedOptions = [],
} = {}) {
  if (draft != null && String(draft).trim()) return String(draft)
  if (boundUserId != null && String(boundUserId).trim()) return String(boundUserId)
  if (Array.isArray(connectedOptions) && connectedOptions.length === 1) {
    const onlyId = connectedOptions[0]?.github_user_id
    if (onlyId != null && String(onlyId).trim()) return String(onlyId)
  }
  return ''
}

export function collectLinkedRepoBranchTargets(linkedProjects, getProjectRepos) {
  const out = []
  const seen = new Set()
  for (const row of linkedProjects || []) {
    const projectId = row?.project_id != null ? String(row.project_id).trim() : ''
    if (!projectId) continue
    const repos = typeof getProjectRepos === 'function' ? getProjectRepos(projectId) : []
    for (const rawUrl of repos || []) {
      const repoUrl = String(rawUrl || '').trim()
      if (!repoUrl) continue
      const key = `${projectId}::${repoUrl}`
      if (seen.has(key)) continue
      seen.add(key)
      out.push({ projectId, repoUrl })
    }
  }
  return out
}

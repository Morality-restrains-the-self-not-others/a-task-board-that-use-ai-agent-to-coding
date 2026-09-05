/**
 * 工作面板：分支命名与 projectSelections → API mappings（纯函数，供 WorkPanel / 创建任务共用）。
 */

export { WORK_BRANCH_PRESET_OPTIONS } from './workBranchPresetOptions.js'

export function formatDateTimeLocal(date) {
  const pad = (num) => String(num).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`
}

export function getDefaultTaskDeadline() {
  const targetDate = new Date()
  targetDate.setSeconds(0, 0)
  targetDate.setDate(targetDate.getDate() + 7)
  const daysUntilThursday = (4 - targetDate.getDay() + 7) % 7
  targetDate.setDate(targetDate.getDate() + daysUntilThursday)
  return formatDateTimeLocal(targetDate)
}

export function formatDateYmd(date = new Date()) {
  const pad = (num) => String(num).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`
}

/** 新建任务用当前时间；已存在任务用 created_at */
export function resolveTaskCreatedAtDate(task) {
  if (task?.created_at) {
    const parsed = new Date(task.created_at)
    if (!Number.isNaN(parsed.getTime())) {
      return parsed
    }
  }
  return new Date()
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
 *        taskTitleSegment：已 sanitize / 翻译后的标题段；传空字符串时用 'task'
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
  // 历史误 sanitize：'${taskTitle}' → '__taskTitle_'
  name = name.replace(/__taskTitle_/g, titleSeg)
  return name
}

export function buildDefaultWorkBranchName(
  presetType = 'feature',
  taskTitle = '',
  companyUserNameStr = '',
  createdAt
) {
  const taskCreatedDateYmd = formatDateYmd(resolveTaskCreatedAtDate(createdAt != null ? { created_at: createdAt } : {}))
  const userName = sanitizeBranchSegment(companyUserNameStr, 'company_user_name')
  const trimmedTitle = String(taskTitle || '').trim()
  // 有标题则写入 sanitize 后的段；无标题保留可替换占位符（切勿再 sanitize，否则变成 __taskTitle_）
  const normalizedTaskTitle = trimmedTitle
    ? sanitizeBranchSegment(trimmedTitle, 'task')
    : '${taskTitle}'

  if (presetType === 'release') {
    return `release/${taskCreatedDateYmd}_aidev\${taskId}`
  }
  return `${presetType}/${taskCreatedDateYmd}_${userName}_aidev\${taskId}_${normalizedTaskTitle}`
}

/** 提交任务前：按任务创建日期刷新工作分支名中的日期段（保留标题翻译等其余片段） */
export function applyTaskCreatedDateToWorkBranchName(workBranchName, task) {
  const taskCreatedDateYmd = formatDateYmd(resolveTaskCreatedAtDate(task))
  const name = String(workBranchName || '').trim()
  if (!name) return name
  return name.replace(
    /^(feature|bugfix|hotfix|release)\/\d{4}-\d{2}-\d{2}_/,
    `$1/${taskCreatedDateYmd}_`
  )
}

export function buildDefaultMergeTargetBranchName(presetType = 'develop') {
  const today = formatDateYmd()
  if (presetType === 'develop' || presetType === 'main') {
    return presetType
  }
  if (presetType === 'release') {
    return `release/${today}_aidev\${taskId}`
  }
  return 'develop'
}

function getProjectByIdFromList(projectId, projectsList) {
  if (!projectId) return null
  return projectsList.find((p) => String(p.id) === String(projectId))
}

/** 将模态框中的 projectSelections 展开为 API 所需的顶层 projects（含 repo_index） */
export function buildBranchStrategyProjectMappings(task, projectsList) {
  const target = String(task.workBranchName || '').trim()
  const selections = Array.isArray(task.projectSelections) ? task.projectSelections : []
  const out = []
  for (const sel of selections) {
    const projectId = sel?.projectId ? String(sel.projectId) : ''
    if (!projectId) continue
    const project = getProjectByIdFromList(projectId, projectsList)
    const repos = Array.isArray(project?.git_repos) ? project.git_repos.filter((u) => u && String(u).trim()) : []
    const repoBranches = Array.isArray(sel.repoBranches) ? sel.repoBranches : []

    const baseForIndex = (i) => {
      const rb = repoBranches.find((r) => Number(r.repoIndex) === i)
      if (rb?.baseBranch != null && String(rb.baseBranch).trim() !== '') {
        return String(rb.baseBranch).trim()
      }
      const byPos = repoBranches[i]
      if (byPos?.baseBranch != null && String(byPos.baseBranch).trim() !== '') {
        return String(byPos.baseBranch).trim()
      }
      return ''
    }

    if (repos.length === 0) {
      out.push({
        project_id: projectId,
        repo_index: 0,
        base_branch: baseForIndex(0),
        target_branch: target,
      })
      continue
    }
    for (let i = 0; i < repos.length; i += 1) {
      out.push({
        project_id: projectId,
        repo_index: i,
        base_branch: baseForIndex(i),
        target_branch: target,
      })
    }
  }
  return out
}

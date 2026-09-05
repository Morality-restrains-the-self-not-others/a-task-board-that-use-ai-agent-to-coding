/**
 * 将任务 projects API 行与工作区项目详情合并，供关联项目只读/编辑展示。
 */
import { gitRepoAddressMismatch } from './gitRepoUrl.js'

export function taskProjectRowId(row) {
  if (!row || typeof row !== 'object') return ''
  const raw = row.project_id != null ? row.project_id : row.projectId
  return raw != null ? String(raw).trim() : ''
}

export function buildTaskProjectsWithDetails(apiRows, workspaceProjects) {
  const rows = Array.isArray(apiRows) ? apiRows : []
  const projectById = {}
  for (const p of Array.isArray(workspaceProjects) ? workspaceProjects : []) {
    projectById[String(p?.id || '')] = p
  }

  const projectMap = new Map()

  for (const row of rows) {
    const projectId = taskProjectRowId(row)
    if (!projectId) continue

    if (!projectMap.has(projectId)) {
      const project = projectById[projectId]
      const catalogRepos = Array.isArray(project?.git_repos)
        ? project.git_repos.filter((u) => u && String(u).trim())
        : []
      const fallbackUrl = String(row?.project_repo_url || row?.stored_repo_address || '').trim()
      const repos = catalogRepos.length > 0 ? catalogRepos : (fallbackUrl ? [fallbackUrl] : [])
      const projectForDisplay = project || (repos.length > 0 ? { id: projectId, git_repos: repos } : project)
      projectMap.set(projectId, {
        project_id: projectId,
        project_name: project?.name ? String(project.name) : '',
        project_missing: !project,
        project: projectForDisplay,
        repos,
        repo_branches: {},
      })
    }

    const projectData = projectMap.get(projectId)
    const idx = Number.isFinite(Number(row.repo_index)) ? Number(row.repo_index) : 0
    const repoAddress = projectData.repos[idx] ? String(projectData.repos[idx]).trim() : ''
    if (repoAddress) {
      projectData.repo_branches[repoAddress] = row.base_branch != null ? String(row.base_branch) : ''
    }
    const stored = String(row.stored_repo_address || '').trim()
    if (row.repo_address_mismatch || gitRepoAddressMismatch(stored, repoAddress)) {
      projectData.repo_address_mismatch = true
      projectData.stored_repo_address = stored
    }
  }

  const result = []
  for (const [projectId, projectData] of projectMap) {
    result.push({
      id: projectId,
      project_id: projectId,
      project_name: projectData.project_name,
      project_missing: Boolean(projectData.project_missing),
      project: projectData.project,
      repo_branches: projectData.repo_branches,
      repo_address_mismatch: Boolean(projectData.repo_address_mismatch),
      stored_repo_address: projectData.stored_repo_address || '',
    })
  }
  return result
}

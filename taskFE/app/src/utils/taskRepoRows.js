/**
 * Build clone/identity repo rows from task projects + workspace catalog.
 * Catalog git_repos win; otherwise fall back to task API URLs so a linked
 * project still appears when the workspace list is empty or stale.
 */
import { taskProjectRowId } from './taskProjectsWithDetails.js'

export function buildTaskRepoRows(apiProjects, workspaceProjects) {
  const rows = []
  const linked = Array.isArray(apiProjects) ? apiProjects : []
  const projectIdsFromTask = new Set()
  for (const p of linked) {
    const pid = taskProjectRowId(p)
    if (pid) projectIdsFromTask.add(pid)
  }
  if (projectIdsFromTask.size === 0) return rows

  for (const p of Array.isArray(workspaceProjects) ? workspaceProjects : []) {
    const projectId = String(p?.id || '')
    if (!projectIdsFromTask.has(projectId)) continue
    const repos = Array.isArray(p?.git_repos) ? p.git_repos.filter((u) => u && String(u).trim()) : []
    for (const raw of repos) {
      const url = String(raw).trim()
      if (!url) continue
      rows.push({
        url,
        projectName: String(p?.name || '').trim() || String(p?.id || ''),
        projectId,
      })
    }
  }

  const emittedProjectIds = new Set(rows.map((row) => row.projectId))
  for (const p of linked) {
    const pid = taskProjectRowId(p)
    if (!pid || emittedProjectIds.has(pid)) continue
    const url = String(p.project_repo_url || p.stored_repo_address || '').trim()
    if (!url) continue
    rows.push({
      url,
      projectName: pid,
      projectId: pid,
    })
    emittedProjectIds.add(pid)
  }
  return rows
}

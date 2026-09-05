/** 从任务 projects API 行收集「任务记录地址 vs 项目当前地址」不一致项。 */
export function collectTaskRepoAddressMismatches(projects) {
  const rows = Array.isArray(projects) ? projects : []
  return rows
    .filter((row) => row && row.repo_address_mismatch)
    .map((row) => ({
      projectId: String(row.project_id || ''),
      storedRepoAddress: String(row.stored_repo_address || '').trim(),
      projectRepoUrl: String(row.project_repo_url || '').trim(),
    }))
    .filter((row) => row.storedRepoAddress && row.projectRepoUrl)
}

/** PATCH 任务 projects，使 TaskProject.repo_address 与项目当前 URL 对齐。 */
export async function syncTaskRepoAddressesFromProjects({
  apiFetch,
  tenantId,
  workspaceId,
  taskId,
  projects,
}) {
  const payload = {
    projects: (projects || [])
      .filter((row) => row && row.project_id)
      .map((row) => ({
        project_id: String(row.project_id),
        repo_index: Number.isFinite(Number(row.repo_index)) ? Number(row.repo_index) : 0,
        base_branch: String(row.base_branch || ''),
        target_branch: String(row.target_branch || ''),
      })),
  }
  const response = await apiFetch(
    `/api/tasks/todos/tenant_id/${tenantId}/workspace_id/${workspaceId}/${taskId}/`,
    {
      method: 'PATCH',
      credentials: 'include',
      headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
      body: JSON.stringify(payload),
    },
  )
  const data = await response.json().catch(() => ({}))
  if (!response.ok) {
    const detail = typeof data.detail === 'string' ? data.detail : '同步仓库地址失败'
    throw new Error(detail)
  }
  return data
}

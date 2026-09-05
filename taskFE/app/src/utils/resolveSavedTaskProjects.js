/**
 * Prefer GET/PATCH response projects (includes stored_repo_address / mismatch)
 * over the send-shaped payload, unless the response has no linked rows.
 */
export function resolveSavedTaskProjects(updatedTask, payloadProjects) {
  const getProjects = Array.isArray(updatedTask?.projects) ? updatedTask.projects : []
  if (getProjects.some((row) => row && row.project_id)) {
    return getProjects
  }
  return Array.isArray(payloadProjects) ? payloadProjects : []
}

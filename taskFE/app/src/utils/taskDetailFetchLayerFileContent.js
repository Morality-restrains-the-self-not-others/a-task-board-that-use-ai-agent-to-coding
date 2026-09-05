/**
 * 拉取层内文件内容；兼容旧 children 无仓库前缀路径（候选 path 依次尝试）。
 */
import { apiFetch } from './apiUtils.js'
import { extractTraceId } from './traceId.js'
import { appendCommentIdPath, fileContentPathCandidates } from './taskDetailProjectFileTreeHelpers.js'

/**
 * @param {{ tenantId: string, workspaceId: string, taskId: string, layerId: string, commentId?: string }} ctx
 * @param {string} relPath
 * @param {string[]} repoPrefixes
 * @returns {Promise<{ ok: true, data: object, path: string } | { ok: false, error: string, traceId: string }>}
 */
export async function fetchLayerFileContentWithPrefixFallback(ctx, relPath, repoPrefixes) {
  const { tenantId, workspaceId, taskId, layerId, commentId } = ctx
  const candidates = fileContentPathCandidates(relPath, repoPrefixes)
  let lastError = '读取文件内容失败'
  let lastTraceId = ''

  for (const candidate of candidates) {
    const apiPath = appendCommentIdPath(
      `/api/cloud/compute/container-layer-file-content/tenant_id/${encodeURIComponent(tenantId)}/workspace_id/${encodeURIComponent(workspaceId)}/task_id/${encodeURIComponent(taskId)}` +
      `?layer_id=${encodeURIComponent(layerId)}&path=${encodeURIComponent(candidate)}`,
      commentId,
    )
    try {
      const resp = await apiFetch(apiPath, {
        credentials: 'include',
        headers: { Accept: 'application/json' },
      })
      const data = await resp.json().catch(() => ({}))
      if (resp.ok) {
        return {
          ok: true,
          data: data && typeof data === 'object' ? data : null,
          path: candidate,
        }
      }
      const detail = data.detail || `读取文件内容失败（HTTP ${resp.status}）`
      lastError = typeof detail === 'string' ? detail : JSON.stringify(detail)
      lastTraceId = extractTraceId(resp) || extractTraceId(data) || ''
      // 仅对 not found 类错误继续尝试下一候选
      if (!/not found/i.test(String(lastError))) {
        break
      }
    } catch (err) {
      lastError = err?.message || '读取文件内容失败'
      lastTraceId = extractTraceId(err) || ''
      break
    }
  }

  return { ok: false, error: lastError, traceId: lastTraceId }
}

/**
 * @param {{ tenantId: string, workspaceId: string, taskId: string, layerId: string, commentId?: string }} ctx
 * @returns {Promise<string[]>}
 */
export async function fetchLayerRepoPathPrefixes(ctx) {
  const { tenantId, workspaceId, taskId, layerId, commentId } = ctx
  const apiPath = appendCommentIdPath(
    `/api/cloud/compute/container-layer-git-repo-identities/tenant_id/${encodeURIComponent(tenantId)}/workspace_id/${encodeURIComponent(workspaceId)}/task_id/${encodeURIComponent(taskId)}` +
    `?layer_id=${encodeURIComponent(layerId)}`,
    commentId,
  )
  try {
    const resp = await apiFetch(apiPath, {
      credentials: 'include',
      headers: { Accept: 'application/json' },
    })
    const data = await resp.json().catch(() => ({}))
    if (!resp.ok) return []
    const repos = Array.isArray(data.repos) ? data.repos : []
    return repos
      .map((r) => String(r?.rel_prefix || '').trim())
      .filter(Boolean)
  } catch {
    return []
  }
}

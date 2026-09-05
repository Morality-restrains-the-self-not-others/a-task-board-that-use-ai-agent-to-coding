/**
 * 文件树按需懒加载子目录逻辑。
 * 配合 TaskDetailProjectFileTree.vue 使用，提取以控制组件行数（500 行限制）。
 */
import { ref } from 'vue'
import { apiFetch } from '../../utils/apiUtils.js'
import { appendCommentIdPath, buildTreeFromFiles } from '../../utils/taskDetailProjectFileTreeHelpers.js'

/**
 * @param {import('vue').Ref<string>|(() => string)} layerIdRef
 * @param {import('vue').Ref<{tenantId: string, workspaceId: string, taskId: string}>} requestCtxRef
 * @param {import('vue').Ref<string[]>} flatFilesRef
 * @param {import('vue').Ref<object[]>} treeNodesRef
 * @param {import('vue').Ref<boolean>|boolean} [isReleasedRef] 服务器已释放时跳过容器文件树请求（OPT-20260823-038）
 */

export function entriesToPathStrings(entries) {
  return (Array.isArray(entries) ? entries : []).map((e) => {
    const rel = String(e.path || '').trim()
    if (!rel) return ''
    return e.type === 'dir' ? rel + '/' : rel
  }).filter(Boolean)
}

function readReleased(isReleasedRef) {
  if (typeof isReleasedRef === 'function') return Boolean(isReleasedRef())
  if (isReleasedRef && typeof isReleasedRef === 'object' && 'value' in isReleasedRef) return Boolean(isReleasedRef.value)
  return Boolean(isReleasedRef)
}

export function useFileTreeLazyChildren(layerIdRef, requestCtxRef, flatFilesRef, treeNodesRef, isReleasedRef) {
  /** @type {Map<string, {type: string, path: string, size: number}[]>} */
  const childrenCache = new Map()
  const loadingDirs = ref(new Set())

  const _layerId = () => String(
    (typeof layerIdRef === 'function' ? layerIdRef() : layerIdRef?.value) || ''
  ).trim()

  // entriesToPathStrings is module-level export (used by TaskDetailProjectFileTree.fetchFiles)

  async function fetchChildren(dirPath) {
    const p = String(dirPath || '').trim()
    if (!p) return []
    if (childrenCache.has(p)) return childrenCache.get(p)
    if (readReleased(isReleasedRef)) return []

    const layerId = _layerId()
    if (!layerId) return []
    const { tenantId, workspaceId, taskId, commentId } = requestCtxRef.value
    if (!tenantId || !workspaceId || !taskId) return []

    const newLoading = new Set(loadingDirs.value)
    newLoading.add(p)
    loadingDirs.value = newLoading

    let entries = []
    try {
      const apiPath = appendCommentIdPath(
        `/api/cloud/compute/container-layer-children/tenant_id/${encodeURIComponent(tenantId)}/workspace_id/${encodeURIComponent(workspaceId)}/task_id/${encodeURIComponent(taskId)}/` +
        `?layer_id=${encodeURIComponent(layerId)}&dir=${encodeURIComponent(p)}`,
        commentId,
      )
      const resp = await apiFetch(apiPath, {
        credentials: 'include',
        headers: { Accept: 'application/json' },
      })
      const data = await resp.json().catch(() => ({}))
      if (!resp.ok) {
        console.warn('[FileTree] fetchChildren error', dirPath, data)
        return []
      }
      entries = Array.isArray(data.entries) ? data.entries : []
      childrenCache.set(p, entries)

      const newPaths = entriesToPathStrings(entries)
      const existing = new Set(flatFilesRef.value)
      const deduped = newPaths.filter((fp) => !existing.has(fp))
      if (deduped.length > 0) {
        flatFilesRef.value = [...flatFilesRef.value, ...deduped]
        treeNodesRef.value = buildTreeFromFiles(flatFilesRef.value)
      }
      return entries
    } catch (err) {
      console.warn('[FileTree] fetchChildren error', dirPath, err?.message || err)
      return []
    } finally {
      const cleared = new Set(loadingDirs.value)
      cleared.delete(p)
      loadingDirs.value = cleared
    }
  }

  function clearChildrenCache() {
    childrenCache.clear()
    loadingDirs.value = new Set()
  }

  return { childrenCache, loadingDirs, fetchChildren, clearChildrenCache }
}

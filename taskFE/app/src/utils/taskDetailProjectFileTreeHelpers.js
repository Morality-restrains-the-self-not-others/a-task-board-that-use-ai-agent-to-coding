/**
 * Pure helpers for TaskDetailProjectFileTree (kept out of the Vue SFC for line-budget).
 */
import { appendCommentIdPath, appendCommentIdQuery } from './containerForwardCommentId.js'

export { appendCommentIdPath, appendCommentIdQuery }

/**
 * 文件内容预览候选路径：原路径优先；若旧 children 返回无仓库前缀路径，
 * 再尝试补上 git-repo-identities 的 rel_prefix（与 files/* 多仓语义对齐）。
 * @param {string} relPath
 * @param {string[]} repoPrefixes
 * @returns {string[]}
 */
export function fileContentPathCandidates(relPath, repoPrefixes) {
  const p = String(relPath || '')
    .trim()
    .replace(/\\/g, '/')
    .replace(/^\/+/, '')
  if (!p) return []
  const prefixes = (Array.isArray(repoPrefixes) ? repoPrefixes : [])
    .map((x) =>
      String(x || '')
        .trim()
        .replace(/\\/g, '/')
        .replace(/(?:^\/+|\/+$)/g, ''),
    )
    .filter(Boolean)
  const alreadyPrefixed = prefixes.some((pref) => p === pref || p.startsWith(`${pref}/`))
  if (alreadyPrefixed || prefixes.length === 0) return [p]
  return [p, ...prefixes.map((pref) => `${pref}/${p}`)]
}

export function normalizeListedFilePath(one) {
  const raw = typeof one === 'string' ? one.trim() : String(one?.path || '').trim()
  return raw.replace(/\/+$/, '')
}

/** 缓存的选中 path 是否仍存在于当前文件列表（文件精确匹配；目录为前缀或目录标记） */
export function selectionPathStillExists(path, kind, files) {
  const p = String(path || '').trim().replace(/\/+$/, '')
  if (!p) return false
  const paths = (Array.isArray(files) ? files : []).map(normalizeListedFilePath).filter(Boolean)
  if (kind === 'git') {
    return paths.some((fp) => fp === p || fp.startsWith(`${p}/`))
  }
  return paths.includes(p)
}

export function buildTreeFromFiles(files) {
  const root = []
  const nodeMap = new Map()
  const ensureDir = (fullPath, name, parentList) => {
    if (nodeMap.has(fullPath)) return nodeMap.get(fullPath)
    const n = { type: 'dir', name, path: fullPath, children: [] }
    nodeMap.set(fullPath, n)
    parentList.push(n)
    return n
  }
  const ensureFile = (fullPath, name, parentList) => {
    if (nodeMap.has(fullPath)) return
    const n = { type: 'file', name, path: fullPath }
    parentList.push(n)
  }
  for (const one of files) {
    const raw = typeof one === 'string' ? one.trim() : String(one?.path || '').trim()
    if (!raw) continue
    // 后端可用 `dirname/` 标记空目录或仅噪声内容的目录
    const isDirMarker = raw.endsWith('/')
    const rel = isDirMarker ? raw.replace(/\/+$/, '') : raw
    const parts = rel.split('/').filter(Boolean)
    if (!parts.length) continue
    let curList = root
    let curPath = ''
    for (let i = 0; i < parts.length; i++) {
      const part = parts[i]
      curPath = curPath ? `${curPath}/${part}` : part
      const isLeaf = i === parts.length - 1
      if (isLeaf) {
        if (isDirMarker) {
          ensureDir(curPath, part, curList)
        } else {
          ensureFile(curPath, part, curList)
        }
      } else {
        const dirNode = ensureDir(curPath, part, curList)
        curList = dirNode.children
      }
    }
  }
  const sortNodes = (nodes) => {
    nodes.sort((a, b) => {
      if (a.type !== b.type) return a.type === 'dir' ? -1 : 1
      return String(a.name).localeCompare(String(b.name))
    })
    for (const n of nodes) {
      if (n.type === 'dir' && Array.isArray(n.children)) sortNodes(n.children)
    }
  }
  sortNodes(root)
  return root
}

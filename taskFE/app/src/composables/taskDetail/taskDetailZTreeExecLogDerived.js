import { LAYER_TREE_NODE_PREFIX } from '../../utils/layerZtreeNodes.js'

/**
 * 执行日志 ZTree 派生纯函数（无 Vue 依赖）。
 * 从 taskDetailZTreeExecLogState.js 拆分（OPT-20260815-027）：层目标解析、
 * layer_changes 规范化/合并/指纹/最新新增层，均与状态工厂解耦。
 */

export function ztreeLogCreatedAtMs(iso) {
  const m = Date.parse(iso || '')
  return Number.isFinite(m) ? m : 0
}

export function resolveZTreeLogTargets(node, jobs) {
  if (!node || node.nodeKind === 'virtual' || node.nodeKind === 'cycle') {
    return { layerId: '', jobId: '' }
  }
  const jlist = Array.isArray(jobs) ? jobs : []
  if (node.nodeKind === 'job') {
    const j = jlist.find((x) => x && String(x.id) === String(node.id))
    return {
      layerId: j?.layer_id ? String(j.layer_id) : '',
      jobId: node.id != null ? String(node.id) : '',
    }
  }
  if (node.nodeKind === 'layer') {
    const raw = node.id != null ? String(node.id) : ''
    const lid = raw.startsWith(LAYER_TREE_NODE_PREFIX)
      ? raw.slice(LAYER_TREE_NODE_PREFIX.length)
      : ''
    const vis = jlist.filter(
      (j) => j && j.layer_id === lid && j.command_kind !== 'clone',
    )
    vis.sort((a, b) => ztreeLogCreatedAtMs(a.created_at) - ztreeLogCreatedAtMs(b.created_at))
    const jobId = vis.length ? String(vis[vis.length - 1].id) : ''
    return { layerId: lid, jobId }
  }
  return { layerId: '', jobId: '' }
}

/** 文件变动列表不包含 .git 树（与层 diff 一致） */
export function layerChangePathIsGitInternal(path) {
  const p = String(path || '')
    .replace(/\\/g, '/')
    .replace(/(?:^\/+|\/+$)/g, '')
  if (!p) return false
  if (p === '.git' || p.startsWith('.git/')) return true
  if (p.includes('/.git/')) return true
  if (p.endsWith('/.git')) return true
  return false
}

export function normalizeLayerChangesPayload(src) {
  if (!src || typeof src !== 'object') return null
  const layerId = src.layer_id != null ? String(src.layer_id).trim() : ''
  if (!layerId) return null
  const raw = Array.isArray(src.changes) ? src.changes : []
  const changes = raw
    .map((one) => ({
      path: one && typeof one.path === 'string' ? one.path : '',
      kind: one && typeof one.kind === 'string' ? one.kind : '',
      git_staged: typeof one?.git_staged === 'boolean' ? one.git_staged : false,
      git_unstaged: typeof one?.git_unstaged === 'boolean' ? one.git_unstaged : true,
      git_layer_diff_only:
        typeof one?.git_layer_diff_only === 'boolean' ? one.git_layer_diff_only : false,
    }))
    .filter((one) => one.path && !layerChangePathIsGitInternal(one.path))
  const rawCount = Number(src.change_count)
  const count =
    Number.isFinite(rawCount) && rawCount >= 0 ? Math.floor(rawCount) : changes.length
  const rawNext = Number(src.next_offset)
  const nextOffset = Number.isFinite(rawNext) && rawNext >= 0 ? Math.floor(rawNext) : changes.length
  return {
    layer_id: layerId,
    parent_layer_id:
      src.parent_layer_id != null && src.parent_layer_id !== ''
        ? String(src.parent_layer_id)
        : null,
    same: count === 0 && changes.length === 0,
    truncated: Boolean(src.truncated),
    has_more: Boolean(src.has_more),
    next_offset: nextOffset,
    changes,
    change_count: count,
    job_ids: Array.isArray(src.job_ids)
      ? src.job_ids.map((x) => String(x || '').trim()).filter(Boolean)
      : [],
    detail: typeof src.detail === 'string' ? src.detail : '',
    /** 最近一次成功拉取该变动集的请求 traceId（截断提示可观测） */
    fetch_trace_id: String(src.fetch_trace_id || '').trim(),
  }
}

/**
 * 将分页续拉结果合并进已有 layer_changes（按 path 去重，追加新项）。
 * @param {ReturnType<typeof normalizeLayerChangesPayload>|null|undefined} prev
 * @param {ReturnType<typeof normalizeLayerChangesPayload>} page
 */
export function mergeLayerChangesPage(prev, page) {
  if (!page) return prev || null
  if (!prev || prev.layer_id !== page.layer_id) return page
  const seen = new Set((prev.changes || []).map((c) => c.path))
  const appended = []
  for (const c of page.changes || []) {
    if (!c?.path || seen.has(c.path)) continue
    seen.add(c.path)
    appended.push(c)
  }
  const changes = [...(prev.changes || []), ...appended]
  const rawCount = Number(page.change_count)
  const count =
    Number.isFinite(rawCount) && rawCount >= 0
      ? Math.floor(rawCount)
      : Math.max(Number(prev.change_count) || 0, changes.length)
  return {
    ...prev,
    ...page,
    changes,
    change_count: count,
    same: count === 0 && changes.length === 0,
    truncated: Boolean(prev.truncated) || Boolean(page.truncated),
    has_more: Boolean(page.has_more),
    next_offset:
      page.next_offset != null
        ? page.next_offset
        : changes.length,
    detail: typeof page.detail === 'string' && page.detail ? page.detail : prev.detail || '',
    fetch_trace_id: page.fetch_trace_id || prev.fetch_trace_id || '',
  }
}

/**
 * Stable fingerprint for layer_changes content.
 * Used to avoid bumping project file tree on identical exec-log poll payloads.
 * @param {ReturnType<typeof normalizeLayerChangesPayload>|null|undefined} normalized
 * @returns {string}
 */
export function layerChangesContentFingerprint(normalized) {
  if (!normalized || typeof normalized !== 'object') return ''
  const layerId = String(normalized.layer_id || '').trim()
  if (!layerId) return ''
  const parts = (Array.isArray(normalized.changes) ? normalized.changes : []).map((c) => {
    const path = typeof c?.path === 'string' ? c.path : ''
    const kind = typeof c?.kind === 'string' ? c.kind : ''
    const staged = c?.git_staged ? 1 : 0
    const unstaged = c?.git_unstaged ? 1 : 0
    const layerOnly = c?.git_layer_diff_only ? 1 : 0
    return `${path}\0${kind}\0${staged}\0${unstaged}\0${layerOnly}`
  })
  parts.sort()
  return `${layerId}|${normalized.truncated ? 1 : 0}|${Number(normalized.change_count) || 0}|${parts.join('\n')}`
}

export function pickNewestAddedLayerIdFromSnapshot(s, addedIds) {
  if (!s || !Array.isArray(addedIds) || addedIds.length === 0) {
    return ''
  }
  const byId = new Map()
  for (const l of s.layers || []) {
    const id = l && l.layer_id != null ? String(l.layer_id).trim() : ''
    if (id) {
      byId.set(id, l)
    }
  }
  let best = ''
  let bestT = -Infinity
  for (const id of addedIds) {
    const l = byId.get(id)
    const t = Date.parse((l && l.created_at) || '') || 0
    if (t >= bestT) {
      bestT = t
      best = id
    }
  }
  return best
}

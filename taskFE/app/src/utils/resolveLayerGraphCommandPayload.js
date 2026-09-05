import { LAYER_TREE_NODE_PREFIX, LAYER_GRAPH_ROOT_ID } from './layerZtreeNodes.js'

function createdAtMs(iso) {
  const m = Date.parse(iso || '')
  return Number.isFinite(m) ? m : 0
}

/**
 * 将 zTree 节点解析为容器 POST /api/jobs 所需字段（与 onlineService 行为对齐）。
 * @param {object} node — buildZTreeNodesSerialFromLayers / buildZTreeNodesFromLayers 产出的节点（含 nodeKind、id）
 * @param {object[]} jobs — 当前快照中的 jobs
 * @returns {{ parent_job_id: string, repo_layer_id: null } | { parent_job_id: null, repo_layer_id: string } | { error: string }}
 */
export function resolveLayerGraphCommandPayload(node, jobs) {
  if (!node || typeof node !== 'object') {
    return { error: '未选择节点' }
  }
  const kind = node.nodeKind
  if (kind === 'virtual' || kind === 'cycle') {
    return { error: '不能对此节点发送指令' }
  }
  const jlist = Array.isArray(jobs) ? jobs : []

  if (kind === 'job') {
    const jid = node.id != null ? String(node.id) : ''
    if (!jid || jid === LAYER_GRAPH_ROOT_ID) {
      return { error: '无效的任务节点' }
    }
    return { parent_job_id: jid, repo_layer_id: null }
  }

  if (kind === 'layer') {
    const raw = node.id != null ? String(node.id) : ''
    const lid = raw.startsWith(LAYER_TREE_NODE_PREFIX)
      ? raw.slice(LAYER_TREE_NODE_PREFIX.length)
      : ''
    if (!lid) {
      return { error: '无法解析可写层 ID' }
    }
    const vis = jlist
      .filter((j) => j && j.layer_id === lid && j.command_kind !== 'clone')
      .sort((a, b) => createdAtMs(a.created_at) - createdAtMs(b.created_at))
    if (vis.length >= 1) {
      const last = vis[vis.length - 1]
      return { parent_job_id: String(last.id), repo_layer_id: null }
    }
    return { parent_job_id: null, repo_layer_id: lid }
  }

  return { error: '不支持的节点类型' }
}

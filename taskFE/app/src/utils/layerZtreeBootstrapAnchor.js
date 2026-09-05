/**
 * 引导空层锚点（OPT-20260820-002）与「真实可写层」判定。
 * 空层 / bootstrap_pending 只是启动时序占位，不得当作层图已就绪。
 */

/**
 * @param {unknown} layer
 * @returns {boolean}
 */
export function isBootstrapAnchorLayer(layer) {
  if (!layer || typeof layer !== 'object') return false
  const rec = /** @type {Record<string, unknown>} */ (layer)
  if (rec.bootstrap_failed === true) return true
  if (rec.bootstrap_pending === true) return true
  if (String(rec.meta_kind || '') === 'empty') return true
  // 旧容器快照可能丢掉 meta_kind / bootstrap_*：无 git、无指令、pending/failed 的占位层仍是锚点。
  const mind = String(rec.mind_state || rec.job_status || '').toLowerCase()
  const cmd = String(rec.command || '').trim()
  const noCmd = !cmd || cmd === '正在准备可写层'
  const noJob = rec.job_id == null || rec.job_id === ''
  const dirty = rec.git_worktree_dirty
  const noGit = dirty === false || dirty == null
  return noCmd && noJob && noGit && (mind === 'pending' || mind === 'failed')
}

/**
 * @param {unknown} layers
 * @returns {boolean}
 */
export function snapshotHasRealWritableLayer(layers) {
  if (!Array.isArray(layers) || layers.length === 0) return false
  return layers.some((l) => !isBootstrapAnchorLayer(l))
}

/**
 * ztree simpleData：仅 layer 行且非引导锚点才算真实可写层。
 * 无 nodeKind 的遗留测例节点视为真实层，避免误开 overlay。
 * @param {unknown} nodes
 * @returns {boolean}
 */
export function ztreeHasRealWritableLayer(nodes) {
  if (!Array.isArray(nodes) || nodes.length === 0) return false
  const layers = nodes.filter((n) => n && n.nodeKind === 'layer')
  if (layers.length === 0) {
    return nodes.some((n) => n && n.nodeKind !== 'virtual')
  }
  return layers.some((n) => n.bootstrapAnchor !== true)
}

/**
 * @param {{ layerGraphHasRealWritableLayer?: boolean, layerGraphZNodes?: unknown }} view
 * @returns {boolean}
 */
export function resolveLayerGraphHasRealWritableLayer(view = {}) {
  if (view.layerGraphHasRealWritableLayer === true) return true
  if (view.layerGraphHasRealWritableLayer === false) return false
  return ztreeHasRealWritableLayer(view.layerGraphZNodes)
}

/**
 * @param {unknown} layer
 * @returns {string}
 */
export function bootstrapAnchorDisplayCommand(layer) {
  if (!isBootstrapAnchorLayer(layer)) return ''
  const rec = /** @type {Record<string, unknown>} */ (layer)
  if (rec.bootstrap_failed === true) {
    const err = String(rec.bootstrap_error || '').trim()
    return err || '引导克隆失败'
  }
  return '正在准备可写层'
}

/**
 * @param {unknown} layer
 * @returns {'active'|null}
 */
export function bootstrapAnchorZtStyle(layer) {
  if (!isBootstrapAnchorLayer(layer)) return null
  return 'active'
}

/**
 * @param {unknown} layer
 * @returns {{ bootstrapAnchor?: true }}
 */
export function layerZtreeBootstrapAnchorFields(layer) {
  if (!isBootstrapAnchorLayer(layer)) return {}
  return { bootstrapAnchor: true }
}

/**
 * @param {unknown} layers
 * @returns {string}
 */
export function bootstrapFailureMessageFromLayers(layers) {
  if (!Array.isArray(layers)) return ''
  for (const raw of layers) {
    if (!raw || typeof raw !== 'object') continue
    const rec = /** @type {Record<string, unknown>} */ (raw)
    if (rec.bootstrap_failed !== true) continue
    const err = String(rec.bootstrap_error || '').trim()
    if (err) return err
  }
  return ''
}

/**
 * 凭证失败已知时，不要让锚点节点继续写「正在准备可写层」。
 * @param {unknown} nodes
 * @param {unknown} failureRaw
 * @returns {unknown}
 */
export function renameBootstrapAnchorPendingNameOnFailure(nodes, failureRaw) {
  if (!Array.isArray(nodes)) return nodes
  const raw = String(failureRaw || '')
  if (!/REPO_CLONE_CREDENTIALS|repo-clone-credentials|Git 授权|克隆凭证/i.test(raw)) {
    return nodes
  }
  return nodes.map((n) => {
    if (!n || typeof n !== 'object' || n.bootstrapAnchor !== true) return n
    const name = String(n.name || '')
    if (!name.includes('正在准备可写层')) return n
    return { ...n, name: name.replace('正在准备可写层', '引导克隆失败') }
  })
}

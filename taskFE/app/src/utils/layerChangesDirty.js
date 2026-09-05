/**
 * 容器上报的 layer_changes（执行日志 / SSE）是否仍表示工作区相对 Git 有待提交变更。
 * 仅「相对父层」且已与 HEAD 一致（git_layer_diff_only）的条目不算 dirty，避免已提交后仍解锁 ztree「提交」。
 *
 * @param {object|null|undefined} payload
 * @returns {boolean}
 */
export function layerChangesPayloadImpliesWorktreeDirty(payload) {
  if (!payload || typeof payload !== 'object') return false
  if (payload.same === true) return false

  const ch = Array.isArray(payload.changes) ? payload.changes : []
  if (ch.length > 0) {
    const hasCommitable = ch.some((c) => {
      if (!c || typeof c !== 'object') return false
      if (c.git_layer_diff_only === true) return false
      if (c.git_staged === true || c.git_unstaged === true) return true
      // 旧载荷未带 git_* 标注时，保守视为可提交（与 normalize 默认 unstaged 对齐）
      return c.git_staged == null && c.git_unstaged == null
    })
    return hasCommitable
  }

  const rawC = Number(payload.change_count)
  if (Number.isFinite(rawC) && rawC > 0) return true
  if (payload.same === false) return true
  return false
}

/**
 * @param {object|null|undefined} payload
 * @returns {number}
 */
export function layerChangesDisplayCount(payload) {
  if (!payload || typeof payload !== 'object') return 0
  const rawC = Number(payload.change_count)
  if (Number.isFinite(rawC) && rawC > 0) return Math.floor(rawC)
  const ch = Array.isArray(payload.changes) ? payload.changes : []
  return ch.length
}

/**
 * zTree「提交」旁文件数文案：可提交变更 vs 仅相对父层差异（避免误以为可点提交）。
 *
 * @param {object|null|undefined} payload
 * @param {{ impliesDirty?: boolean }} [opts]
 * @returns {{ label: string, title: string, parentDiffOnly: boolean }}
 */
export function formatLayerSubmitFileChangesCaption(payload, opts = {}) {
  const c = layerChangesDisplayCount(payload)
  if (c <= 0) {
    return { label: '', title: '', parentDiffOnly: false }
  }
  const truncated = Boolean(payload && payload.truncated)
  const impliesDirty =
    opts.impliesDirty === true || layerChangesPayloadImpliesWorktreeDirty(payload)
  const parentDiffOnly = !impliesDirty
  const suffix = truncated ? '+' : ''
  if (parentDiffOnly) {
    return {
      label: `${c}${suffix} 个相对父层差异`,
      title:
        '相对父层的路径差异（已在当前层 Git HEAD 中，不在暂存区/工作区）；工作区干净时无需再点「提交」',
      parentDiffOnly: true,
    }
  }
  return {
    label: `${c}${suffix} 个文件变化`,
    title: truncated ? '相对父层的变动路径列表已截断' : '含暂存或未暂存的可提交变更',
    parentDiffOnly: false,
  }
}

/**
 * 工作区干净、仅有相对父层差异时的「提交」按钮 title。
 *
 * @param {number} count
 * @returns {string}
 */
export function submitTitleForParentDiffOnly(count) {
  const n = Number(count)
  const c = Number.isFinite(n) && n > 0 ? Math.floor(n) : 0
  if (c > 0) {
    return `该可写层暂无未提交变更（列表中 ${c} 个文件仅为相对父层差异，已在当前层 Git 中）`
  }
  return '该可写层暂无未提交变更'
}

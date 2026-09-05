import { formatTaskDisplayNo, formatTaskIdTitleLabel } from './taskIdDisplay.js'

/**
 * 上层交付物候选是否匹配用户输入的编号或名称（大小写不敏感）。
 * 支持完整 id、`#序号`、标题与「#序号 标题」整段。
 * @param {{ id?: unknown, title?: unknown, workspace_seq?: unknown }|null|undefined} candidate
 * @param {unknown} query
 * @returns {boolean}
 */
export function parentDeliverableMatchesQuery(candidate, query) {
  const qRaw = String(query ?? '').trim().toLowerCase()
  if (!qRaw) return true
  if (!candidate || candidate.id == null || candidate.id === '') return false

  const id = String(candidate.id)
  const idLower = id.toLowerCase()
  const title = candidate.title != null ? String(candidate.title).trim().toLowerCase() : ''
  const seqLabel = formatTaskDisplayNo(candidate.workspace_seq).toLowerCase()
  const label = formatTaskIdTitleLabel(id, candidate.title, candidate.workspace_seq).toLowerCase()
  const qNoHash = qRaw.startsWith('#') ? qRaw.slice(1) : qRaw

  return (
    title.includes(qRaw)
    || idLower.includes(qRaw)
    || idLower.includes(qNoHash)
    || (seqLabel && (seqLabel === qRaw || seqLabel.slice(1) === qNoHash || seqLabel.includes(qRaw)))
    || label.includes(qRaw)
  )
}

/**
 * 按查询过滤候选。
 * 若传入 selectedId，已选中项即使不匹配也会保留（兼容旧 select 展示）。
 * @param {Array<{ id?: unknown, title?: unknown }>} candidates
 * @param {unknown} query
 * @param {unknown} [selectedId]
 * @returns {Array<{ id?: unknown, title?: unknown }>}
 */
export function filterParentDeliverableCandidates(candidates, query, selectedId) {
  const list = Array.isArray(candidates) ? candidates : []
  const selected = selectedId != null && selectedId !== '' ? String(selectedId) : ''
  return list.filter((c) => {
    if (selected && String(c?.id) === selected) return true
    return parentDeliverableMatchesQuery(c, query)
  })
}

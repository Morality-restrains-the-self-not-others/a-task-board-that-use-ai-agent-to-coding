/**
 * Pure helper: merge container layer-graph SSE/HTTP payload into reactive snapshot.
 */

function normalizeJobStatus(raw) {
  return String(raw || '')
    .trim()
    .toLowerCase()
}

function isTerminalJobStatus(raw) {
  const s = normalizeJobStatus(raw)
  return s === 'completed' || s === 'failed' || s === 'interrupted' || s === 'done'
}

function isActiveJobStatus(raw) {
  const s = normalizeJobStatus(raw)
  return s === 'queued' || s === 'pending' || s === 'running'
}

/**
 * 避免 /api/jobs 滞后 running 盖掉层 job_status 终态，或盖掉本地已确认的终态。
 * @param {object[]} nextJobs
 * @param {object[]} prevJobs
 * @param {object[]} layers
 */
export function reconcileLayerGraphJobs(nextJobs, prevJobs, layers) {
  const prevById = new Map(
    (Array.isArray(prevJobs) ? prevJobs : [])
      .filter((j) => j && j.id != null && String(j.id).trim())
      .map((j) => [String(j.id), j]),
  )
  const layerStatusById = new Map()
  for (const L of Array.isArray(layers) ? layers : []) {
    if (!L || L.layer_id == null) continue
    const st = L.job_status != null ? String(L.job_status) : ''
    if (st) layerStatusById.set(String(L.layer_id), st)
  }
  return (Array.isArray(nextJobs) ? nextJobs : []).map((j) => {
    if (!j || typeof j !== 'object') return j
    const jid = j.id != null ? String(j.id) : ''
    let status = j.status
    const prev = jid ? prevById.get(jid) : null
    if (prev && isTerminalJobStatus(prev.status) && isActiveJobStatus(status)) {
      status = prev.status
    }
    const lid = j.layer_id != null ? String(j.layer_id) : ''
    const layerSt = lid ? layerStatusById.get(lid) : ''
    if (isActiveJobStatus(status) && isTerminalJobStatus(layerSt)) {
      status = layerSt
    }
    if (status === j.status) return j
    return { ...j, status }
  })
}

export function createApplyLayerGraphFromPayload(deps) {
  return (data, opts) => {
    if (!data || typeof data !== 'object') {
      return
    }
    const nextLayers = Array.isArray(data.layers) ? data.layers : []
    const forceClear = Boolean(opts && opts.forceClear)
    if (!forceClear && nextLayers.length === 0) {
      const prev = deps.layerGraphSnapshot.value
      if (prev && Array.isArray(prev.layers) && prev.layers.length > 0) {
        return
      }
    }
    deps.containerLayerGraphAuthInvalid.value = false
    const prevLayers = Array.isArray(deps.layerGraphSnapshot.value?.layers)
      ? deps.layerGraphSnapshot.value.layers
      : []
    const prevJobs = Array.isArray(deps.layerGraphSnapshot.value?.jobs)
      ? deps.layerGraphSnapshot.value.jobs
      : []
    const prevById = new Map(
      prevLayers
        .filter((l) => l && l.layer_id)
        .map((l) => [String(l.layer_id), l]),
    )
    const mergedLayers = nextLayers.map((layer) => {
      if (!layer || typeof layer !== 'object') return layer
      const lid = String(layer.layer_id || '').trim()
      const prev = lid ? prevById.get(lid) : null
      const prevGr = prev?.git_remote && typeof prev.git_remote === 'object' ? prev.git_remote : null
      const nextGr = layer.git_remote && typeof layer.git_remote === 'object' ? layer.git_remote : null
      if (!prevGr || !nextGr) return layer
      const nextAhead = nextGr.ahead
      const aheadZero =
        nextAhead === 0 || nextAhead === '0' || (typeof nextAhead === 'number' && nextAhead === 0)
      const prevPushed =
        typeof prevGr.last_pushed_count === 'number' &&
        Number.isFinite(prevGr.last_pushed_count) &&
        prevGr.last_pushed_count > 0
          ? Math.floor(prevGr.last_pushed_count)
          : null
      const prevPrUrl =
        typeof prevGr.pr_html_url === 'string' && prevGr.pr_html_url.trim()
          ? prevGr.pr_html_url.trim()
          : ''
      if (!aheadZero) return layer
      const needPushed = prevPushed != null && !(typeof nextGr.last_pushed_count === 'number' && nextGr.last_pushed_count > 0)
      const needPr =
        Boolean(prevPrUrl) &&
        !(typeof nextGr.pr_html_url === 'string' && nextGr.pr_html_url.trim())
      if (!needPushed && !needPr) return layer
      const mergedGr = { ...nextGr }
      if (needPushed) mergedGr.last_pushed_count = prevPushed
      if (needPr) mergedGr.pr_html_url = prevPrUrl
      return {
        ...layer,
        git_remote: mergedGr,
      }
    })
    const rawJobs = Array.isArray(data.jobs) ? data.jobs : []
    const jobs = reconcileLayerGraphJobs(rawJobs, prevJobs, mergedLayers)
    deps.layerGraphSnapshot.value = {
      layers: mergedLayers,
      jobs,
      layers_root: typeof data.layers_root === 'string' ? data.layers_root : '',
      bootstrap_layer_id:
        data.bootstrap_layer_id != null && data.bootstrap_layer_id !== ''
          ? String(data.bootstrap_layer_id)
          : ''
    }
  }
}

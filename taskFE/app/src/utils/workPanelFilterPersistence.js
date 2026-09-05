/**
 * 工作面板过滤栏持久化：序列化 / 默认 / API URL（含 access_filter）。
 */

import { defaultDeliverableFilterBars } from './workPanelDeliverableFilterBars.js'
import { rootDeliverablePath } from './workPanelDeliverableAggregation.js'

export const WORK_PANEL_FILTER_VERSION = 2
export const MAX_WORK_PANEL_FILTER_BARS = 20

/**
 * taskProjectService handleWorkspacesRoute 分发：workspace_id 位置段在前、
 * kv 键值对在后（与 /api/tasks/{taskId}/tenant_id/{tid}/… 约定一致）。
 *
 * @param {string|number} tenantId
 * @param {string|number} workspaceId
 */
export function workPanelFiltersApiUrl(tenantId, workspaceId) {
  const tid = encodeURIComponent(String(tenantId ?? '').trim())
  const wid = encodeURIComponent(String(workspaceId ?? '').trim())
  return `/api/projects/workspaces/${wid}/work-panel-filters/tenant_id/${tid}`
}

/**
 * @param {unknown} path
 * @returns {Array<{type: string, id?: string, label?: string}>}
 */
function normalizePath(path) {
  if (!Array.isArray(path) || path.length === 0) return rootDeliverablePath()
  const out = []
  for (const seg of path) {
    if (!seg || typeof seg !== 'object') continue
    const type = String(seg.type ?? '').trim().toLowerCase()
    if (type === 'root') {
      out.push({ type: 'root' })
      continue
    }
    if (type === 'category' || type === 'task') {
      const id = seg.id != null && String(seg.id).trim() !== '' ? String(seg.id) : ''
      if (!id) continue
      out.push({
        type,
        id,
        label: seg.label != null ? String(seg.label) : '',
      })
    }
  }
  return out.length ? out : rootDeliverablePath()
}

/**
 * @param {unknown} raw
 * @returns {{ kind: 'person'|'group', id: string, label: string } | null}
 */
export function normalizeAccessFilterPref(raw) {
  if (raw == null || typeof raw !== 'object') return null
  const kind = String(raw.kind ?? '').trim().toLowerCase()
  const id = raw.id != null ? String(raw.id).trim() : ''
  if ((kind !== 'person' && kind !== 'group') || !id) return null
  return {
    kind,
    id,
    label: raw.label != null ? String(raw.label) : '',
  }
}

/**
 * @param {unknown} filter full AccessFilter with memberIds
 * @returns {{ kind: 'person'|'group', id: string, label: string } | null}
 */
export function accessFilterToPref(filter) {
  if (filter == null || typeof filter !== 'object') return null
  return normalizeAccessFilterPref({
    kind: filter.kind,
    id: filter.id,
    label: filter.label,
  })
}

/**
 * @param {unknown} payload
 * @returns {{
 *   version: number,
 *   deliverable_filter_bars: Array<{id: string, path: unknown[]}>,
 *   access_filter: { kind: 'person'|'group', id: string, label: string } | null,
 * }}
 */
export function normalizeWorkPanelFilterPayload(payload) {
  const defaults = {
    version: WORK_PANEL_FILTER_VERSION,
    deliverable_filter_bars: defaultDeliverableFilterBars(),
    access_filter: null,
  }
  if (!payload || typeof payload !== 'object') return defaults
  const rawBars = Array.isArray(payload.deliverable_filter_bars)
    ? payload.deliverable_filter_bars
    : null
  if (!rawBars || rawBars.length === 0) {
    return { ...defaults, access_filter: normalizeAccessFilterPref(payload.access_filter) }
  }
  if (rawBars.length > MAX_WORK_PANEL_FILTER_BARS) return defaults

  const bars = []
  for (let i = 0; i < rawBars.length; i += 1) {
    const bar = rawBars[i]
    if (!bar || typeof bar !== 'object') continue
    const id =
      bar.id != null && String(bar.id).trim() !== ''
        ? String(bar.id)
        : `bar-${i}`
    bars.push({ id, path: normalizePath(bar.path) })
  }
  if (!bars.length) return defaults
  const versionNum = Number(payload.version)
  return {
    version: Number.isFinite(versionNum) && versionNum > 0 ? versionNum : WORK_PANEL_FILTER_VERSION,
    deliverable_filter_bars: bars,
    access_filter: normalizeAccessFilterPref(payload.access_filter),
  }
}

/**
 * @param {unknown} bars
 * @param {unknown} [accessFilter]
 */
export function buildWorkPanelFilterPutBody(bars, accessFilter = null) {
  const normalized = normalizeWorkPanelFilterPayload({
    version: WORK_PANEL_FILTER_VERSION,
    deliverable_filter_bars: bars,
    access_filter: accessFilterToPref(accessFilter) ?? normalizeAccessFilterPref(accessFilter),
  })
  return {
    version: normalized.version,
    deliverable_filter_bars: normalized.deliverable_filter_bars,
    access_filter: normalized.access_filter,
  }
}

/**
 * @param {{ apiFetch: typeof fetch, tenantId: string, workspaceId: string }} opts
 */
export async function fetchWorkPanelFilters({ apiFetch, tenantId, workspaceId }) {
  const tid = String(tenantId ?? '').trim()
  const wid = String(workspaceId ?? '').trim()
  if (!tid || !wid || wid === 'default') {
    return normalizeWorkPanelFilterPayload(null)
  }
  const response = await apiFetch(workPanelFiltersApiUrl(tid, wid), {
    credentials: 'include',
    headers: { Accept: 'application/json' },
  })
  if (!response.ok) {
    const err = new Error(`work-panel-filters GET HTTP ${response.status}`)
    err.status = response.status
    throw err
  }
  const data = await response.json()
  return normalizeWorkPanelFilterPayload(data)
}

/**
 * @param {{ apiFetch: typeof fetch, tenantId: string, workspaceId: string, bars: unknown, accessFilter?: unknown }} opts
 */
export async function saveWorkPanelFilters({ apiFetch, tenantId, workspaceId, bars, accessFilter = null }) {
  const tid = String(tenantId ?? '').trim()
  const wid = String(workspaceId ?? '').trim()
  if (!tid || !wid || wid === 'default') return null
  const body = buildWorkPanelFilterPutBody(bars, accessFilter)
  const response = await apiFetch(workPanelFiltersApiUrl(tid, wid), {
    method: 'PUT',
    credentials: 'include',
    headers: {
      Accept: 'application/json',
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(body),
  })
  if (!response.ok) {
    const err = new Error(`work-panel-filters PUT HTTP ${response.status}`)
    err.status = response.status
    throw err
  }
  return normalizeWorkPanelFilterPayload(await response.json())
}

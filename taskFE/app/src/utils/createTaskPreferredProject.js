/**
 * 工作面板创建任务默认选中的项目：优先用户刚在项目详情看过/授过权的那个，
 * 而不是工作区列表 ORDER BY name 的第一项（否则会出现详情「已授权」、创建弹窗仍「OAuth 绑定」）。
 */

const STORAGE_PREFIX = 'createTask.preferredProjectId.'

export function createTaskPreferredProjectStorageKey(tenantId) {
  const tid = String(tenantId || '').trim()
  if (!tid) return ''
  return `${STORAGE_PREFIX}${tid}`
}

export function rememberCreateTaskPreferredProject(tenantId, projectId) {
  const key = createTaskPreferredProjectStorageKey(tenantId)
  const pid = String(projectId || '').trim()
  if (!key || !pid) return
  try {
    localStorage.setItem(key, pid)
  } catch {
    // 偏好可选：隐私模式 / 配额满时退回列表第一项
  }
}

export function readCreateTaskPreferredProject(tenantId) {
  const key = createTaskPreferredProjectStorageKey(tenantId)
  if (!key) return ''
  try {
    return String(localStorage.getItem(key) || '').trim()
  } catch {
    return ''
  }
}

/**
 * @param {unknown[]} projects
 * @param {Set<string>|string[]} [usedIds]
 * @param {string} [preferredId]
 * @returns {string}
 */
export function pickCreateTaskDefaultProjectId(projects, usedIds, preferredId) {
  const list = Array.isArray(projects) ? projects : []
  const used = usedIds instanceof Set
    ? usedIds
    : new Set((Array.isArray(usedIds) ? usedIds : []).map((id) => String(id || '').trim()).filter(Boolean))
  const preferred = String(preferredId || '').trim()
  if (preferred && !used.has(preferred)) {
    const hit = list.find((row) => String(row?.id || '').trim() === preferred)
    if (hit) return preferred
  }
  const first = list.find((row) => {
    const id = String(row?.id || '').trim()
    return id && !used.has(id)
  })
  return first ? String(first.id) : ''
}

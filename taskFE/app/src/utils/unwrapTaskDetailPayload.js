/**
 * Task-detail GET must be a single task record. Gateway misrouting or
 * list/envelope wrappers would otherwise assign an array to localTask,
 * leaving `.projects` undefined and the 关联项目 panel empty.
 */

export function pickTaskFromList(list, expectedTaskId) {
  const want = String(expectedTaskId || '').trim()
  if (!Array.isArray(list) || !want) return null
  const found = list.find((row) => row && String(row.id) === want)
  return found && typeof found === 'object' ? found : null
}

export function unwrapTaskDetailPayload(raw, expectedTaskId) {
  if (raw == null || typeof raw !== 'object') return null
  const want = String(expectedTaskId || '').trim()

  if (Array.isArray(raw)) {
    return pickTaskFromList(raw, want)
  }

  if (raw.id != null && String(raw.id).trim() !== '') {
    if (want && String(raw.id) !== want) return null
    return raw
  }

  if (Object.prototype.hasOwnProperty.call(raw, 'data')) {
    return unwrapTaskDetailPayload(raw.data, expectedTaskId)
  }
  if (Array.isArray(raw.results)) {
    return pickTaskFromList(raw.results, want)
  }
  return null
}

export function linkedProjectRowsFromTask(task, expectedTaskId) {
  const record = Array.isArray(task)
    ? pickTaskFromList(task, expectedTaskId)
    : (task && typeof task === 'object' ? task : null)
  return Array.isArray(record?.projects) ? record.projects : []
}

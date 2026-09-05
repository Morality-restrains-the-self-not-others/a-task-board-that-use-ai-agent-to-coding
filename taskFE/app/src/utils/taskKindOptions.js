/** 工作区任务类型（task_kind）可选值：默认与归一化。 */

export const DEFAULT_TASK_KIND_OPTIONS = Object.freeze(['bug-fix', 'feature'])

/**
 * @param {unknown} raw
 * @returns {string[]}
 */
export function normalizeTaskKindOptions(raw) {
  const list = Array.isArray(raw) ? raw : []
  const seen = new Set()
  const out = []
  for (const item of list) {
    const v = item == null ? '' : String(item).trim()
    if (!v) continue
    const key = v.toLowerCase()
    if (seen.has(key)) continue
    seen.add(key)
    out.push(v)
  }
  return out.length > 0 ? out : [...DEFAULT_TASK_KIND_OPTIONS]
}

/**
 * @param {unknown} body
 * @returns {string[]}
 */
export function taskKindOptionsFromResponse(body) {
  if (body && typeof body === 'object' && Array.isArray(body.options)) {
    return normalizeTaskKindOptions(body.options)
  }
  return [...DEFAULT_TASK_KIND_OPTIONS]
}

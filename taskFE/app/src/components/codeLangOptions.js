/** 工作区主要编程语言（code_lang）可选值：默认与归一化。 */

export const DEFAULT_CODE_LANG_OPTIONS = Object.freeze(['go', 'rust', 'js'])

/**
 * @param {unknown} raw
 * @returns {string[]}
 */
export function normalizeCodeLangOptions(raw) {
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
  return out.length > 0 ? out : [...DEFAULT_CODE_LANG_OPTIONS]
}

/**
 * @param {unknown} body
 * @returns {string[]}
 */
export function codeLangOptionsFromResponse(body) {
  if (body && typeof body === 'object' && Array.isArray(body.options)) {
    return normalizeCodeLangOptions(body.options)
  }
  return [...DEFAULT_CODE_LANG_OPTIONS]
}

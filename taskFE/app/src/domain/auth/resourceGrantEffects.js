/**
 * 资源组授予效果（v73 / ADR-0004）
 * view=可访问；operate=可编辑执行（蕴含 view）
 */

export const EFFECT_VIEW = 'view'
export const EFFECT_OPERATE = 'operate'

/** @param {string} effect */
export function normalizeGrantEffect(effect) {
  return String(effect || '').toLowerCase() === EFFECT_VIEW ? EFFECT_VIEW : EFFECT_OPERATE
}

/** @param {string} a @param {string} b */
export function maxGrantEffect(a, b) {
  const aa = String(a || '').trim()
  const bb = String(b || '').trim()
  if (!aa) return normalizeGrantEffect(bb)
  if (!bb) return normalizeGrantEffect(aa)
  if (normalizeGrantEffect(aa) === EFFECT_OPERATE || normalizeGrantEffect(bb) === EFFECT_OPERATE) {
    return EFFECT_OPERATE
  }
  return EFFECT_VIEW
}

/**
 * @param {Array<{group_key?:string,effect?:string}>|Record<string,string>|string[]} input
 * @returns {Record<string, 'view'|'operate'>}
 */
export function normalizeGrantsMap(input) {
  const out = {}
  if (!input) return out
  if (Array.isArray(input)) {
    if (input.length && typeof input[0] === 'string') {
      for (const k of input) {
        if (k) out[String(k)] = EFFECT_OPERATE
      }
      return out
    }
    for (const row of input) {
      const key = row?.group_key
      if (!key) continue
      out[String(key)] = maxGrantEffect(out[String(key)], row?.effect)
    }
    return out
  }
  if (typeof input === 'object') {
    for (const [k, v] of Object.entries(input)) {
      if (!k) continue
      out[String(k)] = normalizeGrantEffect(v)
    }
  }
  return out
}

/** @param {Record<string,string>} grantsMap */
export function grantsMapToList(grantsMap) {
  return Object.entries(grantsMap || {})
    .filter(([k]) => k)
    .map(([group_key, effect]) => ({ group_key, effect: normalizeGrantEffect(effect) }))
}

/** @param {Record<string,string>} grantsMap */
export function grantsMapToKeys(grantsMap) {
  return Object.keys(grantsMap || {})
}

/**
 * @param {Record<string,string>} grantsMap
 * @param {string} groupKey
 * @param {'view'|'operate'|null} effect null=清除
 */
export function setGrantEffect(grantsMap, groupKey, effect) {
  const next = { ...(grantsMap || {}) }
  const key = String(groupKey || '')
  if (!key) return next
  if (!effect) {
    delete next[key]
    return next
  }
  next[key] = normalizeGrantEffect(effect)
  return next
}

/**
 * 整页勾选：默认 operate；取消则移除 page + 全部子 region。
 * @param {Record<string,string>} grantsMap
 * @param {{ group_key?: string, children?: Array<{group_key?:string}> }} page
 * @param {boolean} checked
 */
export function togglePageGrants(grantsMap, page, checked) {
  let next = { ...(grantsMap || {}) }
  const keys = []
  if (page?.group_key) keys.push(String(page.group_key))
  for (const c of page?.children || []) {
    if (c?.group_key) keys.push(String(c.group_key))
  }
  if (checked) {
    for (const k of keys) next[k] = EFFECT_OPERATE
  } else {
    for (const k of keys) delete next[k]
  }
  return next
}

/**
 * @param {Record<string,string>} grantsMap
 * @param {{ group_key?: string, children?: Array<{group_key?:string}> }} page
 */
export function isPageFullyGranted(grantsMap, page) {
  const children = page?.children || []
  if (!children.length) return Boolean(grantsMap?.[page?.group_key])
  return children.every((c) => Boolean(grantsMap?.[c.group_key]))
}

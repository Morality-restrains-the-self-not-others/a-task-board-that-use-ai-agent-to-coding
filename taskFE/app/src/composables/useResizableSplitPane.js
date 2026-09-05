/**
 * 左右分栏宽度计算与本地持久化（目录栏 / 内容预览拖动分隔）。
 */

export const DEFAULT_LEFT_WIDTH_PX = 256
export const MIN_LEFT_WIDTH_PX = 192
export const MAX_LEFT_WIDTH_PX = 560
export const MAX_LEFT_WIDTH_RATIO = 0.7
export const KEYBOARD_STEP_PX = 16

/**
 * @param {number} widthPx
 * @param {number} containerWidthPx
 * @param {{ min?: number, max?: number, maxRatio?: number }} [opts]
 * @returns {number}
 */
export function clampLeftWidth(widthPx, containerWidthPx, opts = {}) {
  const min = Number.isFinite(opts.min) ? opts.min : MIN_LEFT_WIDTH_PX
  const max = Number.isFinite(opts.max) ? opts.max : MAX_LEFT_WIDTH_PX
  const maxRatio = Number.isFinite(opts.maxRatio) ? opts.maxRatio : MAX_LEFT_WIDTH_RATIO
  const n = Number(widthPx)
  if (!Number.isFinite(n)) return min
  const containerCap =
    Number.isFinite(containerWidthPx) && containerWidthPx > 0
      ? Math.floor(containerWidthPx * maxRatio)
      : max
  const upper = Math.max(min, Math.min(max, containerCap))
  return Math.min(upper, Math.max(min, Math.round(n)))
}

/**
 * @param {string} storageKey
 * @param {Storage | null | undefined} [storage]
 * @returns {number | null}
 */
export function readStoredLeftWidth(storageKey, storage) {
  const key = String(storageKey || '').trim()
  if (!key || !storage) return null
  try {
    const raw = storage.getItem(key)
    if (raw == null || raw === '') return null
    const n = Number(raw)
    return Number.isFinite(n) ? n : null
  } catch {
    return null
  }
}

/**
 * @param {string} storageKey
 * @param {number} widthPx
 * @param {Storage | null | undefined} [storage]
 */
export function writeStoredLeftWidth(storageKey, widthPx, storage) {
  const key = String(storageKey || '').trim()
  if (!key || !storage || !Number.isFinite(widthPx)) return
  try {
    storage.setItem(key, String(Math.round(widthPx)))
  } catch {
    /* ignore quota / private mode */
  }
}

/**
 * @param {number} clientX
 * @param {number} containerLeft
 * @param {number} containerWidth
 * @returns {number}
 */
export function leftWidthFromPointer(clientX, containerLeft, containerWidth) {
  return clampLeftWidth(clientX - containerLeft, containerWidth)
}

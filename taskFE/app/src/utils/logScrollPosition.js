/** 距底部小于该像素时视为「贴底」，追加内容后自动跟随 */
export const STICKY_LOG_BOTTOM_THRESHOLD_PX = 48

/**
 * 判断滚动容器是否贴在底部（或尚不可滚动）。
 * @param {Element | { scrollHeight: number, clientHeight: number, scrollTop: number } | null | undefined} el
 * @param {number} [thresholdPx]
 * @returns {boolean}
 */
export function isLogScrollNearBottom(el, thresholdPx = STICKY_LOG_BOTTOM_THRESHOLD_PX) {
  if (!el) return true
  const maxScroll = el.scrollHeight - el.clientHeight
  if (maxScroll <= 0) return true
  return maxScroll - el.scrollTop < thresholdPx
}

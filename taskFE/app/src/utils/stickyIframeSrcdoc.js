/**
 * iframe srcdoc 替换时的 sticky 滚动（与 onlineServiceJS setExecRichIframeSrcdocSticky 同语义）。
 * 贴底则跟随最新；非贴底则恢复替换前的 scrollTop。
 */
import { isLogScrollNearBottom, STICKY_LOG_BOTTOM_THRESHOLD_PX } from './logScrollPosition.js'

/**
 * @param {Document | null | undefined} doc
 * @returns {Element | null}
 */
export function iframeLogScrollingEl(doc) {
  if (!doc) return null
  return doc.scrollingElement || doc.documentElement || doc.body
}

/**
 * @param {Document | null | undefined} doc
 * @param {number} [thresholdPx]
 * @returns {boolean}
 */
export function iframeLogNearBottom(doc, thresholdPx = STICKY_LOG_BOTTOM_THRESHOLD_PX) {
  try {
    return isLogScrollNearBottom(iframeLogScrollingEl(doc), thresholdPx)
  } catch {
    return true
  }
}

/**
 * @param {Document | null | undefined} doc
 */
export function iframeLogScrollToBottom(doc) {
  try {
    const el = iframeLogScrollingEl(doc)
    if (el) el.scrollTop = el.scrollHeight
  } catch {
    /* sandbox / cross-origin */
  }
}

/**
 * 写入 iframe.srcdoc，并在 load 后恢复滚动位置。
 * @param {HTMLIFrameElement | null | undefined} fr
 * @param {string} srcdocHtml
 * @param {{ thresholdPx?: number, onAfterRestore?: () => void }} [options]
 */
export function setIframeSrcdocSticky(fr, srcdocHtml, options = {}) {
  if (!fr) return
  const thresholdPx = options.thresholdPx ?? STICKY_LOG_BOTTOM_THRESHOLD_PX
  let stickToBottom = true
  let savedScrollTop = 0
  try {
    const doc = fr.contentDocument
    const el = iframeLogScrollingEl(doc)
    stickToBottom = iframeLogNearBottom(doc, thresholdPx)
    if (!stickToBottom && el) savedScrollTop = el.scrollTop
  } catch {
    stickToBottom = true
  }

  const applyRestore = () => {
    try {
      const doc = fr.contentDocument
      const el = iframeLogScrollingEl(doc)
      if (!el) return
      if (stickToBottom) {
        iframeLogScrollToBottom(doc)
      } else {
        el.scrollTop = savedScrollTop
      }
    } catch {
      /* ignore */
    }
    if (typeof options.onAfterRestore === 'function') {
      options.onAfterRestore()
    }
  }

  const onLoad = () => {
    fr.removeEventListener('load', onLoad)
    requestAnimationFrame(() => {
      applyRestore()
      requestAnimationFrame(applyRestore)
    })
  }
  fr.addEventListener('load', onLoad)
  fr.srcdoc = srcdocHtml
}

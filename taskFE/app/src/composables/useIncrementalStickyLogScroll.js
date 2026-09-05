import { onScopeDispose, unref, watch } from 'vue'
import {
  DEFAULT_LOG_MAX_DOM_CHARS,
  DEFAULT_LOG_MAX_TEXT_NODES,
  syncLogTextContent,
} from '../utils/incrementalLogText.js'
import { isLogScrollNearBottom, STICKY_LOG_BOTTOM_THRESHOLD_PX } from '../utils/logScrollPosition.js'

export { isLogScrollNearBottom, STICKY_LOG_BOTTOM_THRESHOLD_PX }

/**
 * 增量写入日志文本 + sticky 滚动（短日志 / 非虚拟路径）。
 * - 前缀扩展：DOM append + 超阈值 normalize；贴底则滚到底。
 * - 非前缀 / 超 maxDomChars：整段替换后按 sticky 规则恢复。
 *
 * @param {import('vue').Ref<HTMLElement | null>} elRef
 * @param {() => unknown | import('vue').Ref<unknown> | import('vue').ComputedRef<unknown>} contentSource
 * @param {{
 *   thresholdPx?: number,
 *   maxTextNodes?: number,
 *   maxDomChars?: number,
 * }} [options]
 */
export function useIncrementalStickyLogScroll(elRef, contentSource, options = {}) {
  const thresholdPx = options.thresholdPx ?? STICKY_LOG_BOTTOM_THRESHOLD_PX
  const maxTextNodes = options.maxTextNodes ?? DEFAULT_LOG_MAX_TEXT_NODES
  const maxDomChars = options.maxDomChars ?? DEFAULT_LOG_MAX_DOM_CHARS
  /** @type {HTMLElement | null} */
  let boundEl = null
  let savedScrollTop = 0
  let stickToBottom = true
  let rafId = 0

  function readContent() {
    const src = unref(contentSource)
    return typeof src === 'function' ? src() : src
  }

  function writeContent(el) {
    return syncLogTextContent(el, readContent(), { maxTextNodes, maxDomChars })
  }

  function captureFromEl(el) {
    if (!el) {
      stickToBottom = true
      savedScrollTop = 0
      return
    }
    stickToBottom = isLogScrollNearBottom(el, thresholdPx)
    savedScrollTop = el.scrollTop
  }

  function onScroll() {
    captureFromEl(boundEl || elRef.value)
  }

  function unbindScroll() {
    if (boundEl) {
      boundEl.removeEventListener('scroll', onScroll)
      boundEl = null
    }
  }

  function bindScroll(el) {
    if (boundEl === el) return
    unbindScroll()
    if (!el) return
    boundEl = el
    el.addEventListener('scroll', onScroll, { passive: true })
    captureFromEl(el)
  }

  function applyStickyAfterWrite(mode) {
    const el = elRef.value
    if (!el || mode === 'noop') return
    if (stickToBottom) {
      el.scrollTop = el.scrollHeight
      return
    }
    if (mode === 'replace') {
      el.scrollTop = savedScrollTop
    }
  }

  function scheduleRestore(mode) {
    applyStickyAfterWrite(mode)
    if (rafId) cancelAnimationFrame(rafId)
    rafId = requestAnimationFrame(() => {
      rafId = 0
      applyStickyAfterWrite(mode)
    })
  }

  function syncAndSticky() {
    const el = elRef.value
    if (!el) return
    bindScroll(el)
    captureFromEl(el)
    const mode = writeContent(el).mode
    scheduleRestore(mode)
  }

  watch(
    elRef,
    (el) => {
      bindScroll(el || null)
      if (el) {
        const mode = writeContent(el).mode
        if (mode !== 'noop') scheduleRestore(mode)
      }
    },
    { immediate: true, flush: 'post' },
  )

  watch(
    contentSource,
    () => {
      const el = elRef.value
      if (!el) return
      captureFromEl(el)
      const mode = writeContent(el).mode
      scheduleRestore(mode)
    },
    { flush: 'sync' },
  )

  onScopeDispose(() => {
    if (rafId) cancelAnimationFrame(rafId)
    unbindScroll()
  })

  return {
    capture: () => captureFromEl(elRef.value),
    sync: syncAndSticky,
    isNearBottom: () => isLogScrollNearBottom(elRef.value, thresholdPx),
  }
}

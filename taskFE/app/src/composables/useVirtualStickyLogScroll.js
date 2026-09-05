import { nextTick, onScopeDispose, ref, unref, watch } from 'vue'
import {
  computeVirtualLogWindow,
  DEFAULT_LOG_LINE_HEIGHT_PX,
  DEFAULT_LOG_OVERSCAN_LINES,
  splitLogLines,
} from '../utils/incrementalLogText.js'
import { isLogScrollNearBottom, STICKY_LOG_BOTTOM_THRESHOLD_PX } from '../utils/logScrollPosition.js'

export { isLogScrollNearBottom, STICKY_LOG_BOTTOM_THRESHOLD_PX }

/**
 * 行级虚拟列表 + sticky 滚动。
 * 只把可见行写入 bodyEl，用上下 pad 撑起总高度，避免超长日志整段进 DOM。
 *
 * pad 高度同步写到 DOM style（再镜像到 ref），确保 sticky 读到的 scrollHeight 已含新 pad。
 *
 * @param {import('vue').Ref<HTMLElement | null>} viewportRef
 * @param {import('vue').Ref<HTMLElement | null>} bodyRef
 * @param {() => unknown | import('vue').Ref<unknown>} contentSource
 * @param {{
 *   thresholdPx?: number,
 *   lineHeightPx?: number,
 *   overscan?: number,
 *   topPadRef?: import('vue').Ref<HTMLElement | null>,
 *   bottomPadRef?: import('vue').Ref<HTMLElement | null>,
 * }} [options]
 */
export function useVirtualStickyLogScroll(viewportRef, bodyRef, contentSource, options = {}) {
  const thresholdPx = options.thresholdPx ?? STICKY_LOG_BOTTOM_THRESHOLD_PX
  const lineHeightPx = options.lineHeightPx ?? DEFAULT_LOG_LINE_HEIGHT_PX
  const overscan = options.overscan ?? DEFAULT_LOG_OVERSCAN_LINES
  const topPadRef = options.topPadRef ?? null
  const bottomPadRef = options.bottomPadRef ?? null

  const topPadPx = ref(0)
  const bottomPadPx = ref(0)

  /** @type {HTMLElement | null} */
  let boundViewport = null
  let stickToBottom = true
  let savedScrollTop = 0
  let rafId = 0
  /** @type {string[]} */
  let lines = []

  function readContent() {
    const src = unref(contentSource)
    return typeof src === 'function' ? src() : src
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
    const el = boundViewport || viewportRef.value
    captureFromEl(el)
    scheduleRender()
  }

  function unbindScroll() {
    if (boundViewport) {
      boundViewport.removeEventListener('scroll', onScroll)
      boundViewport = null
    }
  }

  function bindScroll(el) {
    if (boundViewport === el) return
    unbindScroll()
    if (!el) return
    boundViewport = el
    el.addEventListener('scroll', onScroll, { passive: true })
    captureFromEl(el)
  }

  function applyPadHeights(topPx, bottomPx) {
    topPadPx.value = topPx
    bottomPadPx.value = bottomPx
    const topEl = topPadRef ? unref(topPadRef) : null
    const bottomEl = bottomPadRef ? unref(bottomPadRef) : null
    if (topEl?.style) topEl.style.height = `${topPx}px`
    if (bottomEl?.style) bottomEl.style.height = `${bottomPx}px`
  }

  function renderWindow() {
    const viewport = viewportRef.value
    const body = bodyRef.value
    if (!viewport || !body) return

    bindScroll(viewport)

    const win = computeVirtualLogWindow(lines, {
      scrollTop: viewport.scrollTop,
      clientHeight: viewport.clientHeight || 1,
      lineHeight: lineHeightPx,
      overscan,
    })
    applyPadHeights(win.topPadPx, win.bottomPadPx)
    if (body.textContent !== win.visibleText) {
      body.textContent = win.visibleText
    }
  }

  function applySticky() {
    const viewport = viewportRef.value
    if (!viewport) return
    if (stickToBottom) {
      viewport.scrollTop = viewport.scrollHeight
      return
    }
    viewport.scrollTop = savedScrollTop
  }

  function scheduleRender() {
    renderWindow()
    if (rafId) cancelAnimationFrame(rafId)
    rafId = requestAnimationFrame(() => {
      rafId = 0
      renderWindow()
      applySticky()
    })
  }

  function syncFromContent() {
    const viewport = viewportRef.value
    if (viewport) {
      const prevHeight = Math.max(lines.length * lineHeightPx, viewport.clientHeight)
      stickToBottom = isLogScrollNearBottom(
        {
          scrollHeight: prevHeight,
          clientHeight: viewport.clientHeight,
          scrollTop: viewport.scrollTop,
        },
        thresholdPx,
      )
      savedScrollTop = viewport.scrollTop
    }
    lines = splitLogLines(readContent())
    renderWindow()
    applySticky()
    nextTick(() => {
      renderWindow()
      applySticky()
      scheduleRender()
    })
  }

  watch(
    viewportRef,
    (el) => {
      bindScroll(el || null)
      if (el) syncFromContent()
    },
    { immediate: true, flush: 'post' },
  )

  watch(
    bodyRef,
    (el) => {
      if (el && viewportRef.value) syncFromContent()
    },
    { flush: 'post' },
  )

  watch(
    contentSource,
    () => {
      syncFromContent()
    },
    { flush: 'sync' },
  )

  onScopeDispose(() => {
    if (rafId) cancelAnimationFrame(rafId)
    unbindScroll()
  })

  return {
    topPadPx,
    bottomPadPx,
    capture: () => captureFromEl(viewportRef.value),
    sync: syncFromContent,
    isNearBottom: () => isLogScrollNearBottom(viewportRef.value, thresholdPx),
  }
}

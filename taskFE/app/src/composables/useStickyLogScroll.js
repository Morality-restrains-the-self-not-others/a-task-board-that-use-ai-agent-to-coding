import { nextTick, onScopeDispose, watch } from 'vue'
import { isLogScrollNearBottom, STICKY_LOG_BOTTOM_THRESHOLD_PX } from '../utils/logScrollPosition.js'

export { isLogScrollNearBottom, STICKY_LOG_BOTTOM_THRESHOLD_PX }

/**
 * 在内容更新前后保留滚动位置：贴底则跟随最新；非贴底则恢复原 scrollTop。
 * 通过 scroll 事件持续记录用户意图，避免文本替换把 scrollTop 打回 0 后无法恢复。
 *
 * 恢复必须在 nextTick（DOM 已更新、尚未 paint）内同步完成；若再推迟到 rAF，
 * 浏览器会先画出一帧 scrollTop=0，高频 SSE 追加时表现为滚动窗口持续抖动。
 *
 * @param {import('vue').Ref<HTMLElement | null>} elRef
 * @param {() => unknown | import('vue').Ref<unknown> | import('vue').ComputedRef<unknown>} contentSource
 * @param {{ thresholdPx?: number }} [options]
 */
export function useStickyLogScroll(elRef, contentSource, options = {}) {
  const thresholdPx = options.thresholdPx ?? STICKY_LOG_BOTTOM_THRESHOLD_PX
  /** @type {HTMLElement | null} */
  let boundEl = null
  let savedScrollTop = 0
  let stickToBottom = true
  let rafId = 0

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

  function restore() {
    const el = elRef.value
    if (!el) return
    bindScroll(el)
    if (stickToBottom) {
      el.scrollTop = el.scrollHeight
    } else {
      el.scrollTop = savedScrollTop
    }
  }

  /**
   * nextTick 内立即恢复（赶在 paint 前）；再补一次 rAF 以覆盖布局稍后结算的场景。
   * 禁止「仅 rAF」——那会让错误的 scrollTop=0 被画出来。
   */
  function scheduleRestore() {
    restore()
    if (rafId) cancelAnimationFrame(rafId)
    rafId = requestAnimationFrame(() => {
      rafId = 0
      restore()
    })
  }

  watch(
    elRef,
    (el) => {
      bindScroll(el || null)
    },
    { immediate: true, flush: 'post' },
  )

  watch(
    contentSource,
    () => {
      const el = elRef.value
      if (el) captureFromEl(el)
      nextTick(() => scheduleRestore())
    },
    { flush: 'sync' },
  )

  onScopeDispose(() => {
    if (rafId) cancelAnimationFrame(rafId)
    unbindScroll()
  })

  return {
    capture: () => captureFromEl(elRef.value),
    restore,
    isNearBottom: () => isLogScrollNearBottom(elRef.value, thresholdPx),
  }
}

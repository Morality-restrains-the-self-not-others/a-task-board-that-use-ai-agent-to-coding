// @vitest-environment node
import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest'
import { ref, nextTick, effectScope } from 'vue'
import {
  isLogScrollNearBottom,
  STICKY_LOG_BOTTOM_THRESHOLD_PX,
  useStickyLogScroll,
} from './useStickyLogScroll.js'

describe('isLogScrollNearBottom', () => {
  it('无元素时视为贴底', () => {
    expect(isLogScrollNearBottom(null)).toBe(true)
  })

  it('内容未溢出时视为贴底', () => {
    const el = { scrollHeight: 100, clientHeight: 100, scrollTop: 0 }
    expect(isLogScrollNearBottom(el)).toBe(true)
  })

  it('距底部小于阈值时贴底', () => {
    const el = {
      scrollHeight: 500,
      clientHeight: 100,
      scrollTop: 500 - 100 - (STICKY_LOG_BOTTOM_THRESHOLD_PX - 1),
    }
    expect(isLogScrollNearBottom(el)).toBe(true)
  })

  it('用户上滚离开底部时非贴底', () => {
    const el = { scrollHeight: 500, clientHeight: 100, scrollTop: 20 }
    expect(isLogScrollNearBottom(el)).toBe(false)
  })
})

describe('useStickyLogScroll', () => {
  let rafQueue
  /** @type {import('vue').EffectScope | null} */
  let scope = null

  beforeEach(() => {
    rafQueue = []
    vi.stubGlobal('requestAnimationFrame', (cb) => {
      rafQueue.push(cb)
      return rafQueue.length
    })
    vi.stubGlobal('cancelAnimationFrame', (id) => {
      rafQueue[id - 1] = null
    })
  })

  afterEach(() => {
    if (scope) {
      scope.stop()
      scope = null
    }
    vi.unstubAllGlobals()
  })

  function flushRaf() {
    const q = [...rafQueue]
    rafQueue.length = 0
    for (const cb of q) {
      if (cb) cb(0)
    }
  }

  function setupSticky(initialText) {
    const content = ref(initialText)
    const el = {
      scrollHeight: 1000,
      clientHeight: 200,
      scrollTop: 0,
      listeners: {},
      addEventListener(type, fn) {
        this.listeners[type] = fn
      },
      removeEventListener(type) {
        delete this.listeners[type]
      },
    }
    const elRef = ref(null)
    scope = effectScope()
    scope.run(() => {
      useStickyLogScroll(elRef, content)
    })
    elRef.value = el
    return { content, el, elRef }
  }

  it('非贴底时内容更新后在 nextTick 内恢复原 scrollTop（不等待 rAF）', async () => {
    const { content, el } = setupSticky('line1\n')
    await nextTick()
    el.scrollTop = 120
    el.listeners.scroll?.()

    content.value = 'line1\nline2\nline3\n'
    el.scrollTop = 0
    el.scrollHeight = 1200
    await nextTick()

    expect(el.scrollTop).toBe(120)
    flushRaf()
    expect(el.scrollTop).toBe(120)
  })

  it('贴底时内容更新后在 nextTick 内跟随到底部', async () => {
    const { content, el } = setupSticky('a')
    await nextTick()
    el.scrollTop = 1000 - 200
    el.listeners.scroll?.()

    content.value = 'a\nb\nc\n'
    el.scrollHeight = 1400
    el.scrollTop = 0
    await nextTick()

    expect(el.scrollTop).toBe(1400)
    flushRaf()
    expect(el.scrollTop).toBe(1400)
  })
})

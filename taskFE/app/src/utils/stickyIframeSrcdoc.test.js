// @vitest-environment node
import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest'
import {
  iframeLogScrollingEl,
  iframeLogNearBottom,
  iframeLogScrollToBottom,
  setIframeSrcdocSticky,
} from './stickyIframeSrcdoc.js'
import { STICKY_LOG_BOTTOM_THRESHOLD_PX } from './logScrollPosition.js'

describe('iframeLogScrollingEl / nearBottom', () => {
  it('优先使用 scrollingElement', () => {
    const scrollingElement = { scrollTop: 1 }
    const doc = { scrollingElement, documentElement: {}, body: {} }
    expect(iframeLogScrollingEl(doc)).toBe(scrollingElement)
  })

  it('空文档视为贴底', () => {
    expect(iframeLogNearBottom(null)).toBe(true)
  })

  it('距底部小于阈值时贴底', () => {
    const el = {
      scrollHeight: 500,
      clientHeight: 100,
      scrollTop: 500 - 100 - (STICKY_LOG_BOTTOM_THRESHOLD_PX - 1),
    }
    const doc = { scrollingElement: el }
    expect(iframeLogNearBottom(doc)).toBe(true)
  })

  it('上滚后非贴底', () => {
    const el = { scrollHeight: 500, clientHeight: 100, scrollTop: 10 }
    expect(iframeLogNearBottom({ scrollingElement: el })).toBe(false)
  })

  it('scrollToBottom 写到 scrollHeight', () => {
    const el = { scrollHeight: 900, clientHeight: 100, scrollTop: 0 }
    iframeLogScrollToBottom({ scrollingElement: el })
    expect(el.scrollTop).toBe(900)
  })
})

describe('setIframeSrcdocSticky', () => {
  let rafQueue

  beforeEach(() => {
    rafQueue = []
    vi.stubGlobal('requestAnimationFrame', (cb) => {
      rafQueue.push(cb)
      return rafQueue.length
    })
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  function flushRaf() {
    const q = [...rafQueue]
    rafQueue.length = 0
    for (const cb of q) {
      if (cb) cb(0)
    }
  }

  function makeFrame(scrollTop, scrollHeight = 1000, clientHeight = 200) {
    const el = { scrollTop, scrollHeight, clientHeight }
    const doc = { scrollingElement: el, documentElement: el, body: el }
    /** @type {any} */
    const fr = {
      contentDocument: doc,
      srcdoc: '',
      listeners: {},
      addEventListener(type, fn) {
        this.listeners[type] = fn
      },
      removeEventListener(type) {
        delete this.listeners[type]
      },
    }
    return { fr, el }
  }

  it('非贴底时 srcdoc 更新后恢复 scrollTop', () => {
    const { fr, el } = makeFrame(120)
    setIframeSrcdocSticky(fr, '<html><body>new</body></html>')
    expect(fr.srcdoc).toContain('new')
    // 模拟浏览器重置
    el.scrollTop = 0
    el.scrollHeight = 1200
    fr.listeners.load?.()
    flushRaf()
    flushRaf()
    expect(el.scrollTop).toBe(120)
  })

  it('贴底时 srcdoc 更新后滚到底', () => {
    const { fr, el } = makeFrame(800)
    setIframeSrcdocSticky(fr, '<html><body>more</body></html>')
    el.scrollHeight = 1400
    el.scrollTop = 0
    fr.listeners.load?.()
    flushRaf()
    flushRaf()
    expect(el.scrollTop).toBe(1400)
  })
})

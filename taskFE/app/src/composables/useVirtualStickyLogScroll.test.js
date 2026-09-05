// @vitest-environment node
import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest'
import { ref, nextTick, effectScope } from 'vue'
import { useVirtualStickyLogScroll } from './useVirtualStickyLogScroll.js'

describe('useVirtualStickyLogScroll', () => {
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

  function setup(initialText, { clientHeight = 90, lineHeight = 10 } = {}) {
    const content = ref(initialText)
    const viewport = {
      clientHeight,
      scrollTop: 0,
      listeners: {},
      addEventListener(type, fn) {
        this.listeners[type] = fn
      },
      removeEventListener(type) {
        delete this.listeners[type]
      },
    }
    Object.defineProperty(viewport, 'scrollHeight', {
      configurable: true,
      get() {
        const lines = String(content.value || '').split('\n')
        return lines.length * lineHeight
      },
    })
    const body = { textContent: '' }
    const topPad = { style: { height: '0px' } }
    const bottomPad = { style: { height: '0px' } }
    const viewportRef = ref(null)
    const bodyRef = ref(null)
    const topPadRef = ref(topPad)
    const bottomPadRef = ref(bottomPad)
    let api
    scope = effectScope()
    scope.run(() => {
      api = useVirtualStickyLogScroll(viewportRef, bodyRef, content, {
        lineHeightPx: lineHeight,
        overscan: 2,
        topPadRef,
        bottomPadRef,
      })
    })
    viewportRef.value = viewport
    bodyRef.value = body
    return { content, viewport, body, api, viewportRef, bodyRef }
  }

  it('只把可见行写入 body，总高度由 pad 表达', async () => {
    const lines = Array.from({ length: 100 }, (_, i) => `L${i}`).join('\n')
    const { body, api, viewport } = setup(lines, { clientHeight: 50, lineHeight: 10 })
    await nextTick()
    flushRaf()
    expect(body.textContent.split('\n').length).toBeLessThan(40)
    expect(api.topPadPx.value + api.bottomPadPx.value).toBeGreaterThan(0)
    expect(viewport.scrollHeight).toBe(1000)
  })

  it('贴底时追加后 scrollTop 跟随到底', async () => {
    const { content, viewport } = setup('L0\nL1\nL2', { clientHeight: 20, lineHeight: 10 })
    await nextTick()
    flushRaf()
    viewport.scrollTop = viewport.scrollHeight - viewport.clientHeight
    viewport.listeners.scroll?.()

    content.value = Array.from({ length: 50 }, (_, i) => `L${i}`).join('\n')
    await nextTick()
    flushRaf()
    expect(viewport.scrollTop).toBe(viewport.scrollHeight)
  })

  it('非贴底时内容追加后恢复 scrollTop', async () => {
    const many = Array.from({ length: 80 }, (_, i) => `L${i}`).join('\n')
    const { content, viewport } = setup(many, { clientHeight: 40, lineHeight: 10 })
    await nextTick()
    flushRaf()
    viewport.scrollTop = 100
    viewport.listeners.scroll?.()

    content.value = many + '\nL80'
    viewport.scrollTop = 0
    await nextTick()
    flushRaf()
    expect(viewport.scrollTop).toBe(100)
  })
})

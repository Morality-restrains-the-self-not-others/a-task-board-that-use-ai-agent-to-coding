// @vitest-environment node
import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest'
import { ref, nextTick, effectScope } from 'vue'
import { useIncrementalStickyLogScroll } from './useIncrementalStickyLogScroll.js'

describe('useIncrementalStickyLogScroll', () => {
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

  function setup(initialText) {
    const content = ref(initialText)
    const textNodes = []
    const el = {
      scrollHeight: 1000,
      clientHeight: 200,
      scrollTop: 0,
      _text: initialText,
      childNodes: textNodes,
      firstChild: null,
      listeners: {},
      get textContent() {
        return this._text
      },
      set textContent(v) {
        this._text = String(v ?? '')
        textNodes.length = 0
        this.firstChild = null
      },
      appendChild(node) {
        const t = node && node.nodeValue != null ? String(node.nodeValue) : ''
        this._text += t
        const n = { nodeValue: t }
        textNodes.push(n)
        if (!this.firstChild) this.firstChild = n
        return n
      },
      addEventListener(type, fn) {
        this.listeners[type] = fn
      },
      removeEventListener(type) {
        delete this.listeners[type]
      },
    }
    // seed as if already rendered via replace
    el.textContent = initialText
    const elRef = ref(null)
    scope = effectScope()
    scope.run(() => {
      useIncrementalStickyLogScroll(elRef, content)
    })
    elRef.value = el
    return { content, el, elRef }
  }

  it('前缀扩展时 append，非贴底保持 scrollTop（不因写入被清零）', async () => {
    const { content, el } = setup('line1\n')
    await nextTick()
    el.scrollTop = 120
    el.listeners.scroll?.()

    content.value = 'line1\nline2\n'
    // 增量 append 不应把 scrollTop 打回 0；即使误清零，replace 路径才会强制恢复
    expect(el.textContent).toBe('line1\nline2\n')
    expect(el.scrollTop).toBe(120)
    flushRaf()
    expect(el.scrollTop).toBe(120)
  })

  it('贴底时 append 后跟随到底部', async () => {
    const { content, el } = setup('a')
    await nextTick()
    el.scrollTop = 1000 - 200
    el.listeners.scroll?.()

    content.value = 'a\nb\nc\n'
    expect(el.scrollTop).toBe(1000)
    el.scrollHeight = 1400
    flushRaf()
    expect(el.scrollTop).toBe(1400)
  })

  it('非前缀替换后恢复非贴底 scrollTop', async () => {
    const { content, el } = setup('abcdefghij')
    await nextTick()
    el.scrollTop = 80
    el.listeners.scroll?.()

    content.value = 'fghij'
    // replace 会清零，sticky 应恢复
    expect(el.textContent).toBe('fghij')
    expect(el.scrollTop).toBe(80)
    flushRaf()
    expect(el.scrollTop).toBe(80)
  })
})

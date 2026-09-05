// @vitest-environment jsdom
import { describe, expect, it } from 'vitest'
import {
  syncLogTextContent,
  maybeNormalizeLogTextNodes,
  windowLogTextToTail,
  splitLogLines,
  computeVirtualLogWindow,
  DEFAULT_LOG_MAX_TEXT_NODES,
} from './incrementalLogText.js'

describe('syncLogTextContent', () => {
  function makeEl(text = '') {
    const el = document.createElement('pre')
    el.textContent = text
    return el
  }

  it('无元素时 noop', () => {
    expect(syncLogTextContent(null, 'a')).toEqual({ mode: 'noop' })
  })

  it('内容相同则 noop', () => {
    const el = makeEl('hello')
    expect(syncLogTextContent(el, 'hello')).toEqual({ mode: 'noop', windowed: false })
    expect(el.textContent).toBe('hello')
  })

  it('前缀扩展时仅 append delta，不整段替换', () => {
    const el = makeEl('line1\n')
    const beforeNode = el.firstChild
    const result = syncLogTextContent(el, 'line1\nline2\n')
    expect(result.mode).toBe('append')
    expect(result.delta).toBe('line2\n')
    expect(el.textContent).toBe('line1\nline2\n')
    expect(el.firstChild).toBe(beforeNode)
    expect(el.childNodes.length).toBeGreaterThan(1)
  })

  it('空→有内容走 replace', () => {
    const el = makeEl('')
    expect(syncLogTextContent(el, 'abc').mode).toBe('replace')
    expect(el.textContent).toBe('abc')
  })

  it('非前缀扩展（截断/重置）走 replace', () => {
    const el = makeEl('abcdefghij')
    const result = syncLogTextContent(el, 'fghij')
    expect(result.mode).toBe('replace')
    expect(el.textContent).toBe('fghij')
  })

  it('清空走 replace', () => {
    const el = makeEl('abc')
    expect(syncLogTextContent(el, '').mode).toBe('replace')
    expect(el.textContent).toBe('')
  })

  it('null/undefined 视为空串', () => {
    const el = makeEl('x')
    expect(syncLogTextContent(el, null).mode).toBe('replace')
    expect(el.textContent).toBe('')
  })

  it('maxDomChars 超限时写入尾部窗口', () => {
    const el = makeEl('')
    const full = 'AAAA\n' + 'B'.repeat(100)
    const result = syncLogTextContent(el, full, { maxDomChars: 20 })
    expect(result.windowed).toBe(true)
    expect(el.textContent.length).toBeLessThanOrEqual(20)
    expect(el.textContent.endsWith('B'.repeat(Math.min(20, 100))) || el.textContent.includes('B')).toBe(
      true,
    )
  })
})

describe('maybeNormalizeLogTextNodes', () => {
  it('节点数未超阈值不 normalize', () => {
    const el = document.createElement('pre')
    el.appendChild(document.createTextNode('a'))
    el.appendChild(document.createTextNode('b'))
    expect(maybeNormalizeLogTextNodes(el, 10)).toBe(false)
    expect(el.childNodes.length).toBe(2)
  })

  it('节点数超阈值时合并为单个文本节点', () => {
    const el = document.createElement('pre')
    for (let i = 0; i < DEFAULT_LOG_MAX_TEXT_NODES; i++) {
      el.appendChild(document.createTextNode(`x${i}`))
    }
    expect(el.childNodes.length).toBe(DEFAULT_LOG_MAX_TEXT_NODES)
    expect(maybeNormalizeLogTextNodes(el, DEFAULT_LOG_MAX_TEXT_NODES)).toBe(true)
    expect(el.childNodes.length).toBe(1)
    expect(el.textContent.startsWith('x0')).toBe(true)
  })

  it('append 路径在超阈值时自动 normalize', () => {
    const el = document.createElement('pre')
    el.textContent = 'base'
    let text = 'base'
    for (let i = 0; i < DEFAULT_LOG_MAX_TEXT_NODES; i++) {
      text += `+${i}`
      syncLogTextContent(el, text, { maxTextNodes: DEFAULT_LOG_MAX_TEXT_NODES })
    }
    expect(el.childNodes.length).toBeLessThan(DEFAULT_LOG_MAX_TEXT_NODES)
    expect(el.textContent).toBe(text)
  })
})

describe('windowLogTextToTail', () => {
  it('未超限不截断', () => {
    expect(windowLogTextToTail('abc', 10)).toEqual({ text: 'abc', truncated: false })
  })

  it('超限优先在换行处切开', () => {
    const s = 'keep-head\n' + 'y'.repeat(50)
    const { text, truncated } = windowLogTextToTail(s, 20)
    expect(truncated).toBe(true)
    expect(text.length).toBeLessThanOrEqual(20)
    expect(text.includes('\n') || text.startsWith('y')).toBe(true)
  })
})

describe('computeVirtualLogWindow', () => {
  it('空行', () => {
    const w = computeVirtualLogWindow([], { scrollTop: 0, clientHeight: 100, lineHeight: 10 })
    expect(w.visibleText).toBe('')
    expect(w.totalHeightPx).toBe(0)
  })

  it('只渲染可见窗口 + overscan，并给出上下垫高', () => {
    const lines = Array.from({ length: 100 }, (_, i) => `L${i}`)
    const w = computeVirtualLogWindow(lines, {
      scrollTop: 200,
      clientHeight: 100,
      lineHeight: 10,
      overscan: 2,
    })
    // floor(200/10)-2 = 18；可见 ceil(100/10)+4 = 14 → end 32
    expect(w.start).toBe(18)
    expect(w.end).toBe(32)
    expect(w.topPadPx).toBe(180)
    expect(w.bottomPadPx).toBe((100 - 32) * 10)
    expect(w.visibleText.startsWith('L18')).toBe(true)
    expect(w.visibleText.endsWith('L31')).toBe(true)
    expect(w.totalHeightPx).toBe(1000)
  })

  it('splitLogLines 按换行拆分', () => {
    expect(splitLogLines('a\nb\n')).toEqual(['a', 'b', ''])
  })
})

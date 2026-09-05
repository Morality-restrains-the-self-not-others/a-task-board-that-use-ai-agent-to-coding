/**
 * 超长日志：行级虚拟窗口计算 + DOM 文本节点合并 / 字符窗口裁剪。
 */

/** append 累积的文本节点超过该数时 normalize 合并 */
export const DEFAULT_LOG_MAX_TEXT_NODES = 48

/** 单次写入 DOM 的最大字符（非虚拟路径兜底） */
export const DEFAULT_LOG_MAX_DOM_CHARS = 80_000

/** 虚拟列表默认行高（text-xs + leading-normal 近似） */
export const DEFAULT_LOG_LINE_HEIGHT_PX = 18

/** 视口上下额外多渲染的行数 */
export const DEFAULT_LOG_OVERSCAN_LINES = 16

/**
 * 将全文拆成行（保留末尾空行语义：split 后若以 \n 结尾会多一个 ''）。
 * @param {unknown} text
 * @returns {string[]}
 */
export function splitLogLines(text) {
  const s = text == null ? '' : String(text)
  if (!s) return []
  return s.split('\n')
}

/**
 * @param {string} text
 * @param {number} maxChars
 * @returns {{ text: string, truncated: boolean }}
 */
export function windowLogTextToTail(text, maxChars) {
  const s = text == null ? '' : String(text)
  if (!maxChars || maxChars <= 0 || s.length <= maxChars) {
    return { text: s, truncated: false }
  }
  let start = s.length - maxChars
  const nl = s.indexOf('\n', start)
  if (nl !== -1 && nl < s.length - 1) start = nl + 1
  return { text: s.slice(start), truncated: true }
}

/**
 * 计算虚拟列表可见窗口。
 *
 * @param {string[]} lines
 * @param {{
 *   scrollTop: number,
 *   clientHeight: number,
 *   lineHeight?: number,
 *   overscan?: number,
 * }} metrics
 * @returns {{
 *   start: number,
 *   end: number,
 *   topPadPx: number,
 *   bottomPadPx: number,
 *   visibleText: string,
 *   totalHeightPx: number,
 * }}
 */
export function computeVirtualLogWindow(lines, metrics) {
  const list = Array.isArray(lines) ? lines : []
  const total = list.length
  const lineHeight = metrics.lineHeight ?? DEFAULT_LOG_LINE_HEIGHT_PX
  const overscan = metrics.overscan ?? DEFAULT_LOG_OVERSCAN_LINES
  const clientHeight = Math.max(0, Number(metrics.clientHeight) || 0)
  const scrollTop = Math.max(0, Number(metrics.scrollTop) || 0)
  const totalHeightPx = total * lineHeight

  if (total === 0) {
    return {
      start: 0,
      end: 0,
      topPadPx: 0,
      bottomPadPx: 0,
      visibleText: '',
      totalHeightPx: 0,
    }
  }

  const visibleCount = Math.max(1, Math.ceil(clientHeight / lineHeight) + overscan * 2)
  let start = Math.max(0, Math.floor(scrollTop / lineHeight) - overscan)
  let end = Math.min(total, start + visibleCount)
  if (end - start < visibleCount && start > 0) {
    start = Math.max(0, end - visibleCount)
  }

  return {
    start,
    end,
    topPadPx: start * lineHeight,
    bottomPadPx: Math.max(0, (total - end) * lineHeight),
    visibleText: list.slice(start, end).join('\n'),
    totalHeightPx,
  }
}

/**
 * 合并相邻文本节点；节点数未超阈值则 noop。
 * @param {Element | null | undefined} el
 * @param {number} [maxNodes]
 * @returns {boolean} 是否执行了合并
 */
export function maybeNormalizeLogTextNodes(el, maxNodes = DEFAULT_LOG_MAX_TEXT_NODES) {
  if (!el || !el.childNodes) return false
  if (el.childNodes.length < maxNodes) return false
  if (typeof el.normalize === 'function') {
    el.normalize()
    return true
  }
  const t = el.textContent ?? ''
  el.textContent = t
  return true
}

/**
 * 将日志字符串同步到 DOM：前缀扩展则 append；否则 replace。
 * 可选：append 后 normalize；超长则裁成尾部窗口（replace）。
 *
 * @param {Element | null | undefined} el
 * @param {unknown} nextText
 * @param {{
 *   maxTextNodes?: number,
 *   maxDomChars?: number,
 * }} [options]
 * @returns {{ mode: 'noop' | 'append' | 'replace', delta?: string, normalized?: boolean, windowed?: boolean }}
 */
export function syncLogTextContent(el, nextText, options = {}) {
  if (!el) return { mode: 'noop' }
  const maxDomChars = options.maxDomChars ?? 0
  let next = nextText == null ? '' : String(nextText)
  let windowed = false
  if (maxDomChars > 0) {
    const w = windowLogTextToTail(next, maxDomChars)
    next = w.text
    windowed = w.truncated
  }
  const current = el.textContent ?? ''
  if (next === current) return { mode: 'noop', windowed }

  if (current.length > 0 && next.length > current.length && next.startsWith(current)) {
    const delta = next.slice(current.length)
    appendTextDelta(el, delta)
    const normalized = maybeNormalizeLogTextNodes(el, options.maxTextNodes)
    return { mode: 'append', delta, normalized, windowed }
  }
  el.textContent = next
  return { mode: 'replace', windowed }
}

/**
 * @param {{ appendChild: (n: unknown) => unknown, ownerDocument?: Document | null }} el
 * @param {string} delta
 */
function appendTextDelta(el, delta) {
  const doc = el.ownerDocument
  if (doc && typeof doc.createTextNode === 'function') {
    el.appendChild(doc.createTextNode(delta))
    return
  }
  if (typeof document !== 'undefined' && typeof document.createTextNode === 'function') {
    el.appendChild(document.createTextNode(delta))
    return
  }
  el.appendChild({ nodeType: 3, nodeValue: delta })
}

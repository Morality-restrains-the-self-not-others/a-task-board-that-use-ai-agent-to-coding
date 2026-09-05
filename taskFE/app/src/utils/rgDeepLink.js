/**
 * 访问管理「打开页面 / 定位」深链：URL 片段 #rg=<group_key> 对应页面 data-rg-key 区块。
 *
 * 页面加载或 hashchange 时解析片段 → 定位目标区块（scrollIntoView）并短暂高亮
 * （.rg-deep-link-highlight），便于用户在长设置页快速找到目标区域。
 *
 * 注：router 为 HTML5 history 模式，#rg= 片段由浏览器保留而非 vue-router 处理，
 * 因此需自行监听 hashchange，并在路由切换后重扫（页面组件异步挂载）。
 */
const HIGHLIGHT_CLASS = 'rg-deep-link-highlight'
const HIGHLIGHT_DURATION_MS = 1500
const RETRY_DELAY_MS = [150, 250, 400]

const timers = new Map()

function prefersReducedMotion() {
  if (typeof window !== 'undefined' && typeof window.matchMedia === 'function') {
    return window.matchMedia('(prefers-reduced-motion: reduce)').matches
  }
  return false
}

/**
 * 从 URL hash 中解析 rg 参数（支持 #rg=key 与 #foo=1&rg=key 两种形态）。
 * @param {string} hash
 * @returns {string}
 */
export function parseRgKey(hash) {
  const raw = String(hash || '').replace(/^#/, '')
  const m = raw.match(/(?:^|&)rg=([^&#]+)/)
  if (!m) return ''
  try {
    return decodeURIComponent(m[1])
  } catch {
    return m[1]
  }
}

/**
 * 在文档中查找 data-rg-key === groupKey 的元素。
 * 逐元素比对属性值，避免 querySelector 属性转义在不同环境下的兼容问题。
 * @param {string} groupKey
 * @param {Document} [doc]
 * @returns {Element|null}
 */
export function findRgTarget(groupKey, doc) {
  const root = doc || (typeof document !== 'undefined' ? document : null)
  if (!groupKey || !root?.querySelectorAll) return null
  let found = null
  root.querySelectorAll('[data-rg-key]').forEach((el) => {
    if (!found && el.getAttribute('data-rg-key') === groupKey) found = el
  })
  return found
}

/**
 * 定位目标区块并短暂高亮。
 * @param {string} groupKey
 * @param {{ doc?: Document, scrollOptions?: ScrollIntoViewOptions }} [opts]
 * @returns {boolean} 是否找到目标元素
 */
export function scrollToRgKey(groupKey, opts) {
  const doc = opts?.doc || (typeof document !== 'undefined' ? document : null)
  const el = findRgTarget(groupKey, doc)
  if (!el) return false
  const reduced = prefersReducedMotion()
  el.scrollIntoView({
    behavior: reduced ? 'auto' : 'smooth',
    block: 'start',
    ...(opts?.scrollOptions || {}),
  })
  el.classList.add(HIGHLIGHT_CLASS)
  const prev = timers.get(el)
  if (prev) clearTimeout(prev)
  timers.set(
    el,
    setTimeout(() => el.classList.remove(HIGHLIGHT_CLASS), HIGHLIGHT_DURATION_MS),
  )
  return true
}

/**
 * 处理当前 URL 的 #rg= 深链：立即尝试定位，未命中则短暂重试（等待异步页面挂载）。
 * @returns {boolean}
 */
export function handleRgDeepLink() {
  const key = parseRgKey(typeof window !== 'undefined' ? window.location.hash : '')
  if (!key) return false
  if (scrollToRgKey(key)) return true
  let done = false
  for (const delay of RETRY_DELAY_MS) {
    setTimeout(() => {
      if (!done && scrollToRgKey(key)) done = true
    }, delay)
  }
  return false
}

/**
 * 初始化深链处理：挂载时执行一次 + 监听 hashchange。
 * 返回清理函数（测试与热替换使用）。
 * @returns {() => void}
 */
export function initRgDeepLink() {
  if (typeof window === 'undefined') return () => {}
  handleRgDeepLink()
  const onHash = () => handleRgDeepLink()
  window.addEventListener('hashchange', onHash)
  return () => {
    window.removeEventListener('hashchange', onHash)
    timers.forEach((t) => clearTimeout(t))
    timers.clear()
  }
}

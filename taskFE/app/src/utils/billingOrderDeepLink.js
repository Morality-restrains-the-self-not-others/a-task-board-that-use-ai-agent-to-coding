/**
 * 订单列表深链：?order_id= 定位到具体订单行（scrollIntoView + 短暂高亮）。
 * 与 rgDeepLink 同类交互，供 BillingOrders / SystemAdminOrderRecords 复用。
 */

export const BILLING_ORDER_HIGHLIGHT_CLASS = 'rg-deep-link-highlight'
const HIGHLIGHT_DURATION_MS = 1500

/** 超管「订单与退款」页路径（含尾斜杠）。 */
export const SYSTEM_ADMIN_ORDER_RECORDS_PATH = '/system-admin/order-records/'

/**
 * @param {Record<string, unknown>|null|undefined} query
 * @returns {string}
 */
export function parseOrderIdQuery(query) {
  return String(query?.order_id ?? '').trim()
}

/**
 * @param {Record<string, unknown>|null|undefined} query
 * @returns {string}
 */
export function parseTenantIdQuery(query) {
  return String(query?.tenant_id ?? '').trim()
}

/**
 * 超管退款审批「关联订单」→ 管理端订单 Tab 深链（非租户侧栏）。
 * @param {{ orderId?: unknown, tenantId?: unknown }} opts
 * @returns {string}
 */
export function buildSystemAdminOrderRecordsHref(opts = {}) {
  const orderId = String(opts.orderId ?? '').trim()
  const tenantId = String(opts.tenantId ?? '').trim()
  const q = new URLSearchParams()
  if (tenantId) q.set('tenant_id', tenantId)
  if (orderId) q.set('order_id', orderId)
  const qs = q.toString()
  return qs ? `${SYSTEM_ADMIN_ORDER_RECORDS_PATH}?${qs}` : SYSTEM_ADMIN_ORDER_RECORDS_PATH
}

/**
 * 在订单列表中按 id（或 order_number）匹配目标订单。
 * @param {Array<{id?: unknown, order_number?: unknown}>|null|undefined} orders
 * @param {string} orderId
 * @returns {object|null}
 */
export function findOrderInList(orders, orderId) {
  const id = String(orderId || '').trim()
  if (!id || !Array.isArray(orders)) return null
  return (
    orders.find(
      (o) => String(o?.id ?? '') === id || String(o?.order_number ?? '') === id,
    ) || null
  )
}

/**
 * 由列表 API 的 offset + pageSize 推导 1-based 页码。
 * @param {number} offset
 * @param {number} pageSize
 * @returns {number}
 */
export function pageFromListOffset(offset, pageSize) {
  const ps = Math.max(1, Number(pageSize) || 1)
  const off = Math.max(0, Number(offset) || 0)
  return Math.floor(off / ps) + 1
}

function prefersReducedMotion() {
  if (typeof window !== 'undefined' && typeof window.matchMedia === 'function') {
    return window.matchMedia('(prefers-reduced-motion: reduce)').matches
  }
  return false
}

/**
 * 在文档中查找 data-order-id === orderId 的行元素。
 * @param {string} orderId
 * @param {ParentNode|Document|null} [root]
 * @returns {Element|null}
 */
export function findOrderRowEl(orderId, root) {
  const id = String(orderId || '').trim()
  const doc = root || (typeof document !== 'undefined' ? document : null)
  if (!id || !doc?.querySelectorAll) return null
  let found = null
  doc.querySelectorAll('[data-order-id]').forEach((el) => {
    if (!found && el.getAttribute('data-order-id') === id) found = el
  })
  return found
}

/**
 * 滚动到订单行并短暂高亮。
 * @param {string} orderId
 * @param {{ root?: ParentNode|Document, highlightClass?: string, durationMs?: number }} [opts]
 * @returns {boolean}
 */
export function scrollAndHighlightOrderRow(orderId, opts = {}) {
  const el = findOrderRowEl(orderId, opts.root)
  if (!el) return false
  const reduced = prefersReducedMotion()
  if (typeof el.scrollIntoView === 'function') {
    el.scrollIntoView({ behavior: reduced ? 'auto' : 'smooth', block: 'center' })
  }
  const cls = opts.highlightClass || BILLING_ORDER_HIGHLIGHT_CLASS
  el.classList.add(cls)
  const duration = opts.durationMs ?? HIGHLIGHT_DURATION_MS
  if (typeof window !== 'undefined' && duration > 0) {
    window.setTimeout(() => el.classList.remove(cls), duration)
  }
  return true
}
